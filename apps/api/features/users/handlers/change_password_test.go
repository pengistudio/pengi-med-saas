package user_handlers

import (
	"go.uber.org/zap"
	"pengi-med-saas/core/auth"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
	"testing"
)

func TestChangePassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	handler := NewUserHandler(db, zap.NewNop())
	user := createResetUser(t, db, "dra.lopez", "lopez@example.com", "old-password-1")

	call := func(current, next string) int {
		c := postJSON(t, map[string]string{"current_password": current, "new_password": next})
		c.Set("user_id", int64(user.ID))
		return handler.ChangePassword(c).Code
	}

	if code := call("wrong-password", "new-password-123"); code != 401 {
		t.Fatalf("wrong current: expected 401, got %d", code)
	}
	if code := call("old-password-1", "old-password-1"); code != 400 {
		t.Fatalf("same password: expected 400, got %d", code)
	}
	if code := call("old-password-1", "short"); code != 400 {
		t.Fatalf("too short: expected 400, got %d", code)
	}
	if code := call("old-password-1", "new-password-123"); code != 200 {
		t.Fatalf("change: expected 200, got %d", code)
	}

	var got user_models.User
	db.First(&got, user.ID)
	if !auth.CompareHashAndPassword(got.Password, "new-password-123") {
		t.Fatal("new password not stored")
	}
}
