package subscription_middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
	auth_middleware "pengi-med-saas/features/users/middleware"
)

const ContextKeyAllowedPermissions = "allowed_permissions"

// SubscriptionMiddleware verifies that the tenant has an active subscription
// and loads the allowed permission IDs from the plan into the Gin context.
// Must run after TenantMiddleware (requires "tenant_id" in context).
func SubscriptionMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.GetUint("tenant_id")
		if tenantID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(
				http.StatusUnauthorized, "Tenant not resolved", core_errors.ErrTenantNotFound,
			))
			return
		}

		// 1. Find company by tenant_id
		var company company_models.Company
		if err := db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(
				http.StatusForbidden, "No company found for this tenant", core_errors.ErrCompanyNotFound,
			))
			return
		}

		// 2. Find active subscription within grace period (3 days after expiry)
		const gracePeriodDays = 3
		graceDeadline := time.Now().Add(-gracePeriodDays * 24 * time.Hour)
		var subscription company_models.Subscription
		err := db.
			Where("company_id = ? AND status = ? AND expires_at > ?", company.ID, "active", graceDeadline).
			Preload("Plan.Features.Permissions").
			First(&subscription).Error

		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(
				http.StatusForbidden, "No active subscription found", core_errors.ErrBackofficeSubscriptionNotFound,
			))
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
			c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(
				http.StatusForbidden, "Your plan does not include this feature", core_errors.ErrPermissionGetError,
			))
			return
		}

		// 2. Find company for this tenant
		tenantID := c.GetUint("tenant_id")
		var company company_models.Company
		if err := db.Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(
				http.StatusForbidden, "Company not found", core_errors.ErrCompanyNotFound,
			))
			return
		}

		// 3. Find user's Environment for this company
		userID, _, ok := auth_middleware.GetUserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(
				http.StatusUnauthorized, "User not authenticated", core_errors.ErrAuthInvalidRequest,
			))
			return
		}

		var env user_models.Environment
		if err := db.
			Where("user_id = ? AND company_id = ?", userID, company.ID).
			Preload("Role.Permissions").
			First(&env).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(
				http.StatusForbidden, "User has no role in this company", core_errors.ErrPermissionGetError,
			))
			return
		}

		// 4. Check role has the required permission
		for _, perm := range env.Role.Permissions {
			if perm.ID == permissionID {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(
			http.StatusForbidden, "Insufficient permissions", core_errors.ErrPermissionGetError,
		))
	}
}
