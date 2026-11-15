package portal_handlers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// Negative tests for the portal's file endpoints: a patient session never
// reads another patient's (same clinic) or another clinic's (same document)
// prescriptions, reports, results or invoices, by id or by list.

func TestDocuments_PrescriptionsAndReportsOfOthersAreHidden(t *testing.T) {
	f := newFixture(t)
	signedAt := time.Now()
	type seeded struct {
		presc  clinical_models.Prescription
		report clinical_models.MedicalReport
	}
	seed := func(tenantID, patientID uint, tag string) seeded {
		var s seeded
		record := clinical_models.MedicalRecord{TenantID: tenantID, PatientID: patientID, Date: time.Now()}
		f.mustCreate(&record)
		s.presc = clinical_models.Prescription{MedicalRecordID: record.ID, Content: tag,
			DocumentSignature: clinical_models.DocumentSignature{SignedAt: &signedAt, SignedFile: "signed/presc-" + tag + ".pdf"}}
		f.mustCreate(&s.presc)
		_ = f.files.Write(tenantID, s.presc.SignedFile, []byte("%PDF-presc-"+tag))
		s.report = clinical_models.MedicalReport{TenantID: tenantID, PatientID: patientID,
			DocumentSignature: clinical_models.DocumentSignature{SignedAt: &signedAt, SignedFile: "signed/report-" + tag + ".pdf"}}
		f.mustCreate(&s.report)
		_ = f.files.Write(tenantID, s.report.SignedFile, []byte("%PDF-report-"+tag))
		return s
	}
	mine := seed(f.own.ID, f.ownPat.ID, "mine")
	neighbour := seed(f.own.ID, f.ownPat2.ID, "neighbour")
	foreign := seed(f.other.ID, f.otherPat.ID, "foreign")

	token := f.signIn(f.own, f.ownPat)
	list := f.do(http.MethodGet, "/portal/documents", token, nil)
	if n := listLen(t, list); n != 2 {
		t.Fatalf("documents = %s", list.Raw)
	}
	if strings.Contains(string(list.Raw), "neighbour") || strings.Contains(string(list.Raw), "foreign") {
		t.Fatalf("documents leak other patients: %s", list.Raw)
	}
	for path, want := range map[string]string{
		fmt.Sprintf("/portal/documents/prescription/%d/file", mine.presc.ID): "%PDF-presc-mine",
		fmt.Sprintf("/portal/documents/report/%d/file", mine.report.ID):      "%PDF-report-mine",
	} {
		if r := f.do(http.MethodGet, path, token, nil); r.Code != http.StatusOK || string(r.Raw) != want {
			t.Fatalf("%s: %d %s", path, r.Code, r.Raw)
		}
	}
	for _, s := range []seeded{neighbour, foreign} {
		for _, path := range []string{
			fmt.Sprintf("/portal/documents/prescription/%d/file", s.presc.ID),
			fmt.Sprintf("/portal/documents/report/%d/file", s.report.ID),
			// The id of one type under another type's route.
			fmt.Sprintf("/portal/documents/certificate/%d/file", s.report.ID),
		} {
			if r := f.do(http.MethodGet, path, token, nil); r.Code != http.StatusNotFound {
				t.Fatalf("%s: %d %s", path, r.Code, r.Raw)
			}
		}
	}

	// The other clinic's patient (same document) sees only their own.
	foreignToken := f.signIn(f.other, f.otherPat)
	if n := listLen(t, f.do(http.MethodGet, "/portal/documents", foreignToken, nil)); n != 2 {
		t.Fatalf("foreign patient documents = %d", n)
	}
	if r := f.do(http.MethodGet, fmt.Sprintf("/portal/documents/prescription/%d/file", mine.presc.ID), foreignToken, nil); r.Code != http.StatusNotFound {
		t.Fatalf("foreign patient reads own clinic's prescription: %d", r.Code)
	}
}

