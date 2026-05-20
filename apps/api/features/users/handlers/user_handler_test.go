package user_handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/auth"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"
	"go.uber.org/zap"
)

func TestLogin_Success(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{})
	logger := zap.NewNop()

	// Create a test user with hashed password
	hashedPassword, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	testUser := user_models.User{
		UserName:      "testuser",
		Email:         "test@example.com",
		Password:      hashedPassword,
		EmailVerified: true,
	}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create handler and test context
	handler := NewUserHandler(db, logger)
	c, w := testutils.NewGinContext(1, int64(testUser.ID))

	// Create login request
	loginPayload := user_dto.LoginDTO{
		UserName: "testuser",
		Password: "password123",
	}
	body, _ := json.Marshal(loginPayload)
	c.Request = httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	response := handler.Login(c)

	// Verify response
	if response.Code != 200 {
		t.Errorf("expected status 200, got %d", response.Code)
	}

	// Verify response data is a map and contains required fields
	// Convert to JSON and back to get consistent map type
	respBytes, _ := json.Marshal(response.Data)
	var respData map[string]interface{}
	json.Unmarshal(respBytes, &respData)

	if _, hasToken := respData["token"]; !hasToken {
		t.Errorf("expected 'token' in response, got: %v", respData)
	}
	if _, hasExchangeToken := respData["exchange_token"]; !hasExchangeToken {
		t.Errorf("expected 'exchange_token' in response, got: %v", respData)
	}
	if _, hasUserID := respData["user_id"]; !hasUserID {
		t.Errorf("expected 'user_id' in response, got: %v", respData)
	}

	// Verify refresh token cookie was set
	if len(w.Result().Cookies()) == 0 {
		t.Errorf("expected refresh token cookie to be set")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{})
	logger := zap.NewNop()

	// Create a test user with hashed password
	hashedPassword, err := auth.HashPassword("correctpassword")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	testUser := user_models.User{
		UserName: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
	}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	handler := NewUserHandler(db, logger)
	c, _ := testutils.NewGinContext(1, 0)

	// Login with wrong password
	loginPayload := user_dto.LoginDTO{
		UserName: "testuser",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginPayload)
	c.Request = httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.Login(c)

	// Expect 401 Unauthorized
	if response.Code != 401 {
		t.Errorf("expected status 401, got %d", response.Code)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{})
	logger := zap.NewNop()

	handler := NewUserHandler(db, logger)
	c, _ := testutils.NewGinContext(1, 0)

	// Login with non-existent user
	loginPayload := user_dto.LoginDTO{
		UserName: "nonexistent",
		Password: "somepassword",
	}
	body, _ := json.Marshal(loginPayload)
	c.Request = httptest.NewRequest("POST", "/login", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.Login(c)

	// Expect 401 Unauthorized (not 404, per design)
	if response.Code != 401 {
		t.Errorf("expected status 401, got %d", response.Code)
	}
}

func TestLogin_MissingBody(t *testing.T) {
	db := testutils.SetupTestDB(t, &user_models.User{})
	logger := zap.NewNop()

	handler := NewUserHandler(db, logger)
	c, _ := testutils.NewGinContext(1, 0)

	// Send empty body
	c.Request = httptest.NewRequest("POST", "/login", bytes.NewReader([]byte{}))
	c.Request.Header.Set("Content-Type", "application/json")

	response := handler.Login(c)

	// Expect 400 Bad Request
	if response.Code != 400 {
		t.Errorf("expected status 400, got %d", response.Code)
	}
}
