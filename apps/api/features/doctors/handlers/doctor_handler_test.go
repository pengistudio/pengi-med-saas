package doctor_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	doctor_data "pengi-med-saas/features/doctors/data"
	doctor_dto "pengi-med-saas/features/doctors/dto"
	doctor_models "pengi-med-saas/features/doctors/models"
	doctor_services "pengi-med-saas/features/doctors/services"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// fixture holds two clinics; tests act as members of "own".
type fixture struct {
	t          *testing.T
	db         *gorm.DB
	h          *DoctorHandler
	own, other tenant_models.Tenant
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &company_models.Company{}, &user_models.User{},
		&user_models.Environment{}, &doctor_models.Doctor{})
	f := &fixture{t: t, db: db, h: NewDoctorHandler(db, zap.NewNop())}
	now := time.Now().UnixNano()
	for i, tenant := range []*tenant_models.Tenant{&f.own, &f.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("dr-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-dr-%d-%d", i, now)}
		f.mustCreate(tenant)
		f.mustCreate(&company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "p", TenantID: tenant.ID})
	}
	return f
}

func (f *fixture) mustCreate(v any) {
	f.t.Helper()
	if err := f.db.Create(v).Error; err != nil {
		f.t.Fatalf("create %T: %v", v, err)
	}
}

// member creates a user who belongs to tenant (through its company).
func (f *fixture) member(tenant tenant_models.Tenant, name string) uint {
	f.t.Helper()
	user := user_models.User{UserName: fmt.Sprintf("%s-%d", name, time.Now().UnixNano())}
	f.mustCreate(&user)
	var company company_models.Company
	if err := f.db.Where("tenant_id = ?", tenant.ID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	f.mustCreate(&user_models.Environment{UserID: user.ID, CompanyID: company.ID})
	return user.ID
}

func (f *fixture) doctor(tenant tenant_models.Tenant, name string, userID *uint) doctor_models.Doctor {
	f.t.Helper()
	d := doctor_models.Doctor{TenantID: tenant.ID, UserID: userID, FullName: name, Specialty: "pediatrics", Color: "#2563EB", Active: true}
	f.mustCreate(&d)
	return d
}

// ctx builds a request context for tenant as userID, with JSON body and :id.
func ctx(tenant tenant_models.Tenant, userID uint, body any, id uint) *gin.Context {
	c, _ := testutils.NewGinContext(tenant.ID, int64(userID))
	c.Set("user_id", int64(userID))
	c.Set("username", "tester")
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/", &buf)
	c.Request.Header.Set("Content-Type", "application/json")
	if id != 0 {
		c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	}
	return c
}

func wantCode(t *testing.T, resp envelope.Response, code int) {
	t.Helper()
	if resp.Code != code {
		t.Fatalf("code = %d, want %d (message %q)", resp.Code, code, resp.Message)
	}
}

func wantError(t *testing.T, resp envelope.Response, code int, appErr core_errors.AppError) {
	t.Helper()
	wantCode(t, resp, code)
	got, ok := resp.Data.(core_errors.AppError)
	if !ok || got.ErrorCode != appErr.ErrorCode {
		t.Fatalf("error code = %s, want %s", got.ErrorCode, appErr.ErrorCode)
	}
}

func (f *fixture) reload(id uint) doctor_models.Doctor {
	f.t.Helper()
	var d doctor_models.Doctor
	if err := f.db.Unscoped().First(&d, id).Error; err != nil {
		f.t.Fatal(err)
	}
	return d
}

func TestCreateDoctor_AssignsTenantAndPaletteColor(t *testing.T) {
	f := newFixture(t)
	resp := f.h.CreateDoctor(ctx(f.own, 1, map[string]any{"full_name": "  Ana Pérez ", "specialty": "cardiology"}, 0))
	wantCode(t, resp, http.StatusOK)
	d := resp.Data.(doctor_models.Doctor)
	if d.TenantID != f.own.ID || d.FullName != "Ana Pérez" || !d.Active || d.Color != doctor_data.Palette[0] {
		t.Fatalf("unexpected doctor %+v", d)
	}
	second := f.h.CreateDoctor(ctx(f.own, 1, map[string]any{"full_name": "Luis", "specialty": "cardiology"}, 0)).Data.(doctor_models.Doctor)
	if second.Color != doctor_data.Palette[1] {
		t.Fatalf("second color = %s, want %s", second.Color, doctor_data.Palette[1])
	}
}

func TestCreateDoctor_ValidatesSpecialtyAndColor(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct {
		name string
		body map[string]any
		err  core_errors.AppError
	}{
		{"unknown specialty", map[string]any{"full_name": "A", "specialty": "astrology"}, core_errors.ErrDoctorInvalidSpecialty},
		{"other without name", map[string]any{"full_name": "A", "specialty": "other"}, core_errors.ErrDoctorInvalidSpecialty},
		{"bad color", map[string]any{"full_name": "A", "specialty": "cardiology", "color": "red"}, core_errors.ErrDoctorInvalidColor},
		{"missing name", map[string]any{"specialty": "cardiology"}, core_errors.ErrDoctorInvalidRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wantError(t, f.h.CreateDoctor(ctx(f.own, 1, tc.body, 0)), http.StatusBadRequest, tc.err)
		})
	}
	resp := f.h.CreateDoctor(ctx(f.own, 1, map[string]any{"full_name": "A", "specialty": "other", "specialty_other": "Geriatría"}, 0))
	wantCode(t, resp, http.StatusOK)
	if got := resp.Data.(doctor_models.Doctor).SpecialtyOther; got != "Geriatría" {
		t.Fatalf("specialty_other = %q", got)
	}
}

