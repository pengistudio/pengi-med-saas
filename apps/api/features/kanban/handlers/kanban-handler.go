package kanban_handlers

import (
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	kanban_dto "pengi-med-saas/features/kanban/dto"
	kanban_models "pengi-med-saas/features/kanban/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type KanbanHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewKanbanHandler(db *gorm.DB, logger *zap.Logger) *KanbanHandler {
	return &KanbanHandler{db: db, logger: logger}
}

// parseDateString safely parses an optional date string ("YYYY-MM-DD") to *time.Time.
func parseDateString(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// formatDatePtr formats a *time.Time to *string ("YYYY-MM-DD"), or nil.
func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

// toTaskResponse maps a kanban_models.Task to a TaskResponse DTO.
func toTaskResponse(task kanban_models.Task) kanban_dto.TaskResponse {
	return kanban_dto.TaskResponse{
		ID:            task.ID,
		TenantID:      task.TenantID,
		Title:         task.Title,
		Description:   task.Description,
		Status:        task.Status,
		Position:      task.Position,
		DueDate:       formatDatePtr(task.DueDate),
		CreatedByName: task.CreatedByName,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
	}
}

// GetTasks returns all tasks for the tenant, grouped by status.
func (h *KanbanHandler) GetTasks(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)

	var tasks []kanban_models.Task
	if err := h.db.Scopes(tenantScope).
		Where("archived_at IS NULL").
		Order("status, position").
		Find(&tasks).Error; err != nil {
		h.logger.Error("failed to fetch tasks", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.tasks.fetch.error", core_errors.ErrInternal)
	}

	grouped := kanban_dto.TasksByStatusResponse{
		Todo:       []kanban_dto.TaskResponse{},
		InProgress: []kanban_dto.TaskResponse{},
		Done:       []kanban_dto.TaskResponse{},
	}

	for _, task := range tasks {
		resp := toTaskResponse(task)
		switch task.Status {
		case "in_progress":
			grouped.InProgress = append(grouped.InProgress, resp)
		case "done":
			grouped.Done = append(grouped.Done, resp)
		default:
			grouped.Todo = append(grouped.Todo, resp)
		}
	}

	return envelope.SuccessResponse(grouped, "kanban.tasks.fetch.success")
}

// CreateTask creates a new task.
func (h *KanbanHandler) CreateTask(c *gin.Context) envelope.Response {
	var req kanban_dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.request", core_errors.ErrInvalidRequest)
	}

	tenantScope := tenant_middleware.TenantScope(c)
	tenantID := c.GetUint("tenant_id")

	status := "todo"
	if req.Status != "" {
		status = req.Status
	}

	// Get max position for the given status
	var maxPosition int
	h.db.Scopes(tenantScope).
		Model(&kanban_models.Task{}).
		Where("status = ?", status).
		Order("position DESC").
		Limit(1).
		Pluck("position", &maxPosition)

	dueDate, err := parseDateString(req.DueDate)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.due_date", core_errors.ErrInvalidRequest)
	}

	// Get creator name from authenticated user if not provided
	createdByName := req.CreatedByName
	if createdByName == "" {
		// Get user from context (set by auth middleware)
		if userName, exists := c.Get("username"); exists {
			if name, ok := userName.(string); ok {
				createdByName = name
			}
		}
	}

	task := kanban_models.Task{
		TenantID:      tenantID,
		Title:         req.Title,
		Description:   req.Description,
		Status:        status,
		Position:      maxPosition + 1,
		DueDate:       dueDate,
		CreatedByName: createdByName,
	}

	if err := h.db.Create(&task).Error; err != nil {
		h.logger.Error("failed to create task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.create.error", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(toTaskResponse(task), "kanban.task.create.success")
}

// UpdateTask updates task details (title, description, status, due_date).
func (h *KanbanHandler) UpdateTask(c *gin.Context) envelope.Response {
	taskID := c.Param("id")
	id, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.id", core_errors.ErrInvalidRequest)
	}

	var req kanban_dto.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.request", core_errors.ErrInvalidRequest)
	}

	tenantScope := tenant_middleware.TenantScope(c)

	var task kanban_models.Task
	if err := h.db.Scopes(tenantScope).First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return envelope.ErrorResponse(http.StatusNotFound, "kanban.task.not.found", core_errors.ErrTenantNotFound)
		}
		h.logger.Error("failed to fetch task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.fetch.error", core_errors.ErrInternal)
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.DueDate != nil {
		dueDate, err := parseDateString(req.DueDate)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.due_date", core_errors.ErrInvalidRequest)
		}
		task.DueDate = dueDate
	}

	if err := h.db.Save(&task).Error; err != nil {
		h.logger.Error("failed to update task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.update.error", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(toTaskResponse(task), "kanban.task.update.success")
}

// MoveTask updates task status and position (drag & drop).
func (h *KanbanHandler) MoveTask(c *gin.Context) envelope.Response {
	taskID := c.Param("id")
	id, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.id", core_errors.ErrInvalidRequest)
	}

	var req kanban_dto.MoveTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.request", core_errors.ErrInvalidRequest)
	}

	tenantScope := tenant_middleware.TenantScope(c)

	var task kanban_models.Task
	if err := h.db.Scopes(tenantScope).First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return envelope.ErrorResponse(http.StatusNotFound, "kanban.task.not.found", core_errors.ErrTenantNotFound)
		}
		h.logger.Error("failed to fetch task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.fetch.error", core_errors.ErrInternal)
	}

	oldStatus := task.Status
	if oldStatus != req.Status {
		// Decrease positions in old status column
		if err := h.db.Scopes(tenantScope).
			Model(&kanban_models.Task{}).
			Where("status = ? AND position > ?", oldStatus, task.Position).
			Update("position", gorm.Expr("position - 1")).Error; err != nil {
			h.logger.Error("failed to shift positions in old status", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.move.error", core_errors.ErrInternal)
		}

		// Increase positions in new status column at insertion point
		if err := h.db.Scopes(tenantScope).
			Model(&kanban_models.Task{}).
			Where("status = ? AND position >= ?", req.Status, req.Position).
			Update("position", gorm.Expr("position + 1")).Error; err != nil {
			h.logger.Error("failed to shift positions in new status", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.move.error", core_errors.ErrInternal)
		}
	} else if task.Position != req.Position {
		// Same column reordering
		if req.Position > task.Position {
			// Moving down: shift others up
			if err := h.db.Scopes(tenantScope).
				Model(&kanban_models.Task{}).
				Where("status = ? AND position > ? AND position <= ?", req.Status, task.Position, req.Position).
				Update("position", gorm.Expr("position - 1")).Error; err != nil {
				h.logger.Error("failed to shift positions down", zap.Error(err))
				return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.move.error", core_errors.ErrInternal)
			}
		} else {
			// Moving up: shift others down
			if err := h.db.Scopes(tenantScope).
				Model(&kanban_models.Task{}).
				Where("status = ? AND position >= ? AND position < ?", req.Status, req.Position, task.Position).
				Update("position", gorm.Expr("position + 1")).Error; err != nil {
				h.logger.Error("failed to shift positions up", zap.Error(err))
				return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.move.error", core_errors.ErrInternal)
			}
		}
	}

	task.Status = req.Status
	task.Position = req.Position
	if err := h.db.Save(&task).Error; err != nil {
		h.logger.Error("failed to move task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.move.error", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(toTaskResponse(task), "kanban.task.move.success")
}

// DeleteTask deletes a task and reorders remaining positions.
func (h *KanbanHandler) DeleteTask(c *gin.Context) envelope.Response {
	taskID := c.Param("id")
	id, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "kanban.task.invalid.id", core_errors.ErrInvalidRequest)
	}

	tenantScope := tenant_middleware.TenantScope(c)

	var task kanban_models.Task
	if err := h.db.Scopes(tenantScope).First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return envelope.ErrorResponse(http.StatusNotFound, "kanban.task.not.found", core_errors.ErrTenantNotFound)
		}
		h.logger.Error("failed to fetch task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.fetch.error", core_errors.ErrInternal)
	}

	// Shift positions after deletion
	if err := h.db.Scopes(tenantScope).
		Model(&kanban_models.Task{}).
		Where("status = ? AND position > ?", task.Status, task.Position).
		Update("position", gorm.Expr("position - 1")).Error; err != nil {
		h.logger.Error("failed to shift positions after deletion", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.delete.error", core_errors.ErrInternal)
	}

	if err := h.db.Delete(&task).Error; err != nil {
		h.logger.Error("failed to delete task", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.task.delete.error", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "kanban.task.delete.success")
}
