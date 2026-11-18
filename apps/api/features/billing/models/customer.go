package billing_models

import "gorm.io/gorm"

// Customer is a third-party buyer a clinic invoices instead of the patient
// (aseguradora, empresa, a relative with their own cédula/RUC). The patient
// stays on the invoice as who was treated; the customer is who pays.
//
// Identification is unique per tenant among non-deleted customers, so a
// deleted customer's identification can be registered again.
type Customer struct {
	gorm.Model
	TenantID       uint   `gorm:"not null;index;uniqueIndex:idx_billing_customers_tenant_identification,where:deleted_at IS NULL" json:"tenant_id"`
	Identification string `gorm:"size:20;not null;uniqueIndex:idx_billing_customers_tenant_identification,where:deleted_at IS NULL" json:"identification"` // cédula, RUC or pasaporte
	Name           string `gorm:"size:300;not null" json:"name"`                                                                                           // razón social / nombres y apellidos
	Address        string `gorm:"size:300;not null;default:''" json:"address"`
	Email          string `gorm:"size:300;not null;default:''" json:"email"`
	Phone          string `gorm:"size:50;not null;default:''" json:"phone"`
}

func (Customer) TableName() string { return "billing_customers" }

// Limits of a customer's fields: the SRI's razonSocialComprador and
// identificacionComprador lengths (ficha técnica offline v2.26).
const (
	CustomerNameMaxLength           = 300
	CustomerIdentificationMaxLength = 20
	CustomerContactMaxLength        = 300
	CustomerPhoneMaxLength          = 50
)
