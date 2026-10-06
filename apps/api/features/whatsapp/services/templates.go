package whatsapp_services

import (
	"fmt"
	"strings"
	"time"

	"pengi-med-saas/core/whatsapp"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReminderLanguage is the language of every catalog template.
const ReminderLanguage = "es"

// Template variables, in the order of the template's {{n}} placeholders.
const (
	VarPatient = "patient"
	VarClinic  = "clinic"
	VarDate    = "date"
	VarTime    = "time"
)

// TemplateSpec is one template of the catalog: what is created on the WABA
// and how it is filled.
type TemplateSpec struct {
	Definition whatsapp.TemplateDefinition
	// Variables names each {{n}} placeholder, in order (Var* constants).
	Variables []string
	// NeedsAppointment: date and time come from an appointment of the patient.
	NeedsAppointment bool
	// Inbox: a user can send it from the inbox. The reminder can't: its
	// buttons carry the id of the reminder message.
	Inbox bool
}

// The catalog templates are message content reviewed by Meta, written in the
// patients' language; they are not interface text, so they are not i18n keys.
// All are UTILITY: neutral wording about the patient's care, no promotions,
// inviting a reply (which opens the 24 h window for free-text replies).
var reminderTemplate = TemplateSpec{
	Definition: whatsapp.TemplateDefinition{
		Name:     whatsapp_models.ReminderTemplate,
		Language: ReminderLanguage,
		Category: "UTILITY",
		Body:     "Hola {{1}}, te recordamos tu cita en {{2}} el {{3}} a las {{4}}. Por favor confirma tu asistencia o cancela si no podrás asistir.",
		Example:  []string{"María Pérez", "Clínica Pengi", "lunes 6 de octubre", "10:30"},
		Buttons:  []whatsapp.TemplateButton{{Text: "Confirmar"}, {Text: "Cancelar"}},
	},
	Variables:        []string{VarPatient, VarClinic, VarDate, VarTime},
	NeedsAppointment: true,
}

// Catalog is every template Pengi creates on a connected WABA.
var Catalog = []TemplateSpec{
	reminderTemplate,
	{
		Definition: whatsapp.TemplateDefinition{
			Name:     whatsapp_models.TemplateContinueConversation,
			Language: ReminderLanguage,
			Category: "UTILITY",
			Body:     "Hola {{1}}, te escribimos de {{2}} para dar seguimiento a tu atención. Responde a este mensaje para continuar la conversación.",
			Example:  []string{"María Pérez", "Clínica Pengi"},
		},
		Variables: []string{VarPatient, VarClinic},
		Inbox:     true,
	},
	{
		Definition: whatsapp.TemplateDefinition{
			Name:     whatsapp_models.TemplateResultsReady,
			Language: ReminderLanguage,
			Category: "UTILITY",
			Body:     "Hola {{1}}, tus resultados ya están disponibles en {{2}}. Responde a este mensaje si deseas coordinar su entrega o revisión.",
			Example:  []string{"María Pérez", "Clínica Pengi"},
		},
		Variables: []string{VarPatient, VarClinic},
		Inbox:     true,
	},
	{
		Definition: whatsapp.TemplateDefinition{
			Name:     whatsapp_models.TemplateReschedule,
			Language: ReminderLanguage,
			Category: "UTILITY",
			Body:     "Hola {{1}}, necesitamos reprogramar tu cita en {{2}} del {{3}} a las {{4}}. Responde a este mensaje para coordinar una nueva fecha.",
			Example:  []string{"María Pérez", "Clínica Pengi", "lunes 6 de octubre", "10:30"},
		},
		Variables:        []string{VarPatient, VarClinic, VarDate, VarTime},
		NeedsAppointment: true,
		Inbox:            true,
	},
	{
		Definition: whatsapp.TemplateDefinition{
			Name:     whatsapp_models.TemplateFollowUp,
			Language: ReminderLanguage,
			Category: "UTILITY",
			Body:     "Hola {{1}}, en {{2}} te recordamos que corresponde tu control de seguimiento. Responde a este mensaje para agendar tu cita.",
			Example:  []string{"María Pérez", "Clínica Pengi"},
		},
		Variables: []string{VarPatient, VarClinic},
		Inbox:     true,
	},
}

// FindTemplate returns the catalog entry named name.
func FindTemplate(name string) (TemplateSpec, bool) {
	for _, t := range Catalog {
		if t.Definition.Name == name {
			return t, true
		}
	}
	return TemplateSpec{}, false
}

// TemplateState is a catalog template's review status on a WABA.
type TemplateState struct {
	Name   string
	Status string
	MetaID string
}

// EnsureTemplates creates every catalog template missing on the WABA and
// returns the status of each one it could look up or create. It keeps going
// after a Graph error, so one failing template doesn't block the others, and
// returns the first error.
func EnsureTemplates(client *whatsapp.Client, token, wabaID string) ([]TemplateState, error) {
	states := make([]TemplateState, 0, len(Catalog))
	var firstErr error
	for _, spec := range Catalog {
		state, err := ensureTemplate(client, token, wabaID, spec.Definition)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		states = append(states, state)
	}
	return states, firstErr
}

func ensureTemplate(client *whatsapp.Client, token, wabaID string, def whatsapp.TemplateDefinition) (TemplateState, error) {
	templates, err := client.ListTemplates(token, wabaID, def.Name)
	if err != nil {
		return TemplateState{}, err
	}
	for _, t := range templates {
		if t.Name == def.Name && t.Language == def.Language {
			return TemplateState{Name: def.Name, Status: t.Status, MetaID: t.ID}, nil
		}
	}
	created, err := client.CreateTemplate(token, wabaID, def)
	if err != nil {
		return TemplateState{}, err
	}
	status := created.Status
	if status == "" {
		status = whatsapp_models.TemplateStatusPending
	}
	return TemplateState{Name: def.Name, Status: status, MetaID: created.ID}, nil
}

// StateOf returns the status of name in states ("" when absent).
func StateOf(states []TemplateState, name string) string {
	for _, s := range states {
		if s.Name == name {
			return s.Status
		}
	}
	return whatsapp_models.TemplateStatusNone
}

// SaveTemplateStates upserts the tenant's template rows (db bound to the
// tenant) with states, clearing their rejection reason.
func SaveTemplateStates(db *gorm.DB, tenantID uint, states []TemplateState) error {
	for _, s := range states {
		row := whatsapp_models.WhatsAppTemplate{TenantID: tenantID, Name: s.Name, Status: s.Status, MetaID: s.MetaID}
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "name"}},
			DoUpdates: clause.Assignments(map[string]any{"status": s.Status, "reason": "", "meta_id": s.MetaID, "updated_at": time.Now(), "deleted_at": nil}),
		}).Create(&row).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// SetTemplateStatus stores one review decision (webhook) on the tenant's row
