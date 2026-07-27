package billing_dto

type CreateInvoiceItem struct {
	ProductID uint    `json:"product_id" binding:"required"`
	Quantity  uint    `json:"quantity" binding:"required"`
	Discount  float64 `json:"discount"`
}

type CreateInvoiceDTO struct {
	PatientID         *uint               `json:"patient_id,omitempty"`
	PaymentMethod     string              `json:"payment_method" binding:"required"`
	Term              uint                `json:"term" binding:"gte=0"`
	TimeUnit          string              `json:"time_unit" binding:"required"`
	EstablishmentCode string              `json:"establishment_code" binding:"required"`
	EmissionPointCode string              `json:"emission_point_code" binding:"required"`
	Items             []CreateInvoiceItem `json:"items" binding:"required,min=1"`
}

type InvoiceDTO struct {
	InvoiceID uint64 `json:"invoice_id"`
}

type InvoiceIDListDTO struct {
	IDList []uint64 `json:"id_list"`
}
