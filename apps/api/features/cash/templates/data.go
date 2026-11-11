package cash_templates

// SessionReportData is the model passed to cash_session_report_template.html.
// Every value is already formatted (money with two decimals, local dates).
type SessionReportData struct {
	ClinicName string
	TaxID      string
	SessionID  string

	OpenedAt  string
	OpenedBy  string
	ClosedAt  string
	ClosedBy  string
	CloseNote string

	// Cash count
	OpeningFloat string
	CashIn       string
	CashRefunded string
	MovementsIn  string
	MovementsOut string
	ExpectedCash string
	CountedCash  string
	Difference   string

	Methods   []SessionReportMethod
	Payments  []SessionReportPayment
	Movements []SessionReportMovement

	GeneratedAt string
}

// SessionReportMethod is the net total of one payment method.
type SessionReportMethod struct {
	Method string
	Count  int
	Total  string
}

// SessionReportPayment is one payment or refund of the session.
type SessionReportPayment struct {
	Time      string
	Invoice   string
	Method    string
	Kind      string
	Reference string
	Amount    string
	Voided    bool
}

// SessionReportMovement is one manual cash movement of the session.
type SessionReportMovement struct {
	Time     string
	Type     string
	Category string
	Reason   string
	Amount   string
}

// MethodLabels are the printed names of the payment methods.
var MethodLabels = map[string]string{
	"cash":        "Efectivo",
	"transfer":    "Transferencia",
	"credit_card": "Tarjeta de crédito",
	"debit_card":  "Tarjeta de débito",
	"check":       "Cheque",
	"other":       "Otros",
}

// KindLabels are the printed names of the payment kinds.
var KindLabels = map[string]string{
	"payment": "Cobro",
	"refund":  "Devolución",
}

// MovementTypeLabels are the printed names of the movement directions.
var MovementTypeLabels = map[string]string{
	"in":  "Ingreso",
	"out": "Egreso",
}

// CategoryLabels are the printed names of the movement categories.
var CategoryLabels = map[string]string{
	"expense":    "Gasto",
	"withdrawal": "Retiro",
	"deposit":    "Depósito",
	"adjustment": "Ajuste",
}

// Label returns labels[key], or key itself when it has no label.
func Label(labels map[string]string, key string) string {
	if l, ok := labels[key]; ok {
		return l
	}
	return key
}
