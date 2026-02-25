package tenant_middleware

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	tenant_models "pengi-med-saas/features/tenants/models"

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

		c.Set("tenant_id", tenant.ID)
		c.Next()
	}
}

func TenantScope(c *gin.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tenantID := c.GetUint("tenant_id")
		return db.Where("tenant_id = ?", tenantID)
	}
}
