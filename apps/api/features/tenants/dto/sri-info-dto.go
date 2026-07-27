package tenant_dto

type UpdateSriInfoDTO struct {
	TaxID             string `json:"tax_id" binding:"required"`
	TradeName         string `json:"trade_name" binding:"required"`
	CorporateName     string `json:"corporate_name" binding:"required"`
	Address           string `json:"address" binding:"required"`
	AccountingObliged bool   `json:"accounting_obliged"`
	// Clasificación tributaria opcional del SRI — vacío/false = no aplica.
	SpecialContributorNumber string `json:"special_contributor_number"`
	MicroenterpriseRegime    bool   `json:"microenterprise_regime"`
	WithholdingAgent         string `json:"withholding_agent"`
	RimpeTaxpayer            string `json:"rimpe_taxpayer"`
}
