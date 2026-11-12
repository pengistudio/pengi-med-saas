package reports_handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	agenda_models "pengi-med-saas/features/agenda/models"
	billing_models "pengi-med-saas/features/billing/models"
	cash_models "pengi-med-saas/features/cash/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	reports_dto "pengi-med-saas/features/reports/dto"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	tenantA uint = 1
	tenantB uint = 2
)

type fakeConverter struct{ html string }

func (f *fakeConverter) GeneratePDFFromHTMLWithOptions(html string, _ utils.PDFOptions) ([]byte, error) {
	f.html = html
	return []byte("%PDF-fake"), nil
}

type fixture struct {
	t    *testing.T
	db   *gorm.DB
	h    *ReportsHandler
	conv *fakeConverter
}

func setup(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{}, &doctor_models.Doctor{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&clinical_models.SOAPRecord{}, &clinical_models.MedicalRecord{}, &clinical_models.Prescription{}, &clinical_models.ExamOrder{},
		&agenda_models.AppointmentType{}, &billing_models.Invoice{}, &cash_models.CashSession{}, &cash_models.Payment{},
	)
	conv := &fakeConverter{}
	h := NewReportsHandler(db, zap.NewNop(), pdfrender.New(tenantfiles.Memory(), conv, allDocs()...))
	f := &fixture{t: t, db: db, h: h, conv: conv}
	for _, id := range []uint{tenantA, tenantB} {
		f.create(&tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", id), Slug: fmt.Sprintf("clinic-%d", id), DisplayToken: fmt.Sprintf("tok-%d", id)})
	}
	return f
}

func allDocs() []pdfrender.Document {
	return []pdfrender.Document{revenueExport.doc, receivablesExport.doc, appointmentsExport.doc, clinicalExport.doc}
}

func (f *fixture) create(v any) {
	f.t.Helper()
	if err := f.db.Create(v).Error; err != nil {
		f.t.Fatal(err)
	}
}

// get calls handler for tenantID with the query and returns the recorder.
func get(tenantID uint, handler func(*gin.Context), query string) *httptest.ResponseRecorder {
	c, w := testutils.NewGinContext(tenantID, 7)
	c.Request = httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	handler(c)
	return w
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var env struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v (%s)", err, w.Body.String())
	}
	return env.Data
}

