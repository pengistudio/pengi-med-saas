package reports_templates

// Header is what every report prints on top. Values are already formatted
// (local dates dd/mm/yyyy, money with two decimals).
type Header struct {
	ClinicName  string
	TaxID       string
	From        string
	To          string
	Doctor      string // empty: all doctors
	GeneratedAt string
}

// RevenueData is the model of reports_revenue_template.html.
type RevenueData struct {
	Header
	Invoiced     string
	Collected    string
	Receivable   string
	InvoiceCount int
	PaymentCount int
	Monthly      bool // Series periods are months
	Series       []RevenueSeriesRow
	Methods      []RevenueMethodRow
	Doctors      []RevenueDoctorRow
}

type RevenueSeriesRow struct{ Period, Invoiced, Collected string }
type RevenueMethodRow struct {
	Method string
	Count  int
	Total  string
}
type RevenueDoctorRow struct{ Doctor, Invoiced, Collected string }

// ReceivablesData is the model of reports_receivables_template.html.
type ReceivablesData struct {
	Header
	Count   int
	Total   string
	Paid    string
	Balance string
	Buckets []ReceivableBucketRow
	Items   []ReceivableRow
}

type ReceivableBucketRow struct {
	Label   string
	Count   int
	Balance string
}
type ReceivableRow struct {
	Invoice, Patient, Date, Total, Paid, Balance string
	AgeDays                                      int
}

// AppointmentsData is the model of reports_appointments_template.html.
type AppointmentsData struct {
	Header
	Total      int
	Completed  int
	Cancelled  int
	NoShow     int
	NoShowRate string // "12.5%"
	NewPatients,
	RecurringPatients int
	Statuses []AppointmentStatusRow
	Doctors  []AppointmentDoctorRow
	Types    []AppointmentTypeRow
	Series   []AppointmentDayRow
}

type AppointmentStatusRow struct {
	Status string
	Count  int
}
type AppointmentDoctorRow struct {
	Doctor                              string
	Total, Completed, Cancelled, NoShow int
}
type AppointmentTypeRow struct {
	Type  string
	Count int
}
type AppointmentDayRow struct {
	Day           string
	Total, NoShow int
}

// ClinicalData is the model of reports_clinical_template.html.
type ClinicalData struct {
	Header
	Consultations int
	Prescriptions int
	ExamOrders    int
	Diagnoses     []DiagnosisRow
	Doctors       []ClinicalDoctorRow
}

type DiagnosisRow struct {
	Code, Title string
	Count       int
}
type ClinicalDoctorRow struct {
	Doctor                                   string
	Consultations, Prescriptions, ExamOrders int
}

// Printed labels (the templates are Spanish, like every printable document).
var (
	MethodLabels = map[string]string{
		"cash": "Efectivo", "transfer": "Transferencia", "credit_card": "Tarjeta de crédito",
		"debit_card": "Tarjeta de débito", "check": "Cheque", "other": "Otros",
	}
	StatusLabels = map[string]string{
		"scheduled": "Programada", "confirmed": "Confirmada", "completed": "Completada",
		"cancelled": "Cancelada", "no_show": "No asistió",
	}
	BucketLabels = map[string]string{"0_30": "0 a 30 días", "31_60": "31 a 60 días", "61_plus": "61 días o más"}
)

// Label returns labels[key], or key itself when it has no label.
func Label(labels map[string]string, key string) string {
	if l, ok := labels[key]; ok {
		return l
	}
	return key
}
