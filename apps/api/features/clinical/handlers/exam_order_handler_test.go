package clinical_handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	clinical_services "pengi-med-saas/features/clinical/services"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	signature_models "pengi-med-saas/features/signatures/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
)

const (
	examDoctorID     = 7
	examAssistantID  = 8
	pngResultContent = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
	pdfResultContent = "%PDF-1.4\n1 0 obj\n<<>>\nendobj\n"
)

// examFixture holds two clinics with their seeded exam catalogs; tests act as
// members of "own" unless they say otherwise.
type examFixture struct {
	t            *testing.T
	db           *gorm.DB
	h            *ExamOrderHandler
	catalog      *ExamCatalogHandler
	disk         *tenantfiles.MemoryStore // what is stored (encrypted)
	attachments  tenantfiles.Store        // the encrypting store results go through
	att          *PatientAttachmentHandler
	own, other   uint
	ownPatient   clinical_models.Patient
	otherPatient clinical_models.Patient
}

func newExamFixture(t *testing.T) *examFixture {
	t.Helper()
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.MedicalRecord{},
		&clinical_models.ExamCatalogItem{}, &clinical_models.ExamProfile{}, &clinical_models.ExamOrder{},
		&clinical_models.ExamOrderItem{}, &clinical_models.ExamOrderCounter{}, &clinical_models.PatientAttachment{},
		&company_models.Company{}, &company_models.Plan{}, &company_models.Subscription{},
		&user_models.User{}, &signature_models.UserSignature{}, &notifications_models.Notification{}, &audit.AuditLog{},
	)
	audit.RegisterCallbacks(db)
	key := sha256.Sum256([]byte("exam-results-test-key"))
	box, err := secretbox.New(key[:])
	if err != nil {
		t.Fatal(err)
	}
	f := &examFixture{t: t, db: db, disk: tenantfiles.Memory()}
	f.attachments = tenantfiles.Encrypted(f.disk, box)
	f.h = NewExamOrderHandler(db, zap.NewNop(), nil, nil, nil, tenantfiles.Memory(), f.attachments)
	f.att = NewPatientAttachmentHandler(db, zap.NewNop(), f.attachments)
	f.catalog = NewExamCatalogHandler(db, zap.NewNop())

	now := time.Now().UnixNano()
	for i, id := range []*uint{&f.own, &f.other} {
		tenant := tenant_models.Tenant{Name: fmt.Sprintf("Exam clinic %d", i), Slug: fmt.Sprintf("exam-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-exam-%d-%d", i, now)}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatalf("create tenant: %v", err)
		}
		*id = tenant.ID
		f.subscribe(tenant.ID, now)
		if err := db.Transaction(func(tx *gorm.DB) error { return clinical_services.SeedExamCatalog(tx, tenant.ID) }); err != nil {
			t.Fatalf("seed exam catalog: %v", err)
		}
	}
	for _, u := range []user_models.User{{Model: gorm.Model{ID: examDoctorID}, UserName: "dra.andrade"}, {Model: gorm.Model{ID: examAssistantID}, UserName: "asistente"}} {
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	f.ownPatient = f.patient(f.own, "OWN")
	f.otherPatient = f.patient(f.other, "OTHER")
	return f
}

// subscribe gives the tenant an active plan with attachment storage to spare.
func (f *examFixture) subscribe(tenantID uint, now int64) {
	plan := company_models.Plan{Name: "Clinic", Code: fmt.Sprintf("EXAM-%d-%d", tenantID, now), StorageQuotaMB: 1024}
	company := company_models.Company{LegalName: "Clinic", TradeName: "Clinic", PlanCode: plan.Code, TenantID: tenantID}
	for _, row := range []any{&plan, &company} {
		if err := f.db.Create(row).Error; err != nil {
			f.t.Fatalf("create plan/company: %v", err)
		}
	}
	if err := f.db.Create(&company_models.Subscription{CompanyID: company.ID, PlanCode: plan.Code, Status: "active", ExpiresAt: time.Now().AddDate(0, 1, 0)}).Error; err != nil {
		f.t.Fatalf("create subscription: %v", err)
	}
}

func (f *examFixture) patient(tenantID uint, tag string) clinical_models.Patient {
	p := clinical_models.Patient{TenantID: tenantID, FirstName: tag, LastName: "Paciente", Institution: "H", Document: fmt.Sprintf("DOC-%s-%d", tag, time.Now().UnixNano())}
	if err := f.db.Create(&p).Error; err != nil {
		f.t.Fatalf("create patient: %v", err)
	}
	return p
}

// ctx is a request of userID in tenantID with an optional JSON body and path params.
func (f *examFixture) ctx(tenantID, userID uint, body any, params ...string) *gin.Context {
	c, _ := f.ctxRec(tenantID, userID, body, params...)
	return c
}

func (f *examFixture) ctxRec(tenantID, userID uint, body any, params ...string) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := testutils.NewGinContext(tenantID, int64(userID))
	c.Set("user_id", int64(userID))
	c.Set("username", "tester")
	for i := 0; i+1 < len(params); i += 2 {
		c.Params = append(c.Params, gin.Param{Key: params[i], Value: params[i+1]})
	}
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

