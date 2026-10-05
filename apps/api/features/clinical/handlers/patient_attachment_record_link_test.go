package clinical_handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// record creates a consultation of the given patient in the given tenant.
func (s *attachments) record(tenantID, patientID uint) clinical_models.MedicalRecord {
	s.t.Helper()
	r := clinical_models.MedicalRecord{TenantID: tenantID, PatientID: patientID, Date: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC), Motive: "control"}
	if err := s.db.Create(&r).Error; err != nil {
		s.t.Fatalf("create record: %v", err)
	}
	return r
}

// secondOwnPatient is another patient of "own".
func (s *attachments) secondOwnPatient() clinical_models.Patient {
	s.t.Helper()
	p := clinical_models.Patient{TenantID: s.own.ID, FirstName: "O", LastName: "Otro", Document: fmt.Sprintf("DOC-L-%d", time.Now().UnixNano())}
	if err := s.db.Create(&p).Error; err != nil {
		s.t.Fatal(err)
	}
	return p
}

func (s *attachments) linkedTo(id uint) *uint {
	s.t.Helper()
	var got clinical_models.PatientAttachment
	if err := s.db.First(&got, id).Error; err != nil {
		s.t.Fatal(err)
	}
	return got.MedicalRecordID
}

func responseErrorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return decode[struct {
		ErrorCode string `json:"error_code"`
	}](t, w).ErrorCode
}

func TestAttachmentLink_UploadWithConsultationOfSamePatient(t *testing.T) {
	s := newAttachments(t)
	r := s.record(s.own.ID, s.ownPatient.ID)

	f := pdfForm()
	f.medicalRecordID = fmt.Sprint(r.ID)
	w := s.upload(s.ownPatient.ID, f)
	if w.Code != http.StatusOK {
		t.Fatalf("upload = %d: %s", w.Code, w.Body.String())
	}
	a := decode[clinical_models.PatientAttachment](t, w)
	if a.MedicalRecordID == nil || *a.MedicalRecordID != r.ID {
		t.Fatalf("response medical_record_id = %v, want %d", a.MedicalRecordID, r.ID)
	}
	if got := s.linkedTo(a.ID); got == nil || *got != r.ID {
		t.Fatalf("stored medical_record_id = %v, want %d", got, r.ID)
	}
}

func TestAttachmentLink_UploadRejectsConsultationOfAnotherPatientOrTenant(t *testing.T) {
	s := newAttachments(t)
	otherPatient := s.secondOwnPatient()
	ofOtherPatient := s.record(s.own.ID, otherPatient.ID)
	ofOtherTenant := s.record(s.other.ID, s.otherPatient.ID)

	cases := []struct {
		name     string
		recordID string
		status   int
		code     string
	}{
		{"another patient", fmt.Sprint(ofOtherPatient.ID), http.StatusBadRequest, core_errors.ErrClinicalAttachmentRecordPatient.ErrorCode},
		{"another tenant", fmt.Sprint(ofOtherTenant.ID), http.StatusNotFound, core_errors.ErrClinicalRecordNotFound.ErrorCode},
		{"missing", "999999", http.StatusNotFound, core_errors.ErrClinicalRecordNotFound.ErrorCode},
		{"not a number", "abc", http.StatusBadRequest, core_errors.ErrClinicalInvalidRequest.ErrorCode},
	}
	for _, tc := range cases {
		f := pdfForm()
		f.medicalRecordID = tc.recordID
		w := s.upload(s.ownPatient.ID, f)
		if w.Code != tc.status {
			t.Errorf("%s: upload = %d, want %d", tc.name, w.Code, tc.status)
			continue
		}
		if code := responseErrorCode(t, w); code != tc.code {
			t.Errorf("%s: error_code = %q, want %q", tc.name, code, tc.code)
		}
	}
	if n := s.count(); n != 0 {
		t.Fatalf("a rejected upload stored %d attachments", n)
	}
}