func wantError(t *testing.T, w *httptest.ResponseRecorder, status int, code core_errors.AppError) {
	t.Helper()
	var env struct {
		Data core_errors.AppError `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if w.Code != status || env.Data.ErrorCode != code.ErrorCode {
		t.Fatalf("got %d %s, want %d %s", w.Code, w.Body.String(), status, code.ErrorCode)
	}
}

func today() string { return time.Now().In(location).Format("2006-01-02") }

func rangeQuery(extra string) string {
	return "from=" + today() + "&to=" + today() + extra
}

// seedCash gives tenantID a cash session (so invoices count as charged through
// the desk), a doctor and a patient of that doctor.
func (f *fixture) seedTenant(tenantID uint) (doctor doctor_models.Doctor, patient clinical_models.Patient, session cash_models.CashSession) {
	f.t.Helper()
	doctor = doctor_models.Doctor{TenantID: tenantID, FullName: fmt.Sprintf("Dr. T%d", tenantID), Specialty: "general"}
	f.create(&doctor)
	patient = clinical_models.Patient{TenantID: tenantID, FirstName: "Ana", LastName: fmt.Sprintf("T%d", tenantID), DoctorID: &doctor.ID}
	f.create(&patient)
	session = cash_models.CashSession{TenantID: tenantID, Status: cash_models.SessionOpen, OpenedAt: time.Now().Add(-48 * time.Hour)}
	f.create(&session)
	return
}

func (f *fixture) invoice(tenantID uint, patientID *uint, total float64, status string) billing_models.Invoice {
	f.t.Helper()
	inv := billing_models.Invoice{TenantID: tenantID, PatientID: patientID, Total: total, Status: status, IssueDate: time.Now(),
		Sequential: "000000001", EstablishmentCode: "001", EmissionPointCode: "001"}
	f.create(&inv)
	return inv
}

func (f *fixture) pay(tenantID, sessionID, invoiceID uint, method string, amount float64, status string) {
	f.t.Helper()
	kind := cash_models.KindPayment
	if amount < 0 {
		kind = cash_models.KindRefund
	}
	f.create(&cash_models.Payment{TenantID: tenantID, InvoiceID: invoiceID, CashSessionID: sessionID, Method: method, Amount: amount, Kind: kind, Status: status})
}

func TestRevenue_CollectedIsValidPaymentsMinusRefunds(t *testing.T) {
	f := setup(t)
	doc, pat, sess := f.seedTenant(tenantA)
	inv := f.invoice(tenantA, &pat.ID, 100, "authorized")
	f.invoice(tenantA, &pat.ID, 999, "draft")    // not invoiced
	f.invoice(tenantA, &pat.ID, 999, "rejected") // not invoiced
	f.pay(tenantA, sess.ID, inv.ID, "cash", 60, cash_models.PaymentValid)
	f.pay(tenantA, sess.ID, inv.ID, "credit_card", 40, cash_models.PaymentValid)
	f.pay(tenantA, sess.ID, inv.ID, "cash", -30, cash_models.PaymentValid) // refund
	f.pay(tenantA, sess.ID, inv.ID, "cash", 500, cash_models.PaymentVoided)

	r := decode[reports_dto.RevenueReport](t, get(tenantA, f.h.Revenue, rangeQuery("")))
	if r.Totals.Invoiced != 100 || r.Totals.InvoiceCount != 1 {
		t.Errorf("invoiced = %+v, want 100 over 1 invoice", r.Totals)
	}
	if r.Totals.Collected != 70 || r.Totals.PaymentCount != 3 {
		t.Errorf("collected = %+v, want 70 (60+40-30) over 3 payments", r.Totals)
	}
	if r.Totals.Receivable != 30 {
		t.Errorf("receivable = %v, want 30", r.Totals.Receivable)
	}
	methods := map[string]float64{}
	for _, m := range r.ByMethod {
		methods[m.Method] = m.Total
	}
	if methods["cash"] != 30 || methods["credit_card"] != 40 || len(methods) != 2 {
		t.Errorf("by method = %v", methods)
	}
	if len(r.ByDoctor) != 1 || r.ByDoctor[0].DoctorName != doc.FullName || r.ByDoctor[0].Invoiced != 100 || r.ByDoctor[0].Collected != 70 {
		t.Errorf("by doctor = %+v", r.ByDoctor)
	}
	if len(r.Series) != 1 || r.Series[0].Period != today() || r.Series[0].Invoiced != 100 || r.Series[0].Collected != 70 || r.Granularity != "day" {
		t.Errorf("series = %+v (%s)", r.Series, r.Granularity)
	}

	// Another doctor sees nothing; the owner doctor sees all.
	none := decode[reports_dto.RevenueReport](t, get(tenantA, f.h.Revenue, rangeQuery("&doctor_id=9999")))
	if none.Totals.Invoiced != 0 || none.Totals.Collected != 0 {
		t.Errorf("doctor filter leaked: %+v", none.Totals)
	}
	own := decode[reports_dto.RevenueReport](t, get(tenantA, f.h.Revenue, rangeQuery(fmt.Sprintf("&doctor_id=%d", doc.ID))))
	if own.Totals.Collected != 70 {
		t.Errorf("own doctor collected = %v, want 70", own.Totals.Collected)
	}
}

func TestRevenue_MonthlySeriesOverNinetyDays(t *testing.T) {
	f := setup(t)
	_, pat, _ := f.seedTenant(tenantA)
	f.invoice(tenantA, &pat.ID, 10, "authorized")
	to := time.Now().In(location)
	r := decode[reports_dto.RevenueReport](t, get(tenantA, f.h.Revenue, "from="+to.AddDate(0, 0, -120).Format("2006-01-02")+"&to="+to.Format("2006-01-02")))
	if r.Granularity != "month" || len(r.Series) != 1 || len(r.Series[0].Period) != 7 {
		t.Fatalf("granularity %s series %+v", r.Granularity, r.Series)
	}
}

func TestReports_TwoTenantIsolation(t *testing.T) {
	f := setup(t)
	_, patA, sessA := f.seedTenant(tenantA)
	_, patB, sessB := f.seedTenant(tenantB)
	invA := f.invoice(tenantA, &patA.ID, 100, "authorized")
	invB := f.invoice(tenantB, &patB.ID, 7000, "authorized")
	f.pay(tenantA, sessA.ID, invA.ID, "cash", 100, cash_models.PaymentValid)
	f.pay(tenantB, sessB.ID, invB.ID, "cash", 5000, cash_models.PaymentValid)
	f.create(&clinical_models.Appointment{TenantID: tenantB, PatientID: patB.ID, Date: time.Now().UTC(), Status: "completed"})
	f.create(&clinical_models.MedicalRecord{TenantID: tenantB, PatientID: patB.ID, Date: time.Now(), Diagnoses: datatypes.JSON(`[{"code":"J00","title":"Resfrío"}]`)})
	f.create(&clinical_models.ExamOrder{TenantID: tenantB, Number: 1, PatientID: patB.ID, OrderedByID: 1})

	rev := decode[reports_dto.RevenueReport](t, get(tenantA, f.h.Revenue, rangeQuery("")))
	if rev.Totals.Invoiced != 100 || rev.Totals.Collected != 100 || len(rev.ByDoctor) != 1 {
		t.Errorf("revenue leaked tenant B: %+v", rev)
	}
	rec := decode[reports_dto.ReceivablesReport](t, get(tenantA, f.h.Receivables, rangeQuery("")))
	if rec.Totals.Count != 0 {
		t.Errorf("receivables leaked tenant B: %+v", rec.Totals)
	}
	ap := decode[reports_dto.AppointmentsReport](t, get(tenantA, f.h.Appointments, rangeQuery("")))
	if ap.Totals.Total != 0 || ap.Patients.New != 0 {
		t.Errorf("appointments leaked tenant B: %+v", ap.Totals)
	}
	cl := decode[reports_dto.ClinicalReport](t, get(tenantA, f.h.Clinical, rangeQuery("")))
	if cl.Totals.Consultations != 0 || cl.Totals.ExamOrders != 0 || len(cl.Top) != 0 {
		t.Errorf("clinical leaked tenant B: %+v", cl)
	}
	// And tenant B does see its own.
	clB := decode[reports_dto.ClinicalReport](t, get(tenantB, f.h.Clinical, rangeQuery("")))
	if clB.Totals.Consultations != 1 || clB.Totals.ExamOrders != 1 || len(clB.Top) != 1 {
		t.Errorf("tenant B clinical = %+v", clB)
	}
}

func TestReports_RangeAndFormatValidation(t *testing.T) {
	f := setup(t)
	wantError(t, get(tenantA, f.h.Revenue, "from=2025-01-01&to=2026-06-01"), http.StatusBadRequest, core_errors.ErrReportsRangeTooLong)
	// 366 days inclusive is allowed (2026-01-01 .. 2027-01-01).
	if w := get(tenantA, f.h.Revenue, "from=2026-01-01&to=2027-01-01"); w.Code != http.StatusOK {
		t.Errorf("366 days rejected: %d %s", w.Code, w.Body.String())
	}
	wantError(t, get(tenantA, f.h.Revenue, "from=2026-01-01&to=2027-01-02"), http.StatusBadRequest, core_errors.ErrReportsRangeTooLong)
	wantError(t, get(tenantA, f.h.Revenue, "from=2026-02-01&to=2026-01-01"), http.StatusBadRequest, core_errors.ErrReportsInvalidFilters)
	wantError(t, get(tenantA, f.h.Revenue, "from=01/02/2026"), http.StatusBadRequest, core_errors.ErrReportsInvalidFilters)
	wantError(t, get(tenantA, f.h.Revenue, "doctor_id=abc"), http.StatusBadRequest, core_errors.ErrReportsInvalidFilters)
	wantError(t, get(tenantA, f.h.Clinical, rangeQuery("&format=xml")), http.StatusBadRequest, core_errors.ErrReportsInvalidFormat)
}

func TestReceivables_AgingAndLegacyInvoices(t *testing.T) {
	f := setup(t)
	_, pat, sess := f.seedTenant(tenantA)
	old := f.invoice(tenantA, &pat.ID, 200, "authorized")
	f.db.Model(&old).Update("issue_date", time.Now().AddDate(0, 0, -45))
	fresh := f.invoice(tenantA, &pat.ID, 100, "authorized")
	f.pay(tenantA, sess.ID, fresh.ID, "cash", 40, cash_models.PaymentValid)
	paid := f.invoice(tenantA, &pat.ID, 50, "authorized")
	f.pay(tenantA, sess.ID, paid.ID, "cash", 50, cash_models.PaymentValid)
	// Issued before the first session and never paid: charged outside the desk.
	legacy := f.invoice(tenantA, &pat.ID, 300, "authorized")
	f.db.Model(&legacy).Update("created_at", time.Now().AddDate(0, 0, -10))
	f.db.Model(&cash_models.CashSession{}).Where("id = ?", sess.ID).Update("opened_at", time.Now().AddDate(0, 0, -1))

	from := time.Now().AddDate(0, 0, -60).In(location).Format("2006-01-02")
	r := decode[reports_dto.ReceivablesReport](t, get(tenantA, f.h.Receivables, "from="+from+"&to="+today()))
	if r.Totals.Count != 2 || r.Totals.Balance != 260 || r.Totals.Total != 300 || r.Totals.Paid != 40 {
		t.Fatalf("totals = %+v, want 2 invoices, balance 260", r.Totals)
	}
	if r.Items[0].Balance != 200 || r.Items[0].PaymentStatus != "unpaid" || r.Items[0].AgeDays < 44 || r.Items[0].AgeDays > 46 {
		t.Errorf("oldest first: %+v", r.Items[0])
	}
	if r.Items[1].Balance != 60 || r.Items[1].PaymentStatus != "partial" || r.Items[1].InvoiceNumber != "001-001-000000001" {
		t.Errorf("second: %+v", r.Items[1])
	}
	if r.Buckets[0].Count != 1 || r.Buckets[0].Balance != 60 || r.Buckets[1].Count != 1 || r.Buckets[1].Balance != 200 || r.Buckets[2].Count != 0 {
		t.Errorf("buckets = %+v", r.Buckets)
	}
}

func TestAppointments_Counts(t *testing.T) {
	f := setup(t)
	doc, pat, _ := f.seedTenant(tenantA)
	other := clinical_models.Patient{TenantID: tenantA, FirstName: "Luis", LastName: "Old"}
	f.create(&other)
	// Appointments hold a plain calendar date: today's date in the tenants' clock.
	now := utcDay(time.Now().In(location))
	appt := func(p uint, d *uint, date time.Time, status string) {
		f.create(&clinical_models.Appointment{TenantID: tenantA, PatientID: p, DoctorID: d, Date: date, Status: status})
	}
	kind := agenda_models.AppointmentType{TenantID: tenantA, Name: "Control", DurationMinutes: 20}
	f.create(&kind)
	f.create(&clinical_models.Appointment{TenantID: tenantA, PatientID: pat.ID, DoctorID: &doc.ID, Date: now, Status: "completed", AppointmentTypeID: &kind.ID})
	appt(pat.ID, &doc.ID, now, "no_show")
	appt(other.ID, nil, now, "cancelled")
	appt(other.ID, &doc.ID, now, "completed")
	appt(other.ID, &doc.ID, now.AddDate(0, -3, 0), "completed") // earlier: makes `other` recurring, outside range

	r := decode[reports_dto.AppointmentsReport](t, get(tenantA, f.h.Appointments, rangeQuery("")))
	if r.Totals.Total != 4 || r.Totals.Completed != 2 || r.Totals.NoShow != 1 || r.Totals.Cancelled != 1 {
		t.Fatalf("totals = %+v", r.Totals)
	}
	if want := 0.3333; r.Totals.NoShowRate != want {
		t.Errorf("no-show rate = %v, want %v", r.Totals.NoShowRate, want)
	}
	if r.Patients.New != 1 || r.Patients.Recurring != 1 {
		t.Errorf("patients = %+v, want 1 new, 1 recurring", r.Patients)
	}
	if len(r.ByType) != 2 || r.ByType[1].TypeName != "Control" && r.ByType[0].TypeName != "Control" {
		t.Errorf("by type = %+v", r.ByType)
	}
	if len(r.ByDoctor) != 2 || len(r.Series) != 1 || r.Series[0].Total != 4 || len(r.ByStatus) != 3 {
		t.Errorf("breakdowns: doctors %+v series %+v status %+v", r.ByDoctor, r.Series, r.ByStatus)
	}
	byDoc := decode[reports_dto.AppointmentsReport](t, get(tenantA, f.h.Appointments, rangeQuery(fmt.Sprintf("&doctor_id=%d", doc.ID))))
	if byDoc.Totals.Total != 3 {
		t.Errorf("doctor filter total = %d, want 3", byDoc.Totals.Total)
	}
}

func TestClinical_TopDiagnosesAndActivity(t *testing.T) {
	f := setup(t)
	doc, pat, _ := f.seedTenant(tenantA)
	rec := func(dx string) clinical_models.MedicalRecord {
		r := clinical_models.MedicalRecord{TenantID: tenantA, PatientID: pat.ID, DoctorID: &doc.ID, Date: time.Now(), Diagnoses: datatypes.JSON(dx)}
		f.create(&r)
		return r
	}
	r1 := rec(`[{"code":"J00","title":"Rinofaringitis"},{"code":"I10","title":"Hipertensión"}]`)
	rec(`[{"code":"J00","title":"Rinofaringitis"}]`)
	rec(`[]`)
	f.create(&clinical_models.Prescription{MedicalRecordID: r1.ID, DoctorID: &doc.ID})
	f.create(&clinical_models.ExamOrder{TenantID: tenantA, Number: 1, PatientID: pat.ID, OrderedByID: 1, DoctorID: &doc.ID})

	r := decode[reports_dto.ClinicalReport](t, get(tenantA, f.h.Clinical, rangeQuery("")))
	if len(r.Top) != 2 || r.Top[0].Code != "J00" || r.Top[0].Count != 2 || r.Top[1].Code != "I10" || r.Top[1].Count != 1 {
		t.Errorf("top = %+v", r.Top)
	}
	if r.Totals.Consultations != 3 || r.Totals.Prescriptions != 1 || r.Totals.ExamOrders != 1 {
		t.Errorf("totals = %+v", r.Totals)
	}
	if len(r.ByDoctor) != 1 || r.ByDoctor[0].DoctorName != doc.FullName {
		t.Errorf("by doctor = %+v", r.ByDoctor)
	}
}

func TestRevenue_CSVHasBOMAndRows(t *testing.T) {
	f := setup(t)
	_, pat, sess := f.seedTenant(tenantA)
	inv := f.invoice(tenantA, &pat.ID, 100, "authorized")
	f.pay(tenantA, sess.ID, inv.ID, "cash", 100, cash_models.PaymentValid)

	w := get(tenantA, f.h.Revenue, rangeQuery("&format=csv"))
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/csv") ||
		!strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("headers: %d %v", w.Code, w.Header())
	}
	body := w.Body.Bytes()
	if len(body) < 3 || body[0] != 0xEF || body[1] != 0xBB || body[2] != 0xBF {
		t.Fatal("CSV must start with a UTF-8 BOM")
	}
	r := csv.NewReader(strings.NewReader(string(body[3:])))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(rows[2], ","); got != "100.00,100.00,0.00" {
		t.Errorf("summary row = %q, rows %v", got, rows)
	}
}

func TestReports_PDFRendersEveryDocument(t *testing.T) {
	f := setup(t)
	f.seedTenant(tenantA)
	for name, handler := range map[string]func(*gin.Context){
		"revenue": f.h.Revenue, "receivables": f.h.Receivables, "appointments": f.h.Appointments, "clinical": f.h.Clinical,
	} {
		f.conv.html = ""
		w := get(tenantA, handler, rangeQuery("&format=pdf"))
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/pdf" || w.Body.String() != "%PDF-fake" {
			t.Errorf("%s: %d %s", name, w.Code, w.Body.String())
		}
		if !strings.Contains(f.conv.html, "Clinic 1") {
			t.Errorf("%s: clinic name missing from the rendered HTML", name)
		}
	}
}
