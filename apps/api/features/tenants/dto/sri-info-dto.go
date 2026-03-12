package tenant_dto

type UpdateSriInfoDTO struct {
	TaxID             string `json:"tax_id" binding:"required"`
	TradeName         string `json:"trade_name" binding:"required"`
	CorporateName     string `json:"corporate_name" binding:"required"`
	Address           string `json:"address" binding:"required"`
	AccountingObliged bool   `json:"accounting_obliged"`
}
