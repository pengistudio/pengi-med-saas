package user_handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"pengi-med-saas/core/auth"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// loginAndGetCookie logs a test user in through the real handler and returns
// the raw refresh_token cookie value the response set.
func loginAndGetCookie(t *testing.T, handler *UserHandler, username string) string {
	t.Helper()
	c, w := testutils.NewGinContext(1, 0)
	payload := user_dto.LoginDTO{UserName: username, Password: "password123"}
	body, _ := json.Marshal(payload)
	c.Request = httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	resp := handler.Login(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("login failed: %+v", resp)
	}

	for _, ck := range w.Result().Cookies() {
		if ck.Name == "refresh_token" {
			return ck.Value
		}
	}
	t.Fatal("refresh_token cookie not set on login response")
	return ""
}

// ginContextWithRefreshCookie builds a context whose incoming request
// carries the given refresh_token cookie, for calls into RotateRefreshToken
// / Logout which read it via c.Cookie(...).
func ginContextWithRefreshCookie(cookieValue string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: cookieValue})
	return c, w
}

func TestLogin_CreatesRefreshFamily(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	logger := zap.NewNop()

	hashedPassword, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	testUser := user_models.User{UserName: "familyuser", Email: "family@example.com", Password: hashedPassword, EmailVerified: true}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	handler := NewUserHandler(db, logger)
	loginAndGetCookie(t, handler, "familyuser")

	var rows []user_models.RefreshToken
	if err := db.Where("user_id = ?", testUser.ID).Find(&rows).Error; err != nil {
		t.Fatalf("failed to query refresh tokens: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 refresh token row after login, got %d", len(rows))
	}
	if rows[0].UsedAt != nil || rows[0].RevokedAt != nil {
		t.Errorf("expected a fresh live row, got UsedAt=%v RevokedAt=%v", rows[0].UsedAt, rows[0].RevokedAt)
	}
	if rows[0].FamilyID == "" {
		t.Error("expected non-empty FamilyID")
	}
}

func TestRotateRefreshToken_RotatesAndSlides(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	logger := zap.NewNop()

	hashedPassword, _ := auth.HashPassword("password123")
	testUser := user_models.User{UserName: "rotateuser", Email: "rotate@example.com", Password: hashedPassword, EmailVerified: true}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	handler := NewUserHandler(db, logger)
	oldCookie := loginAndGetCookie(t, handler, "rotateuser")

	var oldRow user_models.RefreshToken
	if err := db.Where("user_id = ?", testUser.ID).First(&oldRow).Error; err != nil {
		t.Fatalf("failed to load original row: %v", err)
	}

	c, _ := ginContextWithRefreshCookie(oldCookie)
	accessToken, err := RotateRefreshToken(db, oldCookie, c, logger)
	if err != nil {
		t.Fatalf("expected rotation to succeed, got %v", err)
	}
	if accessToken == "" {
		t.Error("expected a non-empty access token")
	}

	var refreshedOldRow user_models.RefreshToken
	if err := db.First(&refreshedOldRow, oldRow.ID).Error; err != nil {
		t.Fatalf("failed to reload old row: %v", err)
	}
	if refreshedOldRow.UsedAt == nil {
		t.Error("expected old row to be marked used after rotation")
	}
	if refreshedOldRow.ReplacedByID == nil {
		t.Error("expected old row to have ReplacedByID set")
	}

	var allRows []user_models.RefreshToken
	db.Where("user_id = ?", testUser.ID).Find(&allRows)
	if len(allRows) != 2 {
		t.Fatalf("expected 2 rows (old + new) after one rotation, got %d", len(allRows))
	}

	var newRow user_models.RefreshToken
	for _, r := range allRows {
		if r.ID != oldRow.ID {
			newRow = r
		}
	}
	if newRow.UsedAt != nil {
		t.Error("expected new row to be live (unused)")
	}
	if !newRow.ExpiresAt.After(oldRow.ExpiresAt) {
		t.Error("expected new row's ExpiresAt to be pushed further out (sliding window)")
	}
	if newRow.FamilyID != oldRow.FamilyID {
		t.Error("expected rotation to stay within the same family")
	}
}

func TestRotateRefreshToken_ReuseDetected(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	logger := zap.NewNop()

	hashedPassword, _ := auth.HashPassword("password123")
	testUser := user_models.User{UserName: "reuseuser", Email: "reuse@example.com", Password: hashedPassword, EmailVerified: true}
	db.Create(&testUser)

	handler := NewUserHandler(db, logger)
	tokenA := loginAndGetCookie(t, handler, "reuseuser")

	c1, _ := ginContextWithRefreshCookie(tokenA)
	if _, err := RotateRefreshToken(db, tokenA, c1, logger); err != nil {
		t.Fatalf("first rotation should succeed: %v", err)
	}

	// Replay the already-rotated-out token A.
	c2, _ := ginContextWithRefreshCookie(tokenA)
	_, err := RotateRefreshToken(db, tokenA, c2, logger)
	if err != ErrRefreshTokenReused {
		t.Fatalf("expected ErrRefreshTokenReused, got %v", err)
	}

	var rows []user_models.RefreshToken
	db.Where("user_id = ?", testUser.ID).Find(&rows)
	for _, r := range rows {
		if r.RevokedAt == nil {
			t.Errorf("expected every row in the family to be revoked after reuse detection, row %d is not", r.ID)
		}
	}
}

