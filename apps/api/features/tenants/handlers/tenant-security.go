package tenant_handlers

import (
	"net/http"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	tenant_models "pengi-med-saas/features/tenants/models"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecuritySettings is the clinic's security policy. RequireMFA makes
// two-step verification mandatory for every user (enforced by
// TenantMiddleware).
type SecuritySettings struct {
	RequireMFA *bool `json:"require_mfa" binding:"required"`
}

// UpdateSecuritySettings turns the clinic's two-step verification
// requirement on or off. Turning it on is refused (409) while the admin
// doing it has no second factor themselves: they would lock themselves out
// (and maybe the clinic, if they are its only admin) on the next request.
func (h *TenantHandler) UpdateSecuritySettings(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound)
	}
	userID, _, ok := auth_middleware.GetUserFromContext(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrAuthInvalidRequest)
	}

	var body SecuritySettings
	if err := c.ShouldBindJSON(&body); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrInvalidRequest)
	}
	requireMFA := *body.RequireMFA

	if requireMFA {
		hasMFA, err := tenant_middleware.HasMFA(h.db, uint(userID))
		if err != nil {
			h.logger.Error("Failed to check the admin's second factor", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
		if !hasMFA {
			return envelope.ErrorResponse(http.StatusConflict, "tenant.security.require_mfa.own_mfa_missing", core_errors.ErrTenantMFARequiresOwnMFA)
		}
	}

	// tenants is the tenant table itself (no tenant_id column): by primary key.
	if err := h.db.Model(&tenant_models.Tenant{}).Where("id = ?", tenantID).Update("require_mfa", requireMFA).Error; err != nil {
		h.logger.Error("Failed to save the clinic's security settings", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(SecuritySettings{RequireMFA: &requireMFA}, "tenant.security.update.success")
}
