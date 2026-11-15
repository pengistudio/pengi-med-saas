package whatsapp_handlers

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
)

// echoAt is an app echo with an explicit id and timestamp.
func echoAt(phoneID, to, id, body string, at time.Time) map[string]any {
	ev := echoEvent(phoneID, to, body)
	echo := ev["entry"].([]any)[0].(map[string]any)["changes"].([]any)[0].(map[string]any)["value"].(map[string]any)["message_echoes"].([]any)[0].(map[string]any)
	echo["id"], echo["timestamp"] = id, fmt.Sprint(at.Unix())
	return ev
}

// accountUpdate is an account_update event with extra value fields.
func accountUpdate(entryID, event string, extra map[string]any) map[string]any {
	value := map[string]any{"event": event}
	for k, v := range extra {
		value[k] = v
	}
	return map[string]any{"object": "whatsapp_business_account", "entry": []any{map[string]any{
		"id": entryID, "changes": []any{map[string]any{"field": "account_update", "value": value}},
	}}}
}

// concurrently delivers ev n times at once (Meta retrying in parallel).
func (f *fixture) concurrently(n int, ev map[string]any) {
	f.t.Helper()
	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = f.signed(ev)
		}(i)
	}
	wg.Wait()
	for _, c := range codes {
		if c != http.StatusOK {
			f.t.Fatalf("codes = %v", codes)
		}
	}
}

func (f *fixture) countWam(tenantID uint, wamID string) int64 {
	var n int64
	f.db.Model(&whatsapp_models.WhatsAppMessage{}).Where("tenant_id = ? AND wam_id = ?", tenantID, wamID).Count(&n)
	return n
}

// Concurrent deliveries of one echo or one inbound message store it once
// (the unique (tenant_id, wam_id) index, not only the check before insert).
func TestWebhook_DuplicateDeliveriesStoreOnce(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")

	f.concurrently(6, echoAt("PHONE-OWN", "593991111111", "wamid.echo.race", "le atiendo", time.Now()))
	if n := f.countWam(f.own, "wamid.echo.race"); n != 1 {
		t.Fatalf("echo rows = %d", n)
	}

	in := messagesEvent("PHONE-OWN", map[string]any{"messages": []any{map[string]any{
		"from": "593992222222", "id": "wamid.in.race", "timestamp": fmt.Sprint(time.Now().Unix()), "type": "text",
		"text": map[string]string{"body": "hola"},
	}}})
	f.concurrently(6, in)
	if n := f.countWam(f.own, "wamid.in.race"); n != 1 {
		t.Fatalf("inbound rows = %d", n)
	}
	if c := f.convByPhone(f.own, "593992222222"); c.UnreadCount != 1 {
		t.Fatalf("unread = %d, want 1", c.UnreadCount)
	}

	// The insert itself refuses a stored id, and leaves the thread alone.
	db := tenantdb.ForTenant(f.db, f.own)
	conv := f.convByPhone(f.own, "593991111111")
	msg := whatsapp_models.WhatsAppMessage{TenantID: f.own, Kind: whatsapp_models.KindApp, WamID: "wamid.echo.race", Status: whatsapp_models.MessageStatusSent, Body: "otra vez"}
	if err := whatsapp_services.RecordOutbound(db, &conv, &msg, time.Now().Add(time.Hour)); !errors.Is(err, whatsapp_services.ErrMessageExists) {
		t.Fatalf("err = %v, want ErrMessageExists", err)
	}
	if after := f.convByPhone(f.own, "593991111111"); after.LastMessagePreview != "le atiendo" {
		t.Fatalf("preview = %q", after.LastMessagePreview)
	}
}

// An app echo is not billed, and the assistant stays quiet afterwards: a
// person is answering from the phone.
func TestWebhook_EchoNotBilledAndBotQuiet(t *testing.T) {
	f := newFixture(t)
	ai := startFakeAI(t)
	ai.set(answerParking)
	f.enableBot(f.own, "PHONE-BOT", botOptions{})
	f.writes("PHONE-BOT", "593991234567", nextWamID(), "Quiero una persona")
	conv := f.convByPhone(f.own, "593991234567")
	if !conv.NeedsHuman() {
		t.Fatal("handed off")
	}

	f.signed(echoAt("PHONE-BOT", "593991234567", "wamid.echo.app1", "Hola, le atiendo", time.Now()))
	if f.convByPhone(f.own, "593991234567").NeedsHuman() {
		t.Fatal("the clinic answering from the phone clears the handoff")
	}
	usage, err := whatsapp_services.MonthlyUsage(tenantdb.ForTenant(f.db, f.own), f.own, time.Now())
	if err != nil || usage.Used != 0 {
		t.Fatalf("usage = %+v, err = %v: an echo is never billed", usage, err)
	}

	f.writes("PHONE-BOT", "593991234567", nextWamID(), "¿Tienen parqueadero?")
	if n := len(f.botMessages(conv.ID)); n != 1 || ai.calls() != 0 {
		t.Fatalf("assistant must stay quiet (%d messages, %d calls)", n, ai.calls())
	}
	evs := f.botEvents(conv.ID)
	if evs[len(evs)-1].Reason != whatsapp_models.BotReasonHumanActive {
		t.Fatalf("last event = %+v", evs[len(evs)-1])
	}
}

