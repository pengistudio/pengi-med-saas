package kanban_workers

import (
	"encoding/json"
	"time"

	kanban_models "pengi-med-saas/features/kanban/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ArchiveScheduler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewArchiveScheduler(db *gorm.DB, logger *zap.Logger) *ArchiveScheduler {
	return &ArchiveScheduler{
		db:     db,
		logger: logger,
	}
}

// Start begins the scheduler loop that runs every 5 minutes
func (s *ArchiveScheduler) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.archiveExpiredTasks()
	}
}

// archiveExpiredTasks finds tasks in "done" status past their archive delay and archives them
func (s *ArchiveScheduler) archiveExpiredTasks() {
	var tasks []kanban_models.Task

	// Find all tasks in "done" status that are not yet archived
	if err := s.db.Where("status = ? AND archived_at IS NULL", "done").Find(&tasks).Error; err != nil {
		s.logger.Error("failed to fetch done tasks for archival", zap.Error(err))
		return
	}

	now := time.Now()
	archivedCount := 0

	for _, task := range tasks {
		// Load tenant settings
		var tenant tenant_models.Tenant
		if err := s.db.First(&tenant, task.TenantID).Error; err != nil {
			s.logger.Warn("tenant not found for task", zap.Uint("task_id", task.ID), zap.Error(err))
			continue
		}

		// Parse tenant settings
		var settings tenant_models.UISettings
		if tenant.UISettings != "" && tenant.UISettings != "{}" {
			if err := json.Unmarshal([]byte(tenant.UISettings), &settings); err != nil {
				s.logger.Warn("failed to parse UISettings", zap.Uint("tenant_id", tenant.ID), zap.Error(err))
				settings = tenant_models.DefaultUISettings()
			}
		} else {
			settings = tenant_models.DefaultUISettings()
		}

		// Check if auto-archive is enabled
		if settings.Kanban.AutoArchiveDelay == "never" {
			continue
		}

		// Calculate archive time
		delay := s.GetDelayDuration(settings.Kanban.AutoArchiveDelay)
		archiveTime := task.UpdatedAt.Add(delay)

		// Archive if delay has passed
		if now.After(archiveTime) {
			if err := s.db.Model(&task).Update("archived_at", now).Error; err != nil {
				s.logger.Error("failed to archive task", zap.Uint("task_id", task.ID), zap.Error(err))
				continue
			}
			archivedCount++
		}
	}

	if archivedCount > 0 {
		s.logger.Info("archived expired tasks", zap.Int("count", archivedCount))
	}
}

// GetDelayDuration converts delay string to duration
func (s *ArchiveScheduler) GetDelayDuration(delay string) time.Duration {
	switch delay {
	case "1_day":
		return 24 * time.Hour
	case "1_week":
		return 7 * 24 * time.Hour
	case "2_weeks":
		return 14 * 24 * time.Hour
	case "1_month":
		return 30 * 24 * time.Hour
	default:
		return 0
	}
}
