package kanban_models

import (
	"gorm.io/gorm"
	"time"
)

type Task struct {
	gorm.Model
	TenantID      uint       `gorm:"index" json:"tenant_id"`
	Title         string     `gorm:"not null" json:"title"`
	Description   string     `json:"description"`
	Status        string     `gorm:"type:varchar(20);default:'todo'" json:"status"`
	Position      int        `json:"position"`
	DueDate       *time.Time `json:"due_date"`
	CreatedByName string     `json:"created_by_name"`
	ArchivedAt    *time.Time `json:"archived_at"`
}

func (Task) TableName() string {
	return "kanban_tasks"
}
