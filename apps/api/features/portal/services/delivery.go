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

// Translator resolves an i18n key in the patient's language.
type Translator func(key string) string

// errNoChannel: the code could not be delivered by any channel.
var errNoChannel = errors.New("portal code not delivered")

// Delivery issues sign-in codes and sends them: by WhatsApp when the clinic
// has a connected number with the portal_code template approved, the patient
// has a valid phone and the plan's monthly WhatsApp cap has room; otherwise
// (or when the WhatsApp send fails) by email when the patient has one;
// otherwise nothing is issued.
type Delivery struct {
	db     *gorm.DB
	logger *zap.Logger
	mailer CodeMailer
	client *whatsapp.Client
	now    func() time.Time
}

func NewDelivery(db *gorm.DB, logger *zap.Logger, m CodeMailer, client *whatsapp.Client) *Delivery {
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
	var account whatsapp_models.WhatsAppAccount
	if d.client == nil {
		return account, "", false
	}
	scoped := tenantdb.ForTenant(d.db, tenantID)
	if err := scoped.Limit(1).Find(&account).Error; err != nil || account.ID == 0 || account.Status != whatsapp_models.AccountStatusConnected {
		return account, "", false
	}
	rows, err := whatsapp_services.TemplateRows(scoped)
	if err != nil || rows[whatsapp_models.TemplatePortalCode].Status != whatsapp_models.TemplateStatusApproved {
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
		if err := d.sendWhatsApp(account, token, patient, waTo, code); err == nil {
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
	if err := d.mailer.SendPortalCode(email, codeEmail(clinic, code, tr)); err != nil {
		return "", err
	}
	return portal_models.ChannelEmail, nil
}

// sendWhatsApp records the message (kind portal_code, counted toward the
// monthly cap, outside any conversation, its body without the code) and sends
// the template.
func (d *Delivery) sendWhatsApp(account whatsapp_models.WhatsAppAccount, token string, patient clinical_models.Patient, to, code string) error {
	scoped := tenantdb.ForTenant(d.db, account.TenantID)
	spec := whatsapp_services.PortalCodeTemplate
	patientID := patient.ID
	msg := whatsapp_models.WhatsAppMessage{
		TenantID:    account.TenantID,
		Kind:        whatsapp_models.KindPortalCode,
		PatientID:   &patientID,
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

// codeEmail is the email's text in the request's language.
func codeEmail(clinic Clinic, code string, tr Translator) mailer.PortalCodeEmail {
	name := clinic.Tenant.DisplayName()
	fill := func(key string) string {
		s := tr(key)
		s = strings.ReplaceAll(s, "{clinic}", name)
		return strings.ReplaceAll(s, "{minutes}", strconv.Itoa(int(CodeTTL.Minutes())))
	}
	return mailer.PortalCodeEmail{
		Subject: fill("portal.email.code.subject"),
		Heading: fill("portal.email.code.heading"),
		Intro:   fill("portal.email.code.intro"),
		Code:    code,
		Expiry:  fill("portal.email.code.expiry"),
		Ignore:  fill("portal.email.code.ignore"),
	}
}

// EnsurePortalTemplate creates the portal_code template on the clinic's WABA
// when the clinic has a connected WhatsApp account that does not have it yet
// (accounts connected before the template joined the catalog). Meta reviews
// it; until it is approved codes go by email. Best effort: failures are logged.
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
	if _, ok := rows[whatsapp_models.TemplatePortalCode]; ok {
		return
	}
	token, err := account.OpenAccessToken()
	if err != nil {
		return
	}
	states, err := whatsapp_services.EnsureTemplates(d.client, token, account.WabaID)
	if err != nil {
		d.logger.Warn("portal: failed to create the portal_code template", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
	if err := whatsapp_services.SaveTemplateStates(scoped, tenantID, states); err != nil {
		d.logger.Error("portal: failed to save whatsapp template states", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}
