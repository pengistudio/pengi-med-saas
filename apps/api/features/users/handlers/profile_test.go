package user_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestUpdateProfile_EmailChangeNeedsCurrentPassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.Role{}, &user_models.Environment{}, &company_models.Company{})
	handler := NewProfileHandler(db, zap.NewNop())
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")
	createResetUser(t, db, "dr.perez", "perez@example.com", "other-password")

	role := user_models.Role{Role: "ADMIN"}
	db.Create(&role)
	company := company_models.Company{LegalName: "Clinica", TradeName: "Clinica", TenantID: 1}
	db.Create(&company)
	env := user_models.Environment{UserID: user.ID, Name: "Clinica", RoleID: role.ID, CompanyID: company.ID}
	db.Create(&env)

	call := func(payload map[string]string) int {
		t.Helper()
		body, _ := json.Marshal(payload)
		c, _ := testutils.NewGinContext(1, int64(user.ID))
		c.Set("user_id", int64(user.ID)) // what AuthMiddleware sets
		c.Request = httptest.NewRequest("PUT", fmt.Sprintf("/users/profile?environment_id=%d", env.ID), bytes.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return handler.UpdateProfile(c).Code
	}
	storedEmail := func() string {
		var got user_models.User
		db.First(&got, user.ID)
		return got.Email
	}

	if code := call(map[string]string{"email": "new@example.com"}); code != 401 {
		t.Fatalf("missing password: expected 401, got %d", code)
	}
	if code := call(map[string]string{"email": "new@example.com", "current_password": "wrong"}); code != 401 {
		t.Fatalf("wrong password: expected 401, got %d", code)
	}
	if storedEmail() != "lopez@example.com" {
		t.Fatal("email changed without the current password")
	}

	// Environment-name-only edits, and the same email in another case, need no password.
	if code := call(map[string]string{"environment_name": "Consultorio"}); code != 200 {
		t.Fatalf("env name only: expected 200, got %d", code)
	}
	if code := call(map[string]string{"email": "LOPEZ@example.com", "environment_name": "Consultorio 2"}); code != 200 {
		t.Fatalf("unchanged email: expected 200, got %d", code)
	}

	if code := call(map[string]string{"email": "not-an-email", "current_password": "old-password-1"}); code != 400 {
		t.Fatalf("invalid email: expected 400, got %d", code)
	}
	if code := call(map[string]string{"email": "PEREZ@example.com", "current_password": "old-password-1"}); code != 409 {
		t.Fatalf("taken email: expected 409, got %d", code)
	}

	if code := call(map[string]string{"email": "New@Example.com", "current_password": "old-password-1"}); code != 200 {
		t.Fatalf("change: expected 200, got %d", code)
	}
	if got := storedEmail(); got != "new@example.com" {
		t.Fatalf("stored email = %q, want new@example.com", got)
	}
}
