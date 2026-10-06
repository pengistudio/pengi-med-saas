package whatsapp_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_dto "pengi-med-saas/features/whatsapp/dto"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"
)

const testAppSecret = "test-app-secret"

// fakeGraph is a scripted Graph API: responses by "METHOD path", requests recorded.
type fakeGraph struct {
	mu        sync.Mutex
	responses map[string]fakeResp
	calls     []string
	bodies    map[string]string   // last body per key
	allBodies map[string][]string // every body per key
}

type fakeResp struct {
	status int
	body   string
}

func newFakeGraph() *fakeGraph {
	return &fakeGraph{bodies: map[string]string{}, allBodies: map[string][]string{}, responses: map[string]fakeResp{
		"GET /v23.0/PHONE-1":                   {200, `{"id":"PHONE-1","display_phone_number":"+1 555-0100","verified_name":"Clínica Norte"}`},
		"POST /v23.0/WABA-1/subscribed_apps":   {200, `{"success":true}`},
		"GET /v23.0/WABA-1/message_templates":  {200, `{"data":[]}`},
		"POST /v23.0/WABA-1/message_templates": {200, `{"id":"TPL-1","status":"PENDING","category":"UTILITY"}`},
		"POST /v23.0/PHONE-1/register":         {200, `{"success":true}`},
		"POST /v23.0/PHONE-1/messages":         {200, `{"messages":[{"id":"wamid.TEST"}]}`},
		"GET /v23.0/oauth/access_token":        {200, `{"access_token":"embedded-token"}`},
	}}
}

func (g *fakeGraph) set(key string, status int, body string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.responses[key] = fakeResp{status, body}
}

// bodyWith returns the first body sent to key containing sub ("" if none).
func (g *fakeGraph) bodyWith(key, sub string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, b := range g.allBodies[key] {
		if strings.Contains(b, sub) {
			return b
		}
	}
	return ""
}

// catalogListing is a message_templates listing with every catalog template in status.
func catalogListing(status string) string {
	names := []string{whatsapp_models.ReminderTemplate, whatsapp_models.TemplateContinueConversation,
		whatsapp_models.TemplateResultsReady, whatsapp_models.TemplateReschedule, whatsapp_models.TemplateFollowUp}
	items := make([]string, 0, len(names))
	for i, n := range names {
		items = append(items, fmt.Sprintf(`{"id":"T%d","name":%q,"status":%q,"language":"es"}`, i, n, status))
	}
	return `{"data":[` + strings.Join(items, ",") + `]}`
}

func (g *fakeGraph) called(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, c := range g.calls {
		if c == key {
			return true
		}
	}
	return false
}

func (g *fakeGraph) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		g.mu.Lock()
		g.calls = append(g.calls, key)
		g.bodies[key] = string(raw)
		g.allBodies[key] = append(g.allBodies[key], string(raw))
		resp, ok := g.responses[key]
		g.mu.Unlock()
		if !ok {
			resp = fakeResp{404, `{"error":{"code":100,"message":"unknown path"}}`}
		}
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

type fixture struct {
	t          *testing.T
	db         *gorm.DB
	graph      *fakeGraph
	h          *WhatsAppHandler
	webhook    *WebhookHandler
	calendar   *fakeCalendar
	own, other uint
}

