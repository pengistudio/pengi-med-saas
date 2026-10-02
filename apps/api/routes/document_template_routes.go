package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	document_template_handlers "pengi-med-saas/features/document-templates/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterDocumentTemplateRoutes exposes the tenant's custom templates for
// printable documents. Every route needs MANAGE_DOCUMENT_TEMPLATES; a document
// of a plan module the tenant lacks answers 404.
func RegisterDocumentTemplateRoutes(router *gin.RouterGroup, db *gorm.DB) {
	h := document_template_handlers.NewDocumentTemplateHandler(db, logger.Log, documentRenderer())

	group := router.Group("/document-templates",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)

	rp := subscription_middleware.RequirePermission

	group.GET("", rp(db, "MANAGE_DOCUMENT_TEMPLATES"), envelope.Handle(h.ListDocumentTemplates))
	group.GET("/:doc/default", rp(db, "MANAGE_DOCUMENT_TEMPLATES"), h.DownloadDefaultTemplate)
	group.GET("/:doc/custom", rp(db, "MANAGE_DOCUMENT_TEMPLATES"), h.DownloadCustomTemplate)
	group.PUT("/:doc", rp(db, "MANAGE_DOCUMENT_TEMPLATES"), envelope.Handle(h.UploadDocumentTemplate))
	group.DELETE("/:doc", rp(db, "MANAGE_DOCUMENT_TEMPLATES"), envelope.Handle(h.DeleteDocumentTemplate))
	group.POST("/:doc/preview", rp(db, "MANAGE_DOCUMENT_TEMPLATES"), h.PreviewDocumentTemplate)
}
