package whatsapp_models

import "gorm.io/gorm"

// Template names of the catalog (whatsapp_services.Catalog). ReminderTemplate
// (message.go) is the fifth one.
const (
	TemplateContinueConversation = "pengi_continuar_conversacion"
	TemplateResultsReady         = "pengi_resultados_listos"
	TemplateReschedule           = "pengi_reprogramar_cita"
	TemplateFollowUp             = "pengi_control_seguimiento"
)

// WhatsAppTemplate is the review status of one catalog template on the
// tenant's WABA (Meta's values: APPROVED, PENDING, REJECTED, PAUSED, ...).
// One row per tenant and template name; the reminder's status is also
// mirrored on WhatsAppAccount.TemplateStatus for the existing settings UI.
// Rows are hard-deleted when the account is disconnected (they belong to the
// WABA, which may change on reconnect).
type WhatsAppTemplate struct {
	gorm.Model
	TenantID uint   `json:"tenant_id" gorm:"not null;uniqueIndex:idx_whatsapp_template_tenant_name"`
	Name     string `json:"name" gorm:"not null;uniqueIndex:idx_whatsapp_template_tenant_name"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
	MetaID   string `json:"meta_id"`
}

func (WhatsAppTemplate) TableName() string { return "whatsapp_templates" }
