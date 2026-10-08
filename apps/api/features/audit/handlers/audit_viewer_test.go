package audit_handlers

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// viewerFixture is two clinics: "own" has a team member (Dra. Ana), a user
// who left the team (exuser) and a patient; "other" has its own user.
type viewerFixture struct {
	db                 *gorm.DB
	own, other         tenant_models.Tenant
	ana, former, outer user_models.User
	patient            clinical_models.Patient
}

func newViewerFixture(t *testing.T) *viewerFixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &audit.AuditLog{}, &user_models.User{},
		&user_models.Role{}, &user_models.Environment{}, &company_models.Company{}, &clinical_models.Patient{})
	f := &viewerFixture{db: db}
	now := time.Now().UnixNano()
	for i, tenant := range []*tenant_models.Tenant{&f.own, &f.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("C%d", i), Slug: fmt.Sprintf("aud-v-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-aud-v-%d-%d", i, now)}
		mustCreate(t, db, tenant)
	}
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "PRO", TenantID: f.own.ID}
	mustCreate(t, db, &company)
	for i, u := range []*user_models.User{&f.ana, &f.former, &f.outer} {
		*u = user_models.User{UserName: []string{"ana", "exuser", "outsider"}[i] + fmt.Sprint(now), Email: fmt.Sprintf("u%d-%d@x.ec", i, now)}
		mustCreate(t, db, u)
	}
	role := user_models.Role{Role: fmt.Sprintf("r-%d", now)}
	mustCreate(t, db, &role)
	mustCreate(t, db, &user_models.Environment{UserID: f.ana.ID, Name: "Dra. Ana", RoleID: role.ID, CompanyID: company.ID})
	f.patient = clinical_models.Patient{TenantID: f.own.ID, FirstName: "Luis", LastName: "Paz", Institution: "H", Document: fmt.Sprintf("D-%d", now)}
	mustCreate(t, db, &f.patient)

	at := time.Now()
	for _, l := range []audit.AuditLog{
		{TenantID: f.own.ID, UserID: f.ana.ID, Action: "READ", EntityType: "patients", EntityID: f.patient.ID, PatientID: &f.patient.ID, CreatedAt: at},
		{TenantID: f.own.ID, UserID: f.former.ID, Action: "UPDATE", EntityType: "appointments", EntityID: 9, CreatedAt: at.Add(-time.Minute)},
		// Older row on the patient itself, written without patient_id.
		{TenantID: f.own.ID, UserID: f.ana.ID, Action: "UPDATE", EntityType: "patients", EntityID: f.patient.ID, CreatedAt: at.Add(-2 * time.Minute)},
		{TenantID: f.other.ID, UserID: f.outer.ID, Action: "READ", EntityType: "patients", EntityID: 1, CreatedAt: at},
	} {
		mustCreate(t, db, &l)
	}
	return f
}

func mustCreate(t *testing.T, db *gorm.DB, v any) {
	t.Helper()
	if err := db.Create(v).Error; err != nil {
		t.Fatalf("create %T: %v", v, err)
	}
}

func (f *viewerFixture) items(t *testing.T, query string) []AuditLogItem {
	t.Helper()
	c, _ := testutils.NewGinContext(f.own.ID, 1)
	c.Request = httptest.NewRequest("GET", "/audit/logs"+query, nil)
	resp := NewAuditLogHandler(f.db, zap.NewNop()).GetAuditLogs(c)
	if resp.Code != 200 {
		t.Fatalf("GetAuditLogs%s = %d %q", query, resp.Code, resp.Message)
	}
	return resp.Data.(envelope.PagedData).Items.([]AuditLogItem)
}

// Rows carry the user's team name (or user name once they left) and the
// patient's name.
func TestGetAuditLogs_ShowsUserAndPatientNames(t *testing.T) {
	f := newViewerFixture(t)
	items := f.items(t, "")
	if len(items) != 3 {
		t.Fatalf("items = %d, want own tenant's 3", len(items))
	}
	if items[0].UserName != "Dra. Ana" || items[0].PatientName != "Luis Paz" {
		t.Errorf("first row = user %q patient %q, want Dra. Ana / Luis Paz", items[0].UserName, items[0].PatientName)
	}
	if items[1].UserName != f.former.UserName || items[1].PatientName != "" {
		t.Errorf("second row = user %q patient %q, want %q / none", items[1].UserName, items[1].PatientName, f.former.UserName)
	}
	if items[2].PatientName != "Luis Paz" {
		t.Errorf("row on the patient without patient_id = patient %q, want Luis Paz", items[2].PatientName)
	}
}

func TestGetAuditLogs_FiltersByUser(t *testing.T) {
	f := newViewerFixture(t)
	items := f.items(t, fmt.Sprintf("?user_id=%d", f.former.ID))
	if len(items) != 1 || items[0].UserID != f.former.ID {
		t.Fatalf("user filter returned %+v", items)
	}
	// Another clinic's user filters to nothing, not to their rows.
	if items := f.items(t, fmt.Sprintf("?user_id=%d", f.outer.ID)); len(items) != 0 {
		t.Fatalf("filtering by another tenant's user returned %d rows", len(items))
	}
}

// The user filter lists who appears in this clinic's trail, team members and
// former ones, never another clinic's users.
func TestGetAuditUsers_OwnTenantOnly(t *testing.T) {
	f := newViewerFixture(t)
	c, _ := testutils.NewGinContext(f.own.ID, 1)
	c.Request = httptest.NewRequest("GET", "/audit/users", nil)
	resp := NewAuditLogHandler(f.db, zap.NewNop()).GetAuditUsers(c)
	if resp.Code != 200 {
		t.Fatalf("GetAuditUsers = %d %q", resp.Code, resp.Message)
	}
	got := map[uint]string{}
	for _, u := range resp.Data.([]AuditUser) {
		got[u.ID] = u.Name
	}
	want := map[uint]string{f.ana.ID: "Dra. Ana", f.former.ID: f.former.UserName}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("users = %v, want %v", got, want)
	}
}
