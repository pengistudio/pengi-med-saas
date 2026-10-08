package clinical_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

func TestCreatePatient_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &clinical_models.Patient{})
	logger := zap.NewNop()

	// Create test tenant with unique slug and token
	now := time.Now().UnixNano() % 1000000 // Use last 6 digits to keep slug short
	slug := fmt.Sprintf("pat-create-s-%d", now)
	token := fmt.Sprintf("tok-create-s-%d", now)
	tenant := &tenant_models.Tenant{
		Name:         "Test Tenant Success",
		Slug:         slug,
		DisplayToken: token,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	// Create handler and test context
	handler := NewPatientHandler(db, logger)
	c, _ := testutils.NewGinContext(tenant.ID, 1)

	// Create patient payload with unique document
	docNum := fmt.Sprintf("DOC-C-%d", now)
	birthDate := time.Now().AddDate(-30, 0, 0)
	payload := clinical_dto.CreatePatientDTO{
		Document:    docNum,
		Phone:       "5551234567",
		FirstName:   "John",
		LastName:    "Doe",
		BirthDate:   &birthDate,
		Institution: "Hospital Central",
		Gender:      "M",
	}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/patients", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	response := handler.CreatePatient(c)

	// Verify response
	if response.Code != 200 {
		t.Errorf("expected status 200, got %d; message: %s", response.Code, response.Message)
	}

	// Verify patient was created in DB with correct tenant_id
	var createdPatient clinical_models.Patient
	if err := db.Where("document = ? AND tenant_id = ?", docNum, tenant.ID).First(&createdPatient).Error; err != nil {
		t.Errorf("expected patient to be created in DB: %v", err)
		return
	}

	if createdPatient.FirstName != "John" {
		t.Errorf("expected first_name 'John', got '%s'", createdPatient.FirstName)
	}
}

func TestCreatePatient_MissingFields(t *testing.T) {
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &clinical_models.Patient{})
	logger := zap.NewNop()

	// Create test tenant with unique slug and token
	now := time.Now().UnixNano() % 1000000 // Use last 6 digits to keep slug short
	slug := fmt.Sprintf("pat-missing-%d", now)
	token := fmt.Sprintf("tok-missing-%d", now)
	tenant := &tenant_models.Tenant{
		Name:         "Test Tenant Missing",
		Slug:         slug,
		DisplayToken: token,
	}
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	// Create handler and test context
	handler := NewPatientHandler(db, logger)
	c, _ := testutils.NewGinContext(tenant.ID, 1)

	// Create patient payload WITHOUT first_name (required field)
	docNum := fmt.Sprintf("DOC-M-%d", now)
	payload := clinical_dto.CreatePatientDTO{
		Document: docNum,
		Phone:    "5551234567",
		// FirstName is missing (required)
		LastName:    "Doe",
		Institution: "Hospital Central",
	}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/patients", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	response := handler.CreatePatient(c)

	// Verify response is 400 (Bad Request)
	if response.Code != 400 {
		t.Errorf("expected status 400 for missing required field, got %d", response.Code)
	}

	// Verify no patient was created with this document in this tenant
	var patientCount int64
	db.Model(&clinical_models.Patient{}).Where("document = ? AND tenant_id = ?", docNum, tenant.ID).Count(&patientCount)
	if patientCount != 0 {
		t.Errorf("expected no patients to be created with document, but found %d", patientCount)
	}
}

func TestGetAllPatientsWithLastFollowUp_TenantIsolation(t *testing.T) {
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{},
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&clinical_models.Appointment{}, &clinical_models.MedicalRecord{}, // read by the list query
	)
	logger := zap.NewNop()

	// Create two test tenants with unique slugs and tokens
	now := time.Now().UnixNano() % 1000000 // Use last 6 digits to keep slug short
	tenant1 := &tenant_models.Tenant{
		Name:         "Tenant 1",
		Slug:         fmt.Sprintf("pat-t1-%d", now),
		DisplayToken: fmt.Sprintf("tok-t1-%d", now),
	}
	tenant2 := &tenant_models.Tenant{
		Name:         "Tenant 2",
		Slug:         fmt.Sprintf("pat-t2-%d", now),
		DisplayToken: fmt.Sprintf("tok-t2-%d", now),
	}
	if err := db.Create([]*tenant_models.Tenant{tenant1, tenant2}).Error; err != nil {
		t.Fatalf("failed to create test tenants: %v", err)
	}

	// Create patients for each tenant
	patient1 := &clinical_models.Patient{
		TenantID:    tenant1.ID,
		Document:    "DOC001",
		FirstName:   "Alice",
		LastName:    "Smith",
		Institution: "Hospital A",
	}
	patient2 := &clinical_models.Patient{
		TenantID:    tenant2.ID,
		Document:    "DOC002",
		FirstName:   "Bob",
		LastName:    "Jones",
		Institution: "Hospital B",
	}
	if err := db.Create([]*clinical_models.Patient{patient1, patient2}).Error; err != nil {
		t.Fatalf("failed to create test patients: %v", err)
	}

	handler := NewPatientHandler(db, logger)

	// Each tenant sees only its own patient.
	for _, tc := range []struct {
		tenantID uint
		want     string
	}{{tenant1.ID, "Alice"}, {tenant2.ID, "Bob"}} {
		c, _ := testutils.NewGinContext(tc.tenantID, 1)
		c.Request = httptest.NewRequest("GET", "/patients/follow-up", nil)
		response := handler.GetAllPatientsWithLastFollowUp(c)
		if response.Code != 200 {
			t.Fatalf("tenant %d: expected status 200, got %d", tc.tenantID, response.Code)
		}

		raw, _ := json.Marshal(response.Data)
		var page struct {
			Items []struct {
				FirstName string `json:"first_name"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			t.Fatalf("tenant %d: decode response: %v", tc.tenantID, err)
		}
		if len(page.Items) != 1 || page.Items[0].FirstName != tc.want {
			t.Errorf("tenant %d: got %+v, want only %q", tc.tenantID, page.Items, tc.want)
		}
	}
}
