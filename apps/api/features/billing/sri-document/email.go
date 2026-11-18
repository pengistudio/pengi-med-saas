package sri_document

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/i18n/catalog"
	i18n_messages "pengi-med-saas/i18n/messages"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The issuer must email every authorized comprobante to its buyer (ficha
// técnica SRI v2.34 §4.7). The lifecycle does it once the SRI authorizes a
// document (emailOnAuthorized) and a user can send it again, to the buyer or
// another address (SendEmail). Only authorized documents are emailed, always
// with their RIDE and the authorized XML (never the signed-only XML).

// Mailer sends the email delivering an authorized comprobante
// (*mailer.Mailer in production).
type Mailer interface {
	SendSriDocument(toEmail string, e mailer.SriDocumentEmail) error
}

// Translator resolves an i18n key in the language of the buyer's email.
type Translator func(key string) string

// EmailContent is what a kind's email carries.
type EmailContent struct {
	BuyerEmail string // the invoice buyer's email (sri_services.InvoiceBuyer), trimmed; "" for Consumidor Final
	Number     string // 001-001-000000123
	Total      float64
	IssueDate  time.Time
	AccessKey  string
	Ride       func() ([]byte, error) // the RIDE PDF, rendered when not stored
}

// EmailCooldown is how long after a successful send a document cannot be
// emailed again, so SendEmail cannot be looped to spam an address.
const EmailCooldown = 60 * time.Second

// Email error codes are i18n keys persisted in a document's email_error.
const (
	EmailErrorSendFailed       = "billing.document.email.error.send_failed"
	EmailErrorInvalidRecipient = "billing.document.email.error.invalid_recipient"
	EmailErrorAttachments      = "billing.document.email.error.attachments"
)

var (
	// ErrNoRecipient: no address was given and the buyer has none (Consumidor
	// Final, or a patient without email).
	ErrNoRecipient = errors.New("no email recipient")
	// ErrInvalidRecipient: the address is not a single valid email address.
	ErrInvalidRecipient = errors.New("invalid email recipient")
	// ErrNotAuthorized: only documents the SRI authorized are emailed.
	ErrNotAuthorized = errors.New("sri document not authorized")
	// ErrEmailTooSoon: the document was emailed less than EmailCooldown ago.
	ErrEmailTooSoon = errors.New("sri document emailed too recently")
	// ErrEmailAttachments: the RIDE or the authorized XML is not available.
	ErrEmailAttachments = errors.New("sri document email attachments unavailable")
	// ErrEmailSendFailed: the email provider did not accept the email.
	ErrEmailSendFailed = errors.New("sri document email not sent")

	errAlreadyEmailed = errors.New("sri document already emailed")
)

// WithMailer enables emailing authorized documents. tr translates the email's
// texts; nil uses the message catalog's default language.
func (l *Lifecycle) WithMailer(m Mailer, tr Translator) *Lifecycle {
	l.mailer = m
	l.translate = tr
	return l
}

// SendEmail emails an authorized document of tenantID with its RIDE and
// authorized XML to "to", or to its buyer when "to" is blank, and returns its
// delivery state. The caller must have checked the document belongs to
// tenantID. Refused with ErrEmailTooSoon within EmailCooldown of the last
// successful send.
func (l *Lifecycle) SendEmail(kind Kind, tenantID, id uint, to string) (billing_models.EmailDelivery, error) {
	return l.deliver(kind, tenantID, id, to, true)
}

// emailOnAuthorized emails a just-authorized document to its buyer unless it
// was already emailed, so a re-run never sends it twice. Failures are only
// logged (and recorded in email_error by deliver).
func (l *Lifecycle) emailOnAuthorized(kind Kind, id uint, tenant tenant_models.Tenant) {
	if l.mailer == nil || kind.Email == nil {
		return
	}
	_, err := l.deliver(kind, tenant.ID, id, "", false)
	if err == nil || errors.Is(err, ErrNoRecipient) || errors.Is(err, errAlreadyEmailed) {
		return
	}
	l.logger.Warn("failed to email authorized SRI document", zap.String("kind", kind.Name), zap.Uint("id", id), zap.Error(err))
}

// emailState is the slice of a document delivery reads.
type emailState struct {
	Status     string
	AccessKey  *string
	EmailedAt  *time.Time
	EmailedTo  *string
	EmailError *string
}

func (l *Lifecycle) deliver(kind Kind, tenantID, id uint, to string, manual bool) (billing_models.EmailDelivery, error) {
	db := tenantdb.ForTenant(l.db, tenantID)
	var st emailState
	if err := db.Model(kind.Model).Where("id = ?", id).Take(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return billing_models.EmailDelivery{}, ErrNotFound
		}
		return billing_models.EmailDelivery{}, fmt.Errorf("load %s %d: %w", kind.Name, id, err)
	}
	prev := billing_models.EmailDelivery{EmailedAt: st.EmailedAt, EmailedTo: st.EmailedTo, EmailError: st.EmailError}
	if st.Status != billing_models.InvoiceStatusAuthorized || st.AccessKey == nil || *st.AccessKey == "" {
		return prev, ErrNotAuthorized
	}
	now := time.Now()
	if !manual && st.EmailedAt != nil {
		return prev, errAlreadyEmailed
	}
	if manual && st.EmailedAt != nil && now.Sub(*st.EmailedAt) < EmailCooldown {
		return prev, ErrEmailTooSoon
	}
	if kind.Email == nil {
		return prev, fmt.Errorf("%s documents are not emailed", kind.Name)
	}
	if l.mailer == nil {
		return prev, fmt.Errorf("%w: no mailer configured", ErrEmailSendFailed)
	}

	tenant, err := l.tenant(tenantID)
	if err != nil {
		return prev, err
	}
	content, err := kind.Email(db, l.docs, id, tenant)
	if err != nil {
		return prev, err
	}

	recipient := strings.TrimSpace(to)
	fromBuyer := recipient == ""
	if fromBuyer {
		recipient = content.BuyerEmail
	}
	if recipient == "" {
		return prev, ErrNoRecipient
	}
	if !validEmail(recipient) {
		if !fromBuyer {
			// A typo in the address the user entered: nothing to record.
			return prev, ErrInvalidRecipient
		}
		return l.recordEmailFailure(kind, db, id, prev, EmailErrorInvalidRecipient, ErrInvalidRecipient)
	}

	// Claimed atomically before sending, so two concurrent runs (a duplicate
	// task, a double click) never both send: the loser sees RowsAffected 0.
	claim := db.Model(kind.Model).Where("id = ?", id)
	if manual {
		claim = claim.Where("(emailed_at IS NULL OR emailed_at < ?)", now.Add(-EmailCooldown))
	} else {
		claim = claim.Where("emailed_at IS NULL")
	}
	res := claim.Updates(map[string]any{"emailed_at": now, "emailed_to": recipient})
	if res.Error != nil {
		return prev, fmt.Errorf("claim email of %s %d: %w", kind.Name, id, res.Error)
	}
	if res.RowsAffected != 1 {
		if manual {
			return prev, ErrEmailTooSoon
		}
		return prev, errAlreadyEmailed
	}

	log := l.logger.With(zap.String("kind", kind.Name), zap.Uint("id", id), zap.Uint("tenant_id", tenantID))
	pdf, err := content.Ride()
	if err != nil {
		log.Error("failed to get RIDE for email", zap.Error(err))
		return l.recordEmailFailure(kind, db, id, prev, EmailErrorAttachments, ErrEmailAttachments)
	}
	xml, err := l.AuthorizedXML(kind, tenantID, content.AccessKey)
	if err != nil {
		log.Error("failed to get authorized XML for email", zap.Error(err))
		return l.recordEmailFailure(kind, db, id, prev, EmailErrorAttachments, ErrEmailAttachments)
	}
	if err := l.mailer.SendSriDocument(recipient, l.composeEmail(kind, tenant, content, pdf, xml)); err != nil {
		// The provider's error, never the address.
		log.Error("failed to send SRI document email", zap.Error(err))
		return l.recordEmailFailure(kind, db, id, prev, EmailErrorSendFailed, ErrEmailSendFailed)
	}

	sent := billing_models.EmailDelivery{EmailedAt: &now, EmailedTo: &recipient}
	if err := db.Model(kind.Model).Where("id = ?", id).Update("email_error", nil).Error; err != nil {
		log.Error("failed to clear email error", zap.Error(err))
		sent.EmailError = prev.EmailError
	}
	return sent, nil
}

