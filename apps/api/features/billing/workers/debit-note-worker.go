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

func InitDebitNoteBroker(ch *amqp.Channel, db *gorm.DB, logger *zap.Logger) {
	q, err := rabbitmq.DeclareQueue(ch, "debit_note_tasks")
	if err != nil {
		logger.Error("Failed to declare queue, debit note signer won't start", zap.Error(err))
		return
	}

	go rabbitmq.StartConsumer(ch, q.Name, handleDebitNoteTask(db, logger))
}

func handleDebitNoteTask(db *gorm.DB, logger *zap.Logger) func(body []byte) error {
	return func(body []byte) error {
		var debitNoteDTO billing_dto.DebitNoteDTO
		if err := json.Unmarshal(body, &debitNoteDTO); err != nil {
			logger.Error("Failed to unmarshal debit note task", zap.Error(err))
			return err
		}

		logger.Info("Processing DebitNote task", zap.Uint64("debit_note_id", debitNoteDTO.DebitNoteID))

		var debitNote billing_models.DebitNote
		if err := db.Unscoped().Preload("Invoice").Preload("Invoice.Patient").First(&debitNote, debitNoteDTO.DebitNoteID).Error; err != nil {
			logger.Error("Failed to find debit note", zap.Error(err))
			return err
		}

		debitNote.Status = billing_models.InvoiceStatusProcessing
		if err := db.Save(&debitNote).Error; err != nil {
			logger.Error("Failed to update debit note status to processing", zap.Uint("debit_note_id", debitNote.ID), zap.Error(err))
			return err
		}

		// 1. Fetch Tenant
		var tenant tenant_models.Tenant
		if err := db.Unscoped().First(&tenant, debitNote.TenantID).Error; err != nil {
			logger.Error("Failed to find Tenant for debit note", zap.Error(err))
			return err
		}

		if tenant.SriP12Path == "" || tenant.SriPassword == "" {
			logger.Error("Tenant missing SRI Signature configuration", zap.Uint("tenant_id", tenant.ID))
			return fmt.Errorf("missing SRI setup for tenant %d", tenant.ID)
		}

		// 2. Fetch dependencies
		var motives []billing_models.DebitNoteMotive
		if err := db.Unscoped().Where("debit_note_id = ?", debitNote.ID).Find(&motives).Error; err != nil {
			return err
		}
		debitNote.Motives = motives

		address := tenant.Address
		if address == "" {
			address = "Dirección no provista"
		}

		// 3. Generate XML
		establishmentCode := debitNote.EstablishmentCode
		emissionCode := debitNote.EmissionPointCode
		sriEnv := sri_services.ResolveSriEnv()
		debitNoteSRI, accessCode, err := sri_services.GenerateDebitNote(debitNote, tenant, establishmentCode, emissionCode, address, sriEnv)
		if err != nil {
			logger.Error("Failed to generate SRI parameters", zap.Error(err))
			return err
		}

		debitNote.AccessKey = &accessCode
		if err := db.Save(&debitNote).Error; err != nil {
			logger.Error("Failed to persist debit note access key", zap.Uint("debit_note_id", debitNote.ID), zap.Error(err))
			return err
		}

		sriXML, err := sri_services.GenerateDebitNoteXml(*debitNoteSRI)
		if err != nil {
			logger.Error("Failed to generate XML structure", zap.Error(err))
			return err
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
			"debit-notes",
			tenant.SriP12Path,
			tenant.SriPassword,
			sriXML,
			accessCode,
			sriEnv,
			func(step string, auth *sri_services.SriAuthorization) error {
				debitNote.Status = statusByStep[step]
				if auth != nil {
					debitNote.AuthorizedAt = &auth.AuthorizedAt
				}
				if err := db.Save(&debitNote).Error; err != nil {
					logger.Error("Failed to update debit note status", zap.Uint("debit_note_id", debitNote.ID), zap.String("status", statusByStep[step]), zap.Error(err))
					return err
				}
				return nil
			},
		)
		if err != nil {
			logger.Error("Failed to run SRI pipeline for debit note", zap.Uint("debit_note_id", debitNote.ID), zap.Error(err))
			return err
		}

		logger.Info("Debit note successfully processed by SRI", zap.Uint64("id", uint64(debitNote.ID)))
		return nil
	}
}