func TestCreateDoctor_UserMustBelongToTenantAndBeUnlinked(t *testing.T) {
	f := newFixture(t)
	ownUser := f.member(f.own, "own")
	otherUser := f.member(f.other, "other")

	wantError(t, f.h.CreateDoctor(ctx(f.own, 1, map[string]any{"full_name": "A", "specialty": "cardiology", "user_id": otherUser}, 0)),
		http.StatusBadRequest, core_errors.ErrDoctorUserNotInTenant)

	wantCode(t, f.h.CreateDoctor(ctx(f.own, 1, map[string]any{"full_name": "A", "specialty": "cardiology", "user_id": ownUser}, 0)), http.StatusOK)
	wantError(t, f.h.CreateDoctor(ctx(f.own, 1, map[string]any{"full_name": "B", "specialty": "cardiology", "user_id": ownUser}, 0)),
		http.StatusConflict, core_errors.ErrDoctorUserAlreadyLinked)

	// The same user may have a profile in another tenant it belongs to.
	f.mustCreate(&user_models.Environment{UserID: ownUser, CompanyID: companyOf(t, f, f.other)})
	wantCode(t, f.h.CreateDoctor(ctx(f.other, 1, map[string]any{"full_name": "A", "specialty": "cardiology", "user_id": ownUser}, 0)), http.StatusOK)
}

func companyOf(t *testing.T, f *fixture, tenant tenant_models.Tenant) uint {
	t.Helper()
	var company company_models.Company
	if err := f.db.Where("tenant_id = ?", tenant.ID).First(&company).Error; err != nil {
		t.Fatal(err)
	}
	return company.ID
}

func TestDoctorUniqueIndex_RejectsSecondProfileForUser(t *testing.T) {
	f := newFixture(t)
	userID := f.member(f.own, "u")
	f.doctor(f.own, "A", &userID)
	dup := doctor_models.Doctor{TenantID: f.own.ID, UserID: &userID, FullName: "B", Specialty: "cardiology"}
	if err := f.db.Create(&dup).Error; err == nil {
		t.Fatal("expected unique violation for a second profile of the same user in a tenant")
	}
	// Several doctors without account are fine.
	f.doctor(f.own, "C", nil)
	f.doctor(f.own, "D", nil)
}

func TestGetDoctors_ActiveFilterAndTenantIsolation(t *testing.T) {
	f := newFixture(t)
	active := f.doctor(f.own, "Activo", nil)
	inactive := f.doctor(f.own, "Inactivo", nil)
	f.db.Model(&inactive).Update("active", false)
	f.doctor(f.other, "Ajeno", nil)

	c := ctx(f.own, 1, nil, 0)
	all := f.h.GetDoctors(c).Data.([]doctor_models.Doctor)
	if len(all) != 2 {
		t.Fatalf("got %d doctors, want 2 (other tenant must be hidden)", len(all))
	}
	c = ctx(f.own, 1, nil, 0)
	c.Request.URL.RawQuery = "active=true"
	only := f.h.GetDoctors(c).Data.([]doctor_models.Doctor)
	if len(only) != 1 || only[0].ID != active.ID {
		t.Fatalf("active filter returned %+v", only)
	}
	c = ctx(f.own, 1, nil, 0)
	c.Request.URL.RawQuery = "active=maybe"
	wantError(t, f.h.GetDoctors(c), http.StatusBadRequest, core_errors.ErrDoctorInvalidRequest)
}

