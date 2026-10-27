package portal_services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/tenantdb"
	portal_models "pengi-med-saas/features/portal/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"gorm.io/gorm"
)

// ErrInvalidContact: a channel other than email/whatsapp, a malformed email
// or a phone that is not a WhatsApp number.
var ErrInvalidContact = errors.New("invalid portal contact")

// maxEmailLength bounds a contact email (the request's column is 255).
const maxEmailLength = 254

// NormalizeEmail lowercases and trims an email; ok is false unless it is a
// plain, syntactically valid address (no display name).
func NormalizeEmail(raw string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > maxEmailLength {
		return "", false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || addr.Name != "" {
		return "", false
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || !strings.Contains(email[at+1:], ".") {
		return "", false
	}
	return email, true
}

// NormalizeContact is the canonical form of a contact on channel: the
// lowercased email, or the phone in E.164 without "+" (Ecuador by default).
func NormalizeContact(channel, raw string) (string, error) {
	switch channel {
	case portal_models.ChannelEmail:
		if email, ok := NormalizeEmail(raw); ok {
			return email, nil
		}
	case portal_models.ChannelWhatsApp:
		if phone, err := whatsapp_services.NormalizePhone(raw); err == nil {
			return phone, nil
		}
	}
	return "", ErrInvalidContact
}

// ContactHash is the stored form of a normalized contact: an HMAC keyed with
// AUTH_KEY over tenant, channel and contact, so the table never holds the
// contact itself.
func ContactHash(tenantID uint, channel, contact string) string {
	mac := hmac.New(sha256.New, []byte("portal-contact:"+config.Get().AuthKey))
	fmt.Fprintf(mac, "%d:%s:%s", tenantID, channel, contact)
	return hex.EncodeToString(mac.Sum(nil))
}

// hashContactCode is the stored form of a contact code (bound to the contact).
func hashContactCode(tenantID uint, contactHash, code string) string {
	mac := hmac.New(sha256.New, []byte("portal-contact-code:"+config.Get().AuthKey))
	fmt.Fprintf(mac, "%d:%s:%s", tenantID, contactHash, code)
	return hex.EncodeToString(mac.Sum(nil))
}

// IssueContactCode creates a new code for the normalized contact on channel
// and expires the contact's previous unused codes. A *LimitError (errors.Is
// ErrTooManyCodes) when a budget is spent: the global or clinic one (shared
// with sign-in codes) first, then the contact's (MaxCodesPerWindow per
// CodeWindow, MaxCodesPerContactPerDay a day).
func IssueContactCode(db *gorm.DB, tenantID uint, channel, contact string, now time.Time) (string, portal_models.PortalContactCode, error) {
	issueMu.Lock()
	defer issueMu.Unlock()
	hash := ContactHash(tenantID, channel, contact)
	if budget, err := spentContactBudget(db, tenantID, hash, now); err != nil {
		return "", portal_models.PortalContactCode{}, err
	} else if budget != "" {
		return "", portal_models.PortalContactCode{}, &LimitError{Budget: budget}
	}
	code, err := GenerateCode()
	if err != nil {
		return "", portal_models.PortalContactCode{}, err
	}
	row := portal_models.PortalContactCode{
		// UTC like every bound here: SQLite (CI) compares timestamps as text.
		Model:       gorm.Model{CreatedAt: now.UTC()},
		TenantID:    tenantID,
		Channel:     channel,
		ContactHash: hash,
		CodeHash:    hashContactCode(tenantID, hash, code),
		ExpiresAt:   now.Add(CodeTTL).UTC(),
	}
	err = tenantdb.ForTenant(db, tenantID).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&portal_models.PortalContactCode{}).
			Where("contact_hash = ? AND used_at IS NULL AND expires_at > ?", hash, now.UTC()).
			Update("expires_at", now.UTC()).Error; err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return "", portal_models.PortalContactCode{}, err
	}
	return code, row, nil
}

// spentContactBudget is the first budget the contact's next code would
// exceed, "" if none.
func spentContactBudget(db *gorm.DB, tenantID uint, contactHash string, now time.Time) (string, error) {
	if budget, err := clinicBudget(db, tenantID, now); err != nil || budget != "" {
		return budget, err
	}
	counts, err := personCodeCounts(db, &portal_models.PortalContactCode{}, tenantID, "contact_hash", contactHash, now)
	if err != nil {
		return "", err
	}
	switch {
	case counts.PersonDay >= MaxCodesPerContactPerDay:
		return BudgetContactDay, nil
	case counts.PersonWindow >= MaxCodesPerWindow:
		return BudgetContactWindow, nil
	}
	return "", nil
}

// VerifyContactCode checks code against the contact's latest code and
// consumes it. Same rules as VerifyCode; every failure is ErrInvalidCode.
func VerifyContactCode(db *gorm.DB, tenantID uint, channel, contact, code string, now time.Time) error {
	code = strings.TrimSpace(code)
	if len(code) != CodeLength {
		return ErrInvalidCode
	}
	scoped := tenantdb.ForTenant(db, tenantID)
	hash := ContactHash(tenantID, channel, contact)
	var row portal_models.PortalContactCode
	if err := scoped.Where("contact_hash = ?", hash).Order("id DESC").Limit(1).Find(&row).Error; err != nil {
		return err
	}
	return consumeCode(scoped, &portal_models.PortalContactCode{}, codeRow{
		ID: row.ID, UsedAt: row.UsedAt, ExpiresAt: row.ExpiresAt, Attempts: row.Attempts, CodeHash: row.CodeHash,
	}, hashContactCode(tenantID, hash, code), now)
}
