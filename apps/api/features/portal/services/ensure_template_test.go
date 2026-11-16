package portal_services_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/zap"

	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	portal_services "pengi-med-saas/features/portal/services"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"
)

// templateGraph lists listing as the WABA's templates and accepts creations,
// counting each request by "METHOD path".
type templateGraph struct {
	mu      sync.Mutex
	listing string
	calls   map[string]int
}

func (g *templateGraph) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	g.calls[r.Method+" "+r.URL.Path]++
	g.mu.Unlock()
	if r.Method == http.MethodGet {
		_, _ = w.Write([]byte(g.listing))
		return
	}
	_, _ = w.Write([]byte(`{"id":"NEW","status":"PENDING"}`))
}

func (g *templateGraph) count(key string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls[key]
}

// A Coexistence account missing a portal template only gets its statuses
// read: no template is created on Meta (it got numbers blocked). Other
// accounts still get the missing ones created.
func TestEnsurePortalTemplate_CoexistenceOnlyReads(t *testing.T) {
	for name, tc := range map[string]struct {
		coexistence bool
		wantPosts   bool
	}{
		"coexistence: read only":  {coexistence: true, wantPosts: false},
		"cloud api: creates them": {coexistence: false, wantPosts: true},
	} {
		t.Run(name, func(t *testing.T) {
			db := testutils.SetupTestDB(t, &whatsapp_models.WhatsAppAccount{}, &whatsapp_models.WhatsAppTemplate{})
			const tenantID = 1
			graph := &templateGraph{calls: map[string]int{},
				listing: `{"data":[{"id":"T1","name":"pengi_cita_recordatorio","status":"APPROVED","language":"es"}]}`}
			srv := httptest.NewServer(graph)
			t.Cleanup(srv.Close)

			box, err := whatsapp_models.Box()
			if err != nil {
				t.Fatal(err)
			}
			account := whatsapp_models.WhatsAppAccount{TenantID: tenantID, WabaID: "WABA", PhoneNumberID: "PH-1",
				Status: whatsapp_models.AccountStatusConnected, Coexistence: tc.coexistence}
			if err := account.SealAccessToken(box, "token"); err != nil {
				t.Fatal(err)
			}
			if err := tenantdb.ForTenant(db, tenantID).Create(&account).Error; err != nil {
				t.Fatal(err)
			}

			d := portal_services.NewDelivery(db, zap.NewNop(), nil, whatsapp.New(srv.URL, "v23.0", "", ""))
			d.EnsurePortalTemplate(tenantID)

			posts := graph.count("POST /v23.0/WABA/message_templates")
			if tc.wantPosts != (posts > 0) {
				t.Fatalf("template creations = %d, want any = %v", posts, tc.wantPosts)
			}
			if gets := graph.count("GET /v23.0/WABA/message_templates"); gets != 1 {
				t.Fatalf("listings = %d, want 1", gets)
			}
			var rows []whatsapp_models.WhatsAppTemplate
			tenantdb.ForTenant(db, tenantID).Find(&rows)
			byName := map[string]string{}
			for _, r := range rows {
				byName[r.Name] = r.Status
			}
			if byName[whatsapp_models.ReminderTemplate] != "APPROVED" {
				t.Fatalf("read status not stored: %v", byName)
			}
			if _, ok := byName[whatsapp_models.TemplatePortalCode]; ok == tc.coexistence {
				t.Fatalf("portal_code row = %v, coexistence %v", byName, tc.coexistence)
			}
		})
	}
}
