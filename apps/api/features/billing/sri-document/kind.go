package sri_document

import (
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// Kind adapts one type of comprobante electrónico (factura, nota de crédito, nota
// de débito) to the lifecycle. The lifecycle owns every status transition; a kind
// only says where the document lives and how its XML is built.
//
// Model's table must have the shared lifecycle columns: tenant_id, status,
// access_key, authorized_at, error_code, error_message, document_code,
// emission_type, sequential, issue_date, establishment_code, emission_point_code.
type Kind struct {
	Name   string // for logs, e.g. "invoice"
	Queue  string // RabbitMQ queue its tasks go through
	Folder string // tenant storage folder for its signed XML
	Model  any    // pointer to a zero value of its GORM model
	// MessageIDField is the JSON field carrying the document ID in a task.
	MessageIDField string
	// BuildXML renders the unsigned comprobante XML with the given access key.
	BuildXML func(db *gorm.DB, id uint, tenant tenant_models.Tenant, accessKey string, sriEnv string) (string, error)
	// OnAuthorized runs side effects (e.g. rendering the RIDE) once the SRI has
	// authorized the document. Optional; its errors never fail the authorized
	// document.
	OnAuthorized func(db *gorm.DB, id uint, tenant tenant_models.Tenant, sriEnv string) error
}
