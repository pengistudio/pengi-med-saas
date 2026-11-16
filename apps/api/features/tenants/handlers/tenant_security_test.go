package tenant_handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantfiles"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func newSecurityHandler(t *testing.T) (*TenantHandler, tenant_models.Tenant, user_models.User) {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &user_models.User{}, &user_models.Passkey{})
	h := NewTenantHandler(db, zap.NewNop(), tenantfiles.Disk(t.TempDir()))
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("security-%d", now), DisplayToken: fmt.Sprintf("tok-security-%d", now)}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	admin := user_models.User{UserName: "admin", Email: fmt.Sprintf("admin-%d@clinic.ec", now)}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return h, tenant, admin
}

func putSecurity(t *testing.T, h *TenantHandler, tenantID, userID uint, body string) (int, any) {
	t.Helper()
	c, _ := testutils.NewGinContext(tenantID, int64(userID))
	c.Set("user_id", int64(userID))
	c.Set("username", "admin")
	c.Request = httptest.NewRequest("PUT", "/tenants/security", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	resp := h.UpdateSecuritySettings(c)
	return resp.Code, resp.Data
}

func storedRequireMFA(t *testing.T, h *TenantHandler, tenantID uint) bool {
	t.Helper()
	var tenant tenant_models.Tenant
	if err := h.db.First(&tenant, tenantID).Error; err != nil {
		t.Fatalf("reload tenant: %v", err)
	}
	return tenant.RequireMFA
}

func TestUpdateSecuritySettings_RefusesAdminWithoutSecondFactor(t *testing.T) {
	h, tenant, admin := newSecurityHandler(t)

	code, data := putSecurity(t, h, tenant.ID, admin.ID, `{"require_mfa":true}`)

	if code != http.StatusConflict {
		t.Fatalf("code = %d, want 409", code)
	}
	if appErr, ok := data.(core_errors.AppError); !ok || appErr.ErrorCode != core_errors.ErrTenantMFARequiresOwnMFA.ErrorCode {
		t.Fatalf("data = %#v, want %s", data, core_errors.ErrTenantMFARequiresOwnMFA.ErrorCode)
	}
	if storedRequireMFA(t, h, tenant.ID) {
		t.Fatalf("the policy was turned on anyway")
	}
}

func TestUpdateSecuritySettings_AdminWithTwoFactorTurnsItOnAndOff(t *testing.T) {
	h, tenant, admin := newSecurityHandler(t)
	if err := h.db.Model(&user_models.User{}).Where("id = ?", admin.ID).Update("two_factor_enabled", true).Error; err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}

	if code, _ := putSecurity(t, h, tenant.ID, admin.ID, `{"require_mfa":true}`); code != http.StatusOK || !storedRequireMFA(t, h, tenant.ID) {
		t.Fatalf("turn on = %d, stored %v", code, storedRequireMFA(t, h, tenant.ID))
	}
	if code, _ := putSecurity(t, h, tenant.ID, admin.ID, `{"require_mfa":false}`); code != http.StatusOK || storedRequireMFA(t, h, tenant.ID) {
		t.Fatalf("turn off = %d, stored %v", code, storedRequireMFA(t, h, tenant.ID))
	}
}

func TestUpdateSecuritySettings_AdminWithPasskeyTurnsItOn(t *testing.T) {
	h, tenant, admin := newSecurityHandler(t)
	passkey := user_models.Passkey{UserID: admin.ID, Name: "phone", CredentialID: []byte(fmt.Sprintf("cred-%d", admin.ID)), Credential: []byte("{}")}
	if err := h.db.Create(&passkey).Error; err != nil {
		t.Fatalf("create passkey: %v", err)
	}

	if code, _ := putSecurity(t, h, tenant.ID, admin.ID, `{"require_mfa":true}`); code != http.StatusOK || !storedRequireMFA(t, h, tenant.ID) {
		t.Fatalf("turn on = %d", code)
	}
}

// Turning it off never needs a second factor, and a body without the field is
// rejected instead of read as false.
func TestUpdateSecuritySettings_OffAlwaysAllowedAndFieldRequired(t *testing.T) {
	h, tenant, admin := newSecurityHandler(t)
	if err := h.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenant.ID).Update("require_mfa", true).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	if code, _ := putSecurity(t, h, tenant.ID, admin.ID, `{}`); code != http.StatusBadRequest || !storedRequireMFA(t, h, tenant.ID) {
		t.Fatalf("empty body = %d, stored %v", code, storedRequireMFA(t, h, tenant.ID))
	}
	if code, _ := putSecurity(t, h, tenant.ID, admin.ID, `{"require_mfa":false}`); code != http.StatusOK || storedRequireMFA(t, h, tenant.ID) {
		t.Fatalf("turn off = %d", code)
	}
}