func TestDoctorHandlers_CannotReachOtherTenant(t *testing.T) {
	f := newFixture(t)
	foreign := f.doctor(f.other, "Ajeno", nil)
	body := map[string]any{"full_name": "Hacked"}
	for name, call := range map[string]func(*gin.Context) envelope.Response{
		"get":        f.h.GetDoctor,
		"update":     f.h.UpdateDoctor,
		"link":       f.h.LinkUser,
		"deactivate": f.h.DeactivateDoctor,
		"activate":   f.h.ActivateDoctor,
		"delete":     f.h.DeleteDoctor,
	} {
		t.Run(name, func(t *testing.T) {
			wantError(t, call(ctx(f.own, 1, body, foreign.ID)), http.StatusNotFound, core_errors.ErrDoctorNotFound)
		})
	}
	got := f.reload(foreign.ID)
	if got.FullName != "Ajeno" || !got.Active {
		t.Fatalf("other tenant's doctor changed: %+v", got)
	}
}

func TestUpdateDoctor_ChangesProfileOnlyAndClearsReview(t *testing.T) {
	f := newFixture(t)
	userID := f.member(f.own, "u")
	d := f.doctor(f.own, "A", &userID)
	f.db.Model(&d).Update("needs_review", true)

	body := map[string]any{"full_name": "Ana", "professional_registry": "MSP-1", "color": "#16a34a", "active": false, "user_id": nil}
	wantCode(t, f.h.UpdateDoctor(ctx(f.own, 1, body, d.ID)), http.StatusOK)
	got := f.reload(d.ID)
	if got.FullName != "Ana" || got.ProfessionalRegistry != "MSP-1" || got.Color != "#16A34A" || got.Specialty != "pediatrics" {
		t.Fatalf("profile not updated: %+v", got)
	}
	if !got.Active || got.UserID == nil || *got.UserID != userID || got.NeedsReview {
		t.Fatalf("active/link must not change and review must clear: %+v", got)
	}
}

func TestDeactivateActivateAndLink(t *testing.T) {
	f := newFixture(t)
	d := f.doctor(f.own, "A", nil)
	wantCode(t, f.h.DeactivateDoctor(ctx(f.own, 1, nil, d.ID)), http.StatusOK)
	if f.reload(d.ID).Active {
		t.Fatal("still active")
	}
	wantCode(t, f.h.ActivateDoctor(ctx(f.own, 1, nil, d.ID)), http.StatusOK)
	if !f.reload(d.ID).Active {
		t.Fatal("still inactive")
	}

	userID := f.member(f.own, "u")
	wantCode(t, f.h.LinkUser(ctx(f.own, 1, map[string]any{"user_id": userID}, d.ID)), http.StatusOK)
	if got := f.reload(d.ID).UserID; got == nil || *got != userID {
		t.Fatalf("user not linked: %v", got)
	}
	other := f.doctor(f.own, "B", nil)
	wantError(t, f.h.LinkUser(ctx(f.own, 1, map[string]any{"user_id": userID}, other.ID)), http.StatusConflict, core_errors.ErrDoctorUserAlreadyLinked)
	wantCode(t, f.h.LinkUser(ctx(f.own, 1, map[string]any{"user_id": nil}, d.ID)), http.StatusOK)
	if got := f.reload(d.ID).UserID; got != nil {
		t.Fatalf("user not unlinked: %v", *got)
	}
}

// referenceRow is a stand-in for a table that gains doctor_id in PR2.
type referenceRow struct {
	ID       uint
	DoctorID *uint
}

func TestDeleteDoctor_OnlyWhenUnreferenced(t *testing.T) {
	f := newFixture(t)
	if err := f.db.Table("doctor_reference_rows").AutoMigrate(&referenceRow{}); err != nil {
		t.Fatal(err)
	}
	restore := doctor_services.ReferencingTables
	doctor_services.ReferencingTables = append([]string{"doctor_reference_rows"}, restore...)
	t.Cleanup(func() { doctor_services.ReferencingTables = restore })

	used := f.doctor(f.own, "Usado", nil)
	free := f.doctor(f.own, "Libre", nil)
	if err := f.db.Table("doctor_reference_rows").Create(map[string]any{"doctor_id": used.ID}).Error; err != nil {
		t.Fatal(err)
	}

	wantError(t, f.h.DeleteDoctor(ctx(f.own, 1, nil, used.ID)), http.StatusConflict, core_errors.ErrDoctorInUse)
	wantCode(t, f.h.DeleteDoctor(ctx(f.own, 1, nil, free.ID)), http.StatusOK)
	var n int64
	f.db.Unscoped().Model(&doctor_models.Doctor{}).Where("id = ?", free.ID).Count(&n)
	if n != 0 {
		t.Fatal("unreferenced doctor must be hard-deleted")
	}
}

