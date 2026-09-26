package backoffice_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"pengi-med-saas/core/envelope"
	company_models "pengi-med-saas/features/companies/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func subscriptionRouter(t *testing.T) (*gin.Engine, *company_models.Subscription) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{})

	plan := company_models.Plan{Name: "Pro", Code: "pro"}
	if err := db.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	company := company_models.Company{TradeName: "Clínica Norte"}
	if err := db.Create(&company).Error; err != nil {
		t.Fatal(err)
	}
	sub := company_models.Subscription{CompanyID: company.ID, PlanCode: "pro", Status: "active", ExpiresAt: time.Now().AddDate(0, 1, 0)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}

	h := NewBackofficeSubscriptionHandler(db, zap.NewNop())
	router := gin.New()
	router.GET("/backoffice/subscriptions/:id", envelope.Handle(h.GetSubscriptionByID))
	return router, &sub
}

func TestGetSubscriptionByID_ReturnsSubscriptionWithPlanAndCompany(t *testing.T) {
	router, sub := subscriptionRouter(t)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/backoffice/subscriptions/"+strconv.FormatUint(uint64(sub.ID), 10), nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data struct {
			ID       uint   `json:"ID"`
			PlanCode string `json:"plan_code"`
			Plan     struct {
				Name string `json:"name"`
			} `json:"plan"`
			Company struct {
				TradeName string `json:"trade_name"`
			} `json:"company"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.ID != sub.ID || body.Data.Plan.Name != "Pro" || body.Data.Company.TradeName != "Clínica Norte" {
		t.Fatalf("unexpected subscription: %+v", body.Data)
	}
}

func TestGetSubscriptionByID_UnknownIDIsNotFound(t *testing.T) {
	router, _ := subscriptionRouter(t)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/backoffice/subscriptions/9999", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