// catalogItem returns a seeded exam of the tenant by name.
func (f *examFixture) catalogItem(tenantID uint, name string) clinical_models.ExamCatalogItem {
	var item clinical_models.ExamCatalogItem
	if err := f.db.Where("tenant_id = ? AND name = ?", tenantID, name).First(&item).Error; err != nil {
		f.t.Fatalf("catalog item %q: %v", name, err)
	}
	return item
}

func order(t *testing.T, resp envelope.Response, wantCode int) *clinical_models.ExamOrder {
	t.Helper()
	if resp.Code != wantCode {
		t.Fatalf("got %d (%s, %+v), want %d", resp.Code, resp.Message, resp.Data, wantCode)
	}
	switch d := resp.Data.(type) {
	case *clinical_models.ExamOrder:
		return d
	case ExamResultUpload:
		return d.Order
	}
	t.Fatalf("data is %T, want *ExamOrder", resp.Data)
	return nil
}

func wantCode(t *testing.T, resp envelope.Response, code int) {
	t.Helper()
	if resp.Code != code {
		t.Fatalf("got %d (%s, %+v), want %d", resp.Code, resp.Message, resp.Data, code)
	}
}

func ptr[T any](v T) *T { return &v }

// createOrder creates an order in "own" with a catalog exam (default
// indications) and an exam written by hand.
func (f *examFixture) createOrder() *clinical_models.ExamOrder {
	glucose := f.catalogItem(f.own, "Glucosa en ayunas")
	req := clinical_dto.CreateExamOrderRequest{
		PatientID: f.ownPatient.ID,
		Diagnoses: []clinical_models.DiagnosisItem{{Code: "E11.9", Title: "Diabetes mellitus tipo 2"}},
		Priority:  "urgent",
		Items: []clinical_dto.ExamOrderItemInput{
			{CatalogItemID: &glucose.ID},
			{Name: "Prueba especial", Category: "other", Indications: ptr("Traer orden impresa.")},
		},
	}
	return order(f.t, f.h.CreateExamOrder(f.ctx(f.own, examDoctorID, req)), http.StatusCreated)
}

type upload struct {
	name    string
	content []byte
}