// Events we don't act on change nothing; a reconnect or a removal never
// touches rejected credentials.
func TestWebhook_AccountUpdateGuards(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")

	for _, ev := range []string{"VERIFIED_ACCOUNT", "DISABLED_UPDATE", ""} {
		f.signed(accountUpdate("WABA-OWN", ev, map[string]any{"waba_info": map[string]string{"waba_id": "WABA-OWN"}}))
		if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusConnected {
			t.Fatalf("event %q changed status to %s", ev, s)
		}
	}

	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", f.own).Update("status", whatsapp_models.AccountStatusTokenInvalid)
	f.signed(accountUpdateEvent("PARTNER-BIZ", "PARTNER_REMOVED", "WABA-OWN", "PRIMARY_INACTIVITY"))
	f.signed(accountUpdateEvent("WABA-OWN", "ACCOUNT_RECONNECTED", "", ""))
	if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusTokenInvalid {
		t.Fatalf("removal + reconnect revived a rejected token: %s", s)
	}
}

// Two clinics on one WABA: an update naming a number touches only that
// clinic's account; one without a number touches both.
func TestWebhook_AccountUpdateSharedWaba(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-SHARED")
	f.seedReminder(f.other, "PHONE-OTHER", "WABA-SHARED")
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", f.own).Update("display_phone", "+593 99 111 1111")
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", f.other).Update("display_phone", "+593 99 222 2222")
	status := func() (string, string) { return f.account(f.own).Status, f.account(f.other).Status }
	waba := map[string]string{"waba_id": "WABA-SHARED"}

	f.signed(accountUpdate("BIZ", "PARTNER_REMOVED", map[string]any{"waba_info": waba, "phone_number_id": "PHONE-OWN"}))
	if own, other := status(); own != whatsapp_models.AccountStatusAppDisconnected || other != whatsapp_models.AccountStatusConnected {
		t.Fatalf("by phone_number_id: own=%s other=%s", own, other)
	}
	f.signed(accountUpdate("WABA-SHARED", "ACCOUNT_RECONNECTED", map[string]any{"phone_number": "593991111111"}))
	if own, other := status(); own != whatsapp_models.AccountStatusConnected || other != whatsapp_models.AccountStatusConnected {
		t.Fatalf("reconnect by display number: own=%s other=%s", own, other)
	}
	f.signed(accountUpdate("BIZ", "PARTNER_REMOVED", map[string]any{"waba_info": waba, "phone_number": "593992222222"}))
	if own, other := status(); own != whatsapp_models.AccountStatusConnected || other != whatsapp_models.AccountStatusAppDisconnected {
		t.Fatalf("by display number: own=%s other=%s", own, other)
	}
	f.signed(accountUpdate("BIZ", "PARTNER_REMOVED", map[string]any{"waba_info": waba}))
	if own, other := status(); own != whatsapp_models.AccountStatusAppDisconnected || other != whatsapp_models.AccountStatusAppDisconnected {
		t.Fatalf("whole WABA: own=%s other=%s", own, other)
	}
	// A number of another WABA is not ours to reconnect.
	f.signed(accountUpdate("WABA-ELSE", "ACCOUNT_RECONNECTED", map[string]any{"phone_number_id": "PHONE-OWN"}))
	if own, _ := status(); own != whatsapp_models.AccountStatusAppDisconnected {
		t.Fatalf("other WABA reconnected own: %s", own)
	}
}

// In the history, what the business number wrote is the clinic's (whatever
// the display number's formatting), the rest the patient's; messages with
// the same timestamp keep their order (created_at, then id).
func TestWebhook_HistoryClassificationAndTieOrder(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.signed(historyEvent("PHONE-OWN", map[string]any{"id": "593994445555", "messages": []any{
		historyMsg("wamid.t1", "593994445555", "1700000000", "uno", "read"),
		historyMsg("wamid.t2", "15550100", "1700000000", "dos", "delivered"),
		historyMsg("wamid.t3", "593994445555", "1700000000", "tres", "read"),
	}}))
	conv := f.convByPhone(f.own, "593994445555")
	page := f.callID(f.h.ListConversationMessages, f.own, http.MethodGet, conv.ID, nil, "").Data.(whatsapp_dto.ThreadResponse)
	if len(page.Items) != 3 || page.Items[0].Body != "uno" || page.Items[1].Body != "dos" || page.Items[2].Body != "tres" {
		t.Fatalf("thread = %+v", page.Items)
	}
	var clinic whatsapp_models.WhatsAppMessage
	f.db.Where("wam_id = ?", "wamid.t2").First(&clinic)
	if clinic.Kind != whatsapp_models.KindApp || clinic.Direction != whatsapp_models.DirectionOutbound || clinic.Status != whatsapp_models.MessageStatusDelivered {
		t.Fatalf("business message = %+v", clinic)
	}
	var patient whatsapp_models.WhatsAppMessage
	f.db.Where("wam_id = ?", "wamid.t1").First(&patient)
	if patient.Kind != whatsapp_models.KindInbound || patient.Direction != whatsapp_models.DirectionInbound {
		t.Fatalf("patient message = %+v", patient)
	}
}

