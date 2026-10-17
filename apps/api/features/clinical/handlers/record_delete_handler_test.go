package clinical_handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// recordDeletion drives delete / list deleted / restore over HTTP with two
// clinics. Requests act as user 42 of "own"; "other" must stay out of reach.
type recordDeletion struct {
	t                      *testing.T
	db                     *gorm.DB
	router                 *gin.Engine
	ownPatient             clinical_models.Patient
	ownRecord, otherRecord clinical_models.MedicalRecord
}

func newRecordDeletion(t *testing.T) *recordDeletion {
	t.Helper()
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &clinical_models.Patient{},
		&clinical_models.MedicalRecord{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{},
		&clinical_models.VitalSigns{}, &user_models.User{}, &audit.AuditLog{})
	audit.RegisterCallbacks(db)
	s := &recordDeletion{t: t, db: db}
	now := time.Now().UnixNano()

	var own, other tenant_models.Tenant
	for i, tenant := range []*tenant_models.Tenant{&own, &other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("rec-del-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-rec-del-%d-%d", i, now)}
		if err := db.Create(tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
	}
	var otherPatient clinical_models.Patient
	for _, p := range []struct {
		patient *clinical_models.Patient
		record  *clinical_models.MedicalRecord
		tenant  uint
	}{{&s.ownPatient, &s.ownRecord, own.ID}, {&otherPatient, &s.otherRecord, other.ID}} {
		*p.patient = clinical_models.Patient{TenantID: p.tenant, FirstName: "P", LastName: "Paciente", Document: fmt.Sprintf("DOC-%d-%d", p.tenant, now)}
		if err := db.Create(p.patient).Error; err != nil {
			t.Fatalf("create patient: %v", err)
		}
		*p.record = clinical_models.MedicalRecord{TenantID: p.tenant, PatientID: p.patient.ID, Date: time.Now(), Motive: "Control", SOAPRecord: clinical_models.SOAPRecord{Subjective: "s"}}
		if err := db.Create(p.record).Error; err != nil {
			t.Fatalf("create record: %v", err)
		}
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Stands in for auth + tenant middleware: user 42 of "own".
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", own.ID)
		c.Set("user_id", int64(42))
		c.Set("username", "doctor")
		c.Next()
	})
	h := NewMedicalRecordHandler(db, zap.NewNop())
	records := router.Group("/records")
	records.DELETE("/:id", envelope.Handle(h.DeleteMedicalRecord))
	records.POST("/bulk-delete", envelope.Handle(h.BulkDeleteMedicalRecords))
	records.GET("/patient/:id/deleted", envelope.Handle(h.ListDeletedMedicalRecords))
	records.POST("/:id/restore", envelope.Handle(h.RestoreMedicalRecord))
	s.router = router
	return s
}

func (s *recordDeletion) do(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *recordDeletion) remove(id uint, body string) *httptest.ResponseRecorder {
	return s.do(http.MethodDelete, fmt.Sprintf("/records/%d", id), body)
}

func (s *recordDeletion) restore(id uint) *httptest.ResponseRecorder {
	return s.do(http.MethodPost, fmt.Sprintf("/records/%d/restore", id), "")
}

func (s *recordDeletion) listDeleted(patientID uint) *httptest.ResponseRecorder {
	return s.do(http.MethodGet, fmt.Sprintf("/records/patient/%d/deleted", patientID), "")
}

// visible reports whether the record shows in normal (scoped) queries.
func (s *recordDeletion) visible(id uint) bool {
	var n int64
	s.db.Model(&clinical_models.MedicalRecord{}).Where("id = ?", id).Count(&n)
	return n == 1
}

func TestRecordDelete_RequiresAReason(t *testing.T) {
	s := newRecordDeletion(t)
	for name, body := range map[string]string{"no body": "", "blank": `{"reason":"   "}`, "too long": `{"reason":"` + strings.Repeat("x", 501) + `"}`} {
		if w := s.remove(s.ownRecord.ID, body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400 (%s)", name, w.Code, w.Body)
		}
	}
	if !s.visible(s.ownRecord.ID) {
		t.Fatal("a rejected delete hid the record")
	}
}

