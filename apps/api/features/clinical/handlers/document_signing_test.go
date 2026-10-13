package clinical_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digitorus/pdfsign/verify"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/pdfsign/pdfsigntest"
	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	signature_models "pengi-med-saas/features/signatures/models"
	signature_services "pengi-med-saas/features/signatures/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

// pdfConverter stands in for Gotenberg: it records the HTML and returns a real PDF.
type pdfConverter struct {
	calls int
	html  string
}

func (p *pdfConverter) GeneratePDFFromHTMLWithOptions(html string, _ utils.PDFOptions) ([]byte, error) {
	p.calls++
	p.html = html
	return pdfsigntest.PDF(), nil
}

type signingFixture struct {
	t       *testing.T
	db      *gorm.DB
	files   *tenantfiles.MemoryStore
	conv    *pdfConverter
	docs    *MedicalDocumentHandler
	records *MedicalRecordHandler
	tenant  tenant_models.Tenant
	patient clinical_models.Patient
	userID  int64
}

func newSigningFixture(t *testing.T) *signingFixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &company_models.Company{}, &clinical_models.Patient{},
		&clinical_models.MedicalRecord{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{},
		&clinical_models.MedicalReport{}, &clinical_models.MedicalCertificate{}, &signature_models.UserSignature{})
	files := tenantfiles.Memory()
	conv := &pdfConverter{}
	renderer := pdfrender.New(files, conv)
	signer := signature_services.NewSigner(db, files)

	now := time.Now().UnixNano()
	f := &signingFixture{t: t, db: db, files: files, conv: conv, userID: now % 1_000_000,
		docs:    NewMedicalDocumentHandler(db, zap.NewNop(), nil, renderer, signer, files),
		records: NewMedicalRecordHandler(db, zap.NewNop()),
	}
	f.tenant = tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("sign-%d", now), DisplayToken: fmt.Sprintf("tok-sign-%d", now)}
	if err := db.Create(&f.tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	f.patient = clinical_models.Patient{TenantID: f.tenant.ID, FirstName: "Ana", LastName: "Paciente", Institution: "H", Document: fmt.Sprintf("DOC-%d", now)}
	if err := db.Create(&f.patient).Error; err != nil {
		t.Fatalf("create patient: %v", err)
	}
	return f
}

// beDoctor links the fixture user to a doctor profile: legacy documents
// without a doctor can only be signed by a user with one.
func (f *signingFixture) beDoctor() {
	uid := uint(f.userID)
	f.doctor("Ana Perez", &uid)
}

// giveSignature stores a valid P12 for the fixture user, as the upload endpoint would.
func (f *signingFixture) giveSignature(name string) {
	f.t.Helper()
	p12 := pdfsigntest.P12(f.t, name, "clave", time.Now().Add(365*24*time.Hour))
	box, err := secretbox.FromEnv()
	if err != nil {
		f.t.Fatal(err)
	}
	enc, _ := box.Seal("clave")
	fileName := signature_services.P12FileName(uint(f.userID))
	if err := f.files.Write(f.tenant.ID, fileName, p12); err != nil {
		f.t.Fatal(err)
	}
	sig := signature_models.UserSignature{TenantID: f.tenant.ID, UserID: uint(f.userID), FileName: fileName, EncryptedPassword: enc, SubjectName: name, NotAfter: time.Now().Add(365 * 24 * time.Hour)}
	if err := f.db.Create(&sig).Error; err != nil {
		f.t.Fatal(err)
	}
}

func (f *signingFixture) ctx(id uint, body any) (*gin.Context, *httptest.ResponseRecorder) {
	c, w := testutils.NewGinContext(f.tenant.ID, f.userID)
	c.Set("user_id", f.userID)
	c.Set("username", "doctor")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func (f *signingFixture) report() clinical_models.MedicalReport {
	r := clinical_models.MedicalReport{TenantID: f.tenant.ID, PatientID: f.patient.ID, Plan: "Control en 1 mes"}
	if err := f.db.Create(&r).Error; err != nil {
		f.t.Fatalf("create report: %v", err)
	}
	return r
}

func errorCode(t *testing.T, data any) string {
	t.Helper()
	appErr, ok := data.(core_errors.AppError)
	if !ok {
		t.Fatalf("response data %T is not an AppError", data)
	}
	return appErr.ErrorCode
}

func TestSignMedicalReport_StoresVerifiableSignedPDFWithStamp(t *testing.T) {
	f := newSigningFixture(t)
	f.beDoctor()
	f.giveSignature("DRA ANA PEREZ")
	report := f.report()

	c, _ := f.ctx(report.ID, nil)
	resp := f.docs.SignMedicalReport(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("sign code = %d, data = %+v", resp.Code, resp.Data)
	}
	if !strings.Contains(f.conv.html, "Firmado electrónicamente por:") || !strings.Contains(f.conv.html, "DRA ANA PEREZ") {
		t.Fatal("rendered HTML has no signature stamp")
	}

	var saved clinical_models.MedicalReport
	if err := f.db.Unscoped().First(&saved, report.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !saved.IsSigned() || saved.SignerName != "DRA ANA PEREZ" || saved.SignedByID == nil || *saved.SignedByID != uint(f.userID) {
		t.Fatalf("saved signature = %+v", saved.DocumentSignature)
	}
	pdf, err := f.files.Read(f.tenant.ID, saved.SignedFile)
	if err != nil {
		t.Fatalf("signed pdf: %v", err)
	}
	result, err := verify.Verify(bytes.NewReader(pdf), int64(len(pdf)))
	if err != nil || len(result.Signers) != 1 || !result.Signers[0].ValidSignature {
		t.Fatalf("stored PDF does not verify: %v %+v", err, result)
	}

	// Download serves the stored signed PDF, without re-rendering.
	calls := f.conv.calls
	dc, w := f.ctx(report.ID, nil)
	f.docs.DownloadMedicalReport(dc)
	if w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), pdf) || f.conv.calls != calls {
		t.Fatalf("download did not serve the stored signed PDF (code %d, renders %d→%d)", w.Code, calls, f.conv.calls)
	}
}