func TestMyDoctor_OwnProfileRules(t *testing.T) {
	f := newFixture(t)
	userID := f.member(f.own, "me")

	wantError(t, f.h.GetMyDoctor(ctx(f.own, userID, nil, 0)), http.StatusNotFound, core_errors.ErrDoctorProfileNotFound)
	wantError(t, f.h.UpdateMyDoctor(ctx(f.own, userID, map[string]any{"full_name": "X"}, 0)), http.StatusNotFound, core_errors.ErrDoctorProfileNotFound)

	created := f.h.CreateMyDoctor(ctx(f.own, userID, map[string]any{"full_name": "Yo", "specialty": "dermatology", "user_id": 999}, 0))
	wantCode(t, created, http.StatusOK)
	d := created.Data.(doctor_models.Doctor)
	if d.UserID == nil || *d.UserID != userID {
		t.Fatalf("own profile must link the caller, got %v", d.UserID)
	}
	wantError(t, f.h.CreateMyDoctor(ctx(f.own, userID, map[string]any{"full_name": "Yo", "specialty": "dermatology"}, 0)),
		http.StatusConflict, core_errors.ErrDoctorProfileExists)

	f.db.Model(&doctor_models.Doctor{}).Where("id = ?", d.ID).Update("active", false)
	otherUser := f.member(f.own, "x")
	body := map[string]any{"full_name": "Dra. Yo", "phone": "099", "active": true, "user_id": otherUser}
	wantCode(t, f.h.UpdateMyDoctor(ctx(f.own, userID, body, 0)), http.StatusOK)
	got := f.reload(d.ID)
	if got.FullName != "Dra. Yo" || got.Phone != "099" {
		t.Fatalf("own fields not updated: %+v", got)
	}
	if got.Active || got.UserID == nil || *got.UserID != userID {
		t.Fatalf("own update must not change active or link: %+v", got)
	}

	// A user with no profile can't edit someone else's through /me.
	wantError(t, f.h.UpdateMyDoctor(ctx(f.own, otherUser, map[string]any{"full_name": "Z"}, 0)), http.StatusNotFound, core_errors.ErrDoctorProfileNotFound)
	// The profile is per tenant.
	wantError(t, f.h.GetMyDoctor(ctx(f.other, userID, nil, 0)), http.StatusNotFound, core_errors.ErrDoctorProfileNotFound)
}

func TestGetStatusAndMarkReviewed(t *testing.T) {
	f := newFixture(t)
	userID := f.member(f.own, "me")
	mine := f.doctor(f.own, "Yo", &userID)
	inactive := f.doctor(f.own, "Inactivo", nil)
	f.db.Model(&inactive).Updates(map[string]any{"active": false, "needs_review": true})
	f.doctor(f.other, "Ajeno", nil)

	status := f.h.GetStatus(ctx(f.own, userID, nil, 0)).Data.(doctor_dto.DoctorStatusResponse)
	if !status.HasProfile || status.Doctor == nil || status.Doctor.ID != mine.ID || status.ActiveDoctors != 1 || !status.NeedsReview {
		t.Fatalf("unexpected status %+v", status)
	}

	wantCode(t, f.h.MarkReviewed(ctx(f.own, userID, nil, 0)), http.StatusOK)
	status = f.h.GetStatus(ctx(f.own, 9999, nil, 0)).Data.(doctor_dto.DoctorStatusResponse)
	if status.HasProfile || status.Doctor != nil || status.NeedsReview {
		t.Fatalf("unexpected status after review %+v", status)
	}
}

func TestGetSpecialties_ListsCatalogWithLabelKeys(t *testing.T) {
	f := newFixture(t)
	list := f.h.GetSpecialties(ctx(f.own, 1, nil, 0)).Data.([]doctor_dto.SpecialtyResponse)
	if len(list) != len(doctor_data.Specialties) || list[0].LabelKey != "doctors.specialty.general_medicine" {
		t.Fatalf("unexpected catalog %+v", list[:1])
	}
}

type sqlStateErr string

func (e sqlStateErr) Error() string    { return "pg error" }
func (e sqlStateErr) SQLState() string { return string(e) }

func TestIsUniqueViolation(t *testing.T) {
	f := newFixture(t)
	userID := f.member(f.own, "u")
	f.doctor(f.own, "A", &userID)
	err := f.db.Create(&doctor_models.Doctor{TenantID: f.own.ID, UserID: &userID, FullName: "B", Specialty: "cardiology"}).Error
	if err == nil || !isUniqueViolation(err) {
		t.Fatalf("duplicate insert error not recognized: %v", err)
	}
	if !isUniqueViolation(fmt.Errorf("wrapped: %w", sqlStateErr("23505"))) {
		t.Fatal("postgres 23505 not recognized")
	}
	if isUniqueViolation(sqlStateErr("23503")) {
		t.Fatal("foreign key violation is not a unique violation")
	}
}
