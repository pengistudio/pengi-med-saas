package billing_dto

type CreateCatalogItemDTO struct {
	Name                 string   `json:"name" binding:"required"`
	SKU                  string   `json:"sku" binding:"required"`
	Description          *string  `json:"description"`
	UnitPrice            float64  `json:"unit_price" binding:"required"`
	Tax                  *float64 `json:"tax"`
	TaxCode              *string  `json:"tax_code"`
	TaxPercentageCode    *string  `json:"tax_percentage_code"`
	IceTax               *float64 `json:"ice_tax"`
	IceTaxCode           *string  `json:"ice_tax_code"`
	IceTaxPercentageCode *string  `json:"ice_tax_percentage_code"`
}

type UpdateCatalogItemDTO struct {
	Name                 *string  `json:"name"`
	SKU                  *string  `json:"sku"`
	Description          *string  `json:"description"`
	UnitPrice            *float64 `json:"unit_price"`
	Tax                  *float64 `json:"tax"`
	TaxCode              *string  `json:"tax_code"`
	TaxPercentageCode    *string  `json:"tax_percentage_code"`
	IceTax               *float64 `json:"ice_tax"`
	IceTaxCode           *string  `json:"ice_tax_code"`
	IceTaxPercentageCode *string  `json:"ice_tax_percentage_code"`
}
