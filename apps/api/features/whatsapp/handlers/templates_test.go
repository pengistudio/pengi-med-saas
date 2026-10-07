package whatsapp_handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm/clause"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"
)

func (f *fixture) templateRows(tenantID uint) map[string]whatsapp_models.WhatsAppTemplate {
	f.t.Helper()
	rows, err := whatsapp_services.TemplateRows(tenantdb.ForTenant(f.db, tenantID))
	if err != nil {
		f.t.Fatal(err)
	}
	return rows
}

func (f *fixture) approveTemplates(tenantID uint, names ...string) {
	f.t.Helper()
	for _, n := range names {
		if err := whatsapp_services.SetTemplateStatus(tenantdb.ForTenant(f.db, tenantID), tenantID, n, "APPROVED", ""); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) consentingPatient(tenantID uint, first, phone string) clinical_models.Patient {
	f.t.Helper()
	p := f.patient(tenantID, first, phone)
	f.db.Model(&p).Update("whatsapp_opt_in", true)
	p.WhatsAppOptIn = true
	return p
}

// planLimit gives tenantID an active subscription whose plan caps WhatsApp
// messages at limit.
func (f *fixture) planLimit(tenantID uint, limit int) {
	f.t.Helper()
	var company company_models.Company
	if err := f.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	code := fmt.Sprintf("WA-%d-%d", tenantID, time.Now().UnixNano())
	plan := company_models.Plan{Name: code, Code: code, Properties: datatypes.JSONMap{whatsapp_services.PlanLimitKey: limit}}
	if err := f.db.Create(&plan).Error; err != nil {
		f.t.Fatal(err)
	}
	sub := company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().AddDate(0, 1, 0), CompanyID: company.ID}
	if err := f.db.Omit(clause.Associations).Create(&sub).Error; err != nil {
		f.t.Fatal(err)
	}
}

// grant gives userIDs of tenantID's company a role holding permissionIDs.
func (f *fixture) grant(tenantID uint, userIDs []uint, permissionIDs ...string) {
	f.t.Helper()
	var company company_models.Company
	if err := f.db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	role := user_models.Role{Role: fmt.Sprintf("r-%d", time.Now().UnixNano())}
	for _, id := range permissionIDs {
		perm := permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id}
		if err := f.db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			f.t.Fatal(err)
		}
		role.Permissions = append(role.Permissions, perm)
	}
	if err := f.db.Create(&role).Error; err != nil {
		f.t.Fatal(err)
	}
	for _, uid := range userIDs {
		var env user_models.Environment
		f.db.Where("user_id = ? AND company_id = ?", uid, company.ID).Limit(1).Find(&env)
		if env.ID == 0 {
			env = user_models.Environment{UserID: uid, CompanyID: company.ID}
		}
		env.RoleID = role.ID
		if err := f.db.Save(&env).Error; err != nil {
			f.t.Fatal(err)
		}
	}
}

func templateEvent(wabaID, name, status string) map[string]any {
	return map[string]any{"object": "whatsapp_business_account", "entry": []any{map[string]any{
		"id": wabaID, "changes": []any{map[string]any{"field": "message_template_status_update", "value": map[string]any{
			"event": status, "message_template_id": 123, "message_template_name": name, "message_template_language": "es", "reason": "NONE",
		}}},
	}}}
}

