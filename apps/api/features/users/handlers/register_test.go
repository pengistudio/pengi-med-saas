package user_handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/auth"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func registerPayload(t *testing.T, company, username, email string) []byte {
	t.Helper()
	token, err := auth.GenerateCompanyRegisterToken()
	if err != nil {
		t.Fatalf("failed to generate register token: %v", err)
	}
	return registerPayloadWithToken(token, company, username, email)
}

func registerPayloadWithToken(token, company, username, email string) []byte {
	body, _ := json.Marshal(user_dto.SelfRegisterDTO{
		Token:       token,
		CompanyName: company,
		Username:    username,
		Email:       email,
		Password:    "password123",
	})
	return body
}

func TestRegister_ReusesCanonicalAdminRole(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&user_models.User{}, &user_models.Environment{}, &user_models.Role{},
		&permission_models.Permission{}, &company_models.Company{}, &company_models.Subscription{},
		&tenant_models.Tenant{},
		// Signup seeds the new tenant's exam catalog.
		&clinical_models.ExamCatalogItem{}, &clinical_models.ExamProfile{},
	)
	logger := zap.NewNop()

	// Seed the canonical admin role like the real seeding migration does.
	adminRole := user_models.Role{Role: role_data.RoleAdmin}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("failed to seed canonical admin role: %v", err)
	}

	handler := NewUserHandler(db, logger)

	// First registration.
	c1, w1 := testutils.NewGinContext(0, 0)
	c1.Request = httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerPayload(t, "Clinic One", "clinicone", "one@example.com")))
	c1.Request.Header.Set("Content-Type", "application/json")
	resp1 := handler.Register(c1)
	if resp1.Code != 201 {
		t.Fatalf("expected 201, got %d (w=%v)", resp1.Code, w1.Body.String())
	}

	// Second registration (different company).
	c2, w2 := testutils.NewGinContext(0, 0)
	c2.Request = httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerPayload(t, "Clinic Two", "clinictwo", "two@example.com")))
	c2.Request.Header.Set("Content-Type", "application/json")
	resp2 := handler.Register(c2)
	if resp2.Code != 201 {
		t.Fatalf("expected 201, got %d (w=%v)", resp2.Code, w2.Body.String())
	}

	// No duplicate "admin" role should have been created.
	var adminRoles []user_models.Role
	if err := db.Where("role = ?", role_data.RoleAdmin).Find(&adminRoles).Error; err != nil {
		t.Fatalf("failed to query admin roles: %v", err)
	}
	if len(adminRoles) != 1 {
		t.Fatalf("expected exactly 1 admin role, got %d", len(adminRoles))
	}

	// Both new Environments should point at the same canonical role.
	var environments []user_models.Environment
	if err := db.Find(&environments).Error; err != nil {
		t.Fatalf("failed to query environments: %v", err)
	}
	if len(environments) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(environments))
	}
	for _, env := range environments {
		if env.RoleID != adminRole.ID {
			t.Errorf("expected environment %d to use canonical admin role %d, got %d", env.ID, adminRole.ID, env.RoleID)
		}
	}
}

func TestRegister_FailsWithoutSeededAdminRole(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&user_models.User{}, &user_models.Environment{}, &user_models.Role{},
		&permission_models.Permission{}, &company_models.Company{}, &company_models.Subscription{},
		&tenant_models.Tenant{},
	)
	logger := zap.NewNop()
	handler := NewUserHandler(db, logger)

	c, _ := testutils.NewGinContext(0, 0)
	c.Request = httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerPayload(t, "Clinic Three", "clinicthree", "three@example.com")))
	c.Request.Header.Set("Content-Type", "application/json")
	resp := handler.Register(c)

	if resp.Code != 500 {
		t.Fatalf("expected 500 when canonical admin role is missing, got %d", resp.Code)
	}
}

func TestRegister_RejectsMissingOrInvalidToken(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&user_models.User{}, &user_models.Environment{}, &user_models.Role{},
		&permission_models.Permission{}, &company_models.Company{}, &company_models.Subscription{},
		&tenant_models.Tenant{},
	)
	handler := NewUserHandler(db, zap.NewNop())

	signupToken, err := auth.GenerateCompanySignupToken(1, 0)
	if err != nil {
		t.Fatalf("failed to generate signup token: %v", err)
	}

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"missing token", "", 400},
		{"garbage token", "not-a-jwt", 401},
		{"company signup token", signupToken, 401},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := testutils.NewGinContext(0, 0)
			c.Request = httptest.NewRequest("POST", "/auth/register", bytes.NewReader(registerPayloadWithToken(tc.token, "Clinic Four", "clinicfour", "four@example.com")))
			c.Request.Header.Set("Content-Type", "application/json")
			resp := handler.Register(c)
			if resp.Code != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, resp.Code)
			}
		})
	}

	var count int64
	db.Model(&tenant_models.Tenant{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no tenant to be created, got %d", count)
	}
}
