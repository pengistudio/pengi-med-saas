package subscription_middleware

import (
	"net/http"
	"pengi-med-saas/core/tenantdb"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	user_models "pengi-med-saas/features/users/models"
)

const ContextKeyAllowedPermissions = "allowed_permissions"

// SubscriptionMiddleware verifies that the tenant has an active subscription
// and loads the allowed permission IDs from the plan into the Gin context.
// Must run after TenantMiddleware (requires "tenant_id" in context).
func SubscriptionMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetUint("tenant_id")
		if tenantID == 0 {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrTenantNotFound))
			return
		}

		// 1. Find company by tenant_id
		var company company_models.Company
		if err := tenantdb.For(c, db).First(&company).Error; err != nil {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusForbidden, "company.not_found", core_errors.ErrCompanyNotFound))
			return
		}

		// 2. Find active subscription within grace period (3 days after expiry)
		var subscription company_models.Subscription
		err := company_services.ActiveSubscriptionQuery(db, company.ID).
			Preload("Plan.Features.Permissions").
			First(&subscription).Error

		if err != nil {
			// E-BO-007 on a 403 is what the web app reads as "subscription expired".
			envelope.Abort(c, envelope.ErrorResponse(http.StatusForbidden, "subscription.error.inactive", core_errors.ErrBackofficeSubscriptionNotFound))
			return
		}

		// 3. Collect allowed permission IDs from plan features into a set
		allowed := make(map[string]bool)
		for _, feature := range subscription.Plan.Features {
			for _, perm := range feature.Permissions {
				allowed[perm.ID] = true
			}
		}

		c.Set(ContextKeyAllowedPermissions, allowed)
		c.Next()
	}
}

// GetAllowedPermissions retrieves the subscription permission set from context.
func GetAllowedPermissions(c *gin.Context) map[string]bool {
	val, exists := c.Get(ContextKeyAllowedPermissions)
	if !exists {
		return nil
	}
	perms, _ := val.(map[string]bool)
	return perms
}

// IsPermissionAllowed checks if a specific permission ID is within the subscription's plan.
func IsPermissionAllowed(c *gin.Context, permissionID string) bool {
	perms := GetAllowedPermissions(c)
	if perms == nil {
		return false
	}
	return perms[permissionID]
}

// RequirePermission returns a middleware that checks:
// 1. The permission is included in the active subscription's plan.
// 2. The authenticated user's role (for this company) has the permission.
// Must run after AuthMiddleware, TenantMiddleware, and SubscriptionMiddleware.
func RequirePermission(db *gorm.DB, permissionID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check subscription allows this permission
		if !IsPermissionAllowed(c, permissionID) {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusForbidden, "subscription.error.feature_not_included", core_errors.ErrPermissionNotInPlan))
			return
		}

		checkRolePermission(db, permissionID)(c)
	}
}

// RequireRolePermission returns a middleware that checks ONLY that the
// authenticated user's role (for this company) has the given permission —
// unlike RequirePermission, it does NOT gate on the tenant's subscription
// plan. Use this for basic account administration (e.g. team management)
// that should never be a paid-tier-gated feature.
// Must run after AuthMiddleware and TenantMiddleware.
func RequireRolePermission(db *gorm.DB, permissionID string) gin.HandlerFunc {
	return checkRolePermission(db, permissionID)
}

// checkRolePermission is the shared role-permission lookup used by both
// RequirePermission and RequireRolePermission.
func checkRolePermission(db *gorm.DB, permissionID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TenantMiddleware already verified the caller holds this Environment (their
		// role in the tenant's company); only its permissions are checked here.
		environmentID := c.GetUint("environment_id")
		var env user_models.Environment
		if environmentID == 0 || db.Preload("Role.Permissions").First(&env, environmentID).Error != nil {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusForbidden, "tenant.error.forbidden", core_errors.ErrTenantForbidden))
			return
		}

		// Check role has the required permission
		for _, perm := range env.Role.Permissions {
			if perm.ID == permissionID {
				c.Next()
				return
			}
		}

		envelope.Abort(c, envelope.ErrorResponse(http.StatusForbidden, "permission.error.insufficient", core_errors.ErrPermissionDenied))
	}
}
