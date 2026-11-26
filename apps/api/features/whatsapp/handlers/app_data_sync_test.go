package whatsapp_handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"pengi-med-saas/core/whatsapp"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// Every attempted app data sync, in order.
func (g *fakeGraph) syncs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var types []string
	for _, b := range g.allBodies["POST /v23.0/PHONE-1/smb_app_data"] {
		var body struct {
			SyncType string `json:"sync_type"`
		}
		_ = json.Unmarshal([]byte(b), &body)
		types = append(types, body.SyncType)
	}
	return types
}

func (f *fixture) connectEmbedded(coexistence bool) int {
	f.t.Helper()
	f.t.Setenv("META_ES_CONFIG_ID", "cfg")
	resp, _ := f.call(f.h.ConnectEmbedded, f.own, http.MethodPost, map[string]any{
		"code": "the-code", "waba_id": "WABA-1", "phone_number_id": "PHONE-1", "coexistence": coexistence,
	}, "")
	return resp.Code
}

func wantSyncs(t *testing.T, f *fixture) {
	t.Helper()
	if got := strings.Join(f.graph.syncs(), ","); got != whatsapp.SyncContacts+","+whatsapp.SyncHistory {
		t.Fatalf("syncs = %q, want contacts then history, once each", got)
	}
}

// failSync makes Meta refuse one sync type.
func failSync(syncType string) bodyFailure {
	return bodyFailure{key: "POST /v23.0/PHONE-1/smb_app_data", sub: `"` + syncType + `"`,
		resp: fakeResp{400, `{"error":{"code":100,"message":"not eligible"}}`}}
}

// A Coexistence connect requests contacts, then history, once the account is
// saved, and stores Meta's request ids.
func TestConnectCoexistence_RequestsAppDataSync(t *testing.T) {
	f := newFixture(t)
	core, logs := observer.New(zap.InfoLevel)
	f.h.logger = zap.New(core)
	if code := f.connectEmbedded(true); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	wantSyncs(t, f)
	a := f.account(f.own)
	if a.ContactsSyncRequestID != "REQ-1" || a.HistorySyncRequestID != "REQ-1" || a.AppDataSyncError != "" || a.AppDataSyncAt == nil {
		t.Fatalf("account = %+v", a)
	}
	if a.Status != whatsapp_models.AccountStatusConnected || !a.Coexistence {
		t.Fatalf("account = %+v", a)
	}
	if n := logs.FilterMessage("whatsapp app data sync requested").FilterField(zap.String("request_id", "REQ-1")).Len(); n != 2 {
		t.Fatalf("logs = %d", n)
	}
	// The sync came after everything else on Graph, so after the save.
	calls := f.graph.calls
	if len(calls) < 2 || calls[len(calls)-2] != calls[len(calls)-1] || !strings.HasSuffix(calls[len(calls)-1], "/smb_app_data") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestConnect_NoAppDataSyncWithoutCoexistence(t *testing.T) {
	f := newFixture(t)
	if resp := f.connectManual(f.own); resp.Code != http.StatusOK {
		t.Fatalf("manual: %d", resp.Code)
	}
	if code := f.connectEmbedded(false); code != http.StatusOK {
		t.Fatalf("embedded: %d", code)
	}
	if got := f.graph.syncs(); len(got) != 0 {
		t.Fatalf("syncs = %v", got)
	}
	if a := f.account(f.own); a.AppDataSyncAt != nil {
		t.Fatalf("account = %+v", a)
	}
}

// Nothing is requested for a connect that fails.
func TestConnectCoexistence_NoAppDataSyncWhenConnectFails(t *testing.T) {
	f := newFixture(t)
	f.graph.set("POST /v23.0/WABA-1/subscribed_apps", 500, `{"error":{"code":2,"message":"down"}}`)
	if code := f.connectEmbedded(true); code == http.StatusOK {
		t.Fatal("connect must fail")
	}
	if got := f.graph.syncs(); len(got) != 0 {
		t.Fatalf("syncs = %v", got)
	}
}

// A failed sync never fails the connect nor is it retried; contacts failing
// doesn't stop history.
func TestConnectCoexistence_AppDataSyncFailure(t *testing.T) {
	for _, tc := range []struct {
		name              string
		fail              []string
		contacts, history string
	}{
		{"contacts", []string{whatsapp.SyncContacts}, "", "REQ-1"},
		{"history", []string{whatsapp.SyncHistory}, "REQ-1", ""},
		{"both", []string{whatsapp.SyncContacts, whatsapp.SyncHistory}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			core, logs := observer.New(zap.WarnLevel)
			f.h.logger = zap.New(core)
			for _, s := range tc.fail {
				f.graph.failWhen = append(f.graph.failWhen, failSync(s))
			}
			if code := f.connectEmbedded(true); code != http.StatusOK {
				t.Fatalf("code = %d", code)
			}
			wantSyncs(t, f)
			a := f.account(f.own)
			if a.ContactsSyncRequestID != tc.contacts || a.HistorySyncRequestID != tc.history || a.Status != whatsapp_models.AccountStatusConnected {
				t.Fatalf("account = %+v", a)
			}
			for _, s := range tc.fail {
				if !strings.Contains(a.AppDataSyncError, s+": ") {
					t.Fatalf("error = %q", a.AppDataSyncError)
				}
			}
			if n := logs.FilterMessage("whatsapp app data sync failed").Len(); n != len(tc.fail) {
				t.Fatalf("warnings = %d", n)
			}
		})
	}
}

