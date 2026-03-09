package billing_models

import (
	"gorm.io/gorm"
)

type CatalogService struct {
	gorm.Model
	TenantID             uint    `gorm:"index" json:"tenant_id"`
	Name                 string  `json:"name"`
	SKU                  string  `json:"sku"`
	Description          string  `json:"description"`
	UnitPrice            float64 `json:"unit_price"`
	Tax                  float64 `json:"tax"`                 // Default tax rate (e.g. 0.12 or 0 depending on medical service)
	TaxCode              string  `json:"tax_code"`            // "2" = IVA, "0" = No objeto
	TaxPercentageCode    string  `json:"tax_percentage_code"` // e.g. "2" for 12%, "0" for 0%
	IceTax               float64 `json:"ice_tax"`
	IceTaxCode           string  `json:"ice_tax_code"`
	IceTaxPercentageCode string  `json:"ice_tax_percentage_code"`
}
