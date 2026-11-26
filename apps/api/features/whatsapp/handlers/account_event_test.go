package whatsapp_handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"pengi-med-saas/core/tenantdb"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// accountEvents are the stored events of a WABA, oldest first.
func (f *fixture) accountEvents(wabaID string) []whatsapp_models.WhatsAppAccountEvent {
	f.t.Helper()
	var evs []whatsapp_models.WhatsAppAccountEvent
	if err := tenantdb.System(f.db).Where("waba_id = ?", wabaID).Order("id").Find(&evs).Error; err != nil {
		f.t.Fatal(err)
	}
	return evs
}

// signedRaw delivers a webhook body as is.
func (f *fixture) signedRaw(body string) {
	f.t.Helper()
	if code := f.signed(json.RawMessage(body)); code != http.StatusOK {
		f.t.Fatalf("code = %d", code)
	}
}

func accountChange(wabaID, time, field, value string) string {
	return `{"object":"whatsapp_business_account","entry":[{"id":"` + wabaID + `","time":` + time +
		`,"changes":[{"field":"` + field + `","value":` + value + `}]}]}`
}

// Meta's notices are stored with their reason and the raw value, whether or
// not an account matches, and never change an account's status.
func TestWebhook_AccountEventsStored(t *testing.T) {
	cases := []struct {
		name, field, value, event, detail string
		level                             zapcore.Level
	}{
		{"violation", "account_update", `{"phone_number":"15550783881","event":"ACCOUNT_VIOLATION","violation_info":{"violation_type":"HEALTHCARE"}}`,
			"ACCOUNT_VIOLATION", "violation_type=HEALTHCARE", zap.WarnLevel},
		{"ban", "account_update", `{"phone_number":"15550783881","event":"DISABLED_UPDATE","ban_info":{"waba_ban_state":"DISABLE","waba_ban_date":"November 16, 2026"}}`,
			"DISABLED_UPDATE", "waba_ban_state=DISABLE waba_ban_date=November 16, 2026", zap.WarnLevel},
		{"restriction", "account_update", `{"phone_number":"15550783881","event":"ACCOUNT_RESTRICTION","restriction_info":[{"restriction_type":"RESTRICTED_BIZ_INITIATED_MESSAGING","expiration":1641330498}]}`,
			"ACCOUNT_RESTRICTION", "restriction_type=RESTRICTED_BIZ_INITIATED_MESSAGING expiration=1641330498", zap.WarnLevel},
		{"offboarded", "account_update", `{"phone_number":"15550783881","event":"ACCOUNT_OFFBOARDED"}`, "ACCOUNT_OFFBOARDED", "", zap.WarnLevel},
		{"deleted", "account_update", `{"phone_number":"15550783881","event":"ACCOUNT_DELETED"}`, "ACCOUNT_DELETED", "", zap.WarnLevel},
		{"partner added", "account_update", `{"phone_number":"15550783881","event":"PARTNER_ADDED"}`, "PARTNER_ADDED", "", zap.InfoLevel},
		{"review rejected", "account_review_update", `{"phone_number":"15550783881","decision":"REJECTED"}`, "", "decision=REJECTED", zap.WarnLevel},
		{"review approved", "account_review_update", `{"phone_number":"15550783881","decision":"APPROVED"}`, "", "decision=APPROVED", zap.InfoLevel},
		{"alert", "account_alerts", `{"phone_number":"15550783881","entity_type":"BUSINESS","entity_id":"9","alert_info":{"alert_type":"INCREASED_CAPABILITIES_ELIGIBILITY_FAILED","alert_severity":"CRITICAL"}}`,
			"", "alert_type=INCREASED_CAPABILITIES_ELIGIBILITY_FAILED alert_severity=CRITICAL entity_type=BUSINESS entity_id=9", zap.InfoLevel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
			core, logs := observer.New(zap.InfoLevel)
			f.webhook.logger = zap.New(core)

			for _, waba := range []string{"WABA-OWN", "WABA-NOT-CONNECTED"} {
				f.signedRaw(accountChange(waba, "1739321024", tc.field, tc.value))
				evs := f.accountEvents(waba)
				if len(evs) != 1 {
					t.Fatalf("%s: stored events = %d", waba, len(evs))
				}
				ev := evs[0]
				if ev.Field != tc.field || ev.Event != tc.event || ev.Detail != tc.detail || ev.PhoneNumber != "15550783881" {
					t.Fatalf("event = %+v", ev)
				}
				if ev.Payload != tc.value {
					t.Fatalf("payload = %s", ev.Payload)
				}
				if ev.OccurredAt == nil || ev.OccurredAt.Unix() != 1739321024 {
					t.Fatalf("occurred at = %v", ev.OccurredAt)
				}
			}
			if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusConnected {
				t.Fatalf("status = %s", s)
			}
			entries := logs.FilterMessage("whatsapp webhook: account event").All()
			if len(entries) != 2 || entries[0].Level != tc.level {
				t.Fatalf("logs = %+v", entries)
			}
			if m := entries[0].ContextMap(); m["waba_id"] != "WABA-OWN" || m["event"] != tc.event || m["detail"] != tc.detail {
				t.Fatalf("log fields = %v", m)
			}
		})
	}
}

