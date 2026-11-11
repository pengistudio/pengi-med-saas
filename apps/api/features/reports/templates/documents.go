package reports_templates

import (
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/utils"
)

// The file names are also the names tenant templates are stored under: never
// rename them.

// Revenue is the revenue and collections report.
var Revenue = pdfrender.Document{
	ID:       "reports_revenue",
	File:     "reports_revenue_template.html",
	Defaults: FS,
	Paper:    utils.A4Portrait,
	Feature:  "reports",
	Sample:   sampleHeaderRevenue(),
	Variants: []any{RevenueData{Header: sampleHeader()}},
	Required: []pdfrender.RequiredField{
		{Field: ".Invoiced", Value: "4820.50"},
		{Field: ".Collected", Value: "3910.00"},
		{Field: ".Receivable", Value: "910.50"},
	},
}

// Receivables is the accounts receivable report.
var Receivables = pdfrender.Document{
	ID:       "reports_receivables",
	File:     "reports_receivables_template.html",
	Defaults: FS,
	Paper:    utils.A4Portrait,
	Feature:  "reports",
	Sample:   sampleReceivables(),
	Variants: []any{ReceivablesData{Header: sampleHeader()}},
	Required: []pdfrender.RequiredField{
		{Field: ".Balance", Value: "910.50"},
	},
}

// Appointments is the appointments and productivity report.
var Appointments = pdfrender.Document{
	ID:       "reports_appointments",
	File:     "reports_appointments_template.html",
	Defaults: FS,
	Paper:    utils.A4Portrait,
	Feature:  "reports",
	Sample:   sampleAppointments(),
	Variants: []any{AppointmentsData{Header: sampleHeader()}},
	Required: []pdfrender.RequiredField{
		{Field: ".NoShowRate", Value: "8.3%"},
	},
}

// Clinical is the clinical activity report.
var Clinical = pdfrender.Document{
	ID:       "reports_clinical",
	File:     "reports_clinical_template.html",
	Defaults: FS,
	Paper:    utils.A4Portrait,
	Feature:  "reports",
	Sample:   sampleClinical(),
	Variants: []any{ClinicalData{Header: sampleHeader()}},
	Required: []pdfrender.RequiredField{
		{Field: ".Consultations", Value: "148"},
	},
}

func sampleHeader() Header {
	return Header{
		ClinicName: "Consultorio Médico Andrade", TaxID: "1792345678001",
		From: "01/09/2026", To: "30/09/2026", Doctor: "Dra. Paula Andrade", GeneratedAt: "01/10/2026 08:15",
	}
}

func sampleHeaderRevenue() RevenueData {
	return RevenueData{
		Header: sampleHeader(), Invoiced: "4820.50", Collected: "3910.00", Receivable: "910.50",
		InvoiceCount: 62, PaymentCount: 71,
		Series: []RevenueSeriesRow{
			{Period: "2026-09-01", Invoiced: "310.00", Collected: "280.00"},
			{Period: "2026-09-02", Invoiced: "190.50", Collected: "150.00"},
		},
		Methods: []RevenueMethodRow{
			{Method: "Efectivo", Count: 40, Total: "2100.00"},
			{Method: "Tarjeta de crédito", Count: 20, Total: "1250.00"},
			{Method: "Transferencia", Count: 11, Total: "560.00"},
		},
		Doctors: []RevenueDoctorRow{
			{Doctor: "Dra. Paula Andrade", Invoiced: "3100.00", Collected: "2700.00"},
			{Doctor: "Sin médico", Invoiced: "1720.50", Collected: "1210.00"},
		},
	}
}

func sampleReceivables() ReceivablesData {
	return ReceivablesData{
		Header: sampleHeader(), Count: 2, Total: "1200.00", Paid: "289.50", Balance: "910.50",
		Buckets: []ReceivableBucketRow{
			{Label: "0 a 30 días", Count: 1, Balance: "410.50"},
			{Label: "31 a 60 días", Count: 1, Balance: "500.00"},
			{Label: "61 días o más", Count: 0, Balance: "0.00"},
		},
		Items: []ReceivableRow{
			{Invoice: "001-001-000001198", Patient: "María Pérez", Date: "02/09/2026", Total: "800.00", Paid: "300.00", Balance: "500.00", AgeDays: 38},
			{Invoice: "001-001-000001230", Patient: "Juan Mora", Date: "20/09/2026", Total: "400.00", Paid: "0.00", Balance: "410.50", AgeDays: 20},
		},
	}
}

func sampleAppointments() AppointmentsData {
	return AppointmentsData{
		Header: sampleHeader(), Total: 130, Completed: 110, Cancelled: 8, NoShow: 10, NoShowRate: "8.3%",
		NewPatients: 24, RecurringPatients: 71,
		Statuses: []AppointmentStatusRow{{Status: "Completada", Count: 110}, {Status: "No asistió", Count: 10}},
		Doctors:  []AppointmentDoctorRow{{Doctor: "Dra. Paula Andrade", Total: 130, Completed: 110, Cancelled: 8, NoShow: 10}},
		Types:    []AppointmentTypeRow{{Type: "Consulta general", Count: 90}, {Type: "Control", Count: 40}},
		Series:   []AppointmentDayRow{{Day: "2026-09-01", Total: 6, NoShow: 1}},
	}
}

func sampleClinical() ClinicalData {
	return ClinicalData{
		Header: sampleHeader(), Consultations: 148, Prescriptions: 120, ExamOrders: 45,
		Diagnoses: []DiagnosisRow{{Code: "J00", Title: "Rinofaringitis aguda", Count: 21}, {Code: "I10", Title: "Hipertensión esencial", Count: 15}},
		Doctors:   []ClinicalDoctorRow{{Doctor: "Dra. Paula Andrade", Consultations: 148, Prescriptions: 120, ExamOrders: 45}},
	}
}
