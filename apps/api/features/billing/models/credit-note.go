package billing_models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreditNoteDocumentCode is the SRI document code for "Nota de Crédito" (04).
const CreditNoteDocumentCode = "04"

type CreditNote struct {
	gorm.Model
	TenantID          uint             `gorm:"index" json:"tenant_id"` // Multi-tenant
	InvoiceID         uint             `json:"invoice_id"`             // Factura que se corrige/anula
	Invoice           Invoice          `gorm:"foreignKey:InvoiceID" json:"invoice"`
	AccessKey         *string          `gorm:"size:49" json:"access_key"`
	AuthorizedAt      *time.Time       `json:"authorized_at"` // fechaAutorizacion del SRI; el número de autorización es el mismo AccessKey
	DocumentCode      string           `json:"document_code"` // "04"
	EmissionType      string           `json:"emission_type"` // "1" Emisión normal (offline)
	Sequential        string           `json:"sequential"`    // "000000001"
	IssueDate         time.Time        `json:"issue_date"`
	Reason            string           `json:"reason"` // Motivo de la nota de crédito
	Subtotal          float64          `json:"subtotal"`
	TaxTotal          float64          `json:"tax_total"`
	Total             float64          `json:"total"`
	Currency          string           `json:"currency"`
	Status            string           `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ErrorMessage      *string          `json:"error_message"`
	Items             []CreditNoteItem `gorm:"foreignKey:CreditNoteID" json:"items"`
	EstablishmentCode string           `json:"establishment_code"`
	EmissionPointCode string           `json:"emission_point_code"`
}

type CreditNoteItem struct {
	gorm.Model
	ProductID        uint    `json:"product_id"`
	CreditNoteID     uint    `json:"credit_note_id"`
	Description      string  `json:"description"`
	Quantity         float64 `json:"quantity"`
	UnitPrice        float64 `json:"unit_price"`
	Discount         float64 `json:"discount"`
	TaxPercentage    string  `json:"tax_percentage"`
	TaxCode          string  `json:"tax_code"`
	TaxRate          float64 `json:"tax_rate"`
	IceTax           float64 `json:"ice_tax"`
	IceTaxCode       string  `json:"ice_tax_code"`
	IceTaxPercentage string  `json:"ice_tax_percentage_code"`
	Subtotal         float64 `json:"subtotal"`
	TaxAmount        float64 `json:"tax_amount"`
	Total            float64 `json:"total"`
}

func (creditNote *CreditNote) GenerateSequential(db *gorm.DB) (string, error) {
	var counter InvoiceCounter

	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND document_type = ? AND establishment = ? AND emission_point = ?",
			creditNote.TenantID, CreditNoteDocumentCode, creditNote.EstablishmentCode, creditNote.EmissionPointCode).
		First(&counter).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			counter = InvoiceCounter{
				TenantID:      creditNote.TenantID,
				DocumentType:  CreditNoteDocumentCode,
				Establishment: creditNote.EstablishmentCode,
				EmissionPoint: creditNote.EmissionPointCode,
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
	creditNote.Sequential = fmt.Sprintf("%09d", counter.CurrentNumber)

	return creditNote.Sequential, nil
}
