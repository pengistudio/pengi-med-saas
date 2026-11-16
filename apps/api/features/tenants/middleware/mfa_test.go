package tenant_middleware

import (
	"encoding/json"
	"testing"

	company_models "pengi-med-saas/features/companies/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
)

// newMFAFixture is the membership fixture plus the user's row (users,
// user_passkeys) and a role in the other clinic too, so the policy of one
// clinic can be checked against the other.
func newMFAFixture(t *testing.T) membershipFixture {
	t.Helper()
	f := newMembershipFixture(t)
	if err := f.db.AutoMigrate(&user_models.User{}, &user_models.Passkey{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	user := user_models.User{UserName: "user", Email: "user@clinic.ec"}
	user.ID = uint(f.userID)
	if err := f.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	var otherCompany company_models.Company
	if err := f.db.Where("tenant_id = ?", f.other.ID).First(&otherCompany).Error; err != nil {
		t.Fatalf("other company: %v", err)
	}
	env := user_models.Environment{UserID: uint(f.userID), Name: "Other", RoleID: f.environment.RoleID, CompanyID: otherCompany.ID}
	if err := f.db.Create(&env).Error; err != nil {
		t.Fatalf("create other environment: %v", err)
	}
	return f
}

func (f membershipFixture) requireMFA(t *testing.T, tenant tenant_models.Tenant) {
	t.Helper()
	if err := f.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenant.ID).Update("require_mfa", true).Error; err != nil {
		t.Fatalf("require mfa: %v", err)
	}
}

func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Data struct {
			ErrorCode string `json:"error_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
	return resp.Data.ErrorCode
}

func TestTenantMiddleware_RequireMFA_RejectsUserWithoutSecondFactor(t *testing.T) {
	f := newMFAFixture(t)
	f.requireMFA(t, f.own)

	c, w := f.request("own-clinic")

	if !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
	if got := errorCode(t, w.Body.Bytes()); got != "E-AUTH-026" {
		t.Fatalf("error_code = %q, want E-AUTH-026", got)
	}
	if _, exists := c.Get("tenant_id"); exists {
		t.Fatalf("tenant_id was set for a user the policy rejects")
	}
}

func TestTenantMiddleware_RequireMFA_AllowsTwoFactorUser(t *testing.T) {
	f := newMFAFixture(t)
	f.requireMFA(t, f.own)
	if err := f.db.Model(&user_models.User{}).Where("id = ?", f.userID).Update("two_factor_enabled", true).Error; err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}

	if c, w := f.request("own-clinic"); c.IsAborted() {
		t.Fatalf("a user with 2FA was rejected: %d %s", w.Code, w.Body.String())
	}
}

func TestTenantMiddleware_RequireMFA_AllowsPasskeyUser(t *testing.T) {
	f := newMFAFixture(t)
	f.requireMFA(t, f.own)
	passkey := user_models.Passkey{UserID: uint(f.userID), Name: "laptop", CredentialID: []byte("cred-1"), Credential: []byte("{}")}
	if err := f.db.Create(&passkey).Error; err != nil {
		t.Fatalf("create passkey: %v", err)
	}

	if c, w := f.request("own-clinic"); c.IsAborted() {
		t.Fatalf("a user with a passkey was rejected: %d %s", w.Code, w.Body.String())
	}
}

// A setup started but not confirmed (secret stored, flag off) is no second
// factor yet.
func TestTenantMiddleware_RequireMFA_PendingTwoFactorSetupIsNotEnough(t *testing.T) {
	f := newMFAFixture(t)
	f.requireMFA(t, f.own)
	if err := f.db.Model(&user_models.User{}).Where("id = ?", f.userID).Update("two_factor_secret", []byte("sealed")).Error; err != nil {
		t.Fatalf("pending setup: %v", err)
	}

	if c, w := f.request("own-clinic"); !c.IsAborted() || w.Code != 403 {
		t.Fatalf("aborted=%v code=%d, want aborted with 403", c.IsAborted(), w.Code)
	}
}

func TestTenantMiddleware_PolicyOff_AllowsUserWithoutSecondFactor(t *testing.T) {
	f := newMFAFixture(t)

	if c, w := f.request("own-clinic"); c.IsAborted() {
		t.Fatalf("policy off rejected the user: %d %s", w.Code, w.Body.String())
	}
}

// One clinic's policy doesn't follow the user into another clinic.
func TestTenantMiddleware_RequireMFA_OtherTenantUnaffected(t *testing.T) {
	f := newMFAFixture(t)
	f.requireMFA(t, f.own)

	if c, _ := f.request("own-clinic"); !c.IsAborted() {
		t.Fatalf("the clinic requiring MFA let the user in")
	}
	c, w := f.request("other-clinic")
	if c.IsAborted() {
		t.Fatalf("the clinic without the policy rejected the user: %d %s", w.Code, w.Body.String())
	}
	if got := c.GetUint("tenant_id"); got != f.other.ID {
		t.Fatalf("tenant_id = %d, want %d", got, f.other.ID)
	}
}
