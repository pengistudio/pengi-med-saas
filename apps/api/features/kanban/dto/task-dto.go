package kanban_dto

import "time"

type CreateTaskRequest struct {
	Title         string  `json:"title" binding:"required"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	DueDate       *string `json:"due_date"`
	CreatedByName string  `json:"created_by_name"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	DueDate     *string `json:"due_date"`
}

type MoveTaskRequest struct {
	Status   string `json:"status" binding:"required"`
	Position int    `json:"position"`
}

type TaskResponse struct {
	ID            uint       `json:"id"`
	TenantID      uint       `json:"tenant_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	Position      int        `json:"position"`
	DueDate       *string    `json:"due_date"`
	CreatedByName string     `json:"created_by_name"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TasksByStatusResponse struct {
	Todo       []TaskResponse `json:"todo"`
	InProgress []TaskResponse `json:"in_progress"`
	Done       []TaskResponse `json:"done"`
}
