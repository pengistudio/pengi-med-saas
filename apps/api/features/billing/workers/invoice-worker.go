package billing_workers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"pengi-med-saas/core/brokers/rabbitmq"
	"pengi-med-saas/core/utils"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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

		invoice.Status = "processing"
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to update invoice status to processing", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return err
		}

		// 1. Fetch Tenant
		var tenant tenant_models.Tenant
		if err := db.Unscoped().First(&tenant, invoice.TenantID).Error; err != nil {
			logger.Error("Failed to find Tenant for invoice", zap.Error(err))
			return err
		}

		if tenant.SriP12Path == "" || tenant.SriPassword == "" {
			logger.Error("Tenant missing SRI Signature configuration", zap.Uint("tenant_id", tenant.ID))
			return fmt.Errorf("missing SRI setup for tenant %d", tenant.ID)
		}

		// 2. Fetch dependencies
		var items []billing_models.InvoiceItem
		if err := db.Unscoped().Where("invoice_id = ?", invoice.ID).Find(&items).Error; err != nil {
			return err
		}
		invoice.Items = items

		// Rebuild CatalogItems specifically for generation
		var products []billing_models.CatalogItem
		for _, item := range items {
			var product billing_models.CatalogItem
			if err := db.Unscoped().First(&product, item.ProductID).Error; err != nil {
				return err
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
		sriInvoice, accessCode, err := sri_services.GenerateInvoice(invoice, products, tenant, establishmentCode, emissionCode, address)
		if err != nil {
			logger.Error("Failed to generate SRI parameters", zap.Error(err))
			return err
		}

		invoice.AccessKey = &accessCode
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to persist invoice access key", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return err
		}

		sriXML, err := sri_services.GenerateInvoiceXml(*sriInvoice)
		if err != nil {
			logger.Error("Failed to generate XML structure", zap.Error(err))
			return err
		}

		// 4. Sign XML
		p12Buffer, err := os.ReadFile(tenant.SriP12Path)
		if err != nil {
			logger.Error("Failed to read P12 file", zap.String("path", tenant.SriP12Path), zap.Error(err))
			return err
		}

		sriClient := utils.NewSriSignerClient()
		responseClient, err := sriClient.SignXML(p12Buffer, tenant.SriPassword, []byte(sriXML))
		if err != nil {
			logger.Error("Failed to sign XML", zap.Error(err))
			return err
		}

		finalXML := responseClient.XML

		// 5. Save Authorized XML to Tenant Storage
		xmlDestDir := filepath.Join("storage", "tenants", fmt.Sprint(tenant.ID), "invoices")
		os.MkdirAll(xmlDestDir, os.ModePerm)

		xmlPath := filepath.Join(xmlDestDir, fmt.Sprintf("%s.xml", accessCode))
		if err := os.WriteFile(xmlPath, []byte(finalXML), 0644); err != nil {
			logger.Error("Failed to save final signed XML", zap.Error(err))
			return err
		}

		invoice.Status = "signed"
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to update invoice status to signed", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return err
		}

		// 6. Validate & Authorize SRI
		sriEnv := os.Getenv("SRI_ENV") // Usually "1" (Pruebas) or "2" (Producción)
		if sriEnv == "" {
			sriEnv = "1"
		}

		if _, err := sriClient.ValidateXMLWithSRI([]byte(finalXML), sriEnv); err != nil {
			logger.Error("Failed to Validate XML with SRI", zap.Error(err))
			return err
		}

		invoice.Status = "validated"
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to update invoice status to validated", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return err
		}

		authResp, err := sriClient.AuthorizeXMLWithSRI(*invoice.AccessKey, sriEnv)
		if err != nil {
			// SRI may still be processing; leave as validated so it can be retried
			if strings.Contains(err.Error(), "autorizacion") || strings.Contains(err.Error(), "EN PROCESAMIENTO") {
				logger.Warn("SRI authorization pending, invoice left as validated", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
				return nil
			}
			logger.Error("Failed to Authorize XML with SRI", zap.Error(err))
			return err
		}
		_ = authResp

		invoice.Status = "authorized"
		if err := db.Save(&invoice).Error; err != nil {
			logger.Error("Failed to update invoice status to authorized", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
			return err
		}

		logger.Info("Invoice successfully processed by SRI", zap.Uint64("id", uint64(invoice.ID)))
		return nil
	}
}
