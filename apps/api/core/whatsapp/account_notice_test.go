package whatsapp_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/whatsapp"
)

// parse decodes a webhook payload, which must never fail.
func parse(t *testing.T, body string) whatsapp.WebhookPayload {
	t.Helper()
	var p whatsapp.WebhookPayload
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("payload dropped: %v\n%s", err, body)
	}
	return p
}

// change parses one change of field with the given raw value.
func change(t *testing.T, field, value string) whatsapp.WebhookChange {
	t.Helper()
	p := parse(t, `{"entry":[{"id":"W","changes":[{"field":"`+field+`","value":`+value+`}]}]}`)
	return p.Entry[0].Changes[0]
}

func TestAccountNotice_Detail(t *testing.T) {
	cases := []struct{ name, field, value, want string }{
		{"violation", "account_update", `{"event":"ACCOUNT_VIOLATION","violation_info":{"violation_type":"HEALTHCARE"}}`, "violation_type=HEALTHCARE"},
		{"ban", "account_update", `{"event":"DISABLED_UPDATE","ban_info":{"waba_ban_state":"DISABLE","waba_ban_date":"April 17, 2025"}}`,
			"waba_ban_state=DISABLE waba_ban_date=April 17, 2025"},
		{"restrictions", "account_update", `{"restriction_info":[{"restriction_type":"A","expiration":1641330498},{"restriction_type":"B","expiration":"2026-11-30","remediation":"Review"}]}`,
			"restriction_type=A expiration=1641330498 restriction_type=B expiration=2026-11-30 remediation=Review"},
		{"disconnection", "account_update", `{"disconnection_info":{"reason":"ACCOUNT_DISCONNECTED","initiated_by":"SYSTEM"}}`, "reason=ACCOUNT_DISCONNECTED initiated_by=SYSTEM"},
		{"review", "account_review_update", `{"decision":"REJECTED"}`, "decision=REJECTED"},
		{"alert", "account_alerts", `{"entity_type":"PHONE_NUMBER","entity_id":"77","alert_info":{"alert_severity":"CRITICAL","alert_status":"NONE","alert_type":"OBA_REJECTED","alert_description":"x"}}`,
			"alert_type=OBA_REJECTED alert_severity=CRITICAL alert_status=NONE entity_type=PHONE_NUMBER entity_id=77 alert_description=x"},
		{"alert as a list", "account_alerts", `{"entity_id":77,"alert_info":[{"alert_type":"A"}]}`, "alert_type=A entity_id=77"},
		{"lone restriction", "account_update", `{"restriction_info":{"restriction_type":"A"}}`, "restriction_type=A"},
		{"nothing known", "account_update", `{"event":"PARTNER_ADDED","waba_info":{"waba_id":"W"}}`, ""},
		{"junk objects", "account_update", `{"violation_info":"x","ban_info":[1],"restriction_info":"y","alert_info":7,"disconnection_info":null}`, ""},
		{"nested scalar", "account_update", `{"violation_info":{"violation_type":{"a":1}}}`, `violation_type={"a":1}`},
		{"not an object", "account_update", `"x"`, ""},
		{"null", "account_update", `null`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := change(t, tc.field, tc.value)
			if got := c.AccountNotice().Detail(); got != tc.want {
				t.Fatalf("detail = %q, want %q", got, tc.want)
			}
			if string(c.Raw) != tc.value {
				t.Fatalf("raw = %s", c.Raw)
			}
		})
	}

	long := change(t, "account_update", `{"violation_info":{"violation_type":"`+strings.Repeat("ñ", 900)+`"}}`)
	if n := len([]rune(long.AccountNotice().Detail())); n != 500 {
		t.Fatalf("detail runes = %d", n)
	}
	nul := change(t, "account_update", "{\"event\":\"A\\u0000B\"}")
	if got := nul.AccountNotice().Event; got != "AB" {
		t.Fatalf("event = %q", got)
	}
}

func TestAccountNotice_PhoneNumberID(t *testing.T) {
	for value, want := range map[string]string{
		`{"phone_number_id":"1"}`:                           "1",
		`{"metadata":{"phone_number_id":"2"}}`:              "2",
		`{"entity_type":"PHONE_NUMBER","entity_id":"3"}`:    "3",
		`{"entity_type":"BUSINESS","entity_id":"4"}`:        "",
		`{"phone_number_id":593991111111,"entity_id":"no"}`: "593991111111",
	} {
		if got := change(t, "account_alerts", value).AccountNotice().PhoneNumberID; got != want {
			t.Fatalf("%s: phone number id = %q, want %q", value, got, want)
		}
	}
}

// The raw value is kept byte for byte (spacing, key order, escapes).
func TestWebhookChange_RawIsExact(t *testing.T) {
	raw := `{ "z":1,  "a": "é/\/", "n": 1.50, "big": 12345678901234567890 }`
	if got := string(change(t, "account_update", raw).Raw); got != raw {
		t.Fatalf("raw = %s", got)
	}
}

// Nothing in a change or in the entry's time may fail the whole payload.
func TestWebhookPayload_OddShapesDoNotDropIt(t *testing.T) {
	for _, odd := range []string{`{"field":"account_update","value":{"phone_number":12,"waba_info":"x"}}`, `{"field":7,"value":{}}`, `"x"`, `null`, `[]`, `{}`} {
		p := parse(t, `{"entry":[{"id":"W","time":{},"changes":[`+odd+`,{"field":"messages","value":{"messages":[{"id":"m1"}]}}]}]}`)
		changes := p.Entry[0].Changes
		if len(changes) != 2 || changes[1].DecodeErr != nil || len(changes[1].Value.Messages) != 1 {
			t.Fatalf("%s: changes = %+v", odd, changes)
		}
	}
	if c := change(t, "account_update", `{"phone_number":12}`); c.DecodeErr == nil || c.AccountNotice().PhoneNumber != "12" {
		t.Fatalf("change = %+v", c)
	}
}

func TestWebhookTime(t *testing.T) {
	at := time.Unix(1739321024, 0)
	for raw, want := range map[string]time.Time{
		`1739321024`: at, `"1739321024"`: at,
		`0`: {}, `-5`: {}, `null`: {}, `"x"`: {}, `{}`: {}, `[1]`: {}, `1.5`: {}, `99999999999999`: {},
	} {
		p := parse(t, `{"entry":[{"id":"W","time":`+raw+`}]}`)
		if got := p.Entry[0].Time.Time; !got.Equal(want) {
			t.Fatalf("time %s = %v, want %v", raw, got, want)
		}
	}
}
