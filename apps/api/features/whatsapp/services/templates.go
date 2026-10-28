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
	VarCode    = "code"
	VarDoctor  = "doctor"
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
// They are UTILITY: neutral wording about the patient's care, no promotions,
// inviting a reply (which opens the 24 h window for free-text replies).
// Except continue-conversation: it names no appointment or result, so Meta
// recategorizes it as MARKETING; declaring it so keeps every WABA consistent.
// Templates already on a WABA are left as they are (EnsureTemplates only
// creates missing ones).
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
			Category: "MARKETING",
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
	PortalCodeTemplate,
	BookingConfirmedTemplate,
	BookingNotConfirmedTemplate,
	NoShowTemplate,
	SlotOfferTemplate,
}

// BookingConfirmedTemplate tells the patient that the appointment they asked
// for in the patient portal is confirmed. Not an inbox template.
var BookingConfirmedTemplate = TemplateSpec{
	Definition: whatsapp.TemplateDefinition{
		Name:     whatsapp_models.TemplateBookingConfirmed,
		Language: ReminderLanguage,
		Category: "UTILITY",
		Body:     "Hola {{1}}, tu cita en {{2}} con {{3}} está confirmada para el {{4}} a las {{5}}. Si no puedes asistir, responde a este mensaje.",
		Example:  []string{"María", "Clínica Pengi", "Dra. Ana Ruiz", "lunes 6 de octubre", "10:30"},
	},
	Variables:        []string{VarPatient, VarClinic, VarDoctor, VarDate, VarTime},
	NeedsAppointment: true,
}

// BookingNotConfirmedTemplate tells the patient that the clinic could not
// confirm the appointment they asked for (rejected or expired) and that they
// can pick another time in the portal. Not an inbox template.
var BookingNotConfirmedTemplate = TemplateSpec{
	Definition: whatsapp.TemplateDefinition{
		Name:     whatsapp_models.TemplateBookingNotConfirmed,
		Language: ReminderLanguage,
		Category: "UTILITY",
		Body:     "Hola {{1}}, {{2}} no pudo confirmar tu solicitud de cita del {{3}} a las {{4}}. Puedes elegir otro horario en el portal de pacientes.",
		Example:  []string{"María", "Clínica Pengi", "lunes 6 de octubre", "10:30"},
	},
	Variables:        []string{VarPatient, VarClinic, VarDate, VarTime},
	NeedsAppointment: true,
}

// NoShowTemplate invites a patient marked as no-show to book a new date. It
// says neither "today" nor that the patient missed it: it may go out the
// next morning (send hours). The button reply depends on the appointment, so
// it is not an inbox template.
var NoShowTemplate = TemplateSpec{
	Definition: whatsapp.TemplateDefinition{
		Name:     whatsapp_models.TemplateNoShow,
		Language: ReminderLanguage,
		Category: "UTILITY",
		Body:     "Hola {{1}}, te esperábamos en {{2}} el {{3}} a las {{4}} para tu cita con {{5}}. Si deseas agendar una nueva fecha, toca \"Reagendar\" o responde a este mensaje.",
		Example:  []string{"María Pérez", "Clínica Pengi", "lunes 6 de octubre", "10:30", "Dra. Ana Torres"},
		Buttons:  []whatsapp.TemplateButton{{Text: "Reagendar"}},
	},
	Variables:        []string{VarPatient, VarClinic, VarDate, VarTime, VarDoctor},
	NeedsAppointment: true,
}

// SlotOfferTemplate offers a freed slot to a patient on the waitlist. It
// promises nothing: the reply leads to the normal booking flow. Not an inbox
// template (date, time and doctor are the freed slot's, not an appointment's).
var SlotOfferTemplate = TemplateSpec{
	Definition: whatsapp.TemplateDefinition{
		Name:     whatsapp_models.TemplateSlotOffer,
		Language: ReminderLanguage,
		Category: "UTILITY",
		Body:     "Hola {{1}}, se liberó un turno en {{2}} el {{3}} a las {{4}} con {{5}}. Como estás en nuestra lista de espera, te lo ofrecemos: si te interesa, toca \"Lo quiero\" o responde a este mensaje.",
		Example:  []string{"María Pérez", "Clínica Pengi", "lunes 6 de octubre", "10:30", "Dra. Ana Torres"},
		Buttons:  []whatsapp.TemplateButton{{Text: "Lo quiero"}},
	},
	Variables: []string{VarPatient, VarClinic, VarDate, VarTime, VarDoctor},
}

// PortalCodeExpirationMinutes is how long a patient portal code lasts; the
// template's footer states it.
const PortalCodeExpirationMinutes = 10

// PortalCodeTemplate sends the patient portal's one-time sign-in code. Meta
// writes the body of AUTHENTICATION templates itself; Body is only the
// preview shown in settings. Not an inbox template.
var PortalCodeTemplate = TemplateSpec{
	Definition: whatsapp.TemplateDefinition{
		Name:                  whatsapp_models.TemplatePortalCode,
		Language:              ReminderLanguage,
		Category:              whatsapp.CategoryAuthentication,
		Body:                  "{{1}} es tu código de verificación. Por tu seguridad, no lo compartas.",
		Example:               []string{"123456"},
		CodeExpirationMinutes: PortalCodeExpirationMinutes,
		CopyCodeText:          "Copiar código",
	},
	Variables: []string{VarCode},
}

// PortalCodeMessage builds the sign-in code message to "to".
func PortalCodeMessage(to, code string) whatsapp.TemplateMessage {
	return whatsapp.TemplateMessage{
		To:         to,
		Name:       PortalCodeTemplate.Definition.Name,
		Language:   PortalCodeTemplate.Definition.Language,
		BodyParams: []string{code},
		CopyCode:   code,
	}
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
	Code        string    // one-time code (VarCode)
	DoctorName  string    // VarDoctor
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
		case VarCode:
			params = append(params, data.Code)
		case VarDoctor:
			params = append(params, nonEmpty(data.DoctorName, "tu médico"))
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