// upload posts a result file (none if files is empty) covering itemIDs, as
// userID of tenantID.
func (f *examFixture) upload(tenantID, userID, orderID uint, itemIDs []uint, files ...upload) envelope.Response {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for _, id := range itemIDs {
		_ = w.WriteField("item_ids", fmt.Sprint(id))
	}
	for _, file := range files {
		part, _ := w.CreateFormFile("file", file.name)
		_, _ = part.Write(file.content)
	}
	_ = w.Close()
	c := f.ctx(tenantID, userID, nil, "id", fmt.Sprint(orderID))
	c.Request = httptest.NewRequest(http.MethodPost, "/", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	return f.h.UploadExamResult(c)
}

func pngFile() upload { return upload{"resultado.png", []byte(pngResultContent)} }

var deleteReason = map[string]string{"reason": "Archivo equivocado"}

// deleteResult deletes a result through the exam order route.
func (f *examFixture) deleteResult(tenantID, orderID, attachmentID uint) envelope.Response {
	return f.h.DeleteExamResult(f.ctx(tenantID, examAssistantID, deleteReason, "id", fmt.Sprint(orderID), "attachmentId", fmt.Sprint(attachmentID)))
}

// deleteAttachment and restoreAttachment go through the patient's attachment routes.
func (f *examFixture) deleteAttachment(tenantID, patientID, attachmentID uint) envelope.Response {
	return f.att.DeleteAttachment(f.ctx(tenantID, examAssistantID, deleteReason, "id", fmt.Sprint(patientID), "attachment_id", fmt.Sprint(attachmentID)))
}

func (f *examFixture) restoreAttachment(tenantID, patientID, attachmentID uint) envelope.Response {
	return f.att.RestoreAttachment(f.ctx(tenantID, examAssistantID, nil, "id", fmt.Sprint(patientID), "attachment_id", fmt.Sprint(attachmentID)))
}

func (f *examFixture) status(orderID uint) string {
	var o clinical_models.ExamOrder
	f.db.First(&o, orderID)
	return o.Status
}

func (f *examFixture) update(o *clinical_models.ExamOrder, mutate func(*clinical_dto.UpdateExamOrderRequest)) envelope.Response {
	var diagnoses []clinical_models.DiagnosisItem
	_ = json.Unmarshal(o.Diagnoses, &diagnoses)
	req := clinical_dto.UpdateExamOrderRequest{
		MedicalRecordID: o.MedicalRecordID, Diagnoses: diagnoses, Priority: o.Priority,
		Notes: o.Notes, DestinationLab: o.DestinationLab,
	}
	for _, item := range o.Items {
		req.Items = append(req.Items, clinical_dto.ExamOrderItemInput{
			ID: ptr(item.ID), CatalogItemID: item.CatalogItemID, Name: item.Name, Category: item.Category,
			Subgroup: item.Subgroup, Indications: ptr(item.Indications),
		})
	}
	mutate(&req)
	return f.h.UpdateExamOrder(f.ctx(f.own, examDoctorID, req, "id", fmt.Sprint(o.ID)))
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestExamOrder_CreateCopiesCatalogAndNumbersPerTenant(t *testing.T) {
	f := newExamFixture(t)
	first := f.createOrder()
	if first.Number != 1 || first.Code != "ORD-000001" || first.Status != clinical_models.ExamOrderStatusIssued {
		t.Fatalf("first order: number %d code %q status %q", first.Number, first.Code, first.Status)
	}
	if first.OrderedByID != examDoctorID || first.OrderedByName != "dra.andrade" || first.Priority != "urgent" {
		t.Fatalf("ordered by %d %q priority %q", first.OrderedByID, first.OrderedByName, first.Priority)
	}
	glucose := first.Items[0]
	if glucose.Name != "Glucosa en ayunas" || glucose.Category != "laboratory" || glucose.Subgroup != "Química sanguínea" || glucose.Indications != "Ayuno de 8 horas." {
		t.Fatalf("catalog exam not copied: %+v", glucose)
	}
	if manual := first.Items[1]; manual.CatalogItemID != nil || manual.Category != "other" || manual.Indications != "Traer orden impresa." {
		t.Fatalf("manual exam: %+v", manual)
	}
	if second := f.createOrder(); second.Number != 2 {
		t.Fatalf("second order number = %d, want 2", second.Number)
	}

	// The other tenant has its own sequence.
	otherGlucose := f.catalogItem(f.other, "Glucosa en ayunas")
	resp := f.h.CreateExamOrder(f.ctx(f.other, examDoctorID, clinical_dto.CreateExamOrderRequest{
		PatientID: f.otherPatient.ID, Items: []clinical_dto.ExamOrderItemInput{{CatalogItemID: &otherGlucose.ID}},
	}))
	if o := order(t, resp, http.StatusCreated); o.Number != 1 || o.Priority != "routine" {
		t.Fatalf("other tenant order: number %d priority %q", o.Number, o.Priority)
	}
}

func TestExamOrder_CorrelativeIsUniqueUnderConcurrency(t *testing.T) {
	f := newExamFixture(t)
	if f.db.Dialector.Name() == "sqlite" {
		t.Skip("in-memory SQLite gives each connection its own database; needs Postgres")
	}
	const n = 12
	var wg sync.WaitGroup
	numbers := make(chan uint, n)
	errs := make(chan string, n)
	glucose := f.catalogItem(f.own, "Glucosa en ayunas")
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := f.h.CreateExamOrder(f.ctx(f.own, examDoctorID, clinical_dto.CreateExamOrderRequest{
				PatientID: f.ownPatient.ID, Items: []clinical_dto.ExamOrderItemInput{{CatalogItemID: &glucose.ID}},
			}))
			if resp.Code != http.StatusCreated {
				errs <- fmt.Sprintf("%d %s %+v", resp.Code, resp.Message, resp.Data)
				return
			}
			numbers <- resp.Data.(*clinical_models.ExamOrder).Number
		}()
	}
	wg.Wait()
	close(numbers)
	close(errs)
	for e := range errs {
		t.Errorf("create failed: %s", e)
	}
	var got []int
	for number := range numbers {
		got = append(got, int(number))
	}
	sort.Ints(got)
	for i, number := range got {
		if number != i+1 {
			t.Fatalf("numbers %v are not 1..%d without gaps or duplicates", got, n)
		}
	}
}

func TestExamOrder_EditWithoutResultsClearsSignature(t *testing.T) {
	f := newExamFixture(t)
	o := f.createOrder()
	signedAt := time.Now()
	f.db.Model(&clinical_models.ExamOrder{}).Where("id = ?", o.ID).Updates(map[string]any{
		"signed_by_id": examDoctorID, "signed_at": signedAt, "signer_name": "DRA", "signed_file": "signed/x.pdf",
	})
	tsh := f.catalogItem(f.own, "TSH")
	// Renaming or deleting the catalog exam later doesn't touch the order's copy.
	f.db.Model(&clinical_models.ExamCatalogItem{}).Where("id = ?", *o.Items[0].CatalogItemID).Update("name", "Glucosa basal")
	f.db.Delete(&clinical_models.ExamCatalogItem{}, *o.Items[0].CatalogItemID)

	resp := f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) {
		req.Notes = "Control trimestral"
		req.Items[0].Indications = ptr("Ayuno de 10 horas.")
		req.Items = append(req.Items[:1], clinical_dto.ExamOrderItemInput{CatalogItemID: &tsh.ID}) // drop the manual one, add TSH
	})
	updated := order(t, resp, http.StatusOK)
	if updated.IsSigned() || updated.SignedByID != nil || updated.SignerName != "" {
		t.Fatalf("signature not cleared: %+v", updated.DocumentSignature)
	}
	if updated.Notes != "Control trimestral" || len(updated.Items) != 2 || updated.Items[0].Indications != "Ayuno de 10 horas." ||
		updated.Items[0].Name != "Glucosa en ayunas" || updated.Items[1].Name != "TSH" {
		t.Fatalf("edit not applied: notes %q items %+v", updated.Notes, updated.Items)
	}
}

