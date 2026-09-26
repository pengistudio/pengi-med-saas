package tenant_middleware

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	tenant_models "pengi-med-saas/features/tenants/models"
	auth_middleware "pengi-med-saas/features/users/middleware"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TenantMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.GetHeader("X-Tenant-Slug")

		if slug == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, envelope.ErrorResponse(http.StatusBadRequest, "X-Tenant-Slug header is missing", core_errors.ErrTenantNotFound))
			return
		}

		var tenant tenant_models.Tenant
		if err := db.Where("slug = ?", slug).First(&tenant).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, envelope.ErrorResponse(http.StatusNotFound, "Tenant not found", core_errors.ErrTenantNotFound))
			return
		}

		// The slug comes from a client header: only trust it for users who hold
		// a role (Environment) in the tenant's company.
		userID, _, ok := auth_middleware.GetUserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.ErrorResponse(http.StatusUnauthorized, "User not authenticated", core_errors.ErrAuthInvalidRequest))
			return
		}
		var membership struct {
			EnvironmentID uint
			CompanyID     uint
		}
		err := db.Model(&user_models.Environment{}).
			Select("environments.id AS environment_id, environments.company_id AS company_id").
			Joins("JOIN companies ON companies.id = environments.company_id AND companies.deleted_at IS NULL").
			Where("companies.tenant_id = ? AND environments.user_id = ?", tenant.ID, userID).
			Take(&membership).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, envelope.ErrorResponse(http.StatusForbidden, "tenant.error.forbidden", core_errors.ErrTenantForbidden))
			return
		}

		c.Set("tenant_id", tenant.ID)
		c.Set("company_id", membership.CompanyID)
		c.Set("environment_id", membership.EnvironmentID)
		c.Next()
	}
}

func TenantScope(c *gin.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID := c.GetUint("tenant_id")
		userID, _, _ := auth_middleware.GetUserFromContext(c)

		return db.Set("audit_tenant_id", tenantID).
			Set("audit_user_id", userID).
			Where("tenant_id = ?", tenantID)
	}
}

// AuditScope inyecta información de auditoría sin aplicar condicionales WHERE.
// Útil para db.Create()
func AuditScope(c *gin.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID := c.GetUint("tenant_id")
		userID, _, _ := auth_middleware.GetUserFromContext(c)

		return db.Set("audit_tenant_id", tenantID).
			Set("audit_user_id", userID)
	}
}
