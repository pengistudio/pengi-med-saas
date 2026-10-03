package backoffice_handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"pengi-med-saas/core/envelope"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func planRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &company_models.Feature{}, &company_models.Plan{}, &company_models.PlanPricing{})
	h := NewBackofficePlanHandler(db, zap.NewNop())
	router := gin.New()
	router.POST("/backoffice/plans", envelope.Handle(h.CreatePlan))
	router.PUT("/backoffice/plans/:id", envelope.Handle(h.UpdatePlan))
	return router, db
}

func sendJSON(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func planQuota(t *testing.T, db *gorm.DB, code string) int64 {
	t.Helper()
	var plan company_models.Plan
	if err := db.Where("code = ?", code).First(&plan).Error; err != nil {
		t.Fatal(err)
	}
	return plan.StorageQuotaMB
}

func TestPlans_CreatePersistsStorageQuota(t *testing.T) {
	router, db := planRouter(t)

	w := sendJSON(router, http.MethodPost, "/backoffice/plans", `{"name":"Clinic","code":"CLINIC","tier":1,"storage_quota_mb":10240}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"storage_quota_mb":10240`) {
		t.Fatalf("response lacks the quota: %s", w.Body.String())
	}
	if got := planQuota(t, db, "CLINIC"); got != 10240 {
		t.Fatalf("storage_quota_mb = %d, want 10240", got)
	}
}

func TestPlans_UpdatePersistsStorageQuotaAndKeepsItWhenOmitted(t *testing.T) {
	router, db := planRouter(t)
	plan := company_models.Plan{Name: "Clinic", Code: "CLINIC", Tier: 1, StorageQuotaMB: 2048}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	path := "/backoffice/plans/" + strconv.FormatUint(uint64(plan.ID), 10)

	if w := sendJSON(router, http.MethodPut, path, `{"storage_quota_mb":51200}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := planQuota(t, db, "CLINIC"); got != 51200 {
		t.Fatalf("storage_quota_mb = %d, want 51200", got)
	}

	if w := sendJSON(router, http.MethodPut, path, `{"name":"Clinic Plus"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := planQuota(t, db, "CLINIC"); got != 51200 {
		t.Fatalf("an update without the field changed the quota to %d", got)
	}

	if w := sendJSON(router, http.MethodPut, path, `{"storage_quota_mb":0}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	if got := planQuota(t, db, "CLINIC"); got != 0 {
		t.Fatalf("storage_quota_mb = %d, want 0", got)
	}
}

func TestPlans_RejectsNegativeStorageQuota(t *testing.T) {
	router, db := planRouter(t)
	plan := company_models.Plan{Name: "Clinic", Code: "CLINIC", Tier: 1, StorageQuotaMB: 2048}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}

	if w := sendJSON(router, http.MethodPut, "/backoffice/plans/"+strconv.FormatUint(uint64(plan.ID), 10), `{"storage_quota_mb":-1}`); w.Code != http.StatusBadRequest {
		t.Fatalf("update: status = %d, want 400", w.Code)
	}
	if w := sendJSON(router, http.MethodPost, "/backoffice/plans", `{"name":"Bad","code":"BAD","tier":1,"storage_quota_mb":-5}`); w.Code != http.StatusBadRequest {
		t.Fatalf("create: status = %d, want 400", w.Code)
	}
	if got := planQuota(t, db, "CLINIC"); got != 2048 {
		t.Fatalf("storage_quota_mb = %d, want 2048", got)
	}
}