func TestRotateRefreshToken_RevokedFamilyRejected(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	logger := zap.NewNop()

	hashedPassword, _ := auth.HashPassword("password123")
	testUser := user_models.User{UserName: "revokeduser", Email: "revoked@example.com", Password: hashedPassword, EmailVerified: true}
	db.Create(&testUser)

	handler := NewUserHandler(db, logger)
	tokenA := loginAndGetCookie(t, handler, "revokeduser")

	c1, w1 := ginContextWithRefreshCookie(tokenA)
	if _, err := RotateRefreshToken(db, tokenA, c1, logger); err != nil {
		t.Fatalf("first rotation should succeed: %v", err)
	}

	// Grab the newly-rotated cookie (token B) from the response.
	var tokenB string
	for _, ck := range w1.Result().Cookies() {
		if ck.Name == "refresh_token" {
			tokenB = ck.Value
		}
	}
	if tokenB == "" {
		t.Fatal("expected a rotated refresh_token cookie to be set on the response")
	}

	// Force reuse detection by replaying token A again -> revokes whole family.
	c2, _ := ginContextWithRefreshCookie(tokenA)
	if _, err := RotateRefreshToken(db, tokenA, c2, logger); err != ErrRefreshTokenReused {
		t.Fatalf("expected reuse detection, got %v", err)
	}

	// Now even the legitimately-latest token (B) must be rejected as revoked.
	c3, _ := ginContextWithRefreshCookie(tokenB)
	if _, err := RotateRefreshToken(db, tokenB, c3, logger); err != ErrRefreshTokenRevoked {
		t.Fatalf("expected ErrRefreshTokenRevoked for the last live token in a revoked family, got %v", err)
	}
}

func TestLogout_RevokesFamily(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	logger := zap.NewNop()

	hashedPassword, _ := auth.HashPassword("password123")
	testUser := user_models.User{UserName: "logoutuser", Email: "logout@example.com", Password: hashedPassword, EmailVerified: true}
	db.Create(&testUser)

	handler := NewUserHandler(db, logger)
	token := loginAndGetCookie(t, handler, "logoutuser")

	c, _ := ginContextWithRefreshCookie(token)
	resp := handler.Logout(c)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected logout to succeed, got %+v", resp)
	}

	c2, _ := ginContextWithRefreshCookie(token)
	if _, err := RotateRefreshToken(db, token, c2, logger); err != ErrRefreshTokenRevoked {
		t.Fatalf("expected refresh after logout to fail as revoked, got %v", err)
	}
}

func TestRotateRefreshToken_LegacyTokenMigrates(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	logger := zap.NewNop()

	hashedPassword, _ := auth.HashPassword("password123")
	testUser := user_models.User{UserName: "legacyuser", Email: "legacy@example.com", Password: hashedPassword, EmailVerified: true}
	db.Create(&testUser)

	// Hand-craft a pre-rotation refresh token: no familyId/jti claims,
	// same shape/secret the old GenerateRefreshToken used to produce.
	secret := os.Getenv("AUTH_KEY")
	legacyClaims := jwt.MapClaims{
		"username": testUser.UserName,
		"userId":   testUser.ID,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	legacyToken := jwt.NewWithClaims(jwt.SigningMethodHS256, legacyClaims)
	legacyRaw, err := legacyToken.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign legacy token: %v", err)
	}

	c, _ := ginContextWithRefreshCookie(legacyRaw)
	accessToken, err := RotateRefreshToken(db, legacyRaw, c, logger)
	if err != nil {
		t.Fatalf("expected legacy token to be upgraded, not rejected, got %v", err)
	}
	if accessToken == "" {
		t.Error("expected a non-empty access token for legacy upgrade")
	}

	var rows []user_models.RefreshToken
	db.Where("user_id = ?", testUser.ID).Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 new refresh token row created for the legacy upgrade, got %d", len(rows))
	}
}

func TestRotateRefreshToken_ConcurrentReplayOnlyOneWins(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{}, &user_models.RefreshToken{})
	// In-memory SQLite hands out a fresh, empty database per connection
	// unless pinned to a single one — force serialization so both
	// goroutines actually race against the same schema/data.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	logger := zap.NewNop()

	hashedPassword, _ := auth.HashPassword("password123")
	testUser := user_models.User{UserName: "raceuser", Email: "race@example.com", Password: hashedPassword, EmailVerified: true}
	db.Create(&testUser)

	handler := NewUserHandler(db, logger)
	token := loginAndGetCookie(t, handler, "raceuser")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c, _ := ginContextWithRefreshCookie(token)
			_, err := RotateRefreshToken(db, token, c, logger)
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	successes, reused := 0, 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else if err == ErrRefreshTokenReused {
			reused++
		}
	}
	if successes != 1 {
		t.Errorf("expected exactly 1 successful rotation out of 2 concurrent attempts, got %d (errs=%v)", successes, errs)
	}
	if reused != 1 {
		t.Errorf("expected the losing concurrent attempt to be treated as reuse, got %d (errs=%v)", reused, errs)
	}
}
