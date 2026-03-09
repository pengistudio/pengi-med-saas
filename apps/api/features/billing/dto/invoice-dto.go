package billing_dto

type CreateInvoiceItem struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  uint `json:"quantity" binding:"required"`
}

type CreateInvoiceDTO struct {
	PatientID         *uint               `json:"patient_id,omitempty"`
	PaymentMethod     string              `json:"payment_method" binding:"required"`
	Term              uint                `json:"term" binding:"required"`
	TimeUnit          string              `json:"time_unit" binding:"required"`
	EstablishmentCode string              `json:"establishment_code" binding:"required"`
	EmissionPointCode string              `json:"emission_point_code" binding:"required"`
	Items             []CreateInvoiceItem `json:"items" binding:"required,min=1"`
	SubTotal          *float64            `json:"subtotal,omitempty"`
	Discount          *float64            `json:"discount,omitempty"`
	TaxTotal          *float64            `json:"tax_total,omitempty"`
	Total             *float64            `json:"total,omitempty"`
}

type InvoiceDTO struct {
	InvoiceID uint64 `json:"invoice_id"`
}

type InvoiceIDListDTO struct {
	IDList []uint64 `json:"id_list"`
}
