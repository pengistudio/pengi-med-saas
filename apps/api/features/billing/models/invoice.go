package billing_models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	clinical_models "pengi-med-saas/features/clinical/models"
)

// FinalConsumerMaxTotal is the largest invoice total (USD) that may be issued to
// Consumidor Final; above it the buyer must be identified (ficha técnica SRI
// offline v2.26 §8.10) or the SRI rejects the invoice (ID 69).
const FinalConsumerMaxTotal = 50.0

type Invoice struct {
	gorm.Model
	TenantID     uint                     `gorm:"index" json:"tenant_id"` // Multi-tenant
	AccessKey    *string                  `gorm:"size:49" json:"access_key"`
	AuthorizedAt *time.Time               `json:"authorized_at"` // fechaAutorizacion del SRI; el número de autorización es el mismo AccessKey
	DocumentCode string                   `json:"document_code"` // Factura
	EmissionType string                   `json:"emission_type"` // "1" Emisión normal (offline)
	Sequential   string                   `json:"sequential"`    // "000000001"
	IssueDate    time.Time                `json:"issue_date"`
	PatientID    *uint                    `json:"patient_id"` // FK a Patient — nil = Consumidor Final (sin paciente)
	Patient      *clinical_models.Patient `gorm:"foreignKey:PatientID" json:"patient"`
	// CustomerID is the third-party buyer (Customer), when it is not the
	// patient. nil: the buyer is the patient, or Consumidor Final.
	CustomerID *uint     `gorm:"index" json:"customer_id"`
	Customer   *Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	BuyerSnapshot
	Subtotal          float64       `json:"subtotal"`
	Discount          float64       `json:"discount"`
	TaxTotal          float64       `json:"tax_total"`
	Total             float64       `json:"total"`
	Currency          string        `json:"currency"`
	PaymentMethod     string        `json:"payment_method"`
	Term              string        `json:"term"`
	TimeUnit          string        `json:"time_unit"`
	Status            string        `gorm:"type:varchar(20);default:'draft'" json:"status"` // "draft", "pending", "processing", "signed", "validated", "authorized", "failed"
	ErrorCode         *string       `gorm:"size:100" json:"error_code"`                     // i18n key describing ErrorMessage for the UI
	ErrorMessage      *string       `json:"error_message"`
	Items             []InvoiceItem `gorm:"foreignKey:InvoiceID" json:"items"`
	CompanyID         uint          `json:"company_id"` // Optional if you need it, or can be replaced by Tenant logic
	EstablishmentCode string        `json:"establishment_code"`
	EmissionPointCode string        `json:"emission_point_code"`
	EmailDelivery

	// Collection status, computed from the cash payments (cash_services.
	// ApplyPaymentStatus), not stored. Null for invoices issued before the
	// clinic's first cash session, which were never charged through the desk.
	PaidAmount    *float64 `gorm:"-" json:"paid_amount"`
	PaymentStatus *string  `gorm:"-" json:"payment_status"` // unpaid | partial | paid
}

// BuyerSnapshot is the buyer an invoice declares to the SRI (comprador), as
// of its last build: filled on create for display and rewritten right before
// each signing attempt, so it is frozen once the SRI receives the invoice
// (docs/adr/0001-clave-de-acceso-inmutable.md). Its credit and debit notes
// declare the same buyer. Empty on invoices created before it existed: their
// buyer is the patient, resolved live (sri_services.InvoiceBuyer).
type BuyerSnapshot struct {
	BuyerIdentification     string `gorm:"size:20;not null;default:''" json:"buyer_identification"`
	BuyerIdentificationType string `gorm:"size:2;not null;default:''" json:"buyer_identification_type"` // 04 RUC, 05 cédula, 06 pasaporte, 07 Consumidor Final
	BuyerName               string `gorm:"size:300;not null;default:''" json:"buyer_name"`
	BuyerAddress            string `gorm:"size:300;not null;default:''" json:"buyer_address"`
	BuyerEmail              string `gorm:"size:300;not null;default:''" json:"buyer_email"`
	BuyerPhone              string `gorm:"size:50;not null;default:''" json:"buyer_phone"`
}

// IsEmpty reports whether no buyer was ever recorded (legacy invoices).
func (s BuyerSnapshot) IsEmpty() bool { return s.BuyerIdentification == "" }

// Columns is the snapshot as column values, zero values included, for Updates.
func (s BuyerSnapshot) Columns() map[string]any {
	return map[string]any{
		"buyer_identification":      s.BuyerIdentification,
		"buyer_identification_type": s.BuyerIdentificationType,
		"buyer_name":                s.BuyerName,
		"buyer_address":             s.BuyerAddress,
		"buyer_email":               s.BuyerEmail,
		"buyer_phone":               s.BuyerPhone,
	}
}

// Invoice collection status (Invoice.PaymentStatus), independent of the SRI status.
const (
	PaymentStatusUnpaid  = "unpaid"
	PaymentStatusPartial = "partial"
	PaymentStatusPaid    = "paid"
)

type InvoiceItem struct {
	gorm.Model
	ProductID        uint    `json:"product_id"`
	InvoiceID        uint    `json:"invoice_id"`
	Description      string  `json:"description"`
	Quantity         float64 `json:"quantity"`
	UnitPrice        float64 `json:"unit_price"`
	Discount         float64 `json:"discount"`
	TaxPercentage    string  `json:"tax_percentage"`          // opcional
	TaxCode          string  `json:"tax_code"`                // opcional
	TaxRate          float64 `json:"tax_rate"`                // e.g. 12 for 12%
	IceTax           float64 `json:"ice_tax"`                 // opcional
	IceTaxCode       string  `json:"ice_tax_code"`            // opcional
	IceTaxPercentage string  `json:"ice_tax_percentage_code"` // opcional
	Subtotal         float64 `json:"subtotal"`                // Quantity * UnitPrice - Discount
	TaxAmount        float64 `json:"tax_amount"`              // Subtotal * TaxRate
	Total            float64 `json:"total"`                   // Subtotal + TaxAmount
}

type InvoiceCounter struct {
	gorm.Model
	TenantID      uint
	DocumentType  string // Ej: "01" = Factura
	Establishment string // Ej: "001"
	EmissionPoint string // Ej: "001"
	CurrentNumber int
}

func (invoice *Invoice) Save(db *gorm.DB) error {
	err := db.Save(&invoice).Error
	return err
}

func (invoice *Invoice) Delete(db *gorm.DB) error {
	err := db.Delete(&invoice).Error
	return err
}

func (invoice *Invoice) Update(db *gorm.DB, dataToUpdate map[string]any) error {
	return db.Model(&invoice).Updates(dataToUpdate).Error
}

func (invoice *Invoice) GenerateSequential(db *gorm.DB) (string, error) {
	invoiceCode := "01"
	var counter InvoiceCounter

	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND document_type = ? AND establishment = ? AND emission_point = ?",
			invoice.TenantID, invoiceCode, invoice.EstablishmentCode, invoice.EmissionPointCode).
		First(&counter).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			counter = InvoiceCounter{
				TenantID:      invoice.TenantID,
				DocumentType:  invoiceCode,
				Establishment: invoice.EstablishmentCode,
				EmissionPoint: invoice.EmissionPointCode,
				CurrentNumber: 0,
			}
			if err := db.Create(&counter).Error; err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	counter.CurrentNumber++
	if err := db.Save(&counter).Error; err != nil {
		return "", err
	}
	invoice.Sequential = fmt.Sprintf("%09d", counter.CurrentNumber)

	return invoice.Sequential, nil
}
