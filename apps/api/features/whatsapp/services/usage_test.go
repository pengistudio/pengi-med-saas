package whatsapp_services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"
)

func usageDB(t *testing.T) (*gorm.DB, uint, uint) {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &company_models.Company{}, &company_models.Feature{}, &company_models.Plan{},
		&company_models.Subscription{}, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&notifications_models.Notification{}, &whatsapp_models.WhatsAppMessage{})
	var tenants [2]uint
	for i := range tenants {
		slug := fmt.Sprintf("wa-use-%d-%d", i, time.Now().UnixNano())
		tenant := tenant_models.Tenant{Name: "C", Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		tenants[i] = tenant.ID
	}
	return db, tenants[0], tenants[1]
}

// withLimit gives tenantID a company on an active plan capping messages at
// limit, and a WhatsApp admin (user 7) plus an inbox-only user (8).
func withLimit(t *testing.T, db *gorm.DB, tenantID uint, limit int) {
	t.Helper()
	code := fmt.Sprintf("P-%d-%d", tenantID, time.Now().UnixNano())
	plan := company_models.Plan{Name: code, Code: code, Properties: datatypes.JSONMap{PlanLimitKey: limit}}
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: code, TenantID: tenantID}
	for _, row := range []any{&plan, &company} {
		if err := db.Omit(clause.Associations).Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	sub := company_models.Subscription{Status: "active", PlanCode: code, ExpiresAt: time.Now().AddDate(1, 0, 0), CompanyID: company.ID}
	if err := db.Omit(clause.Associations).Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	for uid, perm := range map[uint]string{7: "MANAGE_WHATSAPP", 8: "USE_WHATSAPP_INBOX"} {
		p := permission_models.Permission{BaseStringID: database.BaseStringID{ID: perm}, Name: perm}
		if err := db.Where(permission_models.Permission{BaseStringID: p.BaseStringID}).FirstOrCreate(&p).Error; err != nil {
			t.Fatal(err)
		}
		role := user_models.Role{Role: fmt.Sprintf("r-%s-%d", perm, time.Now().UnixNano()), Permissions: []permission_models.Permission{p}}
		if err := db.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&user_models.Environment{UserID: uid, CompanyID: company.ID, RoleID: role.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func sentAt(t *testing.T, db *gorm.DB, tenantID uint, kind string, at *time.Time) {
	t.Helper()
	dir := whatsapp_models.DirectionOutbound
	if kind == whatsapp_models.KindInbound {
		dir = whatsapp_models.DirectionInbound
	}
	m := whatsapp_models.WhatsAppMessage{TenantID: tenantID, Kind: kind, Status: whatsapp_models.MessageStatusSent, SentAt: at, Direction: dir}
	if at == nil {
		m.Status = whatsapp_models.MessageStatusFailed
	}
	if err := db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
}

func TestMonthlyUsage_TimeZoneAndKinds(t *testing.T) {
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	db, own, other := usageDB(t)
	ts := func(s string) *time.Time { v, _ := time.Parse(time.RFC3339, s); return &v }

	// Counted in October (Guayaquil): reminder, test, template.
	sentAt(t, db, own, whatsapp_models.KindReminder, ts("2026-10-01T05:00:00Z")) // Oct 1 00:00 local
	sentAt(t, db, own, whatsapp_models.KindTest, ts("2026-10-15T12:00:00Z"))
	sentAt(t, db, own, whatsapp_models.KindTemplate, ts("2026-11-01T04:30:00Z")) // Oct 31 23:30 local
	// Not counted: other months in local time, free kinds, failed before sending, other tenant.
	sentAt(t, db, own, whatsapp_models.KindReminder, ts("2026-10-01T04:59:00Z")) // Sep 30 23:59 local
	sentAt(t, db, own, whatsapp_models.KindTemplate, ts("2026-11-01T05:00:00Z")) // Nov 1 00:00 local
	sentAt(t, db, own, whatsapp_models.KindReply, ts("2026-10-15T12:00:00Z"))
	sentAt(t, db, own, whatsapp_models.KindSystem, ts("2026-10-15T12:00:00Z"))
	sentAt(t, db, own, whatsapp_models.KindInbound, ts("2026-10-15T12:00:00Z"))
	sentAt(t, db, own, whatsapp_models.KindReminder, nil)
	sentAt(t, db, other, whatsapp_models.KindReminder, ts("2026-10-15T12:00:00Z"))

	now := *ts("2026-11-01T03:00:00Z") // Oct 31 22:00 local
	u, err := monthlyUsage(db, own, now, loc)
	if err != nil {
		t.Fatal(err)
	}
	if u.Used != 3 || u.Limit != -1 {
		t.Fatalf("usage = %+v, want used 3, unlimited", u)
	}
	if want := time.Date(2026, 10, 1, 0, 0, 0, 0, loc); !u.PeriodStart.Equal(want) || !u.PeriodEnd.Equal(want.AddDate(0, 1, 0)) {
		t.Fatalf("period = %v – %v", u.PeriodStart, u.PeriodEnd)
	}
	if ok, _, _ := canSendTemplate(db, own, now, loc); !ok {
		t.Fatal("unlimited plan must allow")
	}

	withLimit(t, db, own, 3)
	if ok, u, _ := canSendTemplate(db, own, now, loc); ok || u.Used != 3 || u.Limit != 3 {
		t.Fatalf("at the cap: ok=%v usage=%+v", ok, u)
	}
	// Next month (local) starts from zero.
	if ok, u, _ := canSendTemplate(db, own, *ts("2026-11-01T06:00:00Z"), loc); !ok || u.Used != 1 {
		t.Fatalf("november: ok=%v usage=%+v", ok, u)
	}
}

func TestNotifyUsage_80And100OncePerMonth(t *testing.T) {
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	db, own, _ := usageDB(t)
	withLimit(t, db, own, 10)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, loc)
	send := func(n int) {
		for i := 0; i < n; i++ {
			at := now.Add(-time.Duration(i+1) * time.Minute)
			sentAt(t, db, own, whatsapp_models.KindReminder, &at)
		}
	}
	notifs := func() []notifications_models.Notification {
		var out []notifications_models.Notification
		db.Where("tenant_id = ? AND resource_type = ?", own, UsageResourceType).Order("id").Find(&out)
		return out
	}

	send(7)
	notifyUsage(db, zap.NewNop(), own, now, loc)
	if n := notifs(); len(n) != 0 {
		t.Fatalf("70%%: %d notifications", len(n))
	}
	send(1)
	notifyUsage(db, zap.NewNop(), own, now, loc)
	notifyUsage(db, zap.NewNop(), own, now, loc)
	n := notifs()
	if len(n) != 1 || n[0].Type != NotificationTypeUsageWarning || n[0].UserID != 7 || n[0].ResourceID != 202610 ||
		n[0].MessageKey != "notification.whatsapp.usage.warning" {
		t.Fatalf("80%%: %+v", n)
	}
	// Read or not, it is not repeated this month.
	db.Model(&notifications_models.Notification{}).Where("id = ?", n[0].ID).Update("read_at", now)
	send(1)
	notifyUsage(db, zap.NewNop(), own, now, loc)
	if len(notifs()) != 1 {
		t.Fatal("80% notified twice")
	}
	send(1)
	notifyUsage(db, zap.NewNop(), own, now, loc)
	notifyUsage(db, zap.NewNop(), own, now, loc)
	n = notifs()
	if len(n) != 2 || n[1].Type != NotificationTypeUsageReached || n[1].UserID != 7 {
		t.Fatalf("100%%: %+v", n)
	}
	// Next month: new period id.
	next := time.Date(2026, 11, 2, 12, 0, 0, 0, loc)
	for i := 0; i < 10; i++ {
		at := next.Add(-time.Duration(i+1) * time.Minute)
		sentAt(t, db, own, whatsapp_models.KindTemplate, &at)
	}
	notifyUsage(db, zap.NewNop(), own, next, loc)
	if n := notifs(); len(n) != 3 || n[2].ResourceID != 202611 || n[2].Type != NotificationTypeUsageReached {
		t.Fatalf("november: %+v", n)
	}
}

func TestIsOptInYes(t *testing.T) {
	for _, s := range []string{"SI", "si", "Sí", " sí. ", "SÍ!", "acepto", "Ok", "ok."} {
		if !IsOptInYes(s) {
			t.Errorf("%q must be yes", s)
		}
	}
	for _, s := range []string{"no", "si claro mañana", "okay", "", "sip"} {
		if IsOptInYes(s) {
			t.Errorf("%q must not be yes", s)
		}
	}
}

func TestPatientText_FromSpanishCatalog(t *testing.T) {
	if got := PatientText(OptInRequestKey); got != "Para recibir recordatorios de tus citas por WhatsApp responde SÍ." {
		t.Fatalf("request text = %q", got)
	}
	if got := PatientText(OptInConfirmedKey); got != "Listo, te enviaremos recordatorios." {
		t.Fatalf("confirmed text = %q", got)
	}
}

func TestCatalog_TemplatesAreUtilityAndRender(t *testing.T) {
	names := map[string]bool{}
	for _, spec := range Catalog {
		d := spec.Definition
		if d.Category != "UTILITY" || d.Language != "es" || names[d.Name] {
			t.Fatalf("template %+v", d)
		}
		names[d.Name] = true
		if len(d.Example) != len(spec.Variables) {
			t.Fatalf("%s: %d examples for %d variables", d.Name, len(d.Example), len(spec.Variables))
		}
		body := RenderTemplateBody(spec, d.Example)
		for i := range spec.Variables {
			if want := fmt.Sprintf("{{%d}}", i+1); !strings.Contains(d.Body, want) || strings.Contains(body, want) {
				t.Fatalf("%s: placeholder %s", d.Name, want)
			}
		}
	}
	if len(names) != 5 || !names[whatsapp_models.ReminderTemplate] {
		t.Fatalf("catalog = %v", names)
	}
	// The reminder is unchanged.
	if reminderTemplate.Definition.Body != "Hola {{1}}, te recordamos tu cita en {{2}} el {{3}} a las {{4}}. Por favor confirma tu asistencia o cancela si no podrás asistir." {
		t.Fatal("reminder template text changed")
	}
}
