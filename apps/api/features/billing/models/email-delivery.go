package billing_models

import "time"

// EmailDelivery is the state of emailing an authorized comprobante to its
// buyer (ficha técnica SRI v2.34 §4.7), embedded in Invoice, CreditNote and
// DebitNote so its columns and JSON fields stay flat. EmailedAt and EmailedTo
// describe the last successful send; EmailError is the i18n key of the last
// failure, cleared by the next success.
type EmailDelivery struct {
	EmailedAt  *time.Time `json:"emailed_at"`
	EmailedTo  *string    `gorm:"size:254" json:"emailed_to"`
	EmailError *string    `gorm:"size:100" json:"email_error"`
}
