package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	signature_handlers "pengi-med-saas/features/signatures/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterSignatureRoutes(router *gin.RouterGroup, db *gorm.DB) {
	handler := signature_handlers.NewSignatureHandler(db, logger.Log, tenantFiles)
	rp := subscription_middleware.RequirePermission

	group := router.Group("/signatures", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db), subscription_middleware.SubscriptionMiddleware(db))
	{
		group.GET("/me", rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(handler.GetMySignature))
		group.PUT("/me", rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(handler.UploadMySignature))
		group.DELETE("/me", rp(db, "SIGN_MEDICAL_DOCUMENT"), envelope.Handle(handler.DeleteMySignature))
	}
}