type fakeCalendar struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeCalendar) SyncStatusChange(tenantID uint, a *clinical_models.Appointment) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("%d:%d:%s", tenantID, a.ID, a.Status))
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppMessage{}, &notifications_models.Notification{},
		&company_models.Company{}, &user_models.Environment{}, &whatsapp_models.WhatsAppConversation{},
		&permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{}, &company_models.Plan{},
		&company_models.Subscription{}, &whatsapp_models.WhatsAppTemplate{})
	g := newFakeGraph()
	client := whatsapp.New(g.server(t).URL, "v23.0", "app-id", testAppSecret)
	f := &fixture{t: t, db: db, graph: g, calendar: &fakeCalendar{}}
	f.h = NewWhatsAppHandler(db, zap.NewNop(), client)
	f.webhook = NewWebhookHandler(db, zap.NewNop(), f.calendar).WithSecrets(testAppSecret, "verify-me").WithClient(client)

	now := time.Now().UnixNano()
	for i, id := range []*uint{&f.own, &f.other} {
		slug := fmt.Sprintf("wa-h-%d-%d", i, now)
		tenant := tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		*id = tenant.ID
		company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "P", TenantID: tenant.ID}
		if err := db.Create(&company).Error; err != nil {
			t.Fatal(err)
		}
		for _, uid := range []uint{uint(100 + i*10), uint(101 + i*10)} {
			if err := db.Create(&user_models.Environment{UserID: uid, CompanyID: company.ID}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	return f
}

func (f *fixture) call(action envelope.Action, tenantID uint, method string, body any, query string) (envelope.Response, *httptest.ResponseRecorder) {
	c, w := testutils.NewGinContext(tenantID, 1)
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	c.Request = httptest.NewRequest(method, "/?"+query, bytes.NewReader(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	return action(c), w
}

func (f *fixture) connectManual(tenantID uint) envelope.Response {
	resp, _ := f.call(f.h.ConnectManual, tenantID, http.MethodPost, map[string]string{
		"waba_id": "WABA-1", "phone_number_id": "PHONE-1", "access_token": " EAAG-plain-token ",
	}, "")
	return resp
}

func (f *fixture) account(tenantID uint) whatsapp_models.WhatsAppAccount {
	var a whatsapp_models.WhatsAppAccount
	f.db.Where("tenant_id = ?", tenantID).Limit(1).Find(&a)
	return a
}

func TestConnectManual_SealsTokenAndCreatesTemplate(t *testing.T) {
	f := newFixture(t)
	resp := f.connectManual(f.own)
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d (%v)", resp.Code, resp.Data)
	}
	data := resp.Data.(whatsapp_dto.AccountResponse)
	if !data.Connected || data.DisplayPhone != "+1 555-0100" || data.TemplateStatus != "PENDING" || data.Mode != whatsapp_models.ModeManual ||
		!data.RemindersEnabled || len(data.ReminderOffsets) != 1 || data.ReminderOffsets[0] != 24 {
		t.Fatalf("account response = %+v", data)
	}
	raw, _ := json.Marshal(resp.Data)
	if strings.Contains(string(raw), "EAAG") || strings.Contains(string(raw), "token") {
		t.Fatalf("response leaks the token: %s", raw)
	}

	stored := f.account(f.own)
	if strings.Contains(stored.AccessTokenEncrypted, "EAAG") || stored.AccessTokenEncrypted == "" {
		t.Fatal("token must be stored sealed")
	}
	if tok, err := stored.OpenAccessToken(); err != nil || tok != "EAAG-plain-token" {
		t.Fatalf("stored token = %q, %v (want trimmed)", tok, err)
	}
	if !f.graph.called("POST /v23.0/WABA-1/subscribed_apps") || !f.graph.called("POST /v23.0/WABA-1/message_templates") {
		t.Fatalf("graph calls = %v", f.graph.calls)
	}
	tpl := f.graph.bodyWith("POST /v23.0/WABA-1/message_templates", `"name":"pengi_cita_recordatorio"`)
	if !strings.Contains(tpl, `"category":"UTILITY"`) || !strings.Contains(tpl, `"QUICK_REPLY"`) {
		t.Fatalf("template body = %s", tpl)
	}

	// GET /account returns the same, still without the token.
	get, _ := f.call(f.h.GetAccount, f.own, http.MethodGet, nil, "")
	if acc := get.Data.(whatsapp_dto.AccountResponse); !acc.Connected || acc.PhoneNumberID != "PHONE-1" {
		t.Fatalf("get = %+v", acc)
	}
	// The other tenant sees nothing.
	other, _ := f.call(f.h.GetAccount, f.other, http.MethodGet, nil, "")
	if other.Data.(whatsapp_dto.AccountResponse).Connected {
		t.Fatal("other tenant must not see the account")
	}
}

func TestConnectManual_ExistingTemplateIsNotRecreated(t *testing.T) {
	f := newFixture(t)
	f.graph.set("GET /v23.0/WABA-1/message_templates", 200, catalogListing("APPROVED"))
	resp := f.connectManual(f.own)
	if resp.Data.(whatsapp_dto.AccountResponse).TemplateStatus != "APPROVED" || f.graph.called("POST /v23.0/WABA-1/message_templates") {
		t.Fatalf("resp = %+v calls = %v", resp.Data, f.graph.calls)
	}
}

func TestConnectManual_Errors(t *testing.T) {
	f := newFixture(t)
	f.graph.set("GET /v23.0/PHONE-1", 401, `{"error":{"code":190,"message":"Invalid OAuth access token"}}`)
	if resp := f.connectManual(f.own); resp.Code != http.StatusBadRequest {
		t.Fatalf("bad token: code = %d", resp.Code)
	}
	if f.account(f.own).ID != 0 {
		t.Fatal("nothing must be stored on failure")
	}

	f.graph.set("GET /v23.0/PHONE-1", 200, `{"id":"PHONE-1"}`)
	if resp := f.connectManual(f.own); resp.Code != http.StatusOK {
		t.Fatalf("connect: %d", resp.Code)
	}
	if resp := f.connectManual(f.other); resp.Code != http.StatusConflict {
		t.Fatalf("same number on another tenant: code = %d", resp.Code)
	}
	// Reconnecting the same tenant updates in place.
	if resp := f.connectManual(f.own); resp.Code != http.StatusOK {
		t.Fatalf("reconnect: %d", resp.Code)
	}
	var count int64
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Count(&count)
	if count != 1 {
		t.Fatalf("accounts = %d", count)
	}

	if resp, _ := f.call(f.h.ConnectManual, f.own, http.MethodPost, map[string]string{"waba_id": "W"}, ""); resp.Code != http.StatusBadRequest {
		t.Fatalf("missing fields: %d", resp.Code)
	}
}

func TestConnectEmbedded(t *testing.T) {
	f := newFixture(t)
	body := map[string]string{"code": "the-code", "waba_id": "WABA-1", "phone_number_id": "PHONE-1"}

	t.Setenv("META_ES_CONFIG_ID", "")
	if resp, _ := f.call(f.h.ConnectEmbedded, f.own, http.MethodPost, body, ""); resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("without config id: %d", resp.Code)
	}

	t.Setenv("META_ES_CONFIG_ID", "cfg")
	resp, _ := f.call(f.h.ConnectEmbedded, f.own, http.MethodPost, body, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d (%v)", resp.Code, resp.Data)
	}
	if !f.graph.called("GET /v23.0/oauth/access_token") || !f.graph.called("POST /v23.0/PHONE-1/register") {
		t.Fatalf("calls = %v", f.graph.calls)
	}
	reg := f.graph.bodies["POST /v23.0/PHONE-1/register"]
	if !strings.Contains(reg, `"messaging_product":"whatsapp"`) || !strings.Contains(reg, `"pin":"`) {
		t.Fatalf("register body = %s", reg)
	}
	a := f.account(f.own)
	if a.Mode != whatsapp_models.ModeEmbedded || a.PinEncrypted == "" {
		t.Fatalf("account = %+v", a)
	}
	if tok, _ := a.OpenAccessToken(); tok != "embedded-token" {
		t.Fatalf("token = %q", tok)
	}
}

func TestUpdateSettings(t *testing.T) {
	f := newFixture(t)
	settings := func(enabled bool, offsets []int) envelope.Response {
		resp, _ := f.call(f.h.UpdateSettings, f.own, http.MethodPut, map[string]any{"reminders_enabled": enabled, "reminder_offsets": offsets}, "")
		return resp
	}
	if resp := settings(true, []int{24}); resp.Code != http.StatusNotFound {
		t.Fatalf("not connected: %d", resp.Code)
	}
	f.connectManual(f.own)
	for _, bad := range [][]int{{0}, {169}, {1, 2, 3, 4}, {}} {
		if resp := settings(true, bad); resp.Code != http.StatusBadRequest {
			t.Fatalf("offsets %v: code = %d", bad, resp.Code)
		}
	}
	resp := settings(true, []int{2, 48, 24})
	if resp.Code != http.StatusOK {
		t.Fatalf("code = %d", resp.Code)
	}
	a := f.account(f.own)
	if !a.RemindersEnabled || len(a.ReminderOffsets) != 3 || a.ReminderOffsets[0] != 48 {
		t.Fatalf("stored = %+v", a.ReminderOffsets)
	}
	if resp := settings(false, nil); resp.Code != http.StatusOK || f.account(f.own).RemindersEnabled {
		t.Fatalf("disable: %d", resp.Code)
	}
}

func TestSendTestAndMessageLog(t *testing.T) {
	f := newFixture(t)
	f.connectManual(f.own)
	send := func(phone string) envelope.Response {
		resp, _ := f.call(f.h.SendTest, f.own, http.MethodPost, map[string]string{"phone": phone}, "")
		return resp
	}
	if resp := send("0991234567"); resp.Code != http.StatusConflict {
		t.Fatalf("template pending: %d", resp.Code)
	}
	f.db.Model(&whatsapp_models.WhatsAppAccount{}).Where("tenant_id = ?", f.own).Update("template_status", "APPROVED")
	if resp := send("123"); resp.Code != http.StatusBadRequest {
		t.Fatalf("bad phone: %d", resp.Code)
	}
	resp := send("0991234567")
	if resp.Code != http.StatusOK {
		t.Fatalf("send: %d %v", resp.Code, resp.Data)
	}
	msg := resp.Data.(whatsapp_dto.MessageResponse)
	if msg.Status != whatsapp_models.MessageStatusSent || msg.Kind != whatsapp_models.KindTest || msg.ToPhone != "593991234567" {
		t.Fatalf("message = %+v", msg)
	}
	sent := f.graph.bodies["POST /v23.0/PHONE-1/messages"]
	if !strings.Contains(sent, `"to":"593991234567"`) || !strings.Contains(sent, `"name":"pengi_cita_recordatorio"`) {
		t.Fatalf("sent = %s", sent)
	}

	f.graph.set("POST /v23.0/PHONE-1/messages", 400, `{"error":{"code":131026,"message":"undeliverable"}}`)
	if resp := send("0991234567"); resp.Code != http.StatusBadGateway {
		t.Fatalf("failed send: %d", resp.Code)
	}

	list, _ := f.call(f.h.ListMessages, f.own, http.MethodGet, nil, "page=1&limit=10")
	paged := list.Data.(envelope.PagedData)
	items := paged.Items.([]whatsapp_dto.MessageResponse)
	if paged.Total != 2 || len(items) != 2 || items[0].Status != whatsapp_models.MessageStatusFailed || items[0].ErrorCode != "131026" {
		t.Fatalf("log = %+v", paged)
	}
	failed, _ := f.call(f.h.ListMessages, f.own, http.MethodGet, nil, "status=sent")
	if failed.Data.(envelope.PagedData).Total != 1 {
		t.Fatal("status filter")
	}
	other, _ := f.call(f.h.ListMessages, f.other, http.MethodGet, nil, "")
	if other.Data.(envelope.PagedData).Total != 0 {
		t.Fatal("other tenant must not see the log")
	}
}

func TestDisconnect(t *testing.T) {
	f := newFixture(t)
	if resp, _ := f.call(f.h.Disconnect, f.own, http.MethodDelete, nil, ""); resp.Code != http.StatusNotFound {
		t.Fatalf("nothing to disconnect: %d", resp.Code)
	}
	f.connectManual(f.own)
	if resp, _ := f.call(f.h.Disconnect, f.own, http.MethodDelete, nil, ""); resp.Code != http.StatusOK {
		t.Fatalf("disconnect: %d", resp.Code)
	}
	var count int64
	f.db.Unscoped().Model(&whatsapp_models.WhatsAppAccount{}).Count(&count)
	if count != 0 {
		t.Fatal("account must be hard-deleted so the number can reconnect")
	}
	// The number is free again, for another tenant too.
	if resp := f.connectManual(f.other); resp.Code != http.StatusOK {
		t.Fatalf("reconnect elsewhere: %d", resp.Code)
	}
}

func TestGetConfig(t *testing.T) {
	f := newFixture(t)
	t.Setenv("META_APP_ID", "app-id")
	t.Setenv("META_ES_CONFIG_ID", "cfg")
	resp, _ := f.call(f.h.GetConfig, f.own, http.MethodGet, nil, "")
	cfg := resp.Data.(whatsapp_dto.ConfigResponse)
	if cfg.AppID != "app-id" || cfg.ConfigID != "cfg" || !cfg.EmbeddedSignupAvailable || cfg.GraphVersion != "v23.0" {
		t.Fatalf("config = %+v", cfg)
	}
	t.Setenv("META_ES_CONFIG_ID", "")
	resp, _ = f.call(f.h.GetConfig, f.own, http.MethodGet, nil, "")
	if resp.Data.(whatsapp_dto.ConfigResponse).EmbeddedSignupAvailable {
		t.Fatal("embedded signup needs a config id")
	}
}
