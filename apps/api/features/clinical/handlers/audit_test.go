package clinical_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestUpdatePatient_CreatesUpdateAuditLog(t *testing.T) {
	rawDB := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&audit.AuditLog{},
	)
	audit.RegisterCallbacks(rawDB)
	logger := zap.NewNop()

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "Audit Update Tenant",
		Slug:         fmt.Sprintf("aud-upd-%d", now),
		DisplayToken: fmt.Sprintf("aud-upd-tok-%d", now),
	}
	if err := rawDB.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	patient := &clinical_models.Patient{
		TenantID:    tenant.ID,
		Document:    "DOC-AUDIT-1",
		FirstName:   "Old",
		LastName:    "Name",
		Institution: "Hospital Audit",
	}
	if err := rawDB.Create(patient).Error; err != nil {
		t.Fatalf("failed to create test patient: %v", err)
	}

	handler := NewPatientHandler(rawDB, logger)
	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(patient.ID)}}

	newFirstName := "New"
	payload := clinical_dto.UpdatePatientDTO{FirstName: &newFirstName}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("PUT", "/patients/"+fmt.Sprint(patient.ID), bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.UpdatePatient(c)
	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d; message: %s", response.Code, response.Message)
	}

	var logs []audit.AuditLog
	if err := rawDB.Where("entity_type = ? AND entity_id = ? AND action = ?", "patients", patient.ID, "UPDATE").Find(&logs).Error; err != nil {
		t.Fatalf("failed to query audit logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected at least one UPDATE audit log for patient %d, got none", patient.ID)
	}
	if len(logs[0].NewValues) == 0 {
		t.Errorf("expected NewValues to be populated on UPDATE audit log")
	}
}

func TestGetPatientByID_CreatesReadAuditLog(t *testing.T) {
	rawDB := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&audit.AuditLog{},
	)
	audit.RegisterCallbacks(rawDB)
	logger := zap.NewNop()

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "Audit Read Tenant",
		Slug:         fmt.Sprintf("aud-read-%d", now),
		DisplayToken: fmt.Sprintf("aud-read-tok-%d", now),
	}
	if err := rawDB.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	patient := &clinical_models.Patient{
		TenantID:    tenant.ID,
		Document:    "DOC-AUDIT-2",
		FirstName:   "Read",
		LastName:    "Me",
		Institution: "Hospital Audit",
	}
	if err := rawDB.Create(patient).Error; err != nil {
		t.Fatalf("failed to create test patient: %v", err)
	}

	handler := NewPatientHandler(rawDB, logger)
	c, _ := testutils.NewGinContext(tenant.ID, 1)
	c.Set("user_id", int64(1))
	c.Set("username", "tester")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(patient.ID)}}
	c.Request = httptest.NewRequest("GET", "/patients/"+fmt.Sprint(patient.ID), nil)

	response := handler.GetPatientByID(c)
	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d; message: %s", response.Code, response.Message)
	}

	var logs []audit.AuditLog
	if err := rawDB.Where("entity_type = ? AND entity_id = ? AND action = ?", "patients", patient.ID, "READ").Find(&logs).Error; err != nil {
		t.Fatalf("failed to query audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected exactly 1 READ audit log for patient %d, got %d", patient.ID, len(logs))
	}
	if logs[0].PatientID == nil || *logs[0].PatientID != patient.ID {
		t.Errorf("expected PatientID to be set to %d on READ audit log", patient.ID)
	}
}

func TestDeleteMultiplePatients_CreatesDeleteAuditLogPerPatient(t *testing.T) {
	rawDB := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&clinical_models.Patient{},
		&clinical_models.Appointment{}, &clinical_models.MedicalRecord{}, // read by the remaining-list query
		&audit.AuditLog{},
	)
	audit.RegisterCallbacks(rawDB)
	logger := zap.NewNop()

	now := time.Now().UnixNano() % 1000000
	tenant := &tenant_models.Tenant{
		Name:         "Audit Delete Tenant",
		Slug:         fmt.Sprintf("aud-del-%d", now),
		DisplayToken: fmt.Sprintf("aud-del-tok-%d", now),
	}
	if err := rawDB.Create(tenant).Error; err != nil {
		t.Fatalf("failed to create test tenant: %v", err)
	}

	var ids []uint
	for _, doc := range []string{"DOC-AUDIT-3", "DOC-AUDIT-4"} {
		patient := &clinical_models.Patient{
			TenantID:    tenant.ID,
			Document:    doc,
			FirstName:   "Delete",
			LastName:    "Me",
			Institution: "Hospital Audit",
		}
		if err := rawDB.Create(patient).Error; err != nil {
			t.Fatalf("failed to create test patient: %v", err)
		}
		ids = append(ids, patient.ID)
	}

	handler := NewPatientHandler(rawDB, logger)
	c, _ := testutils.NewGinContext(tenant.ID, 1)
	body, _ := json.Marshal(clinical_dto.DeletePatientsDTO{IdList: ids})
	c.Request = httptest.NewRequest("POST", "/patients/delete-multiple", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.DeleteMultiplePatients(c)
	if response.Code != 200 {
		t.Fatalf("expected status 200, got %d; message: %s", response.Code, response.Message)
	}

	var logged []uint
	if err := rawDB.Model(&audit.AuditLog{}).Where("entity_type = ? AND action = ?", "patients", "DELETE").
		Order("entity_id").Pluck("entity_id", &logged).Error; err != nil {
		t.Fatalf("failed to query audit logs: %v", err)
	}
	if fmt.Sprint(logged) != fmt.Sprint(ids) {
		t.Fatalf("DELETE audit logs for patients %v, want one per deleted patient %v", logged, ids)
	}
}