func TestExamOrder_ResultsDriveStatusAndLockEditing(t *testing.T) {
	f := newExamFixture(t)
	o := f.createOrder()
	first, second := o.Items[0].ID, o.Items[1].ID

	resp := f.upload(f.own, examAssistantID, o.ID, []uint{first}, pngFile())
	o = order(t, resp, http.StatusCreated)
	created := resp.Data.(ExamResultUpload).Attachment
	if o.Status != clinical_models.ExamOrderStatusPartialResults || !o.PendingReview {
		t.Fatalf("after first result: status %q pending %v", o.Status, o.PendingReview)
	}
	// The result is an Adjunto of the order's patient: lab result (the exam is
	// laboratory), dated today, stored encrypted.
	if file := o.Items[0].Attachments[0]; file.ID != created.ID || file.MimeType != "image/png" || file.PatientID != f.ownPatient.ID ||
		file.Category != clinical_models.AttachmentCategoryLabResult || file.TakenAt.IsZero() {
		t.Fatalf("result attachment: %+v", file)
	}
	if raw, _ := f.disk.Read(f.own, created.StoredName); bytes.Equal(raw, []byte(pngResultContent)) {
		t.Fatal("result stored unencrypted")
	}
	if plain, err := f.attachments.Read(f.own, created.StoredName); err != nil || string(plain) != pngResultContent {
		t.Fatalf("result does not decrypt: %v", err)
	}

	// With results: nothing can be removed or changed...
	wantCode(t, f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) { req.Items = req.Items[:1] }), http.StatusConflict)
	wantCode(t, f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) { req.Notes = "otra" }), http.StatusConflict)
	wantCode(t, f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) { req.Items[1].Indications = ptr("x") }), http.StatusConflict)
	// ...but exams can be added.
	o = order(t, f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) {
		req.Items = append(req.Items, clinical_dto.ExamOrderItemInput{Name: "Ecografía especial", Category: "imaging"})
	}), http.StatusOK)
	if len(o.Items) != 3 || o.Status != clinical_models.ExamOrderStatusPartialResults {
		t.Fatalf("after adding: %d items status %q", len(o.Items), o.Status)
	}
	third := o.Items[2].ID

	// One file covering two exams completes the order.
	o = order(t, f.upload(f.own, examAssistantID, o.ID, []uint{second, third}, upload{"informe.pdf", []byte(pdfResultContent)}), http.StatusCreated)
	if o.Status != clinical_models.ExamOrderStatusCompleteResults {
		t.Fatalf("after covering all: status %q", o.Status)
	}
	pdfFileID := o.Items[1].Attachments[0].ID

	// Deleting a result goes back down, and is audited.
	wantCode(t, f.h.DeleteExamResult(f.ctx(f.own, examAssistantID, nil, "id", fmt.Sprint(o.ID), "attachmentId", fmt.Sprint(pdfFileID))), http.StatusBadRequest) // reason required
	o = order(t, f.deleteResult(f.own, o.ID, pdfFileID), http.StatusOK)
	if o.Status != clinical_models.ExamOrderStatusPartialResults {
		t.Fatalf("after delete: status %q", o.Status)
	}
	var deleted clinical_models.PatientAttachment
	f.db.Unscoped().First(&deleted, pdfFileID)
	if !deleted.DeletedAt.Valid || deleted.DeleteReason != "Archivo equivocado" || deleted.DeletedByID == nil {
		t.Fatalf("result not soft-deleted with who and why: %+v", deleted)
	}
	var audited int64
	f.db.Model(&audit.AuditLog{}).Where("action = ? AND entity_type = ? AND entity_id = ?", "UPDATE", "patient_attachments", pdfFileID).Count(&audited)
	if audited != 1 {
		t.Fatalf("result deletion audited %d times, want 1", audited)
	}
	// Restoring it from the patient's attachments brings the status back up.
	wantCode(t, f.restoreAttachment(f.own, f.ownPatient.ID, pdfFileID), http.StatusOK)
	if got := f.status(o.ID); got != clinical_models.ExamOrderStatusCompleteResults {
		t.Fatalf("after restore: status %q", got)
	}
	// Deleting it from the patient's attachments brings it down again.
	wantCode(t, f.deleteAttachment(f.own, f.ownPatient.ID, pdfFileID), http.StatusOK)
	if got := f.status(o.ID); got != clinical_models.ExamOrderStatusPartialResults {
		t.Fatalf("after attachment delete: status %q", got)
	}
	o = order(t, f.deleteResult(f.own, o.ID, created.ID), http.StatusOK)
	if o.Status != clinical_models.ExamOrderStatusIssued {
		t.Fatalf("after deleting every result: status %q", o.Status)
	}

	// Closing by hand completes it; voiding freezes it.
	o = order(t, f.h.CloseExamOrder(f.ctx(f.own, examDoctorID, nil, "id", fmt.Sprint(o.ID))), http.StatusOK)
	if o.Status != clinical_models.ExamOrderStatusCompleteResults || !o.ClosedManually {
		t.Fatalf("after close: status %q closed %v", o.Status, o.ClosedManually)
	}
	wantCode(t, f.h.VoidExamOrder(f.ctx(f.own, examDoctorID, map[string]string{"reason": " "}, "id", fmt.Sprint(o.ID))), http.StatusBadRequest)
	o = order(t, f.h.VoidExamOrder(f.ctx(f.own, examDoctorID, map[string]string{"reason": "Paciente no acudió"}, "id", fmt.Sprint(o.ID))), http.StatusOK)
	if o.Status != clinical_models.ExamOrderStatusVoided || o.VoidReason != "Paciente no acudió" || o.VoidedByID == nil || *o.VoidedByID != examDoctorID {
		t.Fatalf("after void: %+v", o)
	}
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, []uint{first}, pngFile()), http.StatusConflict)
	wantCode(t, f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) {}), http.StatusConflict)
	wantCode(t, f.h.CloseExamOrder(f.ctx(f.own, examDoctorID, nil, "id", fmt.Sprint(o.ID))), http.StatusConflict)
}

