package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	cash_handlers "pengi-med-saas/features/cash/handlers"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterCashRoutes mounts the cash desk (caja): plan + role, Feature CASH.
func RegisterCashRoutes(router *gin.RouterGroup, db *gorm.DB) {
	h := cash_handlers.NewCashHandler(db, logger.Log, documentRenderer())

	group := router.Group("/cash",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)
	rp := subscription_middleware.RequirePermission

	// Sessions
	group.GET("/sessions/current", rp(db, "READ_CASH"), envelope.Handle(h.GetCurrentSession))
	group.GET("/sessions", rp(db, "READ_CASH"), envelope.Handle(h.GetSessions))
	group.GET("/sessions/:id", rp(db, "READ_CASH"), envelope.Handle(h.GetSession))
	group.GET("/sessions/:id/report", rp(db, "READ_CASH"), h.DownloadSessionReport)
	group.POST("/sessions", rp(db, "MANAGE_CASH_SESSION"), envelope.Handle(h.OpenSession))
	group.POST("/sessions/:id/close", rp(db, "MANAGE_CASH_SESSION"), envelope.Handle(h.CloseSession))

	// Payments
	group.GET("/payments", rp(db, "READ_CASH"), envelope.Handle(h.GetPayments))
	group.POST("/payments", rp(db, "CREATE_PAYMENT"), envelope.Handle(h.CreatePayment))
	group.POST("/payments/:id/void", rp(db, "VOID_PAYMENT"), envelope.Handle(h.VoidPayment))
	group.POST("/payments/:id/refund", rp(db, "VOID_PAYMENT"), envelope.Handle(h.RefundPayment))

	// Movements: cash in needs MANAGE_CASH_SESSION, cash out VOID_PAYMENT
	// (checked in the handler, it depends on the payload).
	group.POST("/movements", subscription_middleware.RequireAnyPermission(db, "MANAGE_CASH_SESSION", "VOID_PAYMENT"), envelope.Handle(h.CreateMovement))
}