func TestResults_FilesOnlyThroughThePatientsOwnReviewedOrders(t *testing.T) {
	f := newFixture(t)
	sys := tenantdb.System(f.db)
	reviewed := time.Now()
	order := func(tenantID, patientID uint, number uint, status string, reviewedAt *time.Time) clinical_models.ExamOrderItem {
		o := clinical_models.ExamOrder{TenantID: tenantID, Number: number, PatientID: patientID, OrderedByID: 1, Status: status}
		f.mustCreate(&o)
		it := clinical_models.ExamOrderItem{TenantID: tenantID, ExamOrderID: o.ID, Name: fmt.Sprintf("Exam %d", number), Category: "laboratory", ReviewedAt: reviewedAt}
		f.mustCreate(&it)
		return it
	}
	attachment := func(tenantID, patientID uint, tag string) clinical_models.PatientAttachment {
		a := clinical_models.PatientAttachment{TenantID: tenantID, PatientID: patientID, Category: "lab_result", FileName: tag + ".pdf", MimeType: "application/pdf", StoredName: "attachments/" + tag}
		f.mustCreate(&a)
		_ = f.attach.Write(tenantID, a.StoredName, []byte("%PDF-"+tag))
		return a
	}
	link := func(it clinical_models.ExamOrderItem, a clinical_models.PatientAttachment) {
		if err := sys.Model(&it).Association("Attachments").Append(&a); err != nil {
			t.Fatal(err)
		}
	}

	ownItem := order(f.own.ID, f.ownPat.ID, 1, clinical_models.ExamOrderStatusCompleteResults, &reviewed)
	mine := attachment(f.own.ID, f.ownPat.ID, "mine")
	link(ownItem, mine)
	// The neighbour's attachment linked (by mistake) to the patient's item.
	strayNeighbour := attachment(f.own.ID, f.ownPat2.ID, "stray-neighbour")
	link(ownItem, strayNeighbour)
	// The patient's attachment linked only to the neighbour's reviewed order.
	neighbourItem := order(f.own.ID, f.ownPat2.ID, 2, clinical_models.ExamOrderStatusCompleteResults, &reviewed)
	viaNeighbour := attachment(f.own.ID, f.ownPat.ID, "via-neighbour")
	link(neighbourItem, viaNeighbour)
	// The patient's attachment on a not yet reviewed item, and one not linked at all.
	pendingItem := order(f.own.ID, f.ownPat.ID, 3, clinical_models.ExamOrderStatusIssued, nil)
	unreviewed := attachment(f.own.ID, f.ownPat.ID, "unreviewed")
	link(pendingItem, unreviewed)
	loose := attachment(f.own.ID, f.ownPat.ID, "loose")
	// The other clinic's patient with the same document.
	foreignItem := order(f.other.ID, f.otherPat.ID, 1, clinical_models.ExamOrderStatusCompleteResults, &reviewed)
	foreign := attachment(f.other.ID, f.otherPat.ID, "foreign")
	link(foreignItem, foreign)

	token := f.signIn(f.own, f.ownPat)
	r := f.do(http.MethodGet, "/portal/results", token, nil)
	if listLen(t, r) != 1 {
		t.Fatalf("results = %s", r.Raw)
	}
	for _, leak := range []string{"stray-neighbour", "via-neighbour", "unreviewed", "loose", "foreign"} {
		if strings.Contains(string(r.Raw), leak) {
			t.Fatalf("results leak %q: %s", leak, r.Raw)
		}
	}
	if got := f.do(http.MethodGet, fmt.Sprintf("/portal/results/files/%d", mine.ID), token, nil); got.Code != http.StatusOK || string(got.Raw) != "%PDF-mine" {
		t.Fatalf("own file: %d %s", got.Code, got.Raw)
	}
	for _, a := range []clinical_models.PatientAttachment{strayNeighbour, viaNeighbour, unreviewed, loose, foreign} {
		if got := f.do(http.MethodGet, fmt.Sprintf("/portal/results/files/%d", a.ID), token, nil); got.Code != http.StatusNotFound {
			t.Fatalf("file %s: %d %s", a.FileName, got.Code, got.Raw)
		}
	}
}

func TestInvoices_OtherPatientsAndClinicsAreHidden(t *testing.T) {
	f := newFixture(t)
	mk := func(tenantID, patientID uint, seq int) billing_models.Invoice {
		key := fmt.Sprintf("%049d", seq)
		inv := billing_models.Invoice{TenantID: tenantID, PatientID: &patientID, Status: billing_models.InvoiceStatusAuthorized, AccessKey: &key,
			EstablishmentCode: "001", EmissionPointCode: "001", Sequential: fmt.Sprintf("%09d", seq), IssueDate: time.Now()}
		f.mustCreate(&inv)
		_ = f.files.Write(tenantID, sri_document.InvoiceXMLName(key), []byte("<factura/>"))
		_ = f.files.Write(tenantID, sri_document.RideName(key), []byte("%PDF"))
		return inv
	}
	mk(f.own.ID, f.ownPat.ID, 1)
	neighbour := mk(f.own.ID, f.ownPat2.ID, 2)
	foreign := mk(f.other.ID, f.otherPat.ID, 3)
	// An invoice with no patient (consumidor final) is nobody's.
	key := fmt.Sprintf("%049d", 4)
	anonymous := billing_models.Invoice{TenantID: f.own.ID, Status: billing_models.InvoiceStatusAuthorized, AccessKey: &key,
		EstablishmentCode: "001", EmissionPointCode: "001", Sequential: "000000004", IssueDate: time.Now()}
	f.mustCreate(&anonymous)

	token := f.signIn(f.own, f.ownPat)
	r := f.do(http.MethodGet, "/portal/invoices", token, nil)
	if listLen(t, r) != 1 || !strings.Contains(string(r.Raw), "001-001-000000001") {
		t.Fatalf("invoices = %s", r.Raw)
	}
	for _, inv := range []billing_models.Invoice{neighbour, foreign, anonymous} {
		for _, suffix := range []string{"ride", "xml"} {
			path := fmt.Sprintf("/portal/invoices/%d/%s", inv.ID, suffix)
			if got := f.do(http.MethodGet, path, token, nil); got.Code != http.StatusNotFound {
				t.Fatalf("%s: %d %s", path, got.Code, got.Raw)
			}
		}
	}
}
