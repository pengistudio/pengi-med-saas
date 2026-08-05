package billing_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func TestGetAllInvoices_TenantScope(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&billing_models.Invoice{},
	)
	logger := zap.NewNop()

	// Create two test tenants
	now := time.Now().UnixNano() % 1000000
	tenant1 := &tenant_models.Tenant{
		Name:         "Tenant 1",
		Slug:         fmt.Sprintf("inv-t1-%d", now),
		DisplayToken: fmt.Sprintf("tok-inv-t1-%d", now),
	}
	tenant2 := &tenant_models.Tenant{
		Name:         "Tenant 2",
		Slug:         fmt.Sprintf("inv-t2-%d", now),
		DisplayToken: fmt.Sprintf("tok-inv-t2-%d", now),
	}
	if err := db.Create([]*tenant_models.Tenant{tenant1, tenant2}).Error; err != nil {
		t.Fatalf("failed to create test tenants: %v", err)
	}

	// Create patients for each tenant
	pat1 := &clinical_models.Patient{
		TenantID:    tenant1.ID,
		FirstName:   "Patient",
		LastName:    "One",
		Institution: "Hosp A",
		Document:    fmt.Sprintf("DOC-P1-%d", now),
	}
	pat2 := &clinical_models.Patient{
		TenantID:    tenant2.ID,
		FirstName:   "Patient",
		LastName:    "Two",
		Institution: "Hosp B",
		Document:    fmt.Sprintf("DOC-P2-%d", now),
	}
	if err := db.Create([]*clinical_models.Patient{pat1, pat2}).Error; err != nil {
		t.Fatalf("failed to create test patients: %v", err)
	}

	// Create invoices for each tenant
	inv1 := &billing_models.Invoice{
		TenantID:   tenant1.ID,
		PatientID:  testutils.Ptr(pat1.ID),
		Sequential: "INV-001",
		Status:     "pending",
		Total:      100.0,
	}
	inv2 := &billing_models.Invoice{
		TenantID:   tenant2.ID,
		PatientID:  testutils.Ptr(pat2.ID),
		Sequential: "INV-002",
		Status:     "pending",
		Total:      200.0,
	}
	if err := db.Create([]*billing_models.Invoice{inv1, inv2}).Error; err != nil {
		t.Fatalf("failed to create test invoices: %v", err)
	}

	// Create handler
	handler := NewInvoiceHandler(db, logger)

	// Test: Query invoices for tenant1
	c1, _ := testutils.NewGinContext(tenant1.ID, 1)
	c1.Request = httptest.NewRequest("GET", "/invoices", nil)
	response1 := handler.GetAllInvoices(c1)

	if response1.Code != 200 {
		t.Errorf("expected status 200, got %d", response1.Code)
	}

	// Verify response contains only tenant1's invoice
	respBytes, _ := json.Marshal(response1.Data)
	var respData map[string]interface{}
	json.Unmarshal(respBytes, &respData)

	if items, ok := respData["items"]; ok {
		itemsList := items.([]interface{})
		if len(itemsList) != 1 {
			t.Errorf("expected 1 invoice for tenant1, got %d", len(itemsList))
		}
	}

	// Test: Query invoices for tenant2
	c2, _ := testutils.NewGinContext(tenant2.ID, 1)
	c2.Request = httptest.NewRequest("GET", "/invoices", nil)
	response2 := handler.GetAllInvoices(c2)

	if response2.Code != 200 {
		t.Errorf("expected status 200, got %d", response2.Code)
	}

	respBytes, _ = json.Marshal(response2.Data)
	json.Unmarshal(respBytes, &respData)

	if items, ok := respData["items"]; ok {
		itemsList := items.([]interface{})
		if len(itemsList) != 1 {
			t.Errorf("expected 1 invoice for tenant2, got %d", len(itemsList))
		}
	}
}

