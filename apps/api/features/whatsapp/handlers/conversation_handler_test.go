package whatsapp_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
	"pengi-med-saas/testutils"
)

const inboxUserID = 42

func errCode(r envelope.Response) string {
	if e, ok := r.Data.(core_errors.AppError); ok {
		return e.ErrorCode
	}
	return ""
}

// callID runs a conversation endpoint for :id as user inboxUserID of tenantID.
func (f *fixture) callID(action envelope.Action, tenantID uint, method string, id uint, body any, query string) envelope.Response {
	c, _ := testutils.NewGinContext(tenantID, inboxUserID)
	c.Set("user_id", int64(inboxUserID))
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest(method, "/?"+query, bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	return action(c)
}

func (f *fixture) patient(tenantID uint, first, phone string) clinical_models.Patient {
	f.t.Helper()
	p := clinical_models.Patient{TenantID: tenantID, FirstName: first, LastName: "Mora", Phone: phone, Document: fmt.Sprintf("D%d", time.Now().UnixNano())}
	if err := f.db.Create(&p).Error; err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fixture) connectedAccount(tenantID uint, phoneID string) {
	f.t.Helper()
	box, err := whatsapp_models.Box()
	if err != nil {
		f.t.Fatal(err)
	}
	sealed, err := box.Seal("token-" + phoneID)
	if err != nil {
		f.t.Fatal(err)
	}
	acc := whatsapp_models.WhatsAppAccount{TenantID: tenantID, WabaID: "WABA-" + phoneID, PhoneNumberID: phoneID, AccessTokenEncrypted: sealed,
		Status: whatsapp_models.AccountStatusConnected, TemplateStatus: whatsapp_models.TemplateStatusApproved, ReminderOffsets: []int{24}}
	if err := f.db.Create(&acc).Error; err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) conversations(tenantID uint) []whatsapp_models.WhatsAppConversation {
	var convs []whatsapp_models.WhatsAppConversation
	f.db.Where("tenant_id = ?", tenantID).Order("id").Find(&convs)
	return convs
}

func (f *fixture) thread(convID uint) []whatsapp_models.WhatsAppMessage {
	var msgs []whatsapp_models.WhatsAppMessage
	f.db.Where("conversation_id = ?", convID).Order("id").Find(&msgs)
	return msgs
}

func inboundEvent(phoneID string, msg map[string]any) map[string]any {
	return messagesEvent(phoneID, map[string]any{"messages": []any{msg}})
}

// writes is the patient sending text from `from` (wamid id) right now.
func (f *fixture) writes(phoneID, from, id, body string) {
	f.t.Helper()
	code := f.signed(inboundEvent(phoneID, map[string]any{
		"from": from, "id": id, "timestamp": fmt.Sprint(time.Now().Unix()), "type": "text",
		"text": map[string]string{"body": body},
	}))
	if code != http.StatusOK {
		f.t.Fatalf("webhook code = %d", code)
	}
}

func TestWebhook_InboundCreatesConversationWithoutDuplicates(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-OWN")
	p := f.patient(f.own, "Luis", "0991234567")

	f.writes("PHONE-OWN", "593991234567", "wamid.A", "¿Puedo pasarla a las 3?")
	f.writes("PHONE-OWN", "593991234567", "wamid.A", "¿Puedo pasarla a las 3?") // Meta retry

	convs := f.conversations(f.own)
	if len(convs) != 1 {
		t.Fatalf("conversations = %d", len(convs))
	}
	c := convs[0]
	if c.Phone != "593991234567" || c.PatientID == nil || *c.PatientID != p.ID || c.UnreadCount != 1 ||
		c.LastInboundAt == nil || c.LastDirection != whatsapp_models.DirectionInbound || c.LastMessagePreview != "¿Puedo pasarla a las 3?" ||
		!c.WindowOpen(time.Now()) {
		t.Fatalf("conversation = %+v", c)
	}
	msgs := f.thread(c.ID)
	if len(msgs) != 1 {
		t.Fatalf("messages = %d, want 1 (deduped by wamid)", len(msgs))
	}
	m := msgs[0]
	if m.Kind != whatsapp_models.KindInbound || m.Direction != whatsapp_models.DirectionInbound || m.Status != whatsapp_models.MessageStatusReceived ||
		m.Body != "¿Puedo pasarla a las 3?" || m.ContentType != whatsapp_models.ContentText || m.WamID != "wamid.A" || m.PatientID == nil || *m.PatientID != p.ID {
		t.Fatalf("message = %+v", m)
	}

	f.writes("PHONE-OWN", "593991234567", "wamid.B", "gracias")
	if c := f.conversations(f.own)[0]; c.UnreadCount != 2 || c.LastMessagePreview != "gracias" {
		t.Fatalf("after second message = %+v", c)
	}
	// Nothing reached the other clinic.
	if len(f.conversations(f.other)) != 0 {
		t.Fatal("conversation leaked to another tenant")
	}
}

func TestWebhook_InboundMediaAndButtons(t *testing.T) {
	f := newFixture(t)
	appt, reminder := f.seedReminder(f.own, "PHONE-OWN", "WABA-OWN")

	f.signed(inboundEvent("PHONE-OWN", map[string]any{
		"from": "593991234567", "id": "wamid.IMG", "timestamp": "1760000000", "type": "image",
		"image": map[string]string{"id": "media-1", "mime_type": "image/jpeg", "caption": "mi orden"},
	}))
	f.signed(inboundEvent("PHONE-OWN", map[string]any{
		"from": "593991234567", "id": "wamid.AUD", "timestamp": "1760000001", "type": "audio",
		"audio": map[string]string{"id": "media-2", "mime_type": "audio/ogg"},
	}))
	f.signed(replyEvent("PHONE-OWN", fmt.Sprintf("confirm:%d", reminder.ID), "593991234567", reminder.WamID))

	convs := f.conversations(f.own)
	if len(convs) != 1 {
		t.Fatalf("conversations = %d", len(convs))
	}
	msgs := f.thread(convs[0].ID)
	if len(msgs) != 3 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if msgs[0].ContentType != whatsapp_models.ContentImage || msgs[0].Body != "mi orden" ||
		msgs[1].ContentType != whatsapp_models.ContentAudio || msgs[1].Body != "" ||
		msgs[2].ContentType != whatsapp_models.ContentButton || msgs[2].Body != "Cancelar" || msgs[2].Reply != whatsapp_models.ReplyConfirm {
		t.Fatalf("messages = %+v", msgs)
	}
	// The button still confirms the appointment after being recorded.
	if s := f.appointmentStatus(appt.ID); s != "confirmed" {
		t.Fatalf("appointment = %s, want confirmed", s)
	}
}

func TestEnsureConversation_PatientMatching(t *testing.T) {
	f := newFixture(t)
	db := f.db // tenant filtered through tenant_id below
	unique := f.patient(f.own, "Ana", "099 111 2222")
	f.patient(f.own, "Ben", "0993334444")
	f.patient(f.own, "Bea", "+593 99 333 4444") // same number as Ben: ambiguous
	f.patient(f.other, "Otra", "0995556666")    // only in another clinic

	cases := []struct {
		phone string
		want  *uint
	}{
		{"593991112222", &unique.ID},
		{"593993334444", nil},
		{"593995556666", nil},
		{"593990000000", nil},
	}
	for _, tc := range cases {
		conv, err := whatsapp_services.EnsureConversation(tenantdb.ForTenant(db, f.own), f.own, tc.phone, nil)
		if err != nil {
			t.Fatal(err)
		}
		if (tc.want == nil) != (conv.PatientID == nil) || (tc.want != nil && *tc.want != *conv.PatientID) {
			t.Fatalf("%s: patient = %v, want %v", tc.phone, conv.PatientID, tc.want)
		}
	}
	// Ensuring again returns the same conversation.
	again, _ := whatsapp_services.EnsureConversation(tenantdb.ForTenant(db, f.own), f.own, "593991112222", nil)
	if len(f.conversations(f.own)) != 4 || again.PatientID == nil {
		t.Fatalf("conversations = %d", len(f.conversations(f.own)))
	}
}

func (f *fixture) seedConversation(tenantID uint, phone string, lastInbound *time.Time, unread int) whatsapp_models.WhatsAppConversation {
	f.t.Helper()
	at := time.Now().Add(-time.Hour)
	if lastInbound != nil {
		at = *lastInbound
	}
	c := whatsapp_models.WhatsAppConversation{TenantID: tenantID, Phone: phone, LastInboundAt: lastInbound, UnreadCount: unread,
		LastMessageAt: &at, LastMessagePreview: "hola", LastDirection: whatsapp_models.DirectionInbound}
	if err := f.db.Create(&c).Error; err != nil {
		f.t.Fatal(err)
	}
	return c
}

func TestSendReply_WindowAndDelivery(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	recent := time.Now().Add(-2 * time.Hour)
	stale := time.Now().Add(-25 * time.Hour)
	open := f.seedConversation(f.own, "593991234567", &recent, 3)
	closed := f.seedConversation(f.own, "593990000001", &stale, 0)
	never := f.seedConversation(f.own, "593990000002", nil, 0)

	reply := func(id uint, body string) envelope.Response {
		return f.callID(f.h.SendReply, f.own, http.MethodPost, id, map[string]string{"body": body}, "")
	}
	for _, c := range []whatsapp_models.WhatsAppConversation{closed, never} {
		if resp := reply(c.ID, "hola"); resp.Code != http.StatusConflict || errCode(resp) != "E-WA-017" {
			t.Fatalf("closed window: %d %s", resp.Code, errCode(resp))
		}
	}
	if resp := reply(open.ID, "   "); resp.Code != http.StatusBadRequest {
		t.Fatalf("empty: %d", resp.Code)
	}
	if resp := reply(open.ID, strings.Repeat("ñ", 4097)); resp.Code != http.StatusBadRequest || errCode(resp) != "E-WA-016" {
		t.Fatalf("too long: %d %s", resp.Code, errCode(resp))
	}
	if f.graph.called("POST /v23.0/PHONE-1/messages") {
		t.Fatal("nothing must be sent for rejected replies")
	}

	resp := reply(open.ID, strings.Repeat("a", 4096))
	if resp.Code != http.StatusOK {
		t.Fatalf("reply: %d %v", resp.Code, resp.Data)
	}
	out := resp.Data.(whatsapp_dto.MessageResponse)
	if out.Kind != whatsapp_models.KindReply || out.Status != whatsapp_models.MessageStatusSent || out.Direction != whatsapp_models.DirectionOutbound ||
		out.SentByUserID == nil || *out.SentByUserID != inboxUserID || out.ConversationID == nil || *out.ConversationID != open.ID {
		t.Fatalf("reply = %+v", out)
	}
	sent := f.graph.bodies["POST /v23.0/PHONE-1/messages"]
	if !strings.Contains(sent, `"type":"text"`) || !strings.Contains(sent, `"to":"593991234567"`) || !strings.Contains(sent, `"body":"aaa`) {
		t.Fatalf("graph body = %s", sent)
	}
	stored := f.message(out.ID)
	if stored.WamID != "wamid.TEST" || stored.SentByUserID == nil || *stored.SentByUserID != inboxUserID {
		t.Fatalf("stored = %+v", stored)
	}
	c := f.conversations(f.own)[0]
	if c.UnreadCount != 0 || c.LastDirection != whatsapp_models.DirectionOutbound {
		t.Fatalf("conversation after reply = %+v", c)
	}

	// Meta rejects it: 502 and a failed row in the thread.
	f.graph.set("POST /v23.0/PHONE-1/messages", 400, `{"error":{"code":131047,"message":"Re-engagement message"}}`)
	if resp := reply(open.ID, "¿sigue ahí?"); resp.Code != http.StatusBadGateway {
		t.Fatalf("rejected: %d", resp.Code)
	}
	msgs := f.thread(open.ID)
	last := msgs[len(msgs)-1]
	if last.Status != whatsapp_models.MessageStatusFailed || last.ErrorCode != "131047" || last.Body != "¿sigue ahí?" {
		t.Fatalf("failed reply = %+v", last)
	}
}

func TestSendReply_NotConnected(t *testing.T) {
	f := newFixture(t)
	recent := time.Now().Add(-time.Hour)
	c := f.seedConversation(f.own, "593991234567", &recent, 0)
	if resp := f.callID(f.h.SendReply, f.own, http.MethodPost, c.ID, map[string]string{"body": "hola"}, ""); resp.Code != http.StatusNotFound || errCode(resp) != "E-WA-001" {
		t.Fatalf("not connected: %d %s", resp.Code, errCode(resp))
	}
}

func TestConversationEndpoints_TenantIsolation(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	f.connectedAccount(f.other, "PHONE-2")
	recent := time.Now().Add(-time.Hour)
	theirs := f.seedConversation(f.other, "593991234567", &recent, 2)
	mine := f.patient(f.own, "Ana", "")

	for name, call := range map[string]func() envelope.Response{
		"get": func() envelope.Response {
			return f.callID(f.h.GetConversation, f.own, http.MethodGet, theirs.ID, nil, "")
		},
		"messages": func() envelope.Response {
			return f.callID(f.h.ListConversationMessages, f.own, http.MethodGet, theirs.ID, nil, "")
		},
		"reply": func() envelope.Response {
			return f.callID(f.h.SendReply, f.own, http.MethodPost, theirs.ID, map[string]string{"body": "x"}, "")
		},
		"read": func() envelope.Response {
			return f.callID(f.h.MarkConversationRead, f.own, http.MethodPost, theirs.ID, nil, "")
		},
		"link": func() envelope.Response {
			return f.callID(f.h.LinkPatient, f.own, http.MethodPut, theirs.ID, map[string]uint{"patient_id": mine.ID}, "")
		},
	} {
		if resp := call(); resp.Code != http.StatusNotFound || errCode(resp) != "E-WA-015" {
			t.Errorf("%s: %d %s, want 404 E-WA-015", name, resp.Code, errCode(resp))
		}
	}
	if f.graph.called("POST /v23.0/PHONE-2/messages") || f.graph.called("POST /v23.0/PHONE-1/messages") {
		t.Fatal("a reply was sent to another tenant's conversation")
	}
	list := f.callID(f.h.ListConversations, f.own, http.MethodGet, 0, nil, "")
	if list.Data.(envelope.PagedData).Total != 0 {
		t.Fatal("list leaked another tenant's conversation")
	}
	unread := f.callID(f.h.UnreadCount, f.own, http.MethodGet, 0, nil, "")
	if u := unread.Data.(whatsapp_dto.UnreadCountResponse); u.UnreadCount != 0 {
		t.Fatalf("unread leaked: %+v", u)
	}
	if f.conversations(f.other)[0].UnreadCount != 2 || f.conversations(f.other)[0].PatientID != nil {
		t.Fatal("another tenant's conversation was modified")
	}
}

func TestConversations_ListReadAndUnreadCount(t *testing.T) {
	f := newFixture(t)
	ana := f.patient(f.own, "Ana", "0991112222")
	f.writes("PHONE-OWN", "593991112222", "wamid.1", "hola") // no account yet: ignored
	f.connectedAccount(f.own, "PHONE-OWN")
	f.writes("PHONE-OWN", "593991112222", "wamid.2", "hola")
	f.writes("PHONE-OWN", "593991112222", "wamid.3", "¿hay turno?")
	f.writes("PHONE-OWN", "593998887777", "wamid.4", "buenas") // unknown number, most recent
	old := time.Now().Add(-48 * time.Hour)
	f.seedConversation(f.own, "593990000009", &old, 0)

	list := func(query string) []whatsapp_dto.ConversationResponse {
		resp := f.callID(f.h.ListConversations, f.own, http.MethodGet, 0, nil, query)
		if resp.Code != http.StatusOK {
			t.Fatalf("list %q: %d", query, resp.Code)
		}
		return resp.Data.(envelope.PagedData).Items.([]whatsapp_dto.ConversationResponse)
	}
	all := list("")
	if len(all) != 3 || all[2].Phone != "593990000009" || all[2].WindowOpen || all[2].WindowExpiresAt == nil {
		t.Fatalf("list = %+v", all)
	}
	var anaRow whatsapp_dto.ConversationResponse
	for _, r := range all {
		if r.PatientID != nil && *r.PatientID == ana.ID {
			anaRow = r
		}
	}
	if anaRow.PatientName != "Ana Mora" || anaRow.UnreadCount != 2 || !anaRow.WindowOpen || anaRow.LastMessagePreview != "¿hay turno?" {
		t.Fatalf("ana = %+v", anaRow)
	}
	if rows := list("search=ana"); len(rows) != 1 || rows[0].ID != anaRow.ID {
		t.Fatalf("search by name = %+v", rows)
	}
	if rows := list("search=099%20888"); len(rows) != 1 || rows[0].Phone != "593998887777" {
		t.Fatalf("search by phone = %+v", rows)
	}
	if rows := list("unread=true"); len(rows) != 2 {
		t.Fatalf("unread filter = %d", len(rows))
	}

	unread := f.callID(f.h.UnreadCount, f.own, http.MethodGet, 0, nil, "").Data.(whatsapp_dto.UnreadCountResponse)
	if unread.UnreadCount != 3 || unread.Conversations != 2 {
		t.Fatalf("unread = %+v", unread)
	}
	read := f.callID(f.h.MarkConversationRead, f.own, http.MethodPost, anaRow.ID, nil, "")
	if read.Code != http.StatusOK || read.Data.(whatsapp_dto.ConversationResponse).UnreadCount != 0 {
		t.Fatalf("read = %d %+v", read.Code, read.Data)
	}
	unread = f.callID(f.h.UnreadCount, f.own, http.MethodGet, 0, nil, "").Data.(whatsapp_dto.UnreadCountResponse)
	if unread.UnreadCount != 1 || unread.Conversations != 1 {
		t.Fatalf("unread after read = %+v", unread)
	}
}

func TestConversation_ThreadPagination(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-OWN")
	for i := 0; i < 5; i++ {
		f.writes("PHONE-OWN", "593991112222", fmt.Sprintf("wamid.p%d", i), fmt.Sprintf("m%d", i))
	}
	conv := f.conversations(f.own)[0]
	page := f.callID(f.h.ListConversationMessages, f.own, http.MethodGet, conv.ID, nil, "limit=3").Data.(whatsapp_dto.ThreadResponse)
	if !page.HasMore || len(page.Items) != 3 || page.Items[0].Body != "m2" || page.Items[2].Body != "m4" {
		t.Fatalf("page 1 = %+v", page)
	}
	older := f.callID(f.h.ListConversationMessages, f.own, http.MethodGet, conv.ID, nil, fmt.Sprintf("limit=3&before=%d", page.Items[0].ID)).Data.(whatsapp_dto.ThreadResponse)
	if older.HasMore || len(older.Items) != 2 || older.Items[0].Body != "m0" || older.Items[1].Body != "m1" {
		t.Fatalf("page 2 = %+v", older)
	}
}

func TestConversation_DetailAndLinkPatient(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-OWN")
	f.writes("PHONE-OWN", "593998887777", "wamid.x", "hola, soy Carla")
	conv := f.conversations(f.own)[0]

	detail := f.callID(f.h.GetConversation, f.own, http.MethodGet, conv.ID, nil, "").Data.(whatsapp_dto.ConversationDetailResponse)
	if detail.Patient != nil || detail.NextAppointment != nil || detail.Phone != "593998887777" || !detail.WindowOpen {
		t.Fatalf("detail = %+v", detail)
	}

	foreign := f.patient(f.other, "Ajena", "")
	if resp := f.callID(f.h.LinkPatient, f.own, http.MethodPut, conv.ID, map[string]uint{"patient_id": foreign.ID}, ""); resp.Code != http.StatusNotFound || errCode(resp) != "E-WA-018" {
		t.Fatalf("another tenant's patient: %d %s", resp.Code, errCode(resp))
	}

	carla := f.patient(f.own, "Carla", "0998887777")
	carla.WhatsAppOptIn = false
	loc := whatsapp_services.ClinicLocation()
	tomorrow := time.Now().In(loc).AddDate(0, 0, 1)
	day := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, loc).UTC()
	past := clinical_models.Appointment{TenantID: f.own, PatientID: carla.ID, Title: "Pasada", Date: day.AddDate(0, 0, -3), StartTime: "09:00", EndTime: "09:30", Status: "scheduled"}
	next := clinical_models.Appointment{TenantID: f.own, PatientID: carla.ID, Title: "Control", Date: day, StartTime: "11:00", EndTime: "11:30", Status: "confirmed"}
	cancelled := clinical_models.Appointment{TenantID: f.own, PatientID: carla.ID, Title: "X", Date: day, StartTime: "08:00", EndTime: "08:30", Status: "cancelled"}
	for _, a := range []*clinical_models.Appointment{&past, &next, &cancelled} {
		if err := f.db.Create(a).Error; err != nil {
			t.Fatal(err)
		}
	}

	resp := f.callID(f.h.LinkPatient, f.own, http.MethodPut, conv.ID, map[string]uint{"patient_id": carla.ID}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("link: %d", resp.Code)
	}
	d := resp.Data.(whatsapp_dto.ConversationDetailResponse)
	if d.Patient == nil || d.Patient.ID != carla.ID || d.PatientName != "Carla Mora" || d.Patient.WhatsAppOptIn ||
		d.NextAppointment == nil || d.NextAppointment.ID != next.ID {
		t.Fatalf("linked detail = %+v (next %+v)", d, d.NextAppointment)
	}
	if m := f.thread(conv.ID)[0]; m.PatientID == nil || *m.PatientID != carla.ID {
		t.Fatalf("message not linked: %+v", m)
	}
}

// The test message is part of the thread of the number it was sent to, and
// the sent log does not list what patients wrote.
func TestSendTest_RecordedInThreadAndLogShowsOnlySent(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	resp, _ := f.call(f.h.SendTest, f.own, http.MethodPost, map[string]string{"phone": "0991234567"}, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("send test: %d", resp.Code)
	}
	convs := f.conversations(f.own)
	if len(convs) != 1 || convs[0].Phone != "593991234567" {
		t.Fatalf("conversations = %+v", convs)
	}
	msgs := f.thread(convs[0].ID)
	if len(msgs) != 1 || !strings.HasPrefix(msgs[0].Body, "Hola Paciente de prueba, te recordamos tu cita en Clinic 0") {
		t.Fatalf("thread = %+v", msgs)
	}

	f.writes("PHONE-1", "593991234567", "wamid.reply", "ok")
	list, _ := f.call(f.h.ListMessages, f.own, http.MethodGet, nil, "")
	if total := list.Data.(envelope.PagedData).Total; total != 1 {
		t.Fatalf("sent log total = %d, want 1 (inbound excluded)", total)
	}
}