// A new connect starts over: the previous outcome is cleared.
func TestConnect_ResetsAppDataSync(t *testing.T) {
	f := newFixture(t)
	f.graph.failWhen = []bodyFailure{failSync(whatsapp.SyncHistory)}
	if code := f.connectEmbedded(true); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if a := f.account(f.own); a.ContactsSyncRequestID == "" || a.AppDataSyncError == "" {
		t.Fatalf("account = %+v", a)
	}
	if resp := f.connectManual(f.own); resp.Code != http.StatusOK {
		t.Fatalf("manual: %d", resp.Code)
	}
	if a := f.account(f.own); a.ContactsSyncRequestID != "" || a.HistorySyncRequestID != "" || a.AppDataSyncError != "" || a.AppDataSyncAt != nil {
		t.Fatalf("account = %+v", a)
	}
}

// A slow Graph doesn't hold the connect: the sync finishes in background and
// leaves alone what changed on the account meanwhile.
func TestConnectCoexistence_SlowAppDataSyncFinishesInBackground(t *testing.T) {
	prev := appDataSyncWait
	appDataSyncWait = 10 * time.Millisecond
	t.Cleanup(func() { appDataSyncWait = prev })

	f := newFixture(t)
	f.graph.mu.Lock()
	f.graph.delay = 100 * time.Millisecond
	f.graph.mu.Unlock()
	if code := f.connectEmbedded(true); code != http.StatusOK {
		t.Fatalf("code = %d", code)
	}
	if a := f.account(f.own); a.HistorySyncRequestID != "" {
		t.Fatal("the connect waited for the sync")
	}
	f.signed(accountUpdateEvent("PARTNER-BIZ", "PARTNER_REMOVED", "WABA-1", "ACCOUNT_DISCONNECTED"))

	deadline := time.Now().Add(5 * time.Second)
	for f.account(f.own).HistorySyncRequestID == "" {
		if time.Now().After(deadline) {
			t.Fatalf("background sync never stored: %+v", f.account(f.own))
		}
		time.Sleep(20 * time.Millisecond)
	}
	if s := f.account(f.own).Status; s != whatsapp_models.AccountStatusAppDisconnected {
		t.Fatalf("the sync overwrote the status: %s", s)
	}
	wantSyncs(t, f)
}
