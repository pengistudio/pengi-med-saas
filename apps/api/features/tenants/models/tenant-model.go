package tenant_models

import (
	"time"

	"gorm.io/gorm"
)

type Tenant struct {
	gorm.Model
	Name              string     `gorm:"not null" json:"name"`
	Slug              string     `gorm:"not null;unique" json:"slug"`
	TradeName         string     `json:"trade_name"`          // Nombre Comercial
	CorporateName     string     `json:"corporate_name"`      // Razón Social
	TaxID             string     `json:"tax_id"`              // RUC
	Address           string     `json:"address"`             // Dirección Matriz
	AccountingObliged bool       `json:"accounting_obliged"`  // Obligado a llevar contabilidad (SRI)
	SriPassword       string     `json:"-"`                   // Hidden from API responses
	SriP12Path        string     `json:"sri_p12_path"`        // Local storage path to the uploaded signature
	SriCertExpiration *time.Time `json:"sri_cert_expiration"` // Date the certificate expires
}

func NewTenant(name string) *Tenant {
	return &Tenant{
		Name: name,
	}
}

func (t *Tenant) Save(db *gorm.DB) error {
	return db.Save(t).Error
}
