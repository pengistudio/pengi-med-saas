package routes

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/whatsapp"
	clinical_models "pengi-med-saas/features/clinical/models"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"
	whatsapp_handlers "pengi-med-saas/features/whatsapp/handlers"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// whatsappRouter mounts the WhatsApp session routes for a caller whose role
// holds exactly permissionIDs, on a plan with every WhatsApp permission.
func whatsappRouter(t *testing.T, permissionIDs []string) (*gin.Engine, whatsapp_models.WhatsAppConversation) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testutils.SetupTestDB(t, &permission_models.Permission{}, &user_models.Role{}, &user_models.Environment{},
		&tenant_models.Tenant{}, &clinical_models.Patient{}, &whatsapp_models.WhatsAppAccount{},
		&whatsapp_models.WhatsAppMessage{}, &whatsapp_models.WhatsAppConversation{}, &whatsapp_models.WhatsAppTemplate{})
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("wa-perm-%d", now), DisplayToken: fmt.Sprintf("tok-wa-perm-%d", now)}
	mustCreate(t, db, &tenant)
	conv := whatsapp_models.WhatsAppConversation{TenantID: tenant.ID, Phone: "593991234567"}
	mustCreate(t, db, &conv)

	role := user_models.Role{Role: fmt.Sprintf("role-%d", now)}
	for _, id := range permissionIDs {
		role.Permissions = append(role.Permissions, permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}, Name: id})
	}
	mustCreate(t, db, &role)
	env := user_models.Environment{UserID: 7, CompanyID: 1, RoleID: role.ID}
	mustCreate(t, db, &env)
	plan := map[string]bool{}
	for _, perm := range permission_data.WhatsAppPermissions {
		plan[perm.ID] = true
	}

	router := gin.New()
	group := router.Group("/whatsapp", func(c *gin.Context) {
		c.Set("tenant_id", tenant.ID)
		c.Set("environment_id", env.ID)
		c.Set(subscription_middleware.ContextKeyAllowedPermissions, plan)
	})
	h := whatsapp_handlers.NewWhatsAppHandler(db, zap.NewNop(), whatsapp.New("http://127.0.0.1:1", "v23.0", "", ""))
	registerWhatsAppTenantRoutes(group, db, h)
	return router, conv
}

func TestWhatsAppRoutes_InboxNeedsItsPermission(t *testing.T) {
	router, conv := whatsappRouter(t, []string{"MANAGE_WHATSAPP"})
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/whatsapp/conversations", ""},
		{http.MethodGet, "/whatsapp/conversations/unread-count", ""},
		{http.MethodGet, fmt.Sprintf("/whatsapp/conversations/%d", conv.ID), ""},
		{http.MethodGet, fmt.Sprintf("/whatsapp/conversations/%d/messages", conv.ID), ""},
		{http.MethodPost, fmt.Sprintf("/whatsapp/conversations/%d/messages", conv.ID), `{"body":"hola"}`},
		{http.MethodPost, fmt.Sprintf("/whatsapp/conversations/%d/read", conv.ID), ""},
		{http.MethodPut, fmt.Sprintf("/whatsapp/conversations/%d/patient", conv.ID), `{"patient_id":1}`},
		{http.MethodGet, "/whatsapp/templates", ""},
		{http.MethodPost, "/whatsapp/conversations/start", `{"patient_id":1,"template":"pengi_continuar_conversacion"}`},
		{http.MethodPost, fmt.Sprintf("/whatsapp/conversations/%d/template", conv.ID), `{"template":"pengi_continuar_conversacion"}`},
	} {
		if w := serve(router, tc.method, tc.path, tc.body); w.Code != http.StatusForbidden {
			t.Errorf("MANAGE_WHATSAPP only: %s %s = %d, want 403", tc.method, tc.path, w.Code)
		}
	}
	// The sent log and the monthly usage stay open to the settings card.
	for _, path := range []string{"/whatsapp/messages", "/whatsapp/usage"} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, w.Code)
		}
	}
}

func TestWhatsAppRoutes_RecepcionistaUsesInboxNotSettings(t *testing.T) {
	router, conv := whatsappRouter(t, role_data.RolePermissionMatrix[role_data.RoleRecepcionista])
	for _, path := range []string{"/whatsapp/conversations", "/whatsapp/conversations/unread-count",
		fmt.Sprintf("/whatsapp/conversations/%d", conv.ID), fmt.Sprintf("/whatsapp/conversations/%d/messages", conv.ID), "/whatsapp/messages",
		"/whatsapp/templates", "/whatsapp/usage"} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusOK {
			t.Errorf("recepcionista GET %s = %d, want 200: %s", path, w.Code, w.Body.String())
		}
	}
	if w := serve(router, http.MethodPost, fmt.Sprintf("/whatsapp/conversations/%d/read", conv.ID), ""); w.Code != http.StatusOK {
		t.Errorf("recepcionista mark read = %d", w.Code)
	}
	for _, path := range []string{"/whatsapp/account", "/whatsapp/config"} {
		if w := serve(router, http.MethodGet, path, ""); w.Code != http.StatusForbidden {
			t.Errorf("recepcionista GET %s = %d, want 403", path, w.Code)
		}
	}
}
