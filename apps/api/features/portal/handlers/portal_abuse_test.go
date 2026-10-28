package portal_handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"

	"go.uber.org/zap"
)

// fakeSiteverify answers Turnstile's siteverify: success only for the token
// "good" sent with the secret "s3cret" and a remote IP.
type fakeSiteverify struct {
	mu    sync.Mutex
	calls int
}

func (s *fakeSiteverify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	_ = r.ParseForm()
	ok := r.PostForm.Get("secret") == "s3cret" && r.PostForm.Get("response") == "good" && r.PostForm.Get("remoteip") != ""
	_, _ = fmt.Fprintf(w, `{"success":%t}`, ok)
}

func (f *fixture) loginWith(tenant tenant_models.Tenant, body map[string]string) result {
	return f.do(http.MethodPost, "/portal/clinics/"+tenant.Slug+"/login", "", body)
}

func TestLogin_CaptchaWhenConfigured(t *testing.T) {
	f := newFixture(t)
	verify := &fakeSiteverify{}
	srv := httptest.NewServer(verify)
	t.Cleanup(srv.Close)
	f.auth.WithCaptcha(portal_services.NewTurnstile("s3cret", srv.URL, srv.Client()))

	for name, token := range map[string]string{"missing": "", "invalid": "bad"} {
		r := f.loginWith(f.own, map[string]string{"document": f.ownPat.Document, "captcha_token": token})
		if r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-008" {
			t.Fatalf("%s token: %d %s", name, r.Code, r.Raw)
		}
	}
	if len(f.codes(f.ownPat.ID)) != 0 || f.mail.calls != 0 {
		t.Fatal("no code without a valid captcha")
	}
	// An unknown document gets the same 400: the captcha says nothing about it.
	if r := f.loginWith(f.own, map[string]string{"document": "0000000000"}); r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-008" {
		t.Fatalf("unknown document: %d %s", r.Code, r.Raw)
	}

	r := f.loginWith(f.own, map[string]string{"document": f.ownPat.Document, "captcha_token": "good"})
	if r.Code != http.StatusOK || r.Body["message"] != "portal.login.code_sent" {
		t.Fatalf("valid token: %d %s", r.Code, r.Raw)
	}
	if len(f.codes(f.ownPat.ID)) != 1 || f.mail.last(f.ownPat.Email) == "" {
		t.Fatal("a valid captcha must send the code")
	}

	// Cloudflare unreachable: fail closed.
	srv.Close()
	if r := f.loginWith(f.own, map[string]string{"document": f.ownPat.Document, "captcha_token": "good"}); r.Code != http.StatusBadRequest || errorCode(r) != "E-PORTAL-008" {
		t.Fatalf("unreachable: %d %s", r.Code, r.Raw)
	}
}

func TestLogin_CaptchaSkippedWithoutSecret(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	if portal_services.TurnstileFromConfig() != nil {
		t.Fatal("no secret: no captcha")
	}
	t.Setenv("TURNSTILE_SECRET_KEY", "s3cret")
	if portal_services.TurnstileFromConfig() == nil {
		t.Fatal("a secret turns the captcha on")
	}

	f := newFixture(t) // WithCaptcha(nil), like an unset secret
	if r := f.login(f.own, f.ownPat.Document); r.Code != http.StatusOK || len(f.codes(f.ownPat.ID)) != 1 {
		t.Fatalf("without captcha: %d %s", r.Code, r.Raw)
	}
}

// seedCodes inserts n codes of patientID created at.
func (f *fixture) seedCodes(tenantID, patientID uint, n int, at time.Time) {
	f.t.Helper()
	rows := make([]portal_models.PortalLoginCode, n)
	for i := range rows {
		rows[i] = portal_models.PortalLoginCode{TenantID: tenantID, PatientID: patientID, Channel: portal_models.ChannelEmail,
			CodeHash: "x", ExpiresAt: at.Add(portal_services.CodeTTL).UTC()}
		rows[i].CreatedAt = at.UTC()
	}
	if err := tenantdb.System(f.db).Create(&rows).Error; err != nil {
		f.t.Fatal(err)
	}
}

