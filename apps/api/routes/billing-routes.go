package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	billing_handlers "pengi-med-saas/features/billing/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterBillingRoutes(router *gin.RouterGroup, db *gorm.DB) {
	invoiceHandler := billing_handlers.NewInvoiceHandler(db, logger.Log)

	billingGroup := router.Group("/billing", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))

	billingGroup.POST("/invoices", envelope.Handle(invoiceHandler.CreateInvoice))
	billingGroup.GET("/invoices", envelope.Handle(invoiceHandler.GetAllInvoices))
	billingGroup.DELETE("/invoices/:id", envelope.Handle(invoiceHandler.DeleteInvoiceByID))

	// RabbitMQ Jobs
	billingGroup.POST("/invoices/:id/sri/process", envelope.Handle(invoiceHandler.SRIInvoiceProcessing))
	billingGroup.POST("/invoices/sri/process-batch", envelope.Handle(invoiceHandler.MultipleSRIInvoiceProcessing))
}
