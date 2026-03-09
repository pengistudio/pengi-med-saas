package billing_models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	clinical_models "pengi-med-saas/features/clinical/models"
)

type Invoice struct {
	gorm.Model
	TenantID          uint                    `gorm:"index" json:"tenant_id"` // Multi-tenant
	AccessKey         *string                 `gorm:"size:49" json:"access_key"`
	DocumentCode      string                  `json:"document_code"` // Factura
	EmissionType      string                  `json:"emission_type"` // "1" Emisión normal (offline)
	Sequential        string                  `json:"sequential"`    // "000000001"
	IssueDate         time.Time               `json:"issue_date"`
	PatientID         uint                    `json:"patient_id"` // FK a Patient
	Patient           clinical_models.Patient `gorm:"foreignKey:PatientID" json:"patient"`
	Subtotal          float64                 `json:"subtotal"`
	Discount          float64                 `json:"discount"`
	TaxTotal          float64                 `json:"tax_total"`
	Total             float64                 `json:"total"`
	Currency          string                  `json:"currency"`
	PaymentMethod     string                  `json:"payment_method"`
	Term              string                  `json:"term"`
	TimeUnit          string                  `json:"time_unit"`
	Status            string                  `gorm:"type:varchar(20);default:'draft'" json:"status"` // "draft", "signed", "sent", "authorized", "rejected"
	Items             []InvoiceItem           `gorm:"foreignKey:InvoiceID" json:"items"`
	CompanyID         uint                    `json:"company_id"` // Optional if you need it, or can be replaced by Tenant logic
	EstablishmentCode string                  `json:"establishment_code"`
	EmissionPointCode string                  `json:"emission_point_code"`
}

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
