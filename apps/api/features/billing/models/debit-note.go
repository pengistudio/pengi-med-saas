package billing_models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DebitNoteDocumentCode is the SRI document code for "Nota de Débito" (05).
const DebitNoteDocumentCode = "05"

type DebitNote struct {
	gorm.Model
	TenantID          uint              `gorm:"index" json:"tenant_id"` // Multi-tenant
	InvoiceID         uint              `json:"invoice_id"`             // Factura sobre la que se cobra el cargo adicional
	Invoice           Invoice           `gorm:"foreignKey:InvoiceID" json:"invoice"`
	AccessKey         *string           `gorm:"size:49" json:"access_key"`
	AuthorizedAt      *time.Time        `json:"authorized_at"` // fechaAutorizacion del SRI; el número de autorización es el mismo AccessKey
	DocumentCode      string            `json:"document_code"` // "05"
	EmissionType      string            `json:"emission_type"` // "1" Emisión normal (offline)
	Sequential        string            `json:"sequential"`    // "000000001"
	IssueDate         time.Time         `json:"issue_date"`
	Subtotal          float64           `json:"subtotal"`
	TaxTotal          float64           `json:"tax_total"`
	Total             float64           `json:"total"`
	Currency          string            `json:"currency"`
	Status            string            `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Motives           []DebitNoteMotive `gorm:"foreignKey:DebitNoteID" json:"motives"`
	EstablishmentCode string            `json:"establishment_code"`
	EmissionPointCode string            `json:"emission_point_code"`
}

type DebitNoteMotive struct {
	gorm.Model
	DebitNoteID       uint    `json:"debit_note_id"`
	Reason            string  `json:"reason"` // Ej: "Interés por mora"
	Value             float64 `json:"value"`
	TaxCode           string  `json:"tax_code"`
	TaxPercentageCode string  `json:"tax_percentage_code"`
	TaxRate           float64 `json:"tax_rate"` // e.g. 0.12 for 12%
}

func (debitNote *DebitNote) GenerateSequential(db *gorm.DB) (string, error) {
	var counter InvoiceCounter

	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND document_type = ? AND establishment = ? AND emission_point = ?",
			debitNote.TenantID, DebitNoteDocumentCode, debitNote.EstablishmentCode, debitNote.EmissionPointCode).
		First(&counter).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			counter = InvoiceCounter{
				TenantID:      debitNote.TenantID,
				DocumentType:  DebitNoteDocumentCode,
				Establishment: debitNote.EstablishmentCode,
				EmissionPoint: debitNote.EmissionPointCode,
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
	debitNote.Sequential = fmt.Sprintf("%09d", counter.CurrentNumber)

	return debitNote.Sequential, nil
}