// portalAdmin makes user uid a MANAGE_PATIENT_PORTAL member of tenant's company.
func (f *fixture) portalAdmin(tenant tenant_models.Tenant, uid uint) {
	f.t.Helper()
	if err := f.db.AutoMigrate(&notifications_models.Notification{}, &user_models.Role{}, &user_models.Environment{}); err != nil {
		f.t.Fatal(err)
	}
	var company company_models.Company
	if err := tenantdb.System(f.db).Where("tenant_id = ?", tenant.ID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	var perm permission_models.Permission
	if err := tenantdb.System(f.db).Where("id = ?", "MANAGE_PATIENT_PORTAL").First(&perm).Error; err != nil {
		f.t.Fatal(err)
	}
	role := user_models.Role{Role: "admin-" + strconv.FormatInt(time.Now().UnixNano(), 10), Permissions: []permission_models.Permission{perm}}
	f.mustCreate(&role)
	f.mustCreate(&user_models.Environment{UserID: uid, CompanyID: company.ID, RoleID: role.ID})
}

func (f *fixture) budgetNotices(tenantID uint) []notifications_models.Notification {
	var out []notifications_models.Notification
	tenantdb.System(f.db).Where("tenant_id = ? AND type = ?", tenantID, portal_services.NotificationTypeCodeBudget).Find(&out)
	return out
}

func TestLogin_BudgetsStopSendingAndAnswerTheSame(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		seed   func(f *fixture)
		notify bool
	}{
		{"patient per day", func(f *fixture) {
			f.seedCodes(f.own.ID, f.ownPat.ID, portal_services.MaxCodesPerPatientPerDay, now.Add(-2*time.Hour))
		}, false},
		{"tenant per hour", func(f *fixture) {
			f.seedCodes(f.own.ID, f.ownPat2.ID, portal_services.MaxCodesPerTenantPerHour, now.Add(-10*time.Minute))
		}, true},
		{"tenant per day", func(f *fixture) {
			f.seedCodes(f.own.ID, f.ownPat2.ID, portal_services.MaxCodesPerTenantPerDay, now.Add(-5*time.Hour))
		}, true},
		{"global per day", func(f *fixture) {
			f.t.Setenv("PORTAL_MAX_CODES_PER_DAY", "4")
			f.seedCodes(f.other.ID, f.otherPat.ID, 4, now.Add(-3*time.Hour))
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.portalAdmin(f.own, 7)
			unknown := f.login(f.own, "0000000000")
			tc.seed(f)
			before := len(f.codes(f.ownPat.ID))

			r := f.login(f.own, f.ownPat.Document)
			if r.Code != http.StatusOK || string(r.Raw) != string(unknown.Raw) {
				t.Fatalf("answer differs from a normal login: %d %s vs %s", r.Code, r.Raw, unknown.Raw)
			}
			if len(f.codes(f.ownPat.ID)) != before || f.mail.calls != 0 {
				t.Fatalf("nothing must be sent: codes %d -> %d, emails %d", before, len(f.codes(f.ownPat.ID)), f.mail.calls)
			}
			f.login(f.own, f.ownPat.Document) // again: still one notice
			notices := f.budgetNotices(f.own.ID)
			if tc.notify {
				if len(notices) != 1 || notices[0].UserID != 7 || notices[0].ActionURL != "/settings/portal" {
					t.Fatalf("want one notice for user 7, got %+v", notices)
				}
			} else if len(notices) != 0 {
				t.Fatalf("no notice expected, got %d", len(notices))
			}
		})
	}
}

func TestLogin_BudgetsCountOnlyTheLastDay(t *testing.T) {
	f := newFixture(t)
	f.seedCodes(f.own.ID, f.ownPat2.ID, portal_services.MaxCodesPerTenantPerDay, time.Now().Add(-25*time.Hour))
	f.seedCodes(f.own.ID, f.ownPat.ID, portal_services.MaxCodesPerPatientPerDay, time.Now().Add(-25*time.Hour))
	before := len(f.codes(f.ownPat.ID))
	if r := f.login(f.own, f.ownPat.Document); r.Code != http.StatusOK {
		t.Fatalf("login: %d %s", r.Code, r.Raw)
	}
	if len(f.codes(f.ownPat.ID)) != before+1 || f.mail.calls != 1 {
		t.Fatal("codes older than a day must not count")
	}
}

func TestNotifyCodeBudget_OncePerDay(t *testing.T) {
	f := newFixture(t)
	f.portalAdmin(f.own, 7)
	day := time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)
	portal_services.NotifyCodeBudget(f.db, zap.NewNop(), f.own.ID, portal_services.BudgetTenantHour, day)
	portal_services.NotifyCodeBudget(f.db, zap.NewNop(), f.own.ID, portal_services.BudgetTenantDay, day.Add(time.Hour))
	if n := f.budgetNotices(f.own.ID); len(n) != 1 || n[0].MessageKey != "notification.portal.code_budget.hour" {
		t.Fatalf("want one notice the first day, got %+v", n)
	}
	portal_services.NotifyCodeBudget(f.db, zap.NewNop(), f.own.ID, portal_services.BudgetTenantDay, day.Add(24*time.Hour))
	if n := f.budgetNotices(f.own.ID); len(n) != 2 || n[1].MessageKey != "notification.portal.code_budget.day" {
		t.Fatalf("want a second notice the next day, got %+v", n)
	}
	if n := f.budgetNotices(f.other.ID); len(n) != 0 {
		t.Fatal("another clinic gets no notice")
	}
}

func TestLogin_PerPatientDailyCap(t *testing.T) {
	f := newFixture(t)
	// Spread over the day, outside the 15-minute window.
	f.seedCodes(f.own.ID, f.ownPat.ID, portal_services.MaxCodesPerPatientPerDay-1, time.Now().Add(-3*time.Hour))
	f.login(f.own, f.ownPat.Document) // the 6th code of the day
	f.login(f.own, f.ownPat.Document) // over the cap
	if n := len(f.codes(f.ownPat.ID)); n != portal_services.MaxCodesPerPatientPerDay || f.mail.calls != 1 {
		t.Fatalf("codes = %d, emails = %d", n, f.mail.calls)
	}
}

// Parallel logins (as from several API instances: the lock lives in
// Postgres, not in the process) never issue more codes than a budget allows.
// Only Postgres serializes them; SQLite has a single writer anyway.
func TestLogin_ParallelRequestsRespectTheBudget(t *testing.T) {
	f := newFixture(t)
	if f.db.Dialector.Name() != "postgres" {
		t.Skip("advisory locks need Postgres")
	}
	var wg sync.WaitGroup
	for i := 0; i < 4*portal_services.MaxCodesPerWindow; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = portal_services.IssueCode(f.db, f.own.ID, f.ownPat.ID, portal_models.ChannelEmail, time.Now())
		}()
	}
	wg.Wait()
	if n := len(f.codes(f.ownPat.ID)); n != portal_services.MaxCodesPerWindow {
		t.Fatalf("codes issued = %d, want %d", n, portal_services.MaxCodesPerWindow)
	}
}
