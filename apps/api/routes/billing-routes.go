package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	billing_handlers "pengi-med-saas/features/billing/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterBillingRoutes(router *gin.RouterGroup, db *gorm.DB) {
	invoiceHandler := billing_handlers.NewInvoiceHandler(db, logger.Log)
	catalogItemHandler := billing_handlers.NewCatalogItemHandler(db, logger.Log)

	billingGroup := router.Group("/billing", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db), subscription_middleware.SubscriptionMiddleware(db))

	rp := subscription_middleware.RequirePermission

	// Invoices
	billingGroup.POST("/invoices", rp(db, "CREATE_BILLING"), envelope.Handle(invoiceHandler.CreateInvoice))
	billingGroup.GET("/invoices", rp(db, "READ_BILLING"), envelope.Handle(invoiceHandler.GetAllInvoices))
	billingGroup.DELETE("/invoices/:id", rp(db, "DELETE_BILLING"), envelope.Handle(invoiceHandler.DeleteInvoiceByID))

	// Catalog Items
	billingGroup.POST("/catalog-items", rp(db, "CREATE_BILLING"), envelope.Handle(catalogItemHandler.CreateCatalogItem))
	billingGroup.GET("/catalog-items", rp(db, "READ_BILLING"), envelope.Handle(catalogItemHandler.GetAllCatalogItems))
	billingGroup.GET("/catalog-items/:id", rp(db, "READ_BILLING"), envelope.Handle(catalogItemHandler.GetCatalogItemByID))
	billingGroup.PUT("/catalog-items/:id", rp(db, "UPDATE_BILLING"), envelope.Handle(catalogItemHandler.UpdateCatalogItem))
	billingGroup.DELETE("/catalog-items/:id", rp(db, "DELETE_BILLING"), envelope.Handle(catalogItemHandler.DeleteCatalogItem))

	// RabbitMQ Jobs
	billingGroup.POST("/invoices/:id/sri/process", rp(db, "MANAGE_SRI_SETTINGS"), envelope.Handle(invoiceHandler.SRIInvoiceProcessing))
	billingGroup.POST("/invoices/sri/process-batch", rp(db, "MANAGE_SRI_SETTINGS"), envelope.Handle(invoiceHandler.MultipleSRIInvoiceProcessing))
}