func TestGetAllInvoices_Pagination(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&billing_models.Invoice{},
	)
	logger := zap.NewNop()

	// Create test tenant
	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "Pagination Tenant",
		Slug:         fmt.Sprintf("inv-pag-%d", now),
		DisplayToken: fmt.Sprintf("tok-inv-pag-%d", now),
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	// Create a patient for invoices
	patient := &clinical_models.Patient{
		TenantID:    tenant.ID,
		FirstName:   "Patient",
		LastName:    "Pagination",
		Institution: "Hospital",
		Document:    fmt.Sprintf("DOC-PAG-%d", now),
	}
	if err := db.Create(patient).Error; err != nil {
		t.Fatalf("failed to create test patient: %v", err)
	}

	// Create 5 invoices for pagination testing
	for i := 1; i <= 5; i++ {
		inv := &billing_models.Invoice{
			TenantID:   tenant.ID,
			PatientID:  testutils.Ptr(patient.ID),
			Sequential: fmt.Sprintf("INV-PAG-%03d", i),
			Status:     "pending",
			Total:      float64(i * 100),
		}
		if err := db.Create(inv).Error; err != nil {
			t.Fatalf("failed to create invoice %d: %v", i, err)
		}
	}

	handler := NewInvoiceHandler(db, logger)

	// Test: Page 1 with limit 2
	c1, _ := testutils.NewGinContext(tenant.ID, 1)
	c1.Request = httptest.NewRequest("GET", "/invoices?page=1&limit=2", nil)
	c1.Request.Header.Set("Content-Type", "application/json")
	response1 := handler.GetAllInvoices(c1)

	if response1.Code != 200 {
		t.Errorf("expected status 200, got %d", response1.Code)
	}

	respBytes, _ := json.Marshal(response1.Data)
	var respData map[string]interface{}
	json.Unmarshal(respBytes, &respData)

	if items, ok := respData["items"]; ok {
		itemsList := items.([]interface{})
		if len(itemsList) != 2 {
			t.Errorf("page 1: expected 2 invoices, got %d", len(itemsList))
		}
	}

	if total, ok := respData["total"]; ok {
		totalVal := int(total.(float64))
		if totalVal != 5 {
			t.Errorf("expected total 5, got %d", totalVal)
		}
	}

	// Test: Page 2 with limit 2
	c2, _ := testutils.NewGinContext(tenant.ID, 1)
	c2.Request = httptest.NewRequest("GET", "/invoices?page=2&limit=2", nil)
	response2 := handler.GetAllInvoices(c2)

	if response2.Code != 200 {
		t.Errorf("expected status 200, got %d", response2.Code)
	}

	respBytes, _ = json.Marshal(response2.Data)
	json.Unmarshal(respBytes, &respData)

	if items, ok := respData["items"]; ok {
		itemsList := items.([]interface{})
		if len(itemsList) != 2 {
			t.Errorf("page 2: expected 2 invoices, got %d", len(itemsList))
		}
	}

	// Test: Page 3 with limit 2 (should have only 1 item)
	c3, _ := testutils.NewGinContext(tenant.ID, 1)
	c3.Request = httptest.NewRequest("GET", "/invoices?page=3&limit=2", nil)
	response3 := handler.GetAllInvoices(c3)

	if response3.Code != 200 {
		t.Errorf("expected status 200, got %d", response3.Code)
	}

	respBytes, _ = json.Marshal(response3.Data)
	json.Unmarshal(respBytes, &respData)

	if items, ok := respData["items"]; ok {
		itemsList := items.([]interface{})
		if len(itemsList) != 1 {
			t.Errorf("page 3: expected 1 invoice, got %d", len(itemsList))
		}
	}
}

func TestCreateInvoice_FinalConsumer_NoPatient(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&billing_models.Invoice{},
		&billing_models.InvoiceItem{},
		&billing_models.InvoiceCounter{},
		&billing_models.CatalogItem{},
	)
	logger := zap.NewNop()

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "Final Consumer Tenant",
		Slug:         fmt.Sprintf("inv-fc-%d", now),
		DisplayToken: fmt.Sprintf("tok-inv-fc-%d", now),
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	product := &billing_models.CatalogItem{
		TenantID:  tenant.ID,
		Name:      "Consulta",
		SKU:       fmt.Sprintf("CONS-%d", now),
		UnitPrice: 50.0,
		Tax:       0.12,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("failed to create test catalog item: %v", err)
	}

	handler := NewInvoiceHandler(db, logger)

	payload := billing_dto.CreateInvoiceDTO{
		// PatientID intentionally omitted — "Consumidor Final" invoice
		PaymentMethod:     "01",
		Term:              0,
		TimeUnit:          "dias",
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Items: []billing_dto.CreateInvoiceItem{
			{ProductID: product.ID, Quantity: 1, Discount: 0},
		},
	}
	body, _ := json.Marshal(payload)

	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Request = httptest.NewRequest("POST", "/invoices", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.CreateInvoice(c)

	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d (data: %+v)", response.Code, response.Data)
	}

	var stored billing_models.Invoice
	if err := db.Where("tenant_id = ?", tenant.ID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load created invoice: %v", err)
	}
	if stored.PatientID != nil {
		t.Errorf("expected PatientID to be nil for a final-consumer invoice, got %v", *stored.PatientID)
	}
}

func TestCreateInvoice_MissingTenantID(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&billing_models.Invoice{},
	)
	logger := zap.NewNop()

	handler := NewInvoiceHandler(db, logger)

	// Create context WITHOUT tenant_id
	c, _ := testutils.NewGinContext(1, 1)
	// Manually remove tenant_id from context
	c.Keys = make(map[any]any)

	// Create invoice payload
	payload := billing_dto.CreateInvoiceDTO{
		PaymentMethod:     "01",
		Term:              0,
		TimeUnit:          "days",
		EstablishmentCode: "001",
		EmissionPointCode: "001",
		Items:             []billing_dto.CreateInvoiceItem{},
	}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/invoices", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	response := handler.CreateInvoice(c)

	// Verify response is 401 (Unauthorized - missing tenant)
	if response.Code != 401 {
		t.Errorf("expected status 401 for missing tenant_id, got %d", response.Code)
	}
}
