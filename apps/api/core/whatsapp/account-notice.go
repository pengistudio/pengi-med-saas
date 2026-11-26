package whatsapp

import (
	"bytes"
	"encoding/json"
	"strings"
)

// Webhook fields that carry Meta's notices about an account (violations, bans,
// restrictions, reviews, alerts). They hold no patient data.
const (
	FieldAccountUpdate       = "account_update"
	FieldAccountReviewUpdate = "account_review_update"
	FieldAccountAlerts       = "account_alerts"
)

// AccountNotice is what we read of an account_update, account_review_update or
// account_alerts change, all of it optional. The change's Raw has the rest.
type AccountNotice struct {
	Event         string // account_update: ACCOUNT_VIOLATION, DISABLED_UPDATE, ...
	WabaID        string // waba_info.waba_id (the entry id may be a partner's)
	PhoneNumberID string
	PhoneNumber   string

	ViolationInfo     ViolationInfo
	BanInfo           BanInfo
	RestrictionInfo   []RestrictionInfo
	DisconnectionInfo DisconnectionInfo

	// account_review_update: APPROVED, REJECTED, PENDING, DEFERRED.
	Decision string

	// account_alerts
	EntityType string
	EntityID   string
	AlertInfo  AlertInfo
}

// ViolationInfo comes with ACCOUNT_VIOLATION.
type ViolationInfo struct{ ViolationType string }

// BanInfo comes with DISABLED_UPDATE. State is DISABLE, REINSTATE or
// SCHEDULE_FOR_DISABLE.
type BanInfo struct{ WabaBanState, WabaBanDate string }

// RestrictionInfo comes with ACCOUNT_RESTRICTION. Expiration is a unix time.
type RestrictionInfo struct{ RestrictionType, Expiration, Remediation string }

// DisconnectionInfo comes with PARTNER_REMOVED on a Coexistence number.
type DisconnectionInfo struct{ Reason, InitiatedBy string }

// AlertInfo is the alert of an account_alerts change.
type AlertInfo struct{ AlertSeverity, AlertStatus, AlertType, AlertDescription string }

// AccountNotice reads the notice out of the change's raw value. It never
// fails: a field of an unexpected type is read as text or left empty.
func (c WebhookChange) AccountNotice() AccountNotice {
	v := looseObject(c.Raw)
	n := AccountNotice{
		Event:         looseText(v["event"]),
		WabaID:        looseText(looseObject(v["waba_info"])["waba_id"]),
		PhoneNumberID: looseText(v["phone_number_id"]),
		PhoneNumber:   looseText(v["phone_number"]),
		Decision:      looseText(v["decision"]),
		EntityType:    looseText(v["entity_type"]),
		EntityID:      looseText(v["entity_id"]),
	}
	if n.PhoneNumberID == "" {
		n.PhoneNumberID = looseText(looseObject(v["metadata"])["phone_number_id"])
	}
	if n.PhoneNumberID == "" && strings.Contains(strings.ToUpper(n.EntityType), "PHONE") {
		n.PhoneNumberID = n.EntityID
	}
	n.ViolationInfo.ViolationType = looseText(looseObject(v["violation_info"])["violation_type"])
	ban := looseObject(v["ban_info"])
	n.BanInfo = BanInfo{WabaBanState: looseText(ban["waba_ban_state"]), WabaBanDate: looseText(ban["waba_ban_date"])}
	for _, r := range looseList(v["restriction_info"]) {
		n.RestrictionInfo = append(n.RestrictionInfo, RestrictionInfo{
			RestrictionType: looseText(r["restriction_type"]),
			Expiration:      looseText(r["expiration"]),
			Remediation:     looseText(r["remediation"]),
		})
	}
	off := looseObject(v["disconnection_info"])
	n.DisconnectionInfo = DisconnectionInfo{Reason: looseText(off["reason"]), InitiatedBy: looseText(off["initiated_by"])}
	if alerts := looseList(v["alert_info"]); len(alerts) > 0 {
		a := alerts[0]
		n.AlertInfo = AlertInfo{
			AlertSeverity:    looseText(a["alert_severity"]),
			AlertStatus:      looseText(a["alert_status"]),
			AlertType:        looseText(a["alert_type"]),
			AlertDescription: looseText(a["alert_description"]),
		}
	}
	return n
}

// maxDetail caps Detail, in runes; the raw payload is kept whole.
const maxDetail = 500

// Detail sums up the reason Meta gave as "key=value" pairs ("" when none).
func (n AccountNotice) Detail() string {
	var parts []string
	add := func(key, value string) {
		if value != "" {
			parts = append(parts, key+"="+value)
		}
	}
	add("violation_type", n.ViolationInfo.ViolationType)
	add("waba_ban_state", n.BanInfo.WabaBanState)
	add("waba_ban_date", n.BanInfo.WabaBanDate)
	for _, r := range n.RestrictionInfo {
		add("restriction_type", r.RestrictionType)
		add("expiration", r.Expiration)
		add("remediation", r.Remediation)
	}
	add("reason", n.DisconnectionInfo.Reason)
	add("initiated_by", n.DisconnectionInfo.InitiatedBy)
	add("decision", n.Decision)
	add("alert_type", n.AlertInfo.AlertType)
	add("alert_severity", n.AlertInfo.AlertSeverity)
	add("alert_status", n.AlertInfo.AlertStatus)
	add("entity_type", n.EntityType)
	add("entity_id", n.EntityID)
	add("alert_description", n.AlertInfo.AlertDescription)
	detail := strings.Join(parts, " ")
	if runes := []rune(detail); len(runes) > maxDetail {
		detail = string(runes[:maxDetail])
	}
	return detail
}

// looseObject decodes raw as an object, or returns nil if it isn't one.
func looseObject(raw json.RawMessage) map[string]json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}
	return obj
}

// looseList decodes raw as a list of objects; a lone object is a list of one.
func looseList(raw json.RawMessage) []map[string]json.RawMessage {
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		items = []json.RawMessage{raw}
	}
	var list []map[string]json.RawMessage
	for _, item := range items {
		if obj := looseObject(item); obj != nil {
			list = append(list, obj)
		}
	}
	return list
}

// looseText reads raw as text: a string's content, or the JSON itself for any
// other type ("" for null). NUL, which Postgres refuses, is dropped.
func looseText(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	s := string(raw)
	if raw[0] == '"' {
		if json.Unmarshal(raw, &s) != nil {
			s = string(raw)
		}
	}
	return strings.ToValidUTF8(strings.ReplaceAll(s, "\x00", ""), "�")
}