func TestTemplates_ConnectProvisionsCatalogAndListsIt(t *testing.T) {
	f := newFixture(t)
	if resp := f.connectManual(f.own); resp.Code != http.StatusOK {
		t.Fatalf("connect: %d", resp.Code)
	}
	if n := len(f.graph.allBodies["POST /v23.0/WABA-1/message_templates"]); n != len(whatsapp_services.Catalog) {
		t.Fatalf("templates created = %d, want %d", n, len(whatsapp_services.Catalog))
	}
	for _, spec := range whatsapp_services.Catalog {
		body := f.graph.bodyWith("POST /v23.0/WABA-1/message_templates", fmt.Sprintf(`"name":%q`, spec.Definition.Name))
		if !strings.Contains(body, `"category":"UTILITY"`) || !strings.Contains(body, `"language":"es"`) {
			t.Fatalf("%s body = %s", spec.Definition.Name, body)
		}
	}
	rows := f.templateRows(f.own)
	if len(rows) != len(whatsapp_services.Catalog) || rows[whatsapp_models.TemplateResultsReady].Status != "PENDING" || rows[whatsapp_models.TemplateResultsReady].MetaID != "TPL-1" {
		t.Fatalf("rows = %+v", rows)
	}
	if len(f.templateRows(f.other)) != 0 {
		t.Fatal("other tenant got template rows")
	}

	f.approveTemplates(f.own, whatsapp_models.TemplateFollowUp, whatsapp_models.ReminderTemplate)
	resp, _ := f.call(f.h.ListTemplates, f.own, http.MethodGet, nil, "")
	items := resp.Data.([]whatsapp_dto.TemplateResponse)
	if len(items) != len(whatsapp_services.Catalog) {
		t.Fatalf("items = %+v", items)
	}
	byName := map[string]whatsapp_dto.TemplateResponse{}
	for _, it := range items {
		byName[it.Name] = it
	}
	if r := byName[whatsapp_models.ReminderTemplate]; r.Inbox || r.Usable || len(r.Buttons) != 2 {
		t.Fatalf("reminder = %+v", r)
	}
	if r := byName[whatsapp_models.TemplateFollowUp]; !r.Usable || !strings.Contains(r.Preview, "María Pérez") || strings.Contains(r.Preview, "{{") {
		t.Fatalf("follow-up = %+v", r)
	}
	if r := byName[whatsapp_models.TemplateReschedule]; r.Usable || !r.NeedsAppointment || len(r.Variables) != 4 || r.Status != "PENDING" {
		t.Fatalf("reschedule = %+v", r)
	}

	// Disconnecting drops the rows (they belong to the WABA).
	f.call(f.h.Disconnect, f.own, http.MethodDelete, nil, "")
	if len(f.templateRows(f.own)) != 0 {
		t.Fatal("template rows must go with the account")
	}
}

func TestTemplates_SyncAndWebhookUpdatePerTemplate(t *testing.T) {
	f := newFixture(t)
	f.connectManual(f.own)
	f.graph.set("GET /v23.0/WABA-1/message_templates", 200, catalogListing("APPROVED"))
	if resp, _ := f.call(f.h.SyncTemplate, f.own, http.MethodPost, nil, ""); resp.Code != http.StatusOK {
		t.Fatalf("sync: %d", resp.Code)
	}
	for name, row := range f.templateRows(f.own) {
		if row.Status != "APPROVED" {
			t.Fatalf("%s = %s after sync", name, row.Status)
		}
	}
	if f.account(f.own).TemplateStatus != "APPROVED" {
		t.Fatal("reminder status must be mirrored on the account")
	}

	// Webhook: per template, by name; the account mirrors only the reminder.
	f.signed(templateEvent("WABA-1", whatsapp_models.TemplateResultsReady, "REJECTED"))
	rows := f.templateRows(f.own)
	if rows[whatsapp_models.TemplateResultsReady].Status != "REJECTED" || rows[whatsapp_models.TemplateResultsReady].Reason != "NONE" ||
		rows[whatsapp_models.TemplateFollowUp].Status != "APPROVED" || f.account(f.own).TemplateStatus != "APPROVED" {
		t.Fatalf("after results REJECTED: rows=%+v account=%s", rows, f.account(f.own).TemplateStatus)
	}
	f.signed(templateEvent("WABA-1", whatsapp_models.ReminderTemplate, "PAUSED"))
	if f.account(f.own).TemplateStatus != "PAUSED" || f.templateRows(f.own)[whatsapp_models.ReminderTemplate].Status != "PAUSED" {
		t.Fatal("reminder event must update row and account")
	}
	f.signed(templateEvent("WABA-1", "someone_elses_template", "APPROVED"))
	if _, ok := f.templateRows(f.own)["someone_elses_template"]; ok {
		t.Fatal("templates outside the catalog must be ignored")
	}
	f.signed(templateEvent("WABA-OTHER", whatsapp_models.TemplateFollowUp, "REJECTED"))
	if f.templateRows(f.own)[whatsapp_models.TemplateFollowUp].Status != "APPROVED" {
		t.Fatal("another WABA's event changed ours")
	}
}

