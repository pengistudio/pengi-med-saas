// Package reports_dto holds the JSON shapes of the reports (features/reports).
// Money is a number rounded to cents; dates are YYYY-MM-DD in the tenant's
// time zone (America/Guayaquil).
package reports_dto

// Filters echoes the range and doctor a report was computed for.
type Filters struct {
	From     string `json:"from"`
	To       string `json:"to"`
	DoctorID *uint  `json:"doctor_id"`
}

// ---- /reports/revenue ----

type RevenueReport struct {
	Filters Filters       `json:"filters"`
	Totals  RevenueTotals `json:"totals"`
	// Granularity of Series: "day", or "month" when the range is over 90 days.
	Granularity string            `json:"granularity"`
	Series      []RevenuePoint    `json:"series"`
	ByMethod    []RevenueMethod   `json:"by_method"`
	ByDoctor    []RevenueByDoctor `json:"by_doctor"`
}

type RevenueTotals struct {
	Invoiced     float64 `json:"invoiced"`      // invoices issued in range (not draft, not rejected)
	Collected    float64 `json:"collected"`     // valid payments minus refunds made in range
	Receivable   float64 `json:"receivable"`    // balance still owed on the invoices issued in range
	InvoiceCount int     `json:"invoice_count"` // invoices behind Invoiced
	PaymentCount int     `json:"payment_count"` // valid payments and refunds behind Collected
}

// RevenuePoint is a day (YYYY-MM-DD) or month (YYYY-MM).
type RevenuePoint struct {
	Period    string  `json:"period"`
	Invoiced  float64 `json:"invoiced"`
	Collected float64 `json:"collected"`
}

// RevenueMethod is the net collected amount of one payment method (cash,
// transfer, credit_card, debit_card, check, other).
type RevenueMethod struct {
	Method string  `json:"method"`
	Total  float64 `json:"total"`
	Count  int     `json:"count"`
}

// RevenueByDoctor attributes each invoice to the patient's attending doctor
// (patients.doctor_id); invoices without one go to DoctorID null.
type RevenueByDoctor struct {
	DoctorID   *uint   `json:"doctor_id"`
	DoctorName string  `json:"doctor_name"`
	Invoiced   float64 `json:"invoiced"`
	Collected  float64 `json:"collected"`
}

// ---- /reports/receivables ----

type ReceivablesReport struct {
	Filters Filters            `json:"filters"`
	Totals  ReceivablesTotals  `json:"totals"`
	Buckets []ReceivableBucket `json:"buckets"` // always 3, in order: 0-30, 31-60, 61+
	Items   []ReceivableItem   `json:"items"`   // oldest first
}

type ReceivablesTotals struct {
	Count   int     `json:"count"`
	Total   float64 `json:"total"`   // invoiced amount of the open invoices
	Paid    float64 `json:"paid"`    // collected so far
	Balance float64 `json:"balance"` // still owed
}

// ReceivableBucket Key is "0_30", "31_60" or "61_plus".
type ReceivableBucket struct {
	Key     string  `json:"key"`
	Count   int     `json:"count"`
	Balance float64 `json:"balance"`
}

type ReceivableItem struct {
	InvoiceID     uint    `json:"invoice_id"`
	InvoiceNumber string  `json:"invoice_number"`
	PatientID     *uint   `json:"patient_id"`
	PatientName   string  `json:"patient_name"` // empty: consumidor final
	IssueDate     string  `json:"issue_date"`
	Total         float64 `json:"total"`
	Paid          float64 `json:"paid"`
	Balance       float64 `json:"balance"`
	AgeDays       int     `json:"age_days"`
	PaymentStatus string  `json:"payment_status"` // unpaid | partial
}

// ---- /reports/appointments ----

type AppointmentsReport struct {
	Filters  Filters               `json:"filters"`
	Totals   AppointmentTotals     `json:"totals"`
	Patients AppointmentPatients   `json:"patients"`
	ByStatus []AppointmentStatus   `json:"by_status"`
	ByDoctor []AppointmentByDoctor `json:"by_doctor"`
	ByType   []AppointmentByType   `json:"by_type"`
	Series   []AppointmentDayPoint `json:"series"` // one point per day with appointments
}

type AppointmentTotals struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Cancelled int `json:"cancelled"`
	NoShow    int `json:"no_show"`
	// NoShowRate is no_show / (completed + no_show), 0..1 (0 when neither).
	NoShowRate float64 `json:"no_show_rate"`
}

// AppointmentPatients counts distinct patients with a non-cancelled
// appointment in range: New had their first appointment ever in the range.
type AppointmentPatients struct {
	New       int `json:"new"`
	Recurring int `json:"recurring"`
}

type AppointmentStatus struct {
	Status string `json:"status"` // scheduled | confirmed | completed | cancelled | no_show
	Count  int    `json:"count"`
}

type AppointmentByDoctor struct {
	DoctorID   *uint  `json:"doctor_id"`
	DoctorName string `json:"doctor_name"` // empty: no doctor assigned
	Total      int    `json:"total"`
	Completed  int    `json:"completed"`
	Cancelled  int    `json:"cancelled"`
	NoShow     int    `json:"no_show"`
}

type AppointmentByType struct {
	TypeID   *uint  `json:"type_id"`
	TypeName string `json:"type_name"` // empty: no type
	Count    int    `json:"count"`
}

type AppointmentDayPoint struct {
	Day    string `json:"day"`
	Total  int    `json:"total"`
	NoShow int    `json:"no_show"`
}

// ---- /reports/clinical ----

type ClinicalReport struct {
	Filters  Filters            `json:"filters"`
	Totals   ClinicalTotals     `json:"totals"`
	Top      []DiagnosisCount   `json:"top_diagnoses"` // up to 20, most frequent first
	ByDoctor []ClinicalByDoctor `json:"by_doctor"`
}

type ClinicalTotals struct {
	Consultations int `json:"consultations"`
	Prescriptions int `json:"prescriptions"`
	ExamOrders    int `json:"exam_orders"`
}

// DiagnosisCount Count is the number of consultations carrying the CIE-10 code.
type DiagnosisCount struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Count int    `json:"count"`
}

type ClinicalByDoctor struct {
	DoctorID      *uint  `json:"doctor_id"`
	DoctorName    string `json:"doctor_name"`
	Consultations int    `json:"consultations"`
	Prescriptions int    `json:"prescriptions"`
	ExamOrders    int    `json:"exam_orders"`
}
