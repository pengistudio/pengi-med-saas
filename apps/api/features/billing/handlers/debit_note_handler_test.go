package billing_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupDebitNoteTestData(t *testing.T) (*gorm.DB, *tenant_models.Tenant, *billing_models.Invoice) {
	t.Helper()

	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&billing_models.Invoice{},
		&billing_models.DebitNote{},
		&billing_models.DebitNoteMotive{},
		&billing_models.InvoiceCounter{},
	)

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "DN Tenant",
		Slug:         fmt.Sprintf("dn-t-%d", now),
		DisplayToken: fmt.Sprintf("tok-dn-t-%d", now),
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	patient := &clinical_models.Patient{
		TenantID:    tenant.ID,
		FirstName:   "Patient",
		LastName:    "DN",
		Institution: "Hospital",
		Document:    fmt.Sprintf("DOC-DN-%d", now),
	}
	if err := db.Create(patient).Error; err != nil {
		t.Fatalf("failed to create test patient: %v", err)
	}

	invoice := &billing_models.Invoice{
		TenantID:          tenant.ID,
		PatientID:         patient.ID,
		Sequential:        "000000001",
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Status:            billing_models.InvoiceStatusAuthorized,
		Total:             56.0,
	}
	if err := db.Create(invoice).Error; err != nil {
		t.Fatalf("failed to create test invoice: %v", err)
	}

	return db, tenant, invoice
}

func TestCreateDebitNote_RequiresAuthorizedInvoice(t *testing.T) {
	db, tenant, invoice := setupDebitNoteTestData(t)
	invoice.Status = billing_models.InvoiceStatusPending
	if err := db.Save(invoice).Error; err != nil {
		t.Fatalf("failed to set invoice pending: %v", err)
	}

	handler := NewDebitNoteHandler(db, zap.NewNop())

	payload := billing_dto.CreateDebitNoteDTO{
		InvoiceID: invoice.ID,
		Motives: []billing_dto.CreateDebitNoteMotive{
			{Reason: "Interés por mora", Value: 10.0, TaxCode: "2", TaxPercentageCode: "2", TaxRate: 0.12},
		},
	}
	body, _ := json.Marshal(payload)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest("POST", "/debit-notes", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.CreateDebitNote(c)

	if response.Code != 400 {
		t.Errorf("expected status 400 for non-authorized invoice, got %d", response.Code)
	}
}

func TestCreateDebitNote_Success(t *testing.T) {
	db, tenant, invoice := setupDebitNoteTestData(t)

	handler := NewDebitNoteHandler(db, zap.NewNop())

	payload := billing_dto.CreateDebitNoteDTO{
		InvoiceID: invoice.ID,
		Motives: []billing_dto.CreateDebitNoteMotive{
			{Reason: "Interés por mora", Value: 10.0, TaxCode: "2", TaxPercentageCode: "2", TaxRate: 0.12},
		},
	}
	body, _ := json.Marshal(payload)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest("POST", "/debit-notes", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.CreateDebitNote(c)

	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	respBytes, _ := json.Marshal(response.Data)
	var debitNote billing_models.DebitNote
	if err := json.Unmarshal(respBytes, &debitNote); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if debitNote.DocumentCode != billing_models.DebitNoteDocumentCode {
		t.Errorf("expected document code %s, got %s", billing_models.DebitNoteDocumentCode, debitNote.DocumentCode)
	}
	if debitNote.InvoiceID != invoice.ID {
		t.Errorf("expected invoice_id %d, got %d", invoice.ID, debitNote.InvoiceID)
	}
	expectedTotal := 10.0 + 10.0*0.12
	if debitNote.Total != expectedTotal {
		t.Errorf("expected total %f, got %f", expectedTotal, debitNote.Total)
	}
}
