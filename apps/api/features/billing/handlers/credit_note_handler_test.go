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

func setupCreditNoteTestData(t *testing.T) (*gorm.DB, *tenant_models.Tenant, *billing_models.CatalogItem, *billing_models.Invoice) {
	t.Helper()

	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&billing_models.Invoice{},
		&billing_models.CatalogItem{},
		&billing_models.CreditNote{},
		&billing_models.CreditNoteItem{},
		&billing_models.InvoiceCounter{},
	)

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "CN Tenant",
		Slug:         fmt.Sprintf("cn-t-%d", now),
		DisplayToken: fmt.Sprintf("tok-cn-t-%d", now),
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	patient := &clinical_models.Patient{
		TenantID:    tenant.ID,
		FirstName:   "Patient",
		LastName:    "CN",
		Institution: "Hospital",
		Document:    fmt.Sprintf("DOC-CN-%d", now),
	}
	if err := db.Create(patient).Error; err != nil {
		t.Fatalf("failed to create test patient: %v", err)
	}

	product := &billing_models.CatalogItem{
		TenantID:  tenant.ID,
		Name:      "Consulta",
		SKU:       "CONS-01",
		UnitPrice: 50.0,
		Tax:       0.12,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("failed to create test catalog item: %v", err)
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

	return db, tenant, product, invoice
}

func TestCreateCreditNote_RequiresAuthorizedInvoice(t *testing.T) {
	db, tenant, product, invoice := setupCreditNoteTestData(t)
	invoice.Status = billing_models.InvoiceStatusPending
	if err := db.Save(invoice).Error; err != nil {
		t.Fatalf("failed to set invoice pending: %v", err)
	}

	handler := NewCreditNoteHandler(db, zap.NewNop())

	payload := billing_dto.CreateCreditNoteDTO{
		InvoiceID: invoice.ID,
		Reason:    "Error en el servicio facturado",
		Items: []billing_dto.CreateInvoiceItem{
			{ProductID: product.ID, Quantity: 1},
		},
	}
	body, _ := json.Marshal(payload)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest("POST", "/credit-notes", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.CreateCreditNote(c)

	if response.Code != 400 {
		t.Errorf("expected status 400 for non-authorized invoice, got %d", response.Code)
	}
}

func TestCreateCreditNote_Success(t *testing.T) {
	db, tenant, product, invoice := setupCreditNoteTestData(t)

	handler := NewCreditNoteHandler(db, zap.NewNop())

	payload := billing_dto.CreateCreditNoteDTO{
		InvoiceID: invoice.ID,
		Reason:    "Anulación por error en el servicio facturado",
		Items: []billing_dto.CreateInvoiceItem{
			{ProductID: product.ID, Quantity: 1},
		},
	}
	body, _ := json.Marshal(payload)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest("POST", "/credit-notes", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.CreateCreditNote(c)

	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	respBytes, _ := json.Marshal(response.Data)
	var creditNote billing_models.CreditNote
	if err := json.Unmarshal(respBytes, &creditNote); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if creditNote.DocumentCode != billing_models.CreditNoteDocumentCode {
		t.Errorf("expected document code %s, got %s", billing_models.CreditNoteDocumentCode, creditNote.DocumentCode)
	}
	if creditNote.InvoiceID != invoice.ID {
		t.Errorf("expected invoice_id %d, got %d", invoice.ID, creditNote.InvoiceID)
	}
	if creditNote.Total <= 0 {
		t.Errorf("expected computed total > 0, got %f", creditNote.Total)
	}
}