func TestExamOrder_ReviewBlocksResultDeletion(t *testing.T) {
	f := newExamFixture(t)
	o := f.createOrder()
	first, second := o.Items[0].ID, o.Items[1].ID

	// Only exams with a result can be reviewed.
	wantCode(t, f.h.ReviewExamResults(f.ctx(f.own, examDoctorID, map[string]any{"item_ids": []uint{first}}, "id", fmt.Sprint(o.ID))), http.StatusConflict)

	o = order(t, f.upload(f.own, examAssistantID, o.ID, []uint{first}, pngFile()), http.StatusCreated)
	var notifications []notifications_models.Notification
	f.db.Where("tenant_id = ? AND user_id = ?", f.own, examDoctorID).Find(&notifications)
	if len(notifications) != 1 || notifications[0].MessageKey != "notification.clinical.exam_order.results_uploaded" || notifications[0].ResourceID != o.ID {
		t.Fatalf("ordering doctor not notified: %+v", notifications)
	}

	o = order(t, f.h.ReviewExamResults(f.ctx(f.own, examDoctorID, map[string]any{"item_ids": []uint{first}}, "id", fmt.Sprint(o.ID))), http.StatusOK)
	if o.Items[0].ReviewedAt == nil || o.Items[0].ReviewedByID == nil || *o.Items[0].ReviewedByID != examDoctorID || o.Items[0].ReviewedByName != "dra.andrade" || o.PendingReview {
		t.Fatalf("review not recorded: %+v pending %v", o.Items[0], o.PendingReview)
	}
	// A reviewed result can't be deleted through either route.
	reviewedID := o.Items[0].Attachments[0].ID
	wantCode(t, f.deleteResult(f.own, o.ID, reviewedID), http.StatusConflict)
	wantCode(t, f.deleteAttachment(f.own, f.ownPatient.ID, reviewedID), http.StatusConflict)
	var still clinical_models.PatientAttachment
	if err := f.db.First(&still, reviewedID).Error; err != nil {
		t.Fatalf("reviewed result was deleted: %v", err)
	}
	// A plain attachment of the patient (no exam) still deletes normally.
	plain := clinical_models.PatientAttachment{TenantID: f.own, PatientID: f.ownPatient.ID, Category: "other", FileName: "x.pdf", StoredName: "attachments/x"}
	f.db.Create(&plain)
	wantCode(t, f.deleteAttachment(f.own, f.ownPatient.ID, plain.ID), http.StatusOK)
	// An item of another order is rejected.
	wantCode(t, f.h.ReviewExamResults(f.ctx(f.own, examDoctorID, map[string]any{"item_ids": []uint{second + 1000}}, "id", fmt.Sprint(o.ID))), http.StatusBadRequest)
}

