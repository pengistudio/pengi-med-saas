package sri_document

import (
	"fmt"
	"os"
	"path/filepath"

	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// Kinds lists every comprobante electrónico the lifecycle processes.
var Kinds = []Kind{Invoice, CreditNote, DebitNote}

var Invoice = Kind{
	Name:           "invoice",
	Queue:          "invoice_tasks",
	Folder:         "invoices",
	Model:          &billing_models.Invoice{},
	MessageIDField: "invoice_id",
	BuildXML: func(db *gorm.DB, id uint, tenant tenant_models.Tenant, accessKey string, sriEnv string) (string, error) {
		var invoice billing_models.Invoice
		if err := db.Unscoped().Preload("Patient").First(&invoice, id).Error; err != nil {
			return "", err
		}
		if err := db.Unscoped().Where("invoice_id = ?", id).Find(&invoice.Items).Error; err != nil {
			return "", err
		}
		var priced []pricedItem
		for _, item := range invoice.Items {
			priced = append(priced, pricedItem{item.ProductID, item.UnitPrice, item.TaxRate, item.IceTax})
		}
		products, err := pricedProducts(db, priced)
		if err != nil {
			return "", err
		}
		sriInvoice, err := sri_services.GenerateInvoice(invoice, products, tenant,
			invoice.EstablishmentCode, invoice.EmissionPointCode, establishmentAddress(tenant), sriEnv, accessKey)
		if err != nil {
			return "", err
		}
		return sri_services.GenerateInvoiceXml(*sriInvoice)
	},
	// The RIDE (printed representation) is stored next to the signed XML. It has
	// tax and legal validity (ficha técnica §8.19), but a failure to render it
	// never fails the already-authorized comprobante; it is re-rendered on download.
	OnAuthorized: func(db *gorm.DB, id uint, tenant tenant_models.Tenant, sriEnv string) error {
		var invoice billing_models.Invoice
		if err := db.Unscoped().Preload("Patient").First(&invoice, id).Error; err != nil {
			return err
		}
		if err := db.Unscoped().Where("invoice_id = ?", id).Find(&invoice.Items).Error; err != nil {
			return err
		}
		if invoice.AccessKey == nil {
			return fmt.Errorf("invoice %d has no access key", invoice.ID)
		}
		pdf, err := sri_services.GenerateInvoiceRide(invoice, tenant, establishmentAddress(tenant), sriEnv)
		if err != nil {
			return fmt.Errorf("render RIDE: %w", err)
		}
		dir := filepath.Join("storage", "tenants", fmt.Sprint(tenant.ID), "invoices")
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, *invoice.AccessKey+".pdf"), pdf, 0644)
	},
}

var CreditNote = Kind{
	Name:           "credit_note",
	Queue:          "credit_note_tasks",
	Folder:         "credit-notes",
	Model:          &billing_models.CreditNote{},
	MessageIDField: "credit_note_id",
	BuildXML: func(db *gorm.DB, id uint, tenant tenant_models.Tenant, accessKey string, sriEnv string) (string, error) {
		var creditNote billing_models.CreditNote
		if err := db.Unscoped().Preload("Invoice").Preload("Invoice.Patient").First(&creditNote, id).Error; err != nil {
			return "", err
		}
		if err := db.Unscoped().Where("credit_note_id = ?", id).Find(&creditNote.Items).Error; err != nil {
			return "", err
		}
		var priced []pricedItem
		for _, item := range creditNote.Items {
			priced = append(priced, pricedItem{item.ProductID, item.UnitPrice, item.TaxRate, item.IceTax})
		}
		products, err := pricedProducts(db, priced)
		if err != nil {
			return "", err
		}
		sriCreditNote, err := sri_services.GenerateCreditNote(creditNote, products, tenant,
			creditNote.EstablishmentCode, creditNote.EmissionPointCode, establishmentAddress(tenant), sriEnv, accessKey)
		if err != nil {
			return "", err
		}
		return sri_services.GenerateCreditNoteXml(*sriCreditNote)
	},
}

var DebitNote = Kind{
	Name:           "debit_note",
	Queue:          "debit_note_tasks",
	Folder:         "debit-notes",
	Model:          &billing_models.DebitNote{},
	MessageIDField: "debit_note_id",
	BuildXML: func(db *gorm.DB, id uint, tenant tenant_models.Tenant, accessKey string, sriEnv string) (string, error) {
		var debitNote billing_models.DebitNote
		if err := db.Unscoped().Preload("Invoice").Preload("Invoice.Patient").First(&debitNote, id).Error; err != nil {
			return "", err
		}
		if err := db.Unscoped().Where("debit_note_id = ?", id).Find(&debitNote.Motives).Error; err != nil {
			return "", err
		}
		sriDebitNote, err := sri_services.GenerateDebitNote(debitNote, tenant,
			debitNote.EstablishmentCode, debitNote.EmissionPointCode, establishmentAddress(tenant), sriEnv, accessKey)
		if err != nil {
			return "", err
		}
		return sri_services.GenerateDebitNoteXml(*sriDebitNote)
	},
}

// pricedItem is the price and taxes frozen on a document line when it was issued.
type pricedItem struct {
	ProductID uint
	UnitPrice float64
	TaxRate   float64
	IceTax    float64
}

// pricedProducts loads each line's catalog product with the price and taxes
// frozen on the line, which is what the XML must declare.
func pricedProducts(db *gorm.DB, items []pricedItem) ([]billing_models.CatalogItem, error) {
	var products []billing_models.CatalogItem
	for _, item := range items {
		var product billing_models.CatalogItem
		if err := db.Unscoped().First(&product, item.ProductID).Error; err != nil {
			return nil, fmt.Errorf("load catalog item %d: %w", item.ProductID, err)
		}
		product.UnitPrice = item.UnitPrice
		product.Tax = item.TaxRate
		product.IceTax = item.IceTax
		products = append(products, product)
	}
	return products, nil
}

// establishmentAddress falls back to a placeholder: Pengi has no per-establishment
// address yet, and the SRI requires one.
func establishmentAddress(tenant tenant_models.Tenant) string {
	if tenant.Address == "" {
		return "Dirección no provista"
	}
	return tenant.Address
}
