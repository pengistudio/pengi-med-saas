package cash_templates

import (
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/utils"
)

// sampleSessionID is the session number every report template must print.
const sampleSessionID = "000042"

// SessionReport is the closing report of a cash session. Its file name is also
// the name tenant templates are stored under: never rename it.
var SessionReport = pdfrender.Document{
	ID:       "cash_session_report",
	File:     "cash_session_report_template.html",
	Defaults: FS,
	Paper:    utils.A4Portrait,
	Feature:  "cash",
	Sample:   sampleSessionReport(),
	// A session without payments or movements.
	Variants: []any{SessionReportData{SessionID: sampleSessionID}},
	// The report must say which session and what was counted against expected.
	Required: []pdfrender.RequiredField{
		{Field: ".SessionID", Value: sampleSessionID},
		{Field: ".ExpectedCash", Value: "212.50"},
		{Field: ".CountedCash", Value: "210.00"},
		{Field: ".Difference", Value: "-2.50"},
	},
}

func sampleSessionReport() SessionReportData {
	return SessionReportData{
		ClinicName:   "Consultorio Médico Andrade",
		TaxID:        "1792345678001",
		SessionID:    sampleSessionID,
		OpenedAt:     "15/09/2026 08:02",
		OpenedBy:     "recepcion",
		ClosedAt:     "15/09/2026 18:10",
		ClosedBy:     "recepcion",
		CloseNote:    "Faltante por vuelto mal entregado.",
		OpeningFloat: "50.00",
		CashIn:       "180.00",
		CashRefunded: "10.00",
		MovementsIn:  "0.00",
		MovementsOut: "7.50",
		ExpectedCash: "212.50",
		CountedCash:  "210.00",
		Difference:   "-2.50",
		Methods: []SessionReportMethod{
			{Method: "Efectivo", Count: 5, Total: "170.00"},
			{Method: "Tarjeta de crédito", Count: 2, Total: "95.00"},
			{Method: "Transferencia", Count: 1, Total: "40.00"},
		},
		Payments: []SessionReportPayment{
			{Time: "08:35", Invoice: "001-001-000001234", Method: "Efectivo", Kind: "Cobro", Amount: "40.00"},
			{Time: "09:10", Invoice: "001-001-000001235", Method: "Tarjeta de crédito", Kind: "Cobro", Reference: "Voucher 5521", Amount: "60.00"},
			{Time: "10:42", Invoice: "001-001-000001236", Method: "Transferencia", Kind: "Cobro", Reference: "Pichincha 99812", Amount: "40.00"},
			{Time: "11:05", Invoice: "001-001-000001236", Method: "Efectivo", Kind: "Cobro", Amount: "25.00", Voided: true},
			{Time: "16:20", Invoice: "001-001-000001198", Method: "Efectivo", Kind: "Devolución", Amount: "-10.00"},
		},
		Movements: []SessionReportMovement{
			{Time: "12:30", Type: "Egreso", Category: "Gasto", Reason: "Agua y café para sala de espera", Amount: "7.50"},
		},
		GeneratedAt: "15/09/2026 18:11",
	}
}