func TestExamResult_FileValidation(t *testing.T) {
	f := newExamFixture(t)
	o := f.createOrder()
	item := []uint{o.Items[0].ID}

	// Text disguised as a PDF (name and part Content-Type) is rejected by content.
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, item, upload{"falso.pdf", []byte("hola, esto no es un PDF")}), http.StatusUnsupportedMediaType)
	// HTML is rejected even if named like an image.
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, item, upload{"x.png", []byte("<html><script>alert(1)</script></html>")}), http.StatusUnsupportedMediaType)
	// WEBP is not an accepted attachment type.
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, item, upload{"foto.webp", []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")}), http.StatusUnsupportedMediaType)
	// Over 15 MB.
	big := append([]byte(pdfResultContent), make([]byte, MaxAttachmentSize)...)
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, item, upload{"grande.pdf", big}), http.StatusRequestEntityTooLarge)
	// No file, no exams, or exams of no order.
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, item), http.StatusBadRequest)
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, nil, pngFile()), http.StatusBadRequest)
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, []uint{999999}, pngFile()), http.StatusBadRequest)

	var stored int64
	f.db.Model(&clinical_models.PatientAttachment{}).Count(&stored)
	if stored != 0 {
		t.Fatalf("rejected uploads stored %d files", stored)
	}

	// A JPEG is accepted; the same content again is stored with a duplicate warning.
	jpeg := upload{"foto.jpg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")}
	first := f.upload(f.own, examAssistantID, o.ID, item, jpeg)
	o = order(t, first, http.StatusCreated)
	if d := first.Data.(ExamResultUpload); d.DuplicateOf != nil || d.Attachment.MimeType != "image/jpeg" {
		t.Fatalf("first upload: %+v", d)
	}
	again := f.upload(f.own, examAssistantID, o.ID, item, jpeg)
	o = order(t, again, http.StatusCreated)
	if d := again.Data.(ExamResultUpload); d.DuplicateOf == nil || d.DuplicateOf.ID != first.Data.(ExamResultUpload).Attachment.ID {
		t.Fatalf("duplicate not flagged: %+v", d.DuplicateOf)
	}
	if len(o.Items[0].Attachments) != 2 {
		t.Fatalf("item has %d results, want 2", len(o.Items[0].Attachments))
	}

	// Over the plan quota.
	f.db.Model(&company_models.Plan{}).Where("code LIKE ?", fmt.Sprintf("EXAM-%d-%%", f.own)).Update("storage_quota_mb", 0)
	wantCode(t, f.upload(f.own, examAssistantID, o.ID, item, pngFile()), http.StatusForbidden)
}

func TestExamOrder_TenantIsolation(t *testing.T) {
	f := newExamFixture(t)
	mine := f.createOrder()

	// An order of the other clinic, with a result.
	otherGlucose := f.catalogItem(f.other, "Glucosa en ayunas")
	theirs := order(t, f.h.CreateExamOrder(f.ctx(f.other, examDoctorID, clinical_dto.CreateExamOrderRequest{
		PatientID: f.otherPatient.ID, Items: []clinical_dto.ExamOrderItemInput{{CatalogItemID: &otherGlucose.ID}},
	})), http.StatusCreated)
	theirs = order(t, f.upload(f.other, examAssistantID, theirs.ID, []uint{theirs.Items[0].ID}, pngFile()), http.StatusCreated)
	theirFile := theirs.Items[0].Attachments[0]
	theirID := fmt.Sprint(theirs.ID)

	wantCode(t, f.h.GetExamOrder(f.ctx(f.own, examDoctorID, nil, "id", theirID)), http.StatusNotFound)
	wantCode(t, f.h.VoidExamOrder(f.ctx(f.own, examDoctorID, map[string]string{"reason": "x"}, "id", theirID)), http.StatusNotFound)
	wantCode(t, f.h.CloseExamOrder(f.ctx(f.own, examDoctorID, nil, "id", theirID)), http.StatusNotFound)
	wantCode(t, f.h.UpdateExamOrder(f.ctx(f.own, examDoctorID, clinical_dto.UpdateExamOrderRequest{Items: []clinical_dto.ExamOrderItemInput{{Name: "x"}}}, "id", theirID)), http.StatusNotFound)
	wantCode(t, f.upload(f.own, examAssistantID, theirs.ID, []uint{theirs.Items[0].ID}, pngFile()), http.StatusNotFound)
	wantCode(t, f.h.ReviewExamResults(f.ctx(f.own, examDoctorID, map[string]any{"item_ids": []uint{theirs.Items[0].ID}}, "id", theirID)), http.StatusNotFound)
	wantCode(t, f.deleteResult(f.own, theirs.ID, theirFile.ID), http.StatusNotFound)
	// Their result through my order's URL, or my patient's attachments, too.
	wantCode(t, f.deleteResult(f.own, mine.ID, theirFile.ID), http.StatusNotFound)
	wantCode(t, f.deleteAttachment(f.own, f.ownPatient.ID, theirFile.ID), http.StatusNotFound)
	// A result of my patient that is not a result of this order.
	mineResult := f.upload(f.own, examAssistantID, mine.ID, []uint{mine.Items[0].ID}, pngFile())
	wantCode(t, mineResult, http.StatusCreated)
	other := f.createOrder()
	wantCode(t, f.deleteResult(f.own, other.ID, mineResult.Data.(ExamResultUpload).Attachment.ID), http.StatusNotFound)

	// Their patient, their catalog exam, their exam in my order.
	wantCode(t, f.h.CreateExamOrder(f.ctx(f.own, examDoctorID, clinical_dto.CreateExamOrderRequest{
		PatientID: f.otherPatient.ID, Items: []clinical_dto.ExamOrderItemInput{{Name: "x"}},
	})), http.StatusNotFound)
	wantCode(t, f.h.CreateExamOrder(f.ctx(f.own, examDoctorID, clinical_dto.CreateExamOrderRequest{
		PatientID: f.ownPatient.ID, Items: []clinical_dto.ExamOrderItemInput{{CatalogItemID: &otherGlucose.ID}},
	})), http.StatusBadRequest)
	wantCode(t, f.update(mine, func(req *clinical_dto.UpdateExamOrderRequest) {
		req.Items = append(req.Items, clinical_dto.ExamOrderItemInput{ID: ptr(theirs.Items[0].ID), Name: "x"})
	}), http.StatusBadRequest)

	// Listing shows only mine.
	resp := f.h.GetExamOrders(f.ctx(f.own, examDoctorID, nil))
	wantCode(t, resp, http.StatusOK)
	paged := resp.Data.(envelope.PagedData)
	list := paged.Items.([]*clinical_models.ExamOrder)
	if paged.Total != 2 || len(list) != 2 {
		t.Fatalf("list has %d orders, want my 2", paged.Total)
	}
	for _, o := range list {
		if o.TenantID != f.own || o.ID == theirs.ID {
			t.Fatalf("list leaked other tenant's order %d", o.ID)
		}
	}

	// Their result is still theirs, untouched.
	var theirsNow clinical_models.PatientAttachment
	if err := f.db.First(&theirsNow, theirFile.ID).Error; err != nil || theirsNow.TenantID != f.other {
		t.Fatalf("their result changed: %v", err)
	}

	// Catalog: their exams are neither listed nor editable.
	wantCode(t, f.catalog.UpdateExamCatalogItem(f.ctx(f.own, examDoctorID, map[string]string{"name": "x"}, "id", fmt.Sprint(otherGlucose.ID))), http.StatusNotFound)
	wantCode(t, f.catalog.DeleteExamCatalogItem(f.ctx(f.own, examDoctorID, nil, "id", fmt.Sprint(otherGlucose.ID))), http.StatusNotFound)
	wantCode(t, f.catalog.CreateExamProfile(f.ctx(f.own, examDoctorID, map[string]any{"name": "x", "item_ids": []uint{otherGlucose.ID}})), http.StatusBadRequest)
	resp = f.catalog.GetExamCatalog(f.ctx(f.own, examDoctorID, nil))
	if items := resp.Data.([]clinical_models.ExamCatalogItem); len(items) != 170 {
		t.Fatalf("own catalog has %d exams, want 170", len(items))
	}
}