func TestRecordDelete_SoftDeletesListsAndRestores(t *testing.T) {
	s := newRecordDeletion(t)

	if w := s.remove(s.ownRecord.ID, `{"reason":"  Consulta duplicada  "}`); w.Code != http.StatusOK {
		t.Fatalf("delete: status %d (%s)", w.Code, w.Body)
	}
	if s.visible(s.ownRecord.ID) {
		t.Fatal("deleted record still shows in normal queries")
	}
	// Soft: the row and its SOAP are still there.
	var stored clinical_models.MedicalRecord
	if err := s.db.Unscoped().First(&stored, s.ownRecord.ID).Error; err != nil {
		t.Fatalf("row is gone: %v", err)
	}
	if stored.DeleteReason != "Consulta duplicada" || stored.DeletedByID == nil || *stored.DeletedByID != 42 {
		t.Fatalf("stored reason %q by %v, want trimmed reason by 42", stored.DeleteReason, stored.DeletedByID)
	}
	var soap int64
	s.db.Model(&clinical_models.SOAPRecord{}).Where("id = ?", stored.SOAPRecordID).Count(&soap)
	if soap != 1 {
		t.Fatal("deleting the record removed its SOAP")
	}

	w := s.listDeleted(s.ownPatient.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("list deleted: status %d (%s)", w.Code, w.Body)
	}
	deleted := decode[[]deletedMedicalRecord](t, w)
	if len(deleted) != 1 || deleted[0].ID != s.ownRecord.ID || deleted[0].DeleteReason != "Consulta duplicada" || !deleted[0].DeletedAt.Valid {
		t.Fatalf("deleted list = %+v, want the record with its reason and deleted_at", deleted)
	}

	if w := s.restore(s.ownRecord.ID); w.Code != http.StatusOK {
		t.Fatalf("restore: status %d (%s)", w.Code, w.Body)
	}
	if !s.visible(s.ownRecord.ID) {
		t.Fatal("restored record doesn't show")
	}
	s.db.First(&stored, s.ownRecord.ID)
	if stored.DeleteReason != "" || stored.DeletedByID != nil {
		t.Fatalf("restore left reason %q by %v", stored.DeleteReason, stored.DeletedByID)
	}
	if got := decode[[]deletedMedicalRecord](t, s.listDeleted(s.ownPatient.ID)); len(got) != 0 {
		t.Fatalf("deleted list after restore = %+v, want empty", got)
	}
}

func TestRecordDelete_RestoreOfARecordNotDeletedIsNotFound(t *testing.T) {
	s := newRecordDeletion(t)
	if w := s.restore(s.ownRecord.ID); w.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", w.Code)
	}
}

func TestRecordDelete_AnotherTenantIsNotFoundOnEveryRoute(t *testing.T) {
	s := newRecordDeletion(t)
	if w := s.remove(s.otherRecord.ID, `{"reason":"x"}`); w.Code != http.StatusNotFound {
		t.Errorf("delete: status %d, want 404", w.Code)
	}
	if !s.visible(s.otherRecord.ID) {
		t.Fatal("another clinic's record was deleted")
	}
	if w := s.listDeleted(s.otherRecord.PatientID); w.Code != http.StatusNotFound {
		t.Errorf("list deleted: status %d, want 404", w.Code)
	}

	// Deleted by its own clinic, it still can't be restored from here.
	s.db.Model(&s.otherRecord).Update("deleted_at", time.Now())
	if w := s.restore(s.otherRecord.ID); w.Code != http.StatusNotFound {
		t.Errorf("restore: status %d, want 404", w.Code)
	}
	if s.visible(s.otherRecord.ID) {
		t.Fatal("another clinic's record was restored")
	}
}

