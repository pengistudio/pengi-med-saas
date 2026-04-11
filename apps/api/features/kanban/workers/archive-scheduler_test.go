package kanban_workers_test

import (
	"testing"
	"time"

	kanban_workers "pengi-med-saas/features/kanban/workers"
)

func TestGetDelayDuration(t *testing.T) {
	scheduler := &kanban_workers.ArchiveScheduler{}

	tests := []struct {
		name     string
		delay    string
		expected time.Duration
	}{
		{"1 day", "1_day", 24 * time.Hour},
		{"1 week", "1_week", 7 * 24 * time.Hour},
		{"2 weeks", "2_weeks", 14 * 24 * time.Hour},
		{"1 month", "1_month", 30 * 24 * time.Hour},
		{"never", "never", 0},
		{"unknown", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.GetDelayDuration(tt.delay)
			if result != tt.expected {
				t.Errorf("GetDelayDuration(%s) = %v, want %v", tt.delay, result, tt.expected)
			}
		})
	}
}

func TestSchedulerArchivesExpiredTasks(t *testing.T) {
	// Integration test - requires database setup
	t.Skip("requires test database setup")
}

func TestSchedulerSkipsNeverArchive(t *testing.T) {
	// Integration test - requires database setup
	t.Skip("requires test database setup")
}

func TestSchedulerSkipsNonDoneTasks(t *testing.T) {
	// Integration test - requires database setup
	t.Skip("requires test database setup")
}