func TestStartConversation_Rules(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	start := func(body map[string]any) envelope.Response {
		resp, _ := f.call(f.h.StartConversation, f.own, http.MethodPost, body, "")
		return resp
	}
	ana := f.consentingPatient(f.own, "Ana", "0991234567")
	noConsent := f.patient(f.own, "Beto", "0991112222")
	noPhone := f.consentingPatient(f.own, "Caro", "")
	foreign := f.consentingPatient(f.other, "Dani", "0993334444")
	tpl := whatsapp_models.TemplateContinueConversation

	cases := []struct {
		name string
		body map[string]any
		code int
		err  string
	}{
		{"unknown template", map[string]any{"patient_id": ana.ID, "template": "nope"}, 400, "E-WA-019"},
		{"reminder is not an inbox template", map[string]any{"patient_id": ana.ID, "template": whatsapp_models.ReminderTemplate}, 400, "E-WA-019"},
		{"other tenant's patient", map[string]any{"patient_id": foreign.ID, "template": tpl}, 404, "E-WA-018"},
		{"no phone", map[string]any{"patient_id": noPhone.ID, "template": tpl}, 400, "E-WA-010"},
		{"no consent", map[string]any{"patient_id": noConsent.ID, "template": tpl}, 409, "E-WA-020"},
		{"not approved", map[string]any{"patient_id": ana.ID, "template": tpl}, 409, "E-WA-021"},
	}
	for _, tc := range cases {
		if resp := start(tc.body); resp.Code != tc.code || errCode(resp) != tc.err {
			t.Fatalf("%s: %d %s, want %d %s", tc.name, resp.Code, errCode(resp), tc.code, tc.err)
		}
	}
	if f.graph.called("POST /v23.0/PHONE-1/messages") {
		t.Fatal("nothing must be sent for rejected requests")
	}

	f.approveTemplates(f.own, tpl, whatsapp_models.TemplateReschedule)
	resp := start(map[string]any{"patient_id": ana.ID, "template": tpl})
	if resp.Code != http.StatusOK {
		t.Fatalf("start: %d %v", resp.Code, resp.Data)
	}
	out := resp.Data.(whatsapp_dto.TemplateSendResponse)
	if out.Message.Kind != whatsapp_models.KindTemplate || out.Message.Status != whatsapp_models.MessageStatusSent ||
		out.Message.Template != tpl || !strings.Contains(out.Message.Body, "Ana Mora") || !strings.Contains(out.Message.Body, "Clinic 0") ||
		out.Conversation.PatientID == nil || *out.Conversation.PatientID != ana.ID || out.Conversation.Phone != "593991234567" {
		t.Fatalf("start = %+v", out)
	}
	sent := f.graph.bodies["POST /v23.0/PHONE-1/messages"]
	if !strings.Contains(sent, `"type":"template"`) || !strings.Contains(sent, fmt.Sprintf(`"name":%q`, tpl)) || !strings.Contains(sent, `"text":"Ana Mora"`) {
		t.Fatalf("graph body = %s", sent)
	}

	// Reprogramar needs an appointment of this patient, and takes date and time from it.
	other := f.consentingPatient(f.own, "Eva", "0995556666")
	loc := whatsapp_services.ClinicLocation()
	day := time.Date(2026, 11, 3, 0, 0, 0, 0, loc)
	appt := clinical_models.Appointment{TenantID: f.own, PatientID: ana.ID, Title: "Control", Date: day, StartTime: "09:15", EndTime: "09:45", Status: "scheduled"}
	evaAppt := clinical_models.Appointment{TenantID: f.own, PatientID: other.ID, Title: "Control", Date: day, StartTime: "11:00", EndTime: "11:30", Status: "scheduled"}
	foreignAppt := clinical_models.Appointment{TenantID: f.other, PatientID: foreign.ID, Title: "Control", Date: day, StartTime: "11:00", EndTime: "11:30", Status: "scheduled"}
	for _, a := range []*clinical_models.Appointment{&appt, &evaAppt, &foreignAppt} {
		if err := f.db.Create(a).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, apptID := range []any{nil, evaAppt.ID, foreignAppt.ID} {
		body := map[string]any{"patient_id": ana.ID, "template": whatsapp_models.TemplateReschedule}
		if apptID != nil {
			body["appointment_id"] = apptID
		}
		if resp := start(body); resp.Code != http.StatusBadRequest || errCode(resp) != "E-WA-022" {
			t.Fatalf("appointment %v: %d %s", apptID, resp.Code, errCode(resp))
		}
	}
	resp = start(map[string]any{"patient_id": ana.ID, "template": whatsapp_models.TemplateReschedule, "appointment_id": appt.ID})
	if resp.Code != http.StatusOK {
		t.Fatalf("reschedule: %d %v", resp.Code, resp.Data)
	}
	if body := resp.Data.(whatsapp_dto.TemplateSendResponse).Message.Body; !strings.Contains(body, "martes 3 de noviembre") || !strings.Contains(body, "09:15") {
		t.Fatalf("reschedule body = %s", body)
	}
	// Same conversation both times.
	if convs := f.conversations(f.own); len(convs) != 1 || len(f.thread(convs[0].ID)) != 2 {
		t.Fatalf("conversations = %+v", convs)
	}
}

func TestStartConversation_MonthlyLimit(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	f.approveTemplates(f.own, whatsapp_models.TemplateFollowUp)
	f.planLimit(f.own, 1)
	ana := f.consentingPatient(f.own, "Ana", "0991234567")
	body := map[string]any{"patient_id": ana.ID, "template": whatsapp_models.TemplateFollowUp}

	if resp, _ := f.call(f.h.StartConversation, f.own, http.MethodPost, body, ""); resp.Code != http.StatusOK {
		t.Fatalf("first: %d %v", resp.Code, resp.Data)
	}
	resp, _ := f.call(f.h.StartConversation, f.own, http.MethodPost, body, "")
	if resp.Code != http.StatusForbidden || errCode(resp) != "E-WA-023" || resp.Message != "plan.limit.whatsapp_messages" {
		t.Fatalf("over the cap: %d %s %s", resp.Code, errCode(resp), resp.Message)
	}
	// The test send is capped too.
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", f.own).Update("template_status", "APPROVED")
	if resp, _ := f.call(f.h.SendTest, f.own, http.MethodPost, map[string]string{"phone": "0991234567"}, ""); resp.Code != http.StatusForbidden || errCode(resp) != "E-WA-023" {
		t.Fatalf("test over the cap: %d %s", resp.Code, errCode(resp))
	}
	// Usage endpoint.
	usage, _ := f.call(f.h.GetUsage, f.own, http.MethodGet, nil, "")
	u := usage.Data.(whatsapp_dto.UsageResponse)
	if u.Used != 1 || u.Limit != 1 || !u.PeriodEnd.After(u.PeriodStart) || u.PeriodStart.Day() != 1 {
		t.Fatalf("usage = %+v", u)
	}
	other, _ := f.call(f.h.GetUsage, f.other, http.MethodGet, nil, "")
	if u := other.Data.(whatsapp_dto.UsageResponse); u.Used != 0 || u.Limit != -1 {
		t.Fatalf("other tenant usage = %+v", u)
	}
}

func TestSendTemplate_ClosedWindowAndConsent(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-1")
	f.approveTemplates(f.own, whatsapp_models.TemplateResultsReady)
	stale := time.Now().Add(-72 * time.Hour)
	ana := f.consentingPatient(f.own, "Ana", "0991234567")
	linked := f.seedConversation(f.own, "593991234567", &stale, 0)
	f.db.Model(&linked).Update("patient_id", ana.ID)
	unlinked := f.seedConversation(f.own, "593990000009", &stale, 0)
	foreign := f.seedConversation(f.other, "593991234567", &stale, 0)

	send := func(id uint) envelope.Response {
		return f.callID(f.h.SendTemplate, f.own, http.MethodPost, id, map[string]any{"template": whatsapp_models.TemplateResultsReady}, "")
	}
	if resp := send(unlinked.ID); resp.Code != http.StatusConflict || errCode(resp) != "E-WA-020" {
		t.Fatalf("unlinked: %d %s", resp.Code, errCode(resp))
	}
	if resp := send(foreign.ID); resp.Code != http.StatusNotFound {
		t.Fatalf("other tenant's conversation: %d", resp.Code)
	}
	resp := send(linked.ID)
	if resp.Code != http.StatusOK {
		t.Fatalf("closed window template: %d %v", resp.Code, resp.Data)
	}
	out := resp.Data.(whatsapp_dto.TemplateSendResponse)
	if out.Message.ConversationID == nil || *out.Message.ConversationID != linked.ID || out.Message.SentByUserID == nil || *out.Message.SentByUserID != inboxUserID ||
		out.Conversation.WindowOpen {
		t.Fatalf("template = %+v", out)
	}
	// Consent withdrawn: refused.
	f.db.Model(&ana).Update("whatsapp_opt_in", false)
	if resp := send(linked.ID); resp.Code != http.StatusConflict || errCode(resp) != "E-WA-020" {
		t.Fatalf("no consent: %d %s", resp.Code, errCode(resp))
	}
}

func (f *fixture) notifications(userID uint, notifType string) []notifications_models.Notification {
	var out []notifications_models.Notification
	f.db.Where("user_id = ? AND type = ?", userID, notifType).Order("id").Find(&out)
	return out
}

func notifCount(t *testing.T, n notifications_models.Notification) (int, string) {
	t.Helper()
	var p map[string]any
	if err := json.Unmarshal(n.Params, &p); err != nil {
		t.Fatal(err)
	}
	c, _ := p["count"].(float64)
	preview, _ := p["preview"].(string)
	return int(c), preview
}

func TestInboundNotifications_UpsertAndMarkRead(t *testing.T) {
	f := newFixture(t)
	f.connectedAccount(f.own, "PHONE-OWN")
	f.patient(f.own, "Ana", "0991234567")
	f.grant(f.own, []uint{100, inboxUserID}, "USE_WHATSAPP_INBOX")
	// 101 has no inbox permission; 110 is the other clinic's.
	f.grant(f.other, []uint{110}, "USE_WHATSAPP_INBOX")

	f.writes("PHONE-OWN", "593991234567", nextWamID(), "hola")
	f.writes("PHONE-OWN", "593991234567", nextWamID(), "¿tienen turno mañana?")
	conv := f.conversations(f.own)[0]

	for _, uid := range []uint{100, inboxUserID} {
		ns := f.notifications(uid, whatsapp_services.NotificationTypeMessage)
		if len(ns) != 1 {
			t.Fatalf("user %d notifications = %d, want 1 (upserted)", uid, len(ns))
		}
		n := ns[0]
		count, preview := notifCount(t, n)
		if count != 2 || preview != "¿tienen turno mañana?" || n.ResourceType != "whatsapp_conversation" || n.ResourceID != conv.ID ||
			n.MessageKey != "notification.whatsapp.message" || n.ActionURL != fmt.Sprintf("/whatsapp?c=%d", conv.ID) || n.TenantID != f.own {
			t.Fatalf("notification = %+v (count %d)", n, count)
		}
		var params map[string]any
		_ = json.Unmarshal(n.Params, &params)
		if params["name"] != "Ana Mora" {
			t.Fatalf("params = %s", n.Params)
		}
	}
	if ns := f.notifications(101, whatsapp_services.NotificationTypeMessage); len(ns) != 0 {
		t.Fatal("users without the inbox permission must not be notified")
	}
	if ns := f.notifications(110, whatsapp_services.NotificationTypeMessage); len(ns) != 0 {
		t.Fatal("the other clinic must not be notified")
	}

	// Opening the conversation reads the caller's notifications only.
	if resp := f.callID(f.h.MarkConversationRead, f.own, http.MethodPost, conv.ID, nil, ""); resp.Code != http.StatusOK {
		t.Fatalf("read: %d", resp.Code)
	}
	if n := f.notifications(inboxUserID, whatsapp_services.NotificationTypeMessage)[0]; n.ReadAt == nil {
		t.Fatal("caller's notification must be read")
	}
	if n := f.notifications(100, whatsapp_services.NotificationTypeMessage)[0]; n.ReadAt != nil {
		t.Fatal("another user's notification must stay unread")
	}
	// A new message after reading starts a new notification for the caller.
	f.writes("PHONE-OWN", "593991234567", nextWamID(), "gracias")
	ns := f.notifications(inboxUserID, whatsapp_services.NotificationTypeMessage)
	if len(ns) != 2 {
		t.Fatalf("after read + new message: %d notifications", len(ns))
	}
	if c, _ := notifCount(t, ns[1]); c != 1 {
		t.Fatalf("new notification count = %d", c)
	}
	if c, _ := notifCount(t, f.notifications(100, whatsapp_services.NotificationTypeMessage)[0]); c != 3 {
		t.Fatalf("unread notification count = %d, want 3", c)
	}
}

// consentAccount connects tenantID on phoneID with reminders active.
func (f *fixture) consentAccount(tenantID uint, phoneID string) {
	f.connectedAccount(tenantID, phoneID)
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", tenantID).Update("reminders_enabled", true)
}

func systemMessages(msgs []whatsapp_models.WhatsAppMessage) []whatsapp_models.WhatsAppMessage {
	var out []whatsapp_models.WhatsAppMessage
	for _, m := range msgs {
		if m.Kind == whatsapp_models.KindSystem {
			out = append(out, m)
		}
	}
	return out
}

func (f *fixture) loadPatient(id uint) clinical_models.Patient {
	var p clinical_models.Patient
	f.db.Where("id = ?", id).First(&p)
	return p
}

func TestConsentByWhatsApp_AskOnceThenYes(t *testing.T) {
	f := newFixture(t)
	f.consentAccount(f.own, "PHONE-1")
	ana := f.patient(f.own, "Ana", "0991234567")

	// A loose "sí" before asking is not consent; it triggers the question.
	f.writes("PHONE-1", "593991234567", nextWamID(), "Sí")
	conv := f.conversations(f.own)[0]
	if f.loadPatient(ana.ID).WhatsAppOptIn {
		t.Fatal("a sí before asking must not grant consent")
	}
	sys := systemMessages(f.thread(conv.ID))
	if len(sys) != 1 || sys[0].Body != whatsapp_services.PatientText(whatsapp_services.OptInRequestKey) ||
		sys[0].Status != whatsapp_models.MessageStatusSent || sys[0].Direction != whatsapp_models.DirectionOutbound {
		t.Fatalf("ask = %+v", sys)
	}
	if !strings.Contains(sys[0].Body, "responde SÍ") {
		t.Fatalf("ask text = %q (must come from the es catalog)", sys[0].Body)
	}
	if f.conversations(f.own)[0].OptInRequestedAt == nil {
		t.Fatal("OptInRequestedAt must be set")
	}
	sent := f.graph.bodies["POST /v23.0/PHONE-1/messages"]
	if !strings.Contains(sent, `"type":"text"`) || !strings.Contains(sent, `"to":"593991234567"`) {
		t.Fatalf("graph body = %s", sent)
	}

	// Asked only once per conversation.
	f.writes("PHONE-1", "593991234567", nextWamID(), "¿a qué hora atienden?")
	if n := len(systemMessages(f.thread(conv.ID))); n != 1 {
		t.Fatalf("system messages = %d, want 1", n)
	}

	f.writes("PHONE-1", "593991234567", nextWamID(), " si. ")
	p := f.loadPatient(ana.ID)
	if !p.WhatsAppOptIn || p.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourceWhatsApp || p.WhatsAppOptInAt == nil {
		t.Fatalf("patient after SÍ = %+v", p)
	}
	sys = systemMessages(f.thread(conv.ID))
	if len(sys) != 2 || sys[1].Body != whatsapp_services.PatientText(whatsapp_services.OptInConfirmedKey) {
		t.Fatalf("confirmation = %+v", sys)
	}
	// Another SÍ changes nothing and sends nothing.
	f.writes("PHONE-1", "593991234567", nextWamID(), "OK")
	if n := len(systemMessages(f.thread(conv.ID))); n != 2 {
		t.Fatalf("system messages after repeated yes = %d", n)
	}
	// System messages don't count toward the monthly cap.
	u, err := whatsapp_services.MonthlyUsage(f.db, f.own, time.Now())
	if err != nil || u.Used != 0 {
		t.Fatalf("usage = %+v, %v", u, err)
	}
}

func TestConsentByWhatsApp_NotAsked(t *testing.T) {
	f := newFixture(t)
	f.consentAccount(f.own, "PHONE-1")

	// After STOP.
	stopped := f.patient(f.own, "Ana", "0991234567")
	f.db.Model(&stopped).Updates(map[string]any{"whatsapp_opt_in_source": clinical_models.WhatsAppOptInSourceStop})
	f.writes("PHONE-1", "593991234567", nextWamID(), "hola")
	// Unknown number.
	f.writes("PHONE-1", "593990000777", nextWamID(), "hola")
	// Already consented.
	f.consentingPatient(f.own, "Beto", "0991112222")
	f.writes("PHONE-1", "593991112222", nextWamID(), "hola")
	for _, c := range f.conversations(f.own) {
		if sys := systemMessages(f.thread(c.ID)); len(sys) != 0 || c.OptInRequestedAt != nil {
			t.Fatalf("conversation %s was asked: %+v", c.Phone, sys)
		}
	}
	// STOP itself sets the source.
	f.writes("PHONE-1", "593991112222", nextWamID(), "STOP")
	var beto clinical_models.Patient
	f.db.Where("tenant_id = ? AND first_name = ?", f.own, "Beto").First(&beto)
	if beto.WhatsAppOptIn || beto.WhatsAppOptInSource != clinical_models.WhatsAppOptInSourceStop {
		t.Fatalf("after STOP = %+v", beto)
	}

	// Reminders disabled: not asked.
	g := newFixture(t)
	g.connectedAccount(g.own, "PHONE-1") // reminders_enabled false
	g.patient(g.own, "Caro", "0993334444")
	g.writes("PHONE-1", "593993334444", nextWamID(), "hola")
	if sys := systemMessages(g.thread(g.conversations(g.own)[0].ID)); len(sys) != 0 {
		t.Fatal("must not ask when reminders are off")
	}
}
