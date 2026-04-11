package routes

import (
	"pengi-med-saas/core/envelope"
	"pengi-med-saas/core/logger"
	subscription_middleware "pengi-med-saas/features/companies/middleware"
	kanban_handlers "pengi-med-saas/features/kanban/handlers"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterKanbanRoutes(router *gin.RouterGroup, db *gorm.DB) {
	kanbanHandler := kanban_handlers.NewKanbanHandler(db, logger.Log)

	kanbanGroup := router.Group("/kanban",
		auth_middleware.AuthMiddleware(),
		tenant_middleware.TenantMiddleware(db),
		subscription_middleware.SubscriptionMiddleware(db),
	)

	rp := subscription_middleware.RequirePermission

	// GET all tasks
	kanbanGroup.GET("/tasks", rp(db, "READ_KANBAN"), envelope.Handle(kanbanHandler.GetTasks))

	// POST create task
	kanbanGroup.POST("/tasks", rp(db, "CREATE_KANBAN"), envelope.Handle(kanbanHandler.CreateTask))

	// PUT update task details
	kanbanGroup.PUT("/tasks/:id", rp(db, "UPDATE_KANBAN"), envelope.Handle(kanbanHandler.UpdateTask))

	// PUT move task (status + position)
	kanbanGroup.PUT("/tasks/:id/move", rp(db, "UPDATE_KANBAN"), envelope.Handle(kanbanHandler.MoveTask))

	// DELETE task
	kanbanGroup.DELETE("/tasks/:id", rp(db, "DELETE_KANBAN"), envelope.Handle(kanbanHandler.DeleteTask))
}
