// Package sri_document owns the lifecycle of a comprobante electrónico: from the
// moment it is queued until the SRI authorizes (or definitively rejects) it.
//
// Status progression, persisted in each document's own table:
//
//	pending → processing → signed → validated (= recibido) → authorized
//	                                         ↘ rejected (NO AUTORIZADO)
//	failed / connection_error: before reception, retried automatically when transient
//	rejected: resent only when a user retries after correcting the cause
//
// Invariant (docs/adr/0001-clave-de-acceso-inmutable.md): a document's access key
// is generated once and never changes. While the SRI holds it (received, awaiting
// authorization) it is never signed or sent again — only its authorization is
// queried. After a NO AUTORIZADO it is corrected and resent with the same key
// (ficha técnica SRI offline v2.26 §5.10, §5.12).
package sri_document

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/tenantdb"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StuckThreshold is how long a document may sit in "processing" or "signed"
// before we assume its worker died mid-run and let another attempt claim it.
// Re-running is safe even if the SRI already received it (ADR 0001).
const StuckThreshold = 15 * time.Minute

type Lifecycle struct {
	db        *gorm.DB
	logger    *zap.Logger
	gateway   Gateway
	storage   Storage
	publisher Publisher
	sriEnv    string // SRI ambiente code: "1" pruebas, "2" producción
}

func New(db *gorm.DB, logger *zap.Logger, gateway Gateway, storage Storage, publisher Publisher, sriEnv string) *Lifecycle {
	// Processing and sweeping span every tenant (docs/adr/0002); Enqueue gets a
	// tenant-bound handle from its caller instead.
	return &Lifecycle{db: tenantdb.System(db), logger: logger, gateway: gateway, storage: storage, publisher: publisher, sriEnv: sriEnv}
}

// document is the slice of a comprobante the lifecycle reads, common to every kind.
type document struct {
	ID                uint
	TenantID          uint
	Status            string
	AccessKey         *string
	DocumentCode      string
	EmissionType      string
	Sequential        string
	IssueDate         time.Time
	EstablishmentCode string
	EmissionPointCode string
}

// Process runs one processing attempt for a document. A received document only
// has its authorization queried; one already being processed, authorized or
// rejected is left alone. Errors for which IsRetryable is true should be retried.
func (l *Lifecycle) Process(kind Kind, id uint64) error {
	doc, err := l.load(l.db.Unscoped(), kind, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		l.logger.Warn("SRI document not found, skipping", zap.String("kind", kind.Name), zap.Uint64("id", id))
		return nil
	}
	if err != nil {
		return fmt.Errorf("%w: %w", errTransient, err)
	}

	if doc.Status == billing_models.InvoiceStatusValidated {
		tenant, err := l.tenant(doc.TenantID)
		if err != nil {
			return err
		}
		l.authorize(kind, doc, tenant, *doc.AccessKey)
		return nil
	}

	claimed, err := l.claim(kind, doc.ID)
	if err != nil {
		return err
	}
	if !claimed {
		l.logger.Info("SRI document not claimable, skipping", zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.String("status", doc.Status))
		return nil
	}
	return l.attempt(kind, doc)
}

// attempt signs, submits and authorizes a claimed document.
func (l *Lifecycle) attempt(kind Kind, doc document) (err error) {
	accessKey := ""
	if doc.AccessKey != nil {
		accessKey = *doc.AccessKey
	}
	fail := func(err error, code string) error {
		return l.fail(kind, doc, accessKey, err, code)
	}
	defer func() {
		if r := recover(); r != nil {
			fail(fmt.Errorf("panic: %v", r), ErrorCodeInternal)
			panic(r)
		}
	}()

	tenant, err := l.tenant(doc.TenantID)
	if err != nil {
		return fail(err, ErrorCodeInternal)
	}
	if tenant.SriP12Path == "" || tenant.SriPassword == "" {
		return fail(fmt.Errorf("missing SRI signature setup for tenant %d", tenant.ID), ErrorCodeMissingSignature)
	}

	if accessKey, err = l.ensureAccessKey(kind, doc, tenant); err != nil {
		return fail(err, ErrorCodeInternal)
	}
	xml, err := kind.BuildXML(l.db, doc.ID, tenant, accessKey, l.sriEnv)
	if err != nil {
		return fail(fmt.Errorf("build XML: %w", err), ErrorCodeInternal)
	}
	p12, err := l.storage.ReadCertificate(tenant.SriP12Path)
	if err != nil {
		return fail(fmt.Errorf("read P12 certificate: %w", err), ErrorCodeMissingSignature)
	}
	signed, err := l.gateway.Sign(p12, tenant.SriPassword, xml)
	if err != nil {
		return fail(fmt.Errorf("sign XML: %w", err), ErrorCodeInternal)
	}
	if err := l.storage.SaveSignedXML(tenant.ID, kind.Folder, accessKey, signed); err != nil {
		return fail(fmt.Errorf("store signed XML: %w", err), ErrorCodeInternal)
	}
	if err := l.update(kind, doc.ID, map[string]any{"status": billing_models.InvoiceStatusSigned}); err != nil {
		return fail(err, ErrorCodeInternal)
	}

	if err := l.gateway.SubmitForReception(signed, l.sriEnv); err != nil {
		var rejection *Rejection
		if errors.As(err, &rejection) {
			return fail(fmt.Errorf("SRI returned the comprobante: %w", err), ErrorCodeReturned)
		}
		return fail(fmt.Errorf("submit to SRI reception: %w", err), ErrorCodeInternal)
	}
	if err := l.update(kind, doc.ID, map[string]any{
		"status":        billing_models.InvoiceStatusValidated,
		"error_code":    nil,
		"error_message": nil,
	}); err != nil {
		// Received but not recorded: leave it signed; a stuck-sweep retry resends
		// the same key and the SRI answers "already registered".
		return fmt.Errorf("%w: %w", errTransient, err)
	}

	l.authorize(kind, doc, tenant, accessKey)
	return nil
}

