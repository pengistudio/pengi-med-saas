package whatsapp_models

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// WhatsAppAccountEvent is one notice from Meta about an account: an
// account_update, account_review_update or account_alerts webhook change. They
// are stored because the container logs, the only other trace of why Meta
// banned a number, are lost on each deploy.
//
// It has NO TenantID on purpose: a ban can arrive before the account row
// exists (or after it is gone), so the WABA may not belong to any tenant. It
// holds no patient data. Written and read through tenantdb.System only, with no
// endpoint: see docs/backend/whatsapp-integration.md for the queries.
type WhatsAppAccountEvent struct {
	gorm.Model
	WabaID        string `gorm:"index"`
	PhoneNumberID string
	PhoneNumber   string
	// Field is the webhook field; Event its "event" (only account_update has
	// one); Detail the reason Meta gave, as "key=value" pairs.
	Field  string
	Event  string
	Detail string `gorm:"type:text"`
	// Payload is the change's value as Meta sent it (text, not jsonb, which
	// would reorder it; cast with payload::jsonb to query it).
	Payload string `gorm:"type:text"`
	// OccurredAt is the entry's time, when Meta sent one.
	OccurredAt *time.Time
	// Fingerprint makes a webhook Meta retries store one row. Nil without an
	// OccurredAt: the same notice twice may then be two events.
	Fingerprint *string `gorm:"size:64;uniqueIndex"`
}

func (WhatsAppAccountEvent) TableName() string { return "whatsapp_account_events" }

// SetFingerprint fills Fingerprint from the rest of the event.
func (e *WhatsAppAccountEvent) SetFingerprint() {
	e.Fingerprint = nil
	if e.OccurredAt == nil {
		return
	}
	sum := sha256.New()
	for _, part := range []string{e.WabaID, e.Field, e.Event, strconv.FormatInt(e.OccurredAt.Unix(), 10), e.Payload} {
		sum.Write([]byte(strconv.Itoa(len(part)) + ":" + part))
	}
	fp := hex.EncodeToString(sum.Sum(nil))
	e.Fingerprint = &fp
}
