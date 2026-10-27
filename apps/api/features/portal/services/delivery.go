package portal_services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_models "pengi-med-saas/features/portal/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CodeMailer sends the sign-in code email (*mailer.Mailer in production).
type CodeMailer interface {
	SendPortalCode(toEmail string, e mailer.PortalCodeEmail) error
}

// Mailer sends every portal email: sign-in codes and booking notices
// (*mailer.Mailer in production).
type Mailer interface {
	CodeMailer
	SendPortalNotice(toEmail string, e mailer.PortalNoticeEmail) error
}

// Translator resolves an i18n key in the patient's language.
type Translator func(key string) string

// errNoChannel / ErrNoChannel: the code could not be delivered by any channel.
var (
	ErrNoChannel = errors.New("portal code not delivered")
	errNoChannel = ErrNoChannel
)

// Delivery issues sign-in codes and sends them: by WhatsApp when the clinic
// has a connected number with the portal_code template approved, the patient
// has a valid phone and the plan's monthly WhatsApp cap has room; otherwise
// (or when the WhatsApp send fails) by email when the patient has one;
// otherwise nothing is issued.
type Delivery struct {
	db     *gorm.DB
	logger *zap.Logger
	mailer Mailer
	client *whatsapp.Client
	now    func() time.Time
}

func NewDelivery(db *gorm.DB, logger *zap.Logger, m Mailer, client *whatsapp.Client) *Delivery {
	return &Delivery{db: db, logger: logger, mailer: m, client: client, now: time.Now}
}

// WhatsAppReady reports whether the clinic can send codes by WhatsApp (a
// connected account and the portal_code template approved); it ignores the
// monthly cap, which only decides each send.
func (d *Delivery) WhatsAppReady(tenantID uint) bool {
	_, _, ok := d.whatsAppAccount(tenantID)
	return ok
}

func (d *Delivery) whatsAppAccount(tenantID uint) (whatsapp_models.WhatsAppAccount, string, bool) {
	return d.accountWithTemplate(tenantID, whatsapp_models.TemplatePortalCode)
}

// accountWithTemplate returns the clinic's connected WhatsApp account and its
// token when template is approved on it.
func (d *Delivery) accountWithTemplate(tenantID uint, template string) (whatsapp_models.WhatsAppAccount, string, bool) {
	var account whatsapp_models.WhatsAppAccount
	if d.client == nil {
		return account, "", false
	}
	scoped := tenantdb.ForTenant(d.db, tenantID)
	if err := scoped.Limit(1).Find(&account).Error; err != nil || account.ID == 0 || account.Status != whatsapp_models.AccountStatusConnected {
		return account, "", false
	}
	rows, err := whatsapp_services.TemplateRows(scoped)
	if err != nil || rows[template].Status != whatsapp_models.TemplateStatusApproved {
		return account, "", false
	}
	token, err := account.OpenAccessToken()
	if err != nil {
		return account, "", false
	}
	return account, token, true
}

// SendLoginCode issues a code for patient and delivers it; it returns the
// channel used ("" when nothing was sent). ErrTooManyCodes when the patient
// asked for too many codes.
func (d *Delivery) SendLoginCode(clinic Clinic, patient clinical_models.Patient, tr Translator) (string, error) {
	tenantID := clinic.Tenant.ID
	now := d.now()

	account, token, waOK := d.whatsAppAccount(tenantID)
	waTo := ""
	if waOK {
		if to, err := whatsapp_services.NormalizePhone(patient.Phone); err == nil {
			waTo = to
		}
	}
	if waTo != "" {
		allowed, _, err := whatsapp_services.CanSendTemplate(d.db, tenantID, now)
		if err != nil {
			d.logger.Error("portal: failed to check whatsapp monthly limit", zap.Uint("tenant_id", tenantID), zap.Error(err))
		}
		if err != nil || !allowed {
			waTo = "" // over the plan's cap: email instead
		}
	}
	email := strings.TrimSpace(patient.Email)
	if waTo == "" && email == "" {
		return "", nil
	}

	channel := portal_models.ChannelEmail
	if waTo != "" {
		channel = portal_models.ChannelWhatsApp
	}
	code, row, err := IssueCode(d.db, tenantID, patient.ID, channel, now)
	if err != nil {
		return "", err
	}
	if !config.Get().IsRelease() {
		// Dev only (no Resend key / no WhatsApp locally): lets a developer sign in.
		d.logger.Info("portal: dev sign-in code", zap.Uint("tenant_id", tenantID), zap.Uint("patient_id", patient.ID), zap.String("code", code))
	}

	if waTo != "" {
		patientID := patient.ID
		if err := d.sendWhatsApp(account, token, &patientID, waTo, code); err == nil {
			return portal_models.ChannelWhatsApp, nil
		} else {
			d.logger.Warn("portal: whatsapp code not sent, falling back to email", zap.Uint("tenant_id", tenantID), zap.Error(err))
		}
		if email == "" {
			return "", errNoChannel
		}
		if err := tenantdb.ForTenant(d.db, tenantID).Model(&row).Update("channel", portal_models.ChannelEmail).Error; err != nil {
			d.logger.Error("portal: failed to record the code's channel", zap.Error(err))
		}
	}
	if err := d.mailer.SendPortalCode(email, codeEmail(clinic, code, tr, codeEmailSignIn)); err != nil {
		return "", err
	}
	return portal_models.ChannelEmail, nil
}