// authorize queries the SRI's decision for a received document. Anything short
// of a decision leaves it validated for the next sweep to query again.
func (l *Lifecycle) authorize(kind Kind, doc document, tenant tenant_models.Tenant, accessKey string) {
	log := l.logger.With(zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.String("access_key", accessKey))

	auth, err := l.gateway.QueryAuthorization(accessKey, l.sriEnv)
	if err != nil {
		log.Warn("SRI authorization query failed, document left as received", zap.Error(err))
		return
	}

	switch auth.Status {
	case AuthorizationPending:
		log.Info("SRI authorization pending, document left as received")

	case NotAuthorized:
		if err := l.update(kind, doc.ID, map[string]any{
			"status":        billing_models.InvoiceStatusRejected,
			"error_code":    ErrorCodeNotAuthorized,
			"error_message": auth.Message,
		}); err != nil {
			log.Error("failed to persist SRI rejection", zap.Error(err))
			return
		}
		if err := l.storage.RemoveSignedXML(doc.TenantID, kind.Folder, accessKey); err != nil {
			log.Error("failed to remove signed XML of rejected document", zap.Error(err))
		}

	case Authorized:
		if err := l.update(kind, doc.ID, map[string]any{
			"status":        billing_models.InvoiceStatusAuthorized,
			"authorized_at": auth.AuthorizedAt,
			"error_code":    nil,
			"error_message": nil,
		}); err != nil {
			log.Error("failed to persist SRI authorization", zap.Error(err))
			return
		}
		if kind.OnAuthorized != nil {
			if err := kind.OnAuthorized(l.db, doc.ID, tenant, l.sriEnv); err != nil {
				log.Error("post-authorization step failed", zap.Error(err))
			}
		}
	}
}

// claimableStatuses can start a processing attempt. "processing" and "signed"
// are claimable only once stale (their worker died mid-run).
var claimableStatuses = []string{
	billing_models.InvoiceStatusPending,
	billing_models.InvoiceStatusFailed,
	billing_models.InvoiceStatusConnectionError,
}

var stuckStatuses = []string{
	billing_models.InvoiceStatusProcessing,
	billing_models.InvoiceStatusSigned,
}

// claim atomically moves a document to "processing" so two deliveries of the
// same task never run concurrently.
func (l *Lifecycle) claim(kind Kind, id uint) (bool, error) {
	res := l.db.Unscoped().Model(kind.Model).
		Where("id = ? AND (status IN ? OR (status IN ? AND updated_at < ?))",
			id, claimableStatuses, stuckStatuses, time.Now().Add(-StuckThreshold)).
		Update("status", billing_models.InvoiceStatusProcessing)
	if res.Error != nil {
		return false, fmt.Errorf("%w: %w", errTransient, res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ensureAccessKey returns the document's access key, generating and persisting it
// on the first attempt only (ADR 0001).
func (l *Lifecycle) ensureAccessKey(kind Kind, doc document, tenant tenant_models.Tenant) (string, error) {
	if doc.AccessKey != nil && *doc.AccessKey != "" {
		return *doc.AccessKey, nil
	}
	accessKey, err := sri_services.GenerateAccessKey(doc.IssueDate, doc.DocumentCode, tenant.TaxID,
		doc.EstablishmentCode, doc.EmissionPointCode, doc.Sequential, doc.EmissionType, l.sriEnv)
	if err != nil {
		return "", err
	}
	if err := l.update(kind, doc.ID, map[string]any{"access_key": accessKey}); err != nil {
		return "", err
	}
	return accessKey, nil
}

// fail records a failed attempt before reception and returns err for the caller.
// Unreachable errors become connection_error regardless of code. The signed XML
// is removed (the next attempt re-signs); the access key is kept.
func (l *Lifecycle) fail(kind Kind, doc document, accessKey string, err error, code string) error {
	status := billing_models.InvoiceStatusFailed
	if errors.Is(err, ErrUnreachable) {
		status, code = billing_models.InvoiceStatusConnectionError, ErrorCodeConnection
	}
	if updErr := l.update(kind, doc.ID, map[string]any{
		"status":        status,
		"error_code":    code,
		"error_message": err.Error(),
	}); updErr != nil {
		l.logger.Error("failed to persist SRI document failure", zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.Error(updErr))
	}
	if accessKey != "" {
		if rmErr := l.storage.RemoveSignedXML(doc.TenantID, kind.Folder, accessKey); rmErr != nil {
			l.logger.Error("failed to remove signed XML after failure", zap.String("kind", kind.Name), zap.Uint("id", doc.ID), zap.Error(rmErr))
		}
	}
	return err
}

func (l *Lifecycle) tenant(id uint) (tenant_models.Tenant, error) {
	var tenant tenant_models.Tenant
	if err := l.db.Unscoped().First(&tenant, id).Error; err != nil {
		return tenant, fmt.Errorf("load tenant %d: %w", id, err)
	}
	return tenant, nil
}

func (l *Lifecycle) load(db *gorm.DB, kind Kind, id uint64) (document, error) {
	var doc document
	err := db.Model(kind.Model).Where("id = ?", id).Take(&doc).Error
	if err != nil {
		return document{}, fmt.Errorf("load %s %d: %w", kind.Name, id, err)
	}
	return doc, nil
}

func (l *Lifecycle) update(kind Kind, id uint, fields map[string]any) error {
	return l.db.Unscoped().Model(kind.Model).Where("id = ?", id).Updates(fields).Error
}