func (s *recordDeletion) bulkRemove(body string) *httptest.ResponseRecorder {
	return s.do(http.MethodPost, "/records/bulk-delete", body)
}

// secondOwnRecord adds another record to the own patient.
func (s *recordDeletion) secondOwnRecord() clinical_models.MedicalRecord {
	r := clinical_models.MedicalRecord{TenantID: s.ownRecord.TenantID, PatientID: s.ownPatient.ID, Date: time.Now(), Motive: "Control 2", SOAPRecord: clinical_models.SOAPRecord{Subjective: "s"}}
	if err := s.db.Create(&r).Error; err != nil {
		s.t.Fatalf("create record: %v", err)
	}
	return r
}

func TestRecordBulkDelete_DeletesAllWithTheReason(t *testing.T) {
	s := newRecordDeletion(t)
	second := s.secondOwnRecord()

	body := fmt.Sprintf(`{"ids":[%d,%d,%d],"reason":"  Duplicadas  "}`, s.ownRecord.ID, second.ID, second.ID)
	if w := s.bulkRemove(body); w.Code != http.StatusOK {
		t.Fatalf("bulk delete: status %d (%s)", w.Code, w.Body)
	}
	for _, id := range []uint{s.ownRecord.ID, second.ID} {
		if s.visible(id) {
			t.Fatalf("record %d still shows", id)
		}
		var stored clinical_models.MedicalRecord
		s.db.Unscoped().First(&stored, id)
		if stored.DeleteReason != "Duplicadas" || stored.DeletedByID == nil || *stored.DeletedByID != 42 {
			t.Fatalf("record %d: reason %q by %v, want trimmed reason by 42", id, stored.DeleteReason, stored.DeletedByID)
		}
	}
	if got := decode[[]deletedMedicalRecord](t, s.listDeleted(s.ownPatient.ID)); len(got) != 2 {
		t.Fatalf("deleted list has %d records, want 2", len(got))
	}
}

func TestRecordBulkDelete_RejectsBadRequests(t *testing.T) {
	s := newRecordDeletion(t)
	id := s.ownRecord.ID
	for name, body := range map[string]string{
		"no body":   "",
		"no ids":    `{"ids":[],"reason":"x"}`,
		"no reason": fmt.Sprintf(`{"ids":[%d],"reason":"  "}`, id),
		"too long":  fmt.Sprintf(`{"ids":[%d],"reason":"%s"}`, id, strings.Repeat("x", 501)),
	} {
		if w := s.bulkRemove(body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400 (%s)", name, w.Code, w.Body)
		}
	}
	if !s.visible(id) {
		t.Fatal("a rejected bulk delete hid the record")
	}
}

// One id out of reach (another clinic's, or already deleted) and nothing is
// deleted.
func TestRecordBulkDelete_IsAllOrNothing(t *testing.T) {
	s := newRecordDeletion(t)
	second := s.secondOwnRecord()

	body := fmt.Sprintf(`{"ids":[%d,%d],"reason":"x"}`, s.ownRecord.ID, s.otherRecord.ID)
	if w := s.bulkRemove(body); w.Code != http.StatusNotFound {
		t.Fatalf("with another clinic's id: status %d, want 404", w.Code)
	}
	if !s.visible(s.ownRecord.ID) || !s.visible(s.otherRecord.ID) {
		t.Fatal("a rejected bulk delete deleted records")
	}

	s.remove(second.ID, `{"reason":"x"}`)
	body = fmt.Sprintf(`{"ids":[%d,%d],"reason":"y"}`, s.ownRecord.ID, second.ID)
	if w := s.bulkRemove(body); w.Code != http.StatusNotFound {
		t.Fatalf("with an already deleted id: status %d, want 404", w.Code)
	}
	if !s.visible(s.ownRecord.ID) {
		t.Fatal("a rejected bulk delete deleted the own record")
	}
}