func TestAttachmentLink_UpdateLinksAndUnlinks(t *testing.T) {
	s := newAttachments(t)
	r := s.record(s.own.ID, s.ownPatient.ID)
	a := s.uploaded()

	w := s.update(s.ownPatient.ID, a.ID, fmt.Sprintf(`{"medical_record_id":%d}`, r.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("link = %d: %s", w.Code, w.Body.String())
	}
	if resp := decode[clinical_models.PatientAttachment](t, w); resp.MedicalRecordID == nil || *resp.MedicalRecordID != r.ID {
		t.Fatalf("link response = %+v", resp)
	}
	if got := s.linkedTo(a.ID); got == nil || *got != r.ID {
		t.Fatalf("after link = %v", got)
	}

	// A field left out keeps the link.
	if w := s.update(s.ownPatient.ID, a.ID, `{"description":"nota"}`); w.Code != http.StatusOK {
		t.Fatalf("partial update = %d", w.Code)
	}
	if got := s.linkedTo(a.ID); got == nil || *got != r.ID {
		t.Fatalf("a partial update dropped the link: %v", got)
	}

	w = s.update(s.ownPatient.ID, a.ID, `{"medical_record_id":null}`)
	if w.Code != http.StatusOK {
		t.Fatalf("unlink = %d: %s", w.Code, w.Body.String())
	}
	if resp := decode[clinical_models.PatientAttachment](t, w); resp.MedicalRecordID != nil {
		t.Fatalf("unlink response = %+v", resp)
	}
	if got := s.linkedTo(a.ID); got != nil {
		t.Fatalf("after unlink = %v", *got)
	}

	// Link and unlink go through the audit like any metadata change.
	updates := 0
	for _, action := range s.auditActions(a.ID) {
		if action == "UPDATE" {
			updates++
		}
	}
	if updates != 3 {
		t.Fatalf("audited updates = %d, want 3 (link, description, unlink)", updates)
	}
}

func TestAttachmentLink_UpdateRejectsConsultationOfAnotherPatientOrTenant(t *testing.T) {
	s := newAttachments(t)
	mine := s.record(s.own.ID, s.ownPatient.ID)
	a := s.uploaded()
	if w := s.update(s.ownPatient.ID, a.ID, fmt.Sprintf(`{"medical_record_id":%d}`, mine.ID)); w.Code != http.StatusOK {
		t.Fatalf("link = %d", w.Code)
	}
	otherPatient := s.secondOwnPatient()
	ofOtherPatient := s.record(s.own.ID, otherPatient.ID)
	ofOtherTenant := s.record(s.other.ID, s.otherPatient.ID)

	cases := map[string]struct {
		body   string
		status int
	}{
		"another patient": {fmt.Sprintf(`{"medical_record_id":%d}`, ofOtherPatient.ID), http.StatusBadRequest},
		"another tenant":  {fmt.Sprintf(`{"medical_record_id":%d}`, ofOtherTenant.ID), http.StatusNotFound},
		"missing":         {`{"medical_record_id":999999}`, http.StatusNotFound},
		"not a number":    {`{"medical_record_id":"x"}`, http.StatusBadRequest},
	}
	for name, tc := range cases {
		if w := s.update(s.ownPatient.ID, a.ID, tc.body); w.Code != tc.status {
			t.Errorf("%s: update = %d, want %d", name, w.Code, tc.status)
		}
	}
	if got := s.linkedTo(a.ID); got == nil || *got != mine.ID {
		t.Fatalf("a rejected update changed the link: %v", got)
	}
}

func TestAttachmentLink_ListFiltersByConsultationAndShowsItsDate(t *testing.T) {
	s := newAttachments(t)
	r1 := s.record(s.own.ID, s.ownPatient.ID)
	r2 := s.record(s.own.ID, s.ownPatient.ID)
	upload := func(recordID uint) uint {
		f := pdfForm()
		f.content = append([]byte{}, samplePDF...)
		f.content = append(f.content, []byte(fmt.Sprint(time.Now().UnixNano()))...)
		if recordID != 0 {
			f.medicalRecordID = fmt.Sprint(recordID)
		}
		w := s.upload(s.ownPatient.ID, f)
		if w.Code != http.StatusOK {
			t.Fatalf("upload = %d: %s", w.Code, w.Body.String())
		}
		return decode[clinical_models.PatientAttachment](t, w).ID
	}
	inR1 := upload(r1.ID)
	upload(r2.ID)
	upload(0)

	w := s.list(s.ownPatient.ID, fmt.Sprintf("?medical_record_id=%d", r1.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("list = %d", w.Code)
	}
	items := decode[AttachmentList](t, w).Items
	if len(items) != 1 || items[0].ID != inR1 {
		t.Fatalf("filtered list = %+v, want only %d", items, inR1)
	}
	if items[0].MedicalRecordDate == nil || !items[0].MedicalRecordDate.Equal(r1.Date) {
		t.Fatalf("medical_record_date = %v, want %v", items[0].MedicalRecordDate, r1.Date)
	}

	all := decode[AttachmentList](t, s.list(s.ownPatient.ID, "")).Items
	if len(all) != 3 {
		t.Fatalf("unfiltered list = %d items, want 3", len(all))
	}
	for _, item := range all {
		if (item.MedicalRecordID == nil) != (item.MedicalRecordDate == nil) {
			t.Errorf("item %d: medical_record_id %v but date %v", item.ID, item.MedicalRecordID, item.MedicalRecordDate)
		}
	}

	if w := s.list(s.ownPatient.ID, "?medical_record_id=abc"); w.Code != http.StatusBadRequest {
		t.Errorf("bad filter = %d, want 400", w.Code)
	}
}
