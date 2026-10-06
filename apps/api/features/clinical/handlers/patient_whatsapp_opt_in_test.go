package clinical_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func optInTenants(t *testing.T) (*PatientHandler, func(uint, string, bool) clinical_models.Patient, func(uint) clinical_models.Patient, [2]uint) {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{})
	var ids [2]uint
	for i := range ids {
		slug := fmt.Sprintf("optin-%d-%d", i, time.Now().UnixNano()%1000000)
		tenant := tenant_models.Tenant{Name: "T", Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		ids[i] = tenant.ID
	}
	create := func(tenantID uint, name string, optIn bool) clinical_models.Patient {
		p := clinical_models.Patient{TenantID: tenantID, FirstName: name, LastName: "X", Document: fmt.Sprintf("D%d", time.Now().UnixNano()), WhatsAppOptIn: optIn}
		if err := db.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
		return p
	}
	load := func(id uint) clinical_models.Patient {
		var p clinical_models.Patient
		db.Where("id = ?", id).First(&p)
		return p
	}
	return NewPatientHandler(db, zap.NewNop()), create, load, ids
}

func jsonCtx(tenantID uint, method string, body any, params gin.Params) *gin.Context {
	c, _ := testutils.NewGinContext(tenantID, 1)
	raw, _ := json.Marshal(body)
	c.Request = httptest.NewRequest(method, "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	return c
}

func TestBulkWhatsAppOptIn_TenantIsolated(t *testing.T) {
	h, create, load, tenants := optInTenants(t)
	own, other := tenants[0], tenants[1]
	a := create(own, "A", false)
	b := create(own, "B", false)
	already := create(own, "C", true)
	foreign := create(other, "F", false)

	resp := h.BulkWhatsAppOptIn(jsonCtx(own, "POST", map[string]any{"ids": []uint{a.ID, b.ID, already.ID, foreign.ID, 999999}, "opt_in": true}, nil))
	if resp.Code != 200 {
		t.Fatalf("code = %d", resp.Code)
	}
	if got := resp.Data.(clinical_dto.BulkWhatsAppOptInResponse).Updated; got != 2 {
		t.Fatalf("updated = %d, want 2 (only own patients that changed)", got)
	}
	for _, id := range []uint{a.ID, b.ID} {
		if p := load(id); !p.WhatsAppOptIn || p.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourceBulk || p.WhatsAppOptInAt == nil {
			t.Fatalf("patient %d = %+v", id, p)
		}
	}
	if p := load(already.ID); p.WhatsAppOptInSource != "" {
		t.Fatalf("unchanged patient got source %q", p.WhatsAppOptInSource)
	}
	if p := load(foreign.ID); p.WhatsAppOptIn {
		t.Fatal("other tenant's patient changed")
	}

	resp = h.BulkWhatsAppOptIn(jsonCtx(own, "POST", map[string]any{"ids": []uint{a.ID}, "opt_in": false}, nil))
	if resp.Data.(clinical_dto.BulkWhatsAppOptInResponse).Updated != 1 || load(a.ID).WhatsAppOptIn {
		t.Fatal("clearing consent failed")
	}
	for _, bad := range []map[string]any{{"ids": []uint{}, "opt_in": true}, {"ids": []uint{a.ID}}} {
		if resp := h.BulkWhatsAppOptIn(jsonCtx(own, "POST", bad, nil)); resp.Code != 400 {
			t.Fatalf("%v: code = %d", bad, resp.Code)
		}
	}
}

func TestPatientOptInSource_CreateAndUpdate(t *testing.T) {
	h, _, load, tenants := optInTenants(t)
	own := tenants[0]
	resp := h.CreatePatient(jsonCtx(own, "POST", map[string]any{
		"document": fmt.Sprintf("D%d", time.Now().UnixNano()), "first_name": "A", "last_name": "B", "institution": "I", "whatsapp_opt_in": true,
	}, nil))
	if resp.Code != 200 {
		t.Fatalf("create: %d", resp.Code)
	}
	created := resp.Data.(*clinical_models.Patient)
	if p := load(created.ID); p.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourceRegistration {
		t.Fatalf("source after create = %q", p.WhatsAppOptInSource)
	}
	resp = h.UpdatePatient(jsonCtx(own, "PUT", map[string]any{"whatsapp_opt_in": false}, gin.Params{{Key: "id", Value: fmt.Sprint(created.ID)}}))
	if resp.Code != 200 {
		t.Fatalf("update: %d", resp.Code)
	}
	if p := load(created.ID); p.WhatsAppOptIn || p.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourceManual {
		t.Fatalf("after update = %+v", p)
	}
}
