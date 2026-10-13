package y2026

import (
	"fmt"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_data "pengi-med-saas/features/doctors/data"
	doctor_models "pengi-med-saas/features/doctors/models"
	permission_models "pengi-med-saas/features/permissions/models"
	signature_models "pengi-med-saas/features/signatures/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

type doctorMigrationFixture struct {
	t           *testing.T
	db          *gorm.DB
	admin, doc  user_models.Role
	tenantCount int
}

func newDoctorMigrationFixture(t *testing.T) *doctorMigrationFixture {
	t.Helper()
	db := tenantdb.System(testutils.SetupTestDB(t, &tenant_models.Tenant{}, &company_models.Company{}, &user_models.User{},
		&permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{}, &signature_models.UserSignature{},
		&clinical_models.Patient{}, &doctor_models.Doctor{}))
	f := &doctorMigrationFixture{t: t, db: db, admin: user_models.Role{Role: role_data.RoleAdmin}, doc: user_models.Role{Role: role_data.RoleDoctor}}
	f.create(&f.admin)
	f.create(&f.doc)
	return f
}

func (f *doctorMigrationFixture) create(v any) {
	f.t.Helper()
	if err := f.db.Create(v).Error; err != nil {
		f.t.Fatalf("create %T: %v", v, err)
	}
}

// tenant creates a tenant whose company is owned by a new admin user.
func (f *doctorMigrationFixture) tenant() (tenant_models.Tenant, company_models.Company, user_models.User) {
	f.t.Helper()
	f.tenantCount++
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("drmig-%d-%d", f.tenantCount, now), DisplayToken: fmt.Sprintf("tok-drmig-%d-%d", f.tenantCount, now)}
	f.create(&tenant)
	owner := f.user(fmt.Sprintf("owner%d", f.tenantCount))
	company := company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "p", TenantID: tenant.ID, OwnerUserID: owner.ID}
	f.create(&company)
	f.create(&user_models.Environment{UserID: owner.ID, CompanyID: company.ID, RoleID: f.admin.ID})
	return tenant, company, owner
}

func (f *doctorMigrationFixture) user(name string) user_models.User {
	f.t.Helper()
	u := user_models.User{UserName: name, Email: name + "@example.com"}
	f.create(&u)
	return u
}

func (f *doctorMigrationFixture) member(company company_models.Company, name string, role user_models.Role) user_models.User {
	f.t.Helper()
	u := f.user(name)
	f.create(&user_models.Environment{UserID: u.ID, CompanyID: company.ID, RoleID: role.ID})
	return u
}

func (f *doctorMigrationFixture) signature(tenantID, userID uint, subject string) {
	f.create(&signature_models.UserSignature{TenantID: tenantID, UserID: userID, SubjectName: subject, NotAfter: time.Now().AddDate(1, 0, 0)})
}

func (f *doctorMigrationFixture) patient(tenantID uint, medic string) {
	f.create(&clinical_models.Patient{TenantID: tenantID, FirstName: "P", LastName: "X", Institution: "H",
		Document: fmt.Sprintf("DOC-%d", time.Now().UnixNano()), Medic: medic})
}

func (f *doctorMigrationFixture) doctors(tenantID uint) []doctor_models.Doctor {
	f.t.Helper()
	var out []doctor_models.Doctor
	if err := f.db.Where("tenant_id = ?", tenantID).Order("id").Find(&out).Error; err != nil {
		f.t.Fatal(err)
	}
	return out
}