func TestExamCatalog_RestoreDefaultsKeepsTenantExams(t *testing.T) {
	f := newExamFixture(t)
	tsh := f.catalogItem(f.own, "TSH")
	urea := f.catalogItem(f.own, "Urea")

	wantCode(t, f.catalog.DeleteExamCatalogItem(f.ctx(f.own, examDoctorID, nil, "id", fmt.Sprint(tsh.ID))), http.StatusOK)
	wantCode(t, f.catalog.UpdateExamCatalogItem(f.ctx(f.own, examDoctorID, map[string]any{"active": false, "name": "Urea sérica"}, "id", fmt.Sprint(urea.ID))), http.StatusOK)
	wantCode(t, f.catalog.CreateExamCatalogItem(f.ctx(f.own, examDoctorID, map[string]any{"name": "Panel propio", "category": "laboratory", "subgroup": "Especiales"})), http.StatusCreated)
	var thyroid clinical_models.ExamProfile
	f.db.Where("tenant_id = ? AND name = ?", f.own, "Perfil tiroideo").First(&thyroid)
	wantCode(t, f.catalog.DeleteExamProfile(f.ctx(f.own, examDoctorID, nil, "id", fmt.Sprint(thyroid.ID))), http.StatusOK)

	resp := f.catalog.RestoreExamCatalogDefaults(f.ctx(f.own, examDoctorID, nil))
	wantCode(t, resp, http.StatusOK)
	if res := resp.Data.(clinical_dto.RestoreExamCatalogResponse); res.RestoredItems != 2 || res.RestoredProfiles != 1 {
		t.Fatalf("restored %+v, want 2 exams and 1 profile", res)
	}
	var restoredUrea clinical_models.ExamCatalogItem
	f.db.First(&restoredUrea, urea.ID)
	if !restoredUrea.Active || restoredUrea.Name != "Urea sérica" {
		t.Fatalf("urea: %+v (re-activated, keeping the tenant's name)", restoredUrea)
	}
	resp = f.catalog.GetExamCatalog(f.ctx(f.own, examDoctorID, nil))
	if items := resp.Data.([]clinical_models.ExamCatalogItem); len(items) != 171 {
		t.Fatalf("catalog has %d exams, want 170 seeded + 1 own", len(items))
	}
	resp = f.catalog.GetExamProfiles(f.ctx(f.own, examDoctorID, nil))
	for _, p := range resp.Data.([]clinical_models.ExamProfile) {
		if p.Name == "Perfil tiroideo" && len(p.Items) != 3 {
			t.Fatalf("restored profile has %d exams, want 3 (TSH included)", len(p.Items))
		}
	}

	// Seeding again is a no-op.
	if err := clinical_services.SeedExamCatalog(f.db, f.own); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	var count int64
	f.db.Model(&clinical_models.ExamCatalogItem{}).Where("tenant_id = ?", f.own).Count(&count)
	if count != 171 {
		t.Fatalf("re-seed changed the catalog: %d exams", count)
	}
}