// The WABA of a partner event is waba_info's, not the entry's (a partner's).
// Only a removal that may be enforcement is a warning.
func TestWebhook_AccountEventWabaOfPartnerEvent(t *testing.T) {
	f := newFixture(t)
	core, logs := observer.New(zap.InfoLevel)
	f.webhook.logger = zap.New(core)
	f.signed(accountUpdateEvent("PARTNER-BIZ", "PARTNER_REMOVED", "WABA-X", "ACCOUNT_DISCONNECTED"))
	if evs := f.accountEvents("WABA-X"); len(evs) != 1 || evs[0].Detail != "reason=ACCOUNT_DISCONNECTED initiated_by=SYSTEM" || evs[0].OccurredAt != nil {
		t.Fatalf("events = %+v", evs)
	}
	// ACCOUNT_DISCONNECTED may be Meta's enforcement; inactivity is routine.
	f.signed(accountUpdateEvent("PARTNER-BIZ", "PARTNER_REMOVED", "WABA-X", "PRIMARY_INACTIVITY"))
	entries := logs.FilterMessage("whatsapp webhook: account event").All()
	if len(entries) != 2 || entries[0].Level != zap.WarnLevel || entries[1].Level != zap.InfoLevel {
		t.Fatalf("logs = %+v", entries)
	}
}

// A retried webhook stores one row; another time, or no time at all, is
// another event.
func TestWebhook_AccountEventDedupe(t *testing.T) {
	f := newFixture(t)
	value := `{"event":"ACCOUNT_VIOLATION","violation_info":{"violation_type":"SCAM"}}`
	var ev map[string]any
	if err := json.Unmarshal([]byte(accountChange("W", "1739321024", "account_update", value)), &ev); err != nil {
		t.Fatal(err)
	}
	f.concurrently(5, ev)
	if n := len(f.accountEvents("W")); n != 1 {
		t.Fatalf("retries stored %d rows", n)
	}
	f.signedRaw(accountChange("W", "1739321025", "account_update", value))
	f.signedRaw(accountChange("W", "null", "account_update", value))
	f.signedRaw(accountChange("W", "null", "account_update", value))
	if n := len(f.accountEvents("W")); n != 4 {
		t.Fatalf("rows = %d, want 4", n)
	}
}