// SendContactCode issues a code for a new patient's normalized contact on
// channel and delivers it there (no fallback: the code must reach the
// contact being verified). By WhatsApp the clinic needs the portal_code
// template approved and room under the plan's monthly cap; otherwise nothing
// is sent (ErrNoChannel). A *LimitError when a budget is spent.
func (d *Delivery) SendContactCode(clinic Clinic, channel, contact string, tr Translator) error {
	tenantID := clinic.Tenant.ID
	now := d.now()
	var account whatsapp_models.WhatsAppAccount
	var token string
	switch channel {
	case portal_models.ChannelWhatsApp:
		var ok bool
		if account, token, ok = d.whatsAppAccount(tenantID); !ok {
			return errNoChannel
		}
		allowed, _, err := whatsapp_services.CanSendTemplate(d.db, tenantID, now)
		if err != nil {
			return err
		}
		if !allowed {
			return errNoChannel
		}
	case portal_models.ChannelEmail:
		if d.mailer == nil {
			return errNoChannel
		}
	default:
		return ErrInvalidContact
	}
	code, _, err := IssueContactCode(d.db, tenantID, channel, contact, now)
	if err != nil {
		return err
	}
	if !config.Get().IsRelease() {
		// Dev only (no Resend key / no WhatsApp locally): lets a developer verify.
		d.logger.Info("portal: dev contact code", zap.Uint("tenant_id", tenantID), zap.String("channel", channel), zap.String("code", code))
	}
	if channel == portal_models.ChannelWhatsApp {
		return d.sendWhatsApp(account, token, nil, contact, code)
	}
	return d.mailer.SendPortalCode(contact, codeEmail(clinic, code, tr, codeEmailNewPatient))
}

// sendWhatsApp records the message (kind portal_code, counted toward the
// monthly cap, outside any conversation, its body without the code) and sends
// the template. patientID is nil for a new patient's contact code.
func (d *Delivery) sendWhatsApp(account whatsapp_models.WhatsAppAccount, token string, patientID *uint, to, code string) error {
	scoped := tenantdb.ForTenant(d.db, account.TenantID)
	spec := whatsapp_services.PortalCodeTemplate
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    account.TenantID,
		Kind:        whatsapp_models.KindPortalCode,
		PatientID:   patientID,
		ToPhone:     to,
		Template:    spec.Definition.Name,
		Status:      whatsapp_models.MessageStatusSending,
		Body:        whatsapp_services.RenderTemplateBody(spec, []string{strings.Repeat("•", CodeLength)}),
		ContentType: whatsapp_models.ContentText,
	}
	if err := scoped.Create(&msg).Error; err != nil {
		return err
	}
	sender := whatsapp_services.NewSender(d.db, d.logger, d.client)
	return sender.DeliverTemplate(scoped, &msg, account, token, whatsapp_services.PortalCodeMessage(to, code))
}

// Kinds of code email (the i18n key prefix of their texts).
const (
	codeEmailSignIn     = "portal.email.code"
	codeEmailNewPatient = "portal.email.new_patient_code"
)

// codeEmail is the email's text in the request's language. kind picks the
// subject, heading and intro (codeEmail*); expiry and ignore are shared.
func codeEmail(clinic Clinic, code string, tr Translator, kind string) mailer.PortalCodeEmail {
	name := clinic.Tenant.DisplayName()
	fill := func(key string) string {
		s := tr(key)
		s = strings.ReplaceAll(s, "{clinic}", name)
		return strings.ReplaceAll(s, "{minutes}", strconv.Itoa(int(CodeTTL.Minutes())))
	}
	return mailer.PortalCodeEmail{
		Subject: fill(kind + ".subject"),
		Heading: fill(kind + ".heading"),
		Intro:   fill(kind + ".intro"),
		Code:    code,
		Expiry:  fill("portal.email.code.expiry"),
		Ignore:  fill("portal.email.code.ignore"),
	}
}

// portalTemplates are the catalog templates the portal sends.
var portalTemplates = []string{
	whatsapp_models.TemplatePortalCode, whatsapp_models.TemplateBookingConfirmed, whatsapp_models.TemplateBookingNotConfirmed,
}

// EnsurePortalTemplate creates the portal's templates (portal_code and the
// booking notices) on the clinic's WABA when the clinic has a connected
// WhatsApp account that misses one of them (accounts connected before they
// joined the catalog). Meta reviews them; until one is approved its messages
// go by email. Best effort: failures are logged.
func (d *Delivery) EnsurePortalTemplate(tenantID uint) {
	if d.client == nil {
		return
	}
	scoped := tenantdb.ForTenant(d.db, tenantID)
	var account whatsapp_models.WhatsAppAccount
	if err := scoped.Limit(1).Find(&account).Error; err != nil || account.ID == 0 || account.Status != whatsapp_models.AccountStatusConnected {
		return
	}
	rows, err := whatsapp_services.TemplateRows(scoped)
	if err != nil {
		d.logger.Error("portal: failed to load whatsapp templates", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	missing := false
	for _, name := range portalTemplates {
		if _, ok := rows[name]; !ok {
			missing = true
		}
	}
	if !missing {
		return
	}
	token, err := account.OpenAccessToken()
	if err != nil {
		return
	}
	states, err := whatsapp_services.EnsureTemplates(d.client, token, account.WabaID)
	if err != nil {
		d.logger.Warn("portal: failed to create the portal templates", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
	if err := whatsapp_services.SaveTemplateStates(scoped, tenantID, states); err != nil {
		d.logger.Error("portal: failed to save whatsapp template states", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}