func TestSignMedicalReport_TwiceIsConflict(t *testing.T) {
	f := newSigningFixture(t)
	f.beDoctor()
	f.giveSignature("DR X")
	report := f.report()

	c, _ := f.ctx(report.ID, nil)
	f.docs.SignMedicalReport(c)
	c2, _ := f.ctx(report.ID, nil)
	resp := f.docs.SignMedicalReport(c2)
	if resp.Code != http.StatusConflict || errorCode(t, resp.Data) != core_errors.ErrSignatureAlreadySigned.ErrorCode {
		t.Fatalf("second sign = %d %+v, want 409 E-SIGN-004", resp.Code, resp.Data)
	}
}

func TestSignMedicalReport_LegacyWithoutDoctorNeedsDoctorProfile(t *testing.T) {
	f := newSigningFixture(t)
	f.giveSignature("RECEPCION")
	report := f.report()
	// No beDoctor(): the signer is not a doctor of the tenant (e.g. a receptionist).

	c, _ := f.ctx(report.ID, nil)
	resp := f.docs.SignMedicalReport(c)
	if resp.Code != http.StatusForbidden || errorCode(t, resp.Data) != core_errors.ErrDoctorSignNotOwner.ErrorCode {
		t.Fatalf("sign legacy report without doctor profile = %d %+v, want 403 E-DR-013", resp.Code, resp.Data)
	}
}

func TestSignMedicalReport_WithoutSignatureIsRejected(t *testing.T) {
	f := newSigningFixture(t)
	f.beDoctor()
	report := f.report()

	c, _ := f.ctx(report.ID, nil)
	resp := f.docs.SignMedicalReport(c)
	if resp.Code != http.StatusBadRequest || errorCode(t, resp.Data) != core_errors.ErrSignatureNotConfigured.ErrorCode {
		t.Fatalf("sign without signature = %d %+v, want 400 E-SIGN-003", resp.Code, resp.Data)
	}
}

func TestUpdatePrescription_ChangedContentClearsSignature(t *testing.T) {
	f := newSigningFixture(t)
	signedAt := time.Now()
	signerID := uint(f.userID)
	prescription := clinical_models.Prescription{Content: "Paracetamol", Indications: "c/8h",
		DocumentSignature: clinical_models.DocumentSignature{SignedByID: &signerID, SignedAt: &signedAt, SignerName: "DR X", SignedFile: "signed/prescription-1"}}
	if err := f.db.Create(&prescription).Error; err != nil {
		t.Fatal(err)
	}
	soap := clinical_models.SOAPRecord{}
	f.db.Create(&soap)
	record := clinical_models.MedicalRecord{TenantID: f.tenant.ID, PatientID: f.patient.ID, Date: time.Now(), SOAPRecordID: soap.ID, PrescriptionID: &prescription.ID}
	if err := f.db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}

	reload := func() clinical_models.Prescription {
		var p clinical_models.Prescription
		f.db.First(&p, prescription.ID)
		return p
	}

	c, _ := f.ctx(record.ID, map[string]string{"content": "Paracetamol", "indications": "c/8h"})
	if resp := f.records.UpdatePrescription(c); resp.Code != http.StatusOK {
		t.Fatalf("unchanged update code = %d", resp.Code)
	}
	if !reload().IsSigned() {
		t.Fatal("an unchanged prescription lost its signature")
	}

	c, _ = f.ctx(record.ID, map[string]string{"content": "Ibuprofeno", "indications": "c/8h"})
	if resp := f.records.UpdatePrescription(c); resp.Code != http.StatusOK {
		t.Fatalf("changed update code = %d", resp.Code)
	}
	if p := reload(); p.IsSigned() || p.SignedByID != nil {
		t.Fatalf("edited prescription kept its signature: %+v", p.DocumentSignature)
	}
}