// A notice whose fields have unexpected types is still stored, changes no
// status and doesn't drop the patient message of the same payload.
func TestWebhook_OddAccountEventKeepsMessages(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	for i, odd := range []string{
		`{"event":"ACCOUNT_RESTRICTION","restriction_info":{"restriction_type":["A"],"expiration":{"x":1}}}`,
		`{"event":"PARTNER_REMOVED","waba_info":{"waba_id":"WABA-OWN"},"phone_number":12,"ban_info":"x","violation_info":[]}`,
		`"not an object"`,
	} {
		id := nextWamID()
		f.signedRaw(`{"object":"whatsapp_business_account","entry":[{"id":"WABA-OWN","time":"soon","changes":[` +
			`{"field":"account_update","value":` + odd + `},` +
			`{"field":"messages","value":{"metadata":{"phone_number_id":"PHONE-OWN"},"messages":[{"from":"593997777777","id":"` + id + `","type":"text","text":{"body":"hola"}}]}}]}]}`)
		if n := f.countWam(f.own, id); n != 1 {
			t.Fatalf("%s: message rows = %d", odd, n)
		}
		evs := f.accountEvents("WABA-OWN")
		if len(evs) != i+1 || evs[i].Payload != odd {
			t.Fatalf("%s: events = %+v", odd, evs)
		}
		if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusConnected {
			t.Fatalf("%s: status = %s", odd, s)
		}
	}
}

// A malformed change is skipped alone: the others of the payload are applied.
func TestWebhook_MalformedChangeIsSkippedAlone(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.signedRaw(`{"entry":[{"id":"WABA-OWN","changes":[{"field":"messages","value":{"metadata":{"phone_number_id":"PHONE-OWN"},"messages":"x"}},` +
		`{"field":"messages","value":{"metadata":{"phone_number_id":"PHONE-OWN"},"messages":[{"from":"593997777777","id":"wamid.ok","type":"text","text":{"body":"hola"}}]}}]}]}`)
	if n := f.countWam(f.own, "wamid.ok"); n != 1 {
		t.Fatalf("message rows = %d", n)
	}
}

// Failing to store the event is logged with its payload and doesn't stop the
// account update it carries.
func TestWebhook_AccountEventStoreFailure(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	core, logs := observer.New(zap.ErrorLevel)
	f.webhook.logger = zap.New(core)
	if err := f.db.Migrator().DropTable(&whatsapp_models.WhatsAppAccountEvent{}); err != nil {
		t.Fatal(err)
	}
	f.signed(accountUpdateEvent("PARTNER-BIZ", "PARTNER_REMOVED", "WABA-OWN", "PRIMARY_INACTIVITY"))
	if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusAppDisconnected {
		t.Fatalf("status = %s", s)
	}
	errs := logs.FilterMessage("whatsapp webhook: failed to store account event").All()
	if len(errs) != 1 || !strings.Contains(errs[0].ContextMap()["payload"].(string), "PRIMARY_INACTIVITY") {
		t.Fatalf("logs = %+v", errs)
	}
}

// Contacts, history and messages are never kept as account events.
func TestWebhook_PatientDataIsNotAnAccountEvent(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.signedRaw(accountChange("WABA-OWN", "1739321024", "smb_app_state_sync",
		`{"metadata":{"phone_number_id":"PHONE-OWN"},"state_sync":[{"type":"contact","contact":{"full_name":"Ana","phone_number":"593991234567"},"action":"add"}]}`))
	f.signed(historyEvent("PHONE-OWN", map[string]any{"id": "593994445555", "messages": []any{
		historyMsg("wamid.h1", "593994445555", "1700000000", "uno", "read"),
	}}))
	f.signed(textEvent("PHONE-OWN", "593991234567", "hola"))
	var n int64
	tenantdb.System(f.db).Model(&whatsapp_models.WhatsAppAccountEvent{}).Count(&n)
	if n != 0 {
		t.Fatalf("account events = %d", n)
	}
}

// A history chunk may weigh more than 1 MB (Meta: up to 3 MB).
func TestWebhook_AcceptsLargePayloads(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	if code := f.signed(historyEvent("PHONE-OWN", map[string]any{"id": "593994445555", "messages": []any{
		historyMsg("wamid.big", "593994445555", "1700000000", strings.Repeat("a", 3<<20-4096), "read"),
	}})); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if n := f.countWam(f.own, "wamid.big"); n != 1 {
		t.Fatalf("rows = %d", n)
	}
}
