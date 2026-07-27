package billing_workers

import (
	"encoding/json"
	"fmt"

	"pengi-med-saas/core/brokers/rabbitmq"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func InitCreditNoteBroker(ch *amqp.Channel, db *gorm.DB, logger *zap.Logger) {
	q, err := rabbitmq.DeclareQueue(ch, "credit_note_tasks")
	if err != nil {
		logger.Error("Failed to declare queue, credit note signer won't start", zap.Error(err))
		return
	}

	go rabbitmq.StartConsumer(ch, q.Name, handleCreditNoteTask(db, logger))
}

func handleCreditNoteTask(db *gorm.DB, logger *zap.Logger) func(body []byte) error {
	return func(body []byte) error {
		var creditNoteDTO billing_dto.CreditNoteDTO
		if err := json.Unmarshal(body, &creditNoteDTO); err != nil {
			logger.Error("Failed to unmarshal credit note task", zap.Error(err))
			return err
		}

		logger.Info("Processing CreditNote task", zap.Uint64("credit_note_id", creditNoteDTO.CreditNoteID))

		var creditNote billing_models.CreditNote
		if err := db.Unscoped().Preload("Invoice").Preload("Invoice.Patient").First(&creditNote, creditNoteDTO.CreditNoteID).Error; err != nil {
			logger.Error("Failed to find credit note", zap.Error(err))
			return err
		}

		fail := func(err error) error {
			creditNote.Status = billing_models.InvoiceStatusFailed
			msg := err.Error()
			creditNote.ErrorMessage = &msg
			if saveErr := db.Save(&creditNote).Error; saveErr != nil {
				logger.Error("Failed to persist credit note failure status", zap.Uint("credit_note_id", creditNote.ID), zap.Error(saveErr))
			}
			return err
		}

		creditNote.Status = billing_models.InvoiceStatusProcessing
		if err := db.Save(&creditNote).Error; err != nil {
			logger.Error("Failed to update credit note status to processing", zap.Uint("credit_note_id", creditNote.ID), zap.Error(err))
			return err
		}

		// 1. Fetch Tenant
		var tenant tenant_models.Tenant
		if err := db.Unscoped().First(&tenant, creditNote.TenantID).Error; err != nil {
			logger.Error("Failed to find Tenant for credit note", zap.Error(err))
			return fail(err)
		}

		if tenant.SriP12Path == "" || tenant.SriPassword == "" {
			logger.Error("Tenant missing SRI Signature configuration", zap.Uint("tenant_id", tenant.ID))
			return fail(fmt.Errorf("missing SRI setup for tenant %d", tenant.ID))
		}

		// 2. Fetch dependencies
		var items []billing_models.CreditNoteItem
		if err := db.Unscoped().Where("credit_note_id = ?", creditNote.ID).Find(&items).Error; err != nil {
			return fail(err)
		}
		creditNote.Items = items

		var products []sri_services.CatalogItem
		for _, item := range items {
			var product billing_models.CatalogItem
			if err := db.Unscoped().First(&product, item.ProductID).Error; err != nil {
				return fail(err)
			}
			product.UnitPrice = item.UnitPrice
			product.Tax = item.TaxRate
			product.IceTax = item.IceTax
			products = append(products, product)
		}

		address := tenant.Address
		if address == "" {
			address = "Dirección no provista"
		}

		// 3. Generate XML
		establishmentCode := creditNote.EstablishmentCode
		emissionCode := creditNote.EmissionPointCode
		sriEnv := sri_services.ResolveSriEnv()
		creditNoteSRI, accessCode, err := sri_services.GenerateCreditNote(creditNote, products, tenant, establishmentCode, emissionCode, address, sriEnv)
		if err != nil {
			logger.Error("Failed to generate SRI parameters", zap.Error(err))
			return fail(err)
		}

		creditNote.AccessKey = &accessCode
		if err := db.Save(&creditNote).Error; err != nil {
			logger.Error("Failed to persist credit note access key", zap.Uint("credit_note_id", creditNote.ID), zap.Error(err))
			return fail(err)
		}

		sriXML, err := sri_services.GenerateCreditNoteXml(*creditNoteSRI)
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
			"credit-notes",
			tenant.SriP12Path,
			tenant.SriPassword,
			sriXML,
			accessCode,
			sriEnv,
			func(step string, auth *sri_services.SriAuthorization) error {
				creditNote.Status = statusByStep[step]
				if auth != nil {
					creditNote.AuthorizedAt = &auth.AuthorizedAt
				}
				if err := db.Save(&creditNote).Error; err != nil {
					logger.Error("Failed to update credit note status", zap.Uint("credit_note_id", creditNote.ID), zap.String("status", statusByStep[step]), zap.Error(err))
					return err
				}
				return nil
			},
		)
		if err != nil {
			logger.Error("Failed to run SRI pipeline for credit note", zap.Uint("credit_note_id", creditNote.ID), zap.Error(err))
			return fail(err)
		}

		logger.Info("Credit note successfully processed by SRI", zap.Uint64("id", uint64(creditNote.ID)))
		return nil
	}
}
