package billing_workers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"pengi-med-saas/core/brokers/rabbitmq"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// generateAndStoreInvoiceRide renders the RIDE PDF for an authorized invoice and
// saves it next to the signed XML in tenant storage.
func generateAndStoreInvoiceRide(invoice billing_models.Invoice, tenant tenant_models.Tenant, establishmentAddress string, sriEnv string) error {
	if invoice.AccessKey == nil {
		return fmt.Errorf("invoice %d has no access key", invoice.ID)
	}

	pdfBytes, err := sri_services.GenerateInvoiceRide(invoice, tenant, establishmentAddress, sriEnv)
	if err != nil {
		return fmt.Errorf("failed to render RIDE: %w", err)
	}

	destDir := filepath.Join("storage", "tenants", fmt.Sprint(tenant.ID), "invoices")
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create storage dir: %w", err)
	}

	pdfPath := filepath.Join(destDir, fmt.Sprintf("%s.pdf", *invoice.AccessKey))
	if err := os.WriteFile(pdfPath, pdfBytes, 0644); err != nil {
		return fmt.Errorf("failed to save RIDE PDF: %w", err)
	}

	return nil
}

func InitInvoiceBroker(ch *amqp.Channel, db *gorm.DB, logger *zap.Logger) {
	q, err := rabbitmq.DeclareQueue(ch, "invoice_tasks")
	if err != nil {
		logger.Error("Failed to declare queue, invoice signer won't start", zap.Error(err))
		return
	}

	go rabbitmq.StartConsumer(ch, q.Name, handleInvoiceTask(db, logger))
}

func handleInvoiceTask(db *gorm.DB, logger *zap.Logger) func(body []byte) error {
	return func(body []byte) error {
		var invoiceDTO billing_dto.InvoiceDTO
		if err := json.Unmarshal(body, &invoiceDTO); err != nil {
			logger.Error("Failed to unmarshal invoice task", zap.Error(err))
			return err
		}

		logger.Info("Processing Invoice task", zap.Uint64("invoice_id", invoiceDTO.InvoiceID))

		var invoice billing_models.Invoice
		if err := db.Unscoped().Preload("Patient").First(&invoice, invoiceDTO.InvoiceID).Error; err != nil {
			logger.Error("Failed to find invoice", zap.Error(err))
			return err
		}

		fail := func(err error) error {
			invoice.Status = billing_models.InvoiceStatusFailed
			msg := err.Error()
			invoice.ErrorMessage = &msg
			if saveErr := db.Save(&invoice).Error; saveErr != nil {
				logger.Error("Failed to persist invoice failure status", zap.Uint("invoice_id", invoice.ID), zap.Error(saveErr))
			}
			return err
		}

		invoice.Status = billing_models.InvoiceStatusProcessing
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to update invoice status to processing", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return err
		}

		// 1. Fetch Tenant
		var tenant tenant_models.Tenant
		if err := db.Unscoped().First(&tenant, invoice.TenantID).Error; err != nil {
			logger.Error("Failed to find Tenant for invoice", zap.Error(err))
			return fail(err)
		}

		if tenant.SriP12Path == "" || tenant.SriPassword == "" {
			logger.Error("Tenant missing SRI Signature configuration", zap.Uint("tenant_id", tenant.ID))
			return fail(fmt.Errorf("missing SRI setup for tenant %d", tenant.ID))
		}

		// 2. Fetch dependencies
		var items []billing_models.InvoiceItem
		if err := db.Unscoped().Where("invoice_id = ?", invoice.ID).Find(&items).Error; err != nil {
			return fail(err)
		}
		invoice.Items = items

		// Rebuild CatalogItems specifically for generation
		var products []billing_models.CatalogItem
		for _, item := range items {
			var product billing_models.CatalogItem
			if err := db.Unscoped().First(&product, item.ProductID).Error; err != nil {
				return fail(err)
			}
			product.UnitPrice = item.UnitPrice
			// product.Discount = item.Discount (no longer part of CatalogItem schema)
			product.Tax = item.TaxRate
			product.IceTax = item.IceTax
			products = append(products, product)
		}

		// Stub out establishment address since it's missing in Pengi's schema right now
		address := tenant.Address
		if address == "" {
			address = "Dirección no provista"
		}

		// 3. Generate XML
		establishmentCode := invoice.EstablishmentCode
		emissionCode := invoice.EmissionPointCode
		sriEnv := sri_services.ResolveSriEnv()
		sriInvoice, accessCode, err := sri_services.GenerateInvoice(invoice, products, tenant, establishmentCode, emissionCode, address, sriEnv)
		if err != nil {
			logger.Error("Failed to generate SRI parameters", zap.Error(err))
			return fail(err)
		}

		invoice.AccessKey = &accessCode
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to persist invoice access key", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return fail(err)
		}

		sriXML, err := sri_services.GenerateInvoiceXml(*sriInvoice)
		if err != nil {
			logger.Error("Failed to generate XML structure", zap.Error(err))
			return fail(err)
		}

		// 4. Sign, validate & authorize (shared pipeline, agnostic to document type)
		statusByStep := map[string]string{
			sri_services.StatusSigned:     billing_models.InvoiceStatusSigned,
			sri_services.StatusValidated:  billing_models.InvoiceStatusValidated,
			sri_services.StatusAuthorized: billing_models.InvoiceStatusAuthorized,
		}

		err = sri_services.RunSriPipeline(
			logger,
			tenant.ID,
			"invoices",
			tenant.SriP12Path,
			tenant.SriPassword,
			sriXML,
			accessCode,
			sriEnv,
			func(step string, auth *sri_services.SriAuthorization) error {
				invoice.Status = statusByStep[step]
				if auth != nil {
					invoice.AuthorizedAt = &auth.AuthorizedAt
				}
				if err := db.Save(&invoice).Error; err != nil {
					logger.Error("Failed to update invoice status", zap.Uint("invoice_id", invoice.ID), zap.String("status", statusByStep[step]), zap.Error(err))
					return err
				}

				if step == sri_services.StatusAuthorized {
					// The RIDE is a courtesy PDF, not the legal document (the XML is).
					// Never fail the pipeline over a RIDE generation error.
					if err := generateAndStoreInvoiceRide(invoice, tenant, address, sriEnv); err != nil {
						logger.Error("Failed to generate RIDE for invoice", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
					}
				}
				return nil
			},
		)
		if err != nil {
			logger.Error("Failed to run SRI pipeline for invoice", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return fail(err)
		}

		logger.Info("Invoice successfully processed by SRI", zap.Uint64("id", uint64(invoice.ID)))
		return nil
	}
}