func runCreateDoctors(t *testing.T, db *gorm.DB) {
	t.Helper()
	m, ok := database.GlobalDBMap[createDoctorsFromUsersID]
	if !ok {
		t.Fatalf("migration %s not registered", createDoctorsFromUsersID)
	}
	if err := m.Execute(db); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDoctorsFromUsers(t *testing.T) {
	f := newDoctorMigrationFixture(t)

	// Clinic with doctor-role users: one per doctor, the admin owner is left out.
	clinic, clinicCo, _ := f.tenant()
	ana := f.member(clinicCo, "ana", f.doc)
	luis := f.member(clinicCo, "luis", f.doc)
	f.member(clinicCo, "otro-admin", f.admin)
	f.signature(clinic.ID, ana.ID, "ANA MARIA PEREZ")

	// Owner with a signature.
	signed, _, signedOwner := f.tenant()
	f.signature(signed.ID, signedOwner.ID, "DR. OWNER FIRMADO")
	f.patient(signed.ID, "Dr. Otro")

	// Owner without signature: most frequent non-empty patients.medic.
	medics, _, _ := f.tenant()
	for _, m := range []string{"Dr. Ruiz", " Dr. Ruiz ", "Dr. Paz", "", ""} {
		f.patient(medics.ID, m)
	}

	// Owner with nothing else: user name.
	bare, _, bareOwner := f.tenant()

	runCreateDoctors(t, f.db)

	got := f.doctors(clinic.ID)
	if len(got) != 2 {
		t.Fatalf("clinic: got %d doctors, want 2", len(got))
	}
	byUser := map[uint]doctor_models.Doctor{}
	for _, d := range got {
		if d.UserID == nil {
			t.Fatalf("clinic doctor without user: %+v", d)
		}
		byUser[*d.UserID] = d
	}
	if byUser[ana.ID].FullName != "ANA MARIA PEREZ" || byUser[luis.ID].FullName != "luis" {
		t.Fatalf("clinic names: %+v", byUser)
	}
	if d := byUser[luis.ID]; !d.Active || !d.NeedsReview || d.Specialty != doctor_data.DefaultSpecialty || d.Email != "luis@example.com" || d.Color == "" {
		t.Fatalf("clinic doctor fields: %+v", d)
	}
	if byUser[ana.ID].Color == byUser[luis.ID].Color {
		t.Fatal("doctors of one tenant should get different colors")
	}

	for _, tc := range []struct {
		name   string
		tenant tenant_models.Tenant
		want   string
		user   uint
	}{
		{"owner signature", signed, "DR. OWNER FIRMADO", signedOwner.ID},
		{"most frequent medic", medics, "Dr. Ruiz", 0},
		{"user name", bare, bareOwner.UserName, bareOwner.ID},
	} {
		ds := f.doctors(tc.tenant.ID)
		if len(ds) != 1 || ds[0].FullName != tc.want || ds[0].UserID == nil || !ds[0].NeedsReview {
			t.Fatalf("%s: got %+v", tc.name, ds)
		}
		if tc.user != 0 && *ds[0].UserID != tc.user {
			t.Fatalf("%s: linked to %d, want owner %d", tc.name, *ds[0].UserID, tc.user)
		}
	}

	// Idempotent: a second run creates nothing.
	var before, after int64
	f.db.Model(&doctor_models.Doctor{}).Count(&before)
	runCreateDoctors(t, f.db)
	f.db.Model(&doctor_models.Doctor{}).Count(&after)
	if before != after || before != 5 {
		t.Fatalf("doctors before/after second run = %d/%d, want 5/5", before, after)
	}
}

func TestCreateDoctorsFromUsers_AddsMissingDoctorUserOnly(t *testing.T) {
	f := newDoctorMigrationFixture(t)
	clinic, clinicCo, _ := f.tenant()
	ana := f.member(clinicCo, "ana", f.doc)
	anaID := ana.ID
	f.create(&doctor_models.Doctor{TenantID: clinic.ID, UserID: &anaID, FullName: "Ya existe", Specialty: "cardiology", Active: true})
	luis := f.member(clinicCo, "luis", f.doc)

	runCreateDoctors(t, f.db)

	got := f.doctors(clinic.ID)
	if len(got) != 2 || got[0].FullName != "Ya existe" || got[0].NeedsReview || got[1].UserID == nil || *got[1].UserID != luis.ID {
		t.Fatalf("got %+v", got)
	}
}

func TestAddDoctorPermissions_AdminAndClinicalFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	admin := user_models.Role{Role: role_data.RoleAdmin}
	doctor := user_models.Role{Role: role_data.RoleDoctor}
	clinical := company_models.Feature{Code: "CLINICAL", Name: "Clinical"}
	for _, v := range []any{&admin, &doctor, &clinical} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Twice: it must be idempotent.
	for i := 0; i < 2; i++ {
		if err := database.GlobalDBMap[addDoctorPermissionsID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	has := func(model any, assoc string) bool {
		var perms []permission_models.Permission
		if err := db.Model(model).Association(assoc).Find(&perms); err != nil {
			t.Fatal(err)
		}
		for _, p := range perms {
			if p.ID == "MANAGE_DOCTORS" {
				return true
			}
		}
		return false
	}
	if !has(&admin, "Permissions") || !has(&clinical, "Permissions") {
		t.Fatal("MANAGE_DOCTORS must be on the admin role and the CLINICAL feature")
	}
	if has(&doctor, "Permissions") {
		t.Fatal("MANAGE_DOCTORS is admin-only")
	}
}

func TestAddDoctorPermissions_CreatesPermissionWithoutAdminOrFeature(t *testing.T) {
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &company_models.Feature{})
	if err := database.GlobalDBMap[addDoctorPermissionsID].Execute(db); err != nil {
		t.Fatalf("must not fail when the admin role or CLINICAL feature is missing: %v", err)
	}
	var n int64
	db.Model(&permission_models.Permission{}).Where("id = ?", "MANAGE_DOCTORS").Count(&n)
	if n != 1 {
		t.Fatal("MANAGE_DOCTORS was not created")
	}
}