// A message id is unique per clinic, not across clinics: the same echo or
// history id on two numbers lands in each clinic's own thread only.
func TestWebhook_EchoAndHistoryTenantIsolation(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	f.seedReminder(f.other, "PHONE-OTHER", "WABA-OTHER")

	f.signed(echoAt("PHONE-OWN", "593996667777", "wamid.shared.echo", "propio", time.Now()))
	f.signed(echoAt("PHONE-OTHER", "593996667777", "wamid.shared.echo", "ajeno", time.Now()))
	thread := historyMsg("wamid.shared.hist", "593996667777", "1700000000", "historia", "read")
	f.signed(historyEvent("PHONE-OWN", map[string]any{"id": "593996667777", "messages": []any{thread}}))
	f.signed(historyEvent("PHONE-OTHER", map[string]any{"id": "593996667777", "messages": []any{thread}}))

	for _, tenant := range []uint{f.own, f.other} {
		if n := f.countWam(tenant, "wamid.shared.echo"); n != 1 {
			t.Fatalf("tenant %d echo rows = %d", tenant, n)
		}
		if n := f.countWam(tenant, "wamid.shared.hist"); n != 1 {
			t.Fatalf("tenant %d history rows = %d", tenant, n)
		}
		conv := f.convByPhone(tenant, "593996667777")
		var foreign int64
		f.db.Model(&whatsapp_models.WhatsAppMessage{}).Where("conversation_id = ? AND tenant_id <> ?", conv.ID, tenant).Count(&foreign)
		if foreign != 0 {
			t.Fatalf("tenant %d thread holds %d foreign messages", tenant, foreign)
		}
	}
	var own whatsapp_models.WhatsAppMessage
	f.db.Where("tenant_id = ? AND wam_id = ?", f.own, "wamid.shared.echo").First(&own)
	if own.Body != "propio" {
		t.Fatalf("own echo = %q", own.Body)
	}
}

// Payloads with missing pieces are ignored without panicking.
func TestWebhook_IncompletePayloadsDoNotPanic(t *testing.T) {
	f := newFixture(t)
	f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")
	change := func(field string, value any) map[string]any {
		return map[string]any{"object": "whatsapp_business_account", "entry": []any{map[string]any{
			"id": "WABA-OWN", "changes": []any{map[string]any{"field": field, "value": value}},
		}}}
	}
	meta := map[string]string{"phone_number_id": "PHONE-OWN"}
	payloads := []any{
		map[string]any{},
		map[string]any{"entry": []any{map[string]any{}}},
		change("messages", map[string]any{}),
		change("messages", map[string]any{"metadata": meta, "messages": []any{map[string]any{}}, "statuses": []any{map[string]any{}}}),
		change("messages", map[string]any{"metadata": meta, "messages": []any{map[string]any{"type": "text", "id": "wamid.x"}}}),
		change("messages", map[string]any{"metadata": meta, "messages": []any{map[string]any{"type": "button", "from": "593991234567", "id": "wamid.y"}}}),
		change("messages", map[string]any{"metadata": meta, "messages": []any{map[string]any{"type": "interactive", "from": "593991234567", "id": "wamid.z"}}}),
		change("smb_message_echoes", map[string]any{"metadata": meta, "message_echoes": []any{map[string]any{}, map[string]any{"to": "593991234567"}, map[string]any{"id": "wamid.e"}}}),
		change("history", map[string]any{"metadata": meta, "history": []any{map[string]any{}, map[string]any{"threads": []any{map[string]any{}, map[string]any{"id": "593991234567", "messages": []any{map[string]any{}}}}}}}),
		change("history", map[string]any{"metadata": meta, "history": []any{map[string]any{"errors": []any{}}}}),
		change("account_update", map[string]any{}),
		change("account_update", map[string]any{"event": "PARTNER_REMOVED", "waba_info": map[string]any{}}),
		change("message_template_status_update", map[string]any{}),
		change("unknown_field", map[string]any{"x": 1}),
	}
	for i, p := range payloads {
		if code := f.signed(p); code != http.StatusOK {
			t.Fatalf("payload %d: code %d", i, code)
		}
	}
	if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusAppDisconnected {
		// PARTNER_REMOVED without waba_info falls back to the entry id.
		t.Fatalf("status = %s", s)
	}
}