// recordEmailFailure restores the last successful send (undoing the claim)
// and records code in email_error. It returns the persisted state and errSent.
func (l *Lifecycle) recordEmailFailure(kind Kind, db *gorm.DB, id uint, prev billing_models.EmailDelivery, code string, errSent error) (billing_models.EmailDelivery, error) {
	fields := map[string]any{"emailed_at": nil, "emailed_to": nil, "email_error": code}
	if prev.EmailedAt != nil {
		fields["emailed_at"] = *prev.EmailedAt
	}
	if prev.EmailedTo != nil {
		fields["emailed_to"] = *prev.EmailedTo
	}
	if err := db.Model(kind.Model).Where("id = ?", id).Updates(fields).Error; err != nil {
		l.logger.Error("failed to record SRI document email failure", zap.String("kind", kind.Name), zap.Uint("id", id), zap.Error(err))
		return prev, errSent
	}
	prev.EmailError = &code
	return prev, errSent
}

// composeEmail is the buyer's email in the translator's language.
func (l *Lifecycle) composeEmail(kind Kind, tenant tenant_models.Tenant, content EmailContent, pdf, xml []byte) mailer.SriDocumentEmail {
	text := l.translate
	if text == nil {
		text = catalogText
	}
	// Fields collapses newlines and runs of spaces: the name goes into the Subject.
	issuer := strings.Join(strings.Fields(tenant.CorporateName), " ")
	if issuer == "" {
		issuer = tenant.DisplayName()
	}
	fill := strings.NewReplacer(
		"{document}", text(kind.EmailTitle),
		"{number}", content.Number,
		"{issuer}", issuer,
		"{total}", fmt.Sprintf("$%.2f", content.Total),
		"{date}", content.IssueDate.Format("02/01/2006"),
		"{access_key}", content.AccessKey,
	).Replace
	return mailer.SriDocumentEmail{
		Subject: fill(text("billing.document.email.subject")),
		Heading: fill(text("billing.document.email.heading")),
		Lines: []string{
			fill(text("billing.document.email.intro")),
			fill(text("billing.document.email.summary")),
			fill(text("billing.document.email.attachments")),
		},
		Footer: fill(text("billing.document.email.footer")),
		Attachments: []mailer.Attachment{
			{Filename: content.AccessKey + ".pdf", Content: pdf},
			{Filename: content.AccessKey + ".xml", Content: xml},
		},
	}
}

// validEmail reports whether s is exactly one bare email address (no display
// name, no list).
func validEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && strings.Contains(s, ".")
}

// documentNumber is the printed number of a comprobante: 001-001-000000123.
func documentNumber(establishment, emissionPoint, sequential string) string {
	return establishment + "-" + emissionPoint + "-" + sequential
}

var (
	emailCatalogOnce sync.Once
	emailCatalog     *catalog.Catalog
)

// catalogText is the text of key in the catalog's default language (Spanish,
// the language patients get their texts in): a document is emailed outside
// any request, and the buyer's language is not known.
func catalogText(key string) string {
	emailCatalogOnce.Do(func() {
		if c, err := catalog.Load(i18n_messages.FS); err == nil {
			emailCatalog = c
		}
	})
	if emailCatalog == nil {
		return key
	}
	return emailCatalog.Translate(catalog.DefaultLanguage, key)
}