func TestExamResult_LinkedToTheOrdersConsultation(t *testing.T) {
	f := newExamFixture(t)
	record := clinical_models.MedicalRecord{TenantID: f.own, PatientID: f.ownPatient.ID, Date: time.Now()}
	if err := f.db.Create(&record).Error; err != nil {
		t.Fatalf("create record: %v", err)
	}
	ecg := f.catalogItem(f.own, "Electrocardiograma")
	o := order(t, f.h.CreateExamOrder(f.ctx(f.own, examDoctorID, clinical_dto.CreateExamOrderRequest{
		PatientID: f.ownPatient.ID, MedicalRecordID: &record.ID, Items: []clinical_dto.ExamOrderItemInput{{CatalogItemID: &ecg.ID}},
	})), http.StatusCreated)

	resp := f.upload(f.own, examAssistantID, o.ID, []uint{o.Items[0].ID}, upload{"ecg.pdf", []byte(pdfResultContent)})
	wantCode(t, resp, http.StatusCreated)
	a := resp.Data.(ExamResultUpload).Attachment
	if a.MedicalRecordID == nil || *a.MedicalRecordID != record.ID || a.PatientID != f.ownPatient.ID ||
		a.Category != clinical_models.AttachmentCategoryExternalReport || a.MimeType != "application/pdf" {
		t.Fatalf("result attachment: %+v", a)
	}
}

// An edit that starts while a result upload holds the order (same row lock)
// must see that result once it gets the lock, and be refused.
func TestExamOrder_EditWaitsForConcurrentResultAndRespectsLock(t *testing.T) {
	f := newExamFixture(t)
	if f.db.Dialector.Name() == "sqlite" {
		t.Skip("row locks need Postgres")
	}
	o := f.createOrder()

	// An upload in progress: it holds the lock and has linked a result.
	upload := tenantdb.ForTenant(f.db, f.own).Begin()
	defer upload.Rollback()
	if _, err := lockExamOrder(upload, o.ID); err != nil {
		t.Fatalf("lock: %v", err)
	}
	result := clinical_models.PatientAttachment{TenantID: f.own, PatientID: f.ownPatient.ID, Category: "lab_result", FileName: "r.pdf", StoredName: "attachments/r"}
	if err := upload.Create(&result).Error; err != nil {
		t.Fatalf("create result: %v", err)
	}
	link := map[string]any{"exam_order_item_id": o.Items[0].ID, "patient_attachment_id": result.ID}
	if err := upload.Table("exam_order_item_attachments").Create(&link).Error; err != nil {
		t.Fatalf("link result: %v", err)
	}

	// Meanwhile, an edit that removes an exam (allowed only without results).
	done := make(chan envelope.Response, 1)
	go func() {
		done <- f.update(o, func(req *clinical_dto.UpdateExamOrderRequest) { req.Items = req.Items[:1] })
	}()
	select {
	case resp := <-done:
		t.Fatalf("edit did not wait for the upload's lock: %d", resp.Code)
	case <-time.After(300 * time.Millisecond):
	}
	if err := upload.Commit().Error; err != nil {
		t.Fatalf("commit upload: %v", err)
	}
	select {
	case resp := <-done:
		wantCode(t, resp, http.StatusConflict)
	case <-time.After(10 * time.Second):
		t.Fatal("edit never finished")
	}
	var items int64
	f.db.Model(&clinical_models.ExamOrderItem{}).Where("exam_order_id = ?", o.ID).Count(&items)
	if items != 2 {
		t.Fatalf("order has %d exams after the refused edit, want 2", items)
	}
}

func TestExamResult_AttachmentListFlagsReviewedResults(t *testing.T) {
	f := newExamFixture(t)
	o := f.createOrder()
	first, second := o.Items[0].ID, o.Items[1].ID
	o = order(t, f.upload(f.own, examAssistantID, o.ID, []uint{first}, pngFile()), http.StatusCreated)
	o = order(t, f.upload(f.own, examAssistantID, o.ID, []uint{second}, upload{"b.pdf", []byte(pdfResultContent)}), http.StatusCreated)
	order(t, f.h.ReviewExamResults(f.ctx(f.own, examDoctorID, map[string]any{"item_ids": []uint{first}}, "id", fmt.Sprint(o.ID))), http.StatusOK)
	reviewedID := o.Items[0].Attachments[0].ID

	resp := f.att.ListAttachments(f.ctx(f.own, examAssistantID, nil, "id", fmt.Sprint(f.ownPatient.ID)))
	wantCode(t, resp, http.StatusOK)
	list, ok := resp.Data.(AttachmentList)
	if !ok || len(list.Items) != 2 {
		t.Fatalf("unexpected list: %#v", resp.Data)
	}
	for _, item := range list.Items {
		if item.LockedByReview != (item.ID == reviewedID) {
			t.Fatalf("attachment %d locked_by_review = %v", item.ID, item.LockedByReview)
		}
	}
}