// of name, creating it.
func SetTemplateStatus(db *gorm.DB, tenantID uint, name, status, reason string) error {
	row := whatsapp_models.WhatsAppTemplate{TenantID: tenantID, Name: name, Status: status, Reason: reason}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "name"}},
		DoUpdates: clause.Assignments(map[string]any{"status": status, "reason": reason, "updated_at": time.Now(), "deleted_at": nil}),
	}).Create(&row).Error
}

// TemplateRows returns the tenant's template rows by name (db bound to the tenant).
func TemplateRows(db *gorm.DB) (map[string]whatsapp_models.WhatsAppTemplate, error) {
	var rows []whatsapp_models.WhatsAppTemplate
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]whatsapp_models.WhatsAppTemplate, len(rows))
	for _, r := range rows {
		out[r.Name] = r
	}
	return out, nil
}

// ReminderData fills the reminder template's variables.
type ReminderData struct {
	PatientName string
	ClinicName  string
	Start       time.Time // appointment start, in the clinic's time zone
}

// TemplateData fills any catalog template; Start is only used by templates
// with date / time variables.
type TemplateData = ReminderData

// ReminderMessage builds the template message for message msgID; the buttons
// carry "confirm:<id>" / "cancel:<id>" so the webhook knows what was answered.
func ReminderMessage(to string, msgID uint, data ReminderData) whatsapp.TemplateMessage {
	return whatsapp.TemplateMessage{
		To:         to,
		Name:       whatsapp_models.ReminderTemplate,
		Language:   ReminderLanguage,
		BodyParams: TemplateParams(reminderTemplate, data),
		Buttons: []whatsapp.QuickReplyButton{
			{Payload: fmt.Sprintf("%s:%d", whatsapp_models.ReplyConfirm, msgID)},
			{Payload: fmt.Sprintf("%s:%d", whatsapp_models.ReplyCancel, msgID)},
		},
	}
}

// TemplateMessageFor builds the message of a catalog template without
// buttons (the inbox templates have none).
func TemplateMessageFor(spec TemplateSpec, to string, data TemplateData) whatsapp.TemplateMessage {
	return whatsapp.TemplateMessage{
		To:         to,
		Name:       spec.Definition.Name,
		Language:   spec.Definition.Language,
		BodyParams: TemplateParams(spec, data),
	}
}

// TemplateParams are spec's body parameters filled from data.
func TemplateParams(spec TemplateSpec, data TemplateData) []string {
	params := make([]string, 0, len(spec.Variables))
	for _, v := range spec.Variables {
		switch v {
		case VarPatient:
			params = append(params, nonEmpty(data.PatientName, "paciente"))
		case VarClinic:
			params = append(params, nonEmpty(data.ClinicName, "la clínica"))
		case VarDate:
			params = append(params, SpanishDate(data.Start))
		case VarTime:
			params = append(params, data.Start.Format("15:04"))
		}
	}
	return params
}

// RenderTemplateBody is spec's text as the patient sees it, with params in
// place of its {{n}} placeholders; stored as the message Body for the thread.
func RenderTemplateBody(spec TemplateSpec, params []string) string {
	body := spec.Definition.Body
	for i, p := range params {
		body = strings.ReplaceAll(body, fmt.Sprintf("{{%d}}", i+1), p)
	}
	return body
}

// RenderReminderBody is the reminder's text as the patient sees it.
func RenderReminderBody(data ReminderData) string {
	return RenderTemplateBody(reminderTemplate, TemplateParams(reminderTemplate, data))
}

var spanishDays = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
var spanishMonths = [...]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

// SpanishDate formats t as "lunes 6 de octubre" for the Spanish template.
func SpanishDate(t time.Time) string {
	return fmt.Sprintf("%s %d de %s", spanishDays[t.Weekday()], t.Day(), spanishMonths[t.Month()-1])
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
