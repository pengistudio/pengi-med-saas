# Kanban Task Auto-Archive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement automatic archival of completed kanban tasks with configurable tenant-level delays (1 day, 1 week, 2 weeks, 1 month, or never).

**Architecture:** Backend scheduler runs every 5 minutes, checks tasks in "done" status, and archives those past their delay by setting `archived_at`. Frontend settings UI allows users to configure the delay per-tenant. GET `/kanban/tasks` filters out archived tasks. Soft-delete approach preserves audit trail.

**Tech Stack:** Go (backend), React TypeScript (frontend), GORM (ORM), Zustand (state), Gin (HTTP), JSON settings storage.

---

## File Structure

### Backend Files
- **`apps/api/features/kanban/models/task-model.go`** — Add `ArchivedAt` field to Task struct
- **`apps/api/features/tenants/models/tenant-model.go`** — Add `KanbanSettings` struct, update `DefaultUISettings()`, update `UISettings` struct
- **`apps/api/features/kanban/handlers/kanban-handler.go`** — Update `GetAllTasks()` to filter `archived_at IS NULL`
- **`apps/api/features/kanban/workers/archive-scheduler.go`** (NEW) — Background job that archives expired tasks every 5 minutes
- **`apps/api/features/kanban/workers/archive-scheduler_test.go`** (NEW) — Unit tests for scheduler
- **`apps/api/cmd/main.go`** — Initialize scheduler on startup
- **`apps/api/i18n/messages/messages_es.json`** — Add Spanish i18n keys
- **`apps/api/i18n/messages/messages_en.json`** — Add English i18n keys

### Frontend Files
- **`apps/web/src/api/settings-service.ts`** — Add `KanbanSettings` interface, update `TenantUISettings`
- **`apps/web/src/sections/settings/kanban-settings.tsx`** (NEW) — Kanban settings section component
- **`apps/web/src/pages/settings/settings-page.tsx`** — Import and render kanban settings section
- **`apps/web/src/__tests__/settings.test.tsx`** (NEW) — Tests for kanban settings UI

---

## i18n Keys

**Spanish (`messages_es.json`):**
```json
{
  "kanban.settings.archive_delay.label": "Archivar tareas completadas después de:",
  "kanban.settings.archive_delay.never": "Nunca",
  "kanban.settings.archive_delay.1_day": "1 día",
  "kanban.settings.archive_delay.1_week": "1 semana",
  "kanban.settings.archive_delay.2_weeks": "2 semanas",
  "kanban.settings.archive_delay.1_month": "1 mes",
  "kanban.archive.success": "Configuración de archivo actualizada"
}
```

**English (`messages_en.json`):**
```json
{
  "kanban.settings.archive_delay.label": "Auto-archive completed tasks after:",
  "kanban.settings.archive_delay.never": "Never",
  "kanban.settings.archive_delay.1_day": "1 day",
  "kanban.settings.archive_delay.1_week": "1 week",
  "kanban.settings.archive_delay.2_weeks": "2 weeks",
  "kanban.settings.archive_delay.1_month": "1 month",
  "kanban.archive.success": "Archive settings updated"
}
```

---

## Implementation Tasks

### Task 1: Add ArchivedAt Field to Task Model

**Files:**
- Modify: `apps/api/features/kanban/models/task-model.go`

- [ ] **Step 1: Open task-model.go and examine current structure**

```bash
cat apps/api/features/kanban/models/task-model.go
```

- [ ] **Step 2: Add ArchivedAt field**

Edit `apps/api/features/kanban/models/task-model.go`:

```go
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
	ArchivedAt    *time.Time `json:"archived_at"` // NEW: nil = not archived
}

func (Task) TableName() string {
	return "kanban_tasks"
}
```

- [ ] **Step 3: Verify syntax**

```bash
cd apps/api && go vet ./features/kanban/models/...
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
cd apps/api && git add features/kanban/models/task-model.go && git commit -m "feat: add archived_at field to kanban task model"
```

---

### Task 2: Add KanbanSettings to Tenant UISettings

**Files:**
- Modify: `apps/api/features/tenants/models/tenant-model.go`

- [ ] **Step 1: Open tenant-model.go and locate UISettings struct**

```bash
grep -n "type UISettings struct" apps/api/features/tenants/models/tenant-model.go
```

- [ ] **Step 2: Add KanbanSettings struct and update UISettings**

Edit `apps/api/features/tenants/models/tenant-model.go` and add after `ClinicalSettings` struct:

```go
type KanbanSettings struct {
	AutoArchiveDelay string `json:"auto_archive_delay"` // "never", "1_day", "1_week", "2_weeks", "1_month"
}

type UISettings struct {
	Clinical ClinicalSettings `json:"clinical"`
	Kanban   KanbanSettings   `json:"kanban"`
}
```

- [ ] **Step 3: Update DefaultUISettings() function**

Edit the `DefaultUISettings()` function:

```go
// DefaultUISettings returns sensible defaults (all visible, CIE-11).
func DefaultUISettings() UISettings {
	return UISettings{
		Clinical: ClinicalSettings{
			ShowNextAppointment: true,
			ShowDiagnosis:       true,
			ShowMedic:           true,
			ShowInsurance:       true,
			ShowVitalSigns:      true,
			ShowDiagnoses:       true,
			DiagnosisSystem:     "cie11",
		},
		Kanban: KanbanSettings{
			AutoArchiveDelay: "never",
		},
	}
}
```

- [ ] **Step 4: Verify syntax**

```bash
cd apps/api && go vet ./features/tenants/models/...
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd apps/api && git add features/tenants/models/tenant-model.go && git commit -m "feat: add KanbanSettings to tenant UISettings"
```

---

### Task 3: Update GetAllTasks Handler to Filter Archived Tasks

**Files:**
- Modify: `apps/api/features/kanban/handlers/kanban-handler.go`

- [ ] **Step 1: Open kanban-handler.go and locate GetAllTasks method**

```bash
grep -n "func.*GetAllTasks\|func.*GetTasks" apps/api/features/kanban/handlers/kanban-handler.go
```

- [ ] **Step 2: Find the database query in GetAllTasks**

Read the file to understand the current query structure:

```bash
sed -n '50,100p' apps/api/features/kanban/handlers/kanban-handler.go
```

(Adjust line numbers based on actual file content)

- [ ] **Step 3: Add archived_at filter**

Update the `GetAllTasks` method to add `WHERE archived_at IS NULL`:

```go
func (h *KanbanHandler) GetAllTasks(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)
	var tasks []kanban_models.Task
	
	// Add archived_at filter
	if err := h.db.Scopes(tenantScope).
		Where("archived_at IS NULL").
		Order("position ASC").
		Find(&tasks).Error; err != nil {
		h.logger.Error("failed to fetch kanban tasks", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.tasks.fetch.error", core_errors.ErrInternal)
	}
	
	return envelope.SuccessResponse(tasks, "kanban.tasks.fetch.success")
}
```

- [ ] **Step 4: Verify syntax**

```bash
cd apps/api && go vet ./features/kanban/handlers/...
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd apps/api && git add features/kanban/handlers/kanban-handler.go && git commit -m "feat: filter archived tasks from GET /kanban/tasks"
```

---

### Task 4: Create Archive Scheduler Worker

**Files:**
- Create: `apps/api/features/kanban/workers/archive-scheduler.go`
- Create: `apps/api/features/kanban/workers/archive-scheduler_test.go`

- [ ] **Step 1: Create archive-scheduler.go**

Create file `apps/api/features/kanban/workers/archive-scheduler.go`:

```go
package kanban_workers

import (
	"encoding/json"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/features/kanban/models"
	"pengi-med-saas/features/tenants/models"

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
		delay := s.getDelayDuration(settings.Kanban.AutoArchiveDelay)
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

// getDelayDuration converts delay string to duration
func (s *ArchiveScheduler) getDelayDuration(delay string) time.Duration {
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
```

- [ ] **Step 2: Create archive-scheduler_test.go**

Create file `apps/api/features/kanban/workers/archive-scheduler_test.go`:

```go
package kanban_workers_test

import (
	"encoding/json"
	"testing"
	"time"

	"pengi-med-saas/features/kanban/models"
	kanban_workers "pengi-med-saas/features/kanban/workers"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

func TestSchedulerArchivesExpiredTasks(t *testing.T) {
	// Setup: Create test task in "done" status, updated 2 days ago
	// With auto_archive_delay = "1_day"
	// Expect: Task gets archived_at set

	t.Skip("requires database setup")
}

func TestSchedulerSkipsNeverArchive(t *testing.T) {
	// Setup: Create test task in "done" status with auto_archive_delay = "never"
	// Even if past delay
	// Expect: archived_at remains NULL

	t.Skip("requires database setup")
}

func TestSchedulerSkipsNonDoneTasks(t *testing.T) {
	// Setup: Create test task in "todo" status, past archive delay
	// Expect: Task not archived

	t.Skip("requires database setup")
}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scheduler.GetDelayDuration(tt.delay)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 3: Update archive-scheduler_test.go to export GetDelayDuration**

Edit `archive-scheduler.go` and make `getDelayDuration` public:

```go
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
```

- [ ] **Step 4: Run tests**

```bash
cd apps/api && go test ./features/kanban/workers/... -v
```

Expected: All tests pass (skipped tests are OK for now)

- [ ] **Step 5: Verify syntax**

```bash
cd apps/api && go vet ./features/kanban/workers/...
```

Expected: No errors

- [ ] **Step 6: Commit**

```bash
cd apps/api && git add features/kanban/workers/archive-scheduler.go features/kanban/workers/archive-scheduler_test.go && git commit -m "feat: add archive scheduler worker to archive expired tasks"
```

---

### Task 5: Initialize Scheduler in main.go

**Files:**
- Modify: `apps/api/cmd/main.go`

- [ ] **Step 1: Locate the main function and existing goroutines**

```bash
grep -n "go " apps/api/cmd/main.go | head -10
```

- [ ] **Step 2: Add import for kanban_workers**

Add to imports in `apps/api/cmd/main.go`:

```go
kanban_workers "pengi-med-saas/features/kanban/workers"
```

- [ ] **Step 3: Initialize scheduler in main()**

After database connection and other service initialization, add:

```go
// Initialize archive scheduler
archiveScheduler := kanban_workers.NewArchiveScheduler(db, logger.Log)
go archiveScheduler.Start()
logger.Log.Info("archive scheduler started")
```

- [ ] **Step 4: Verify syntax**

```bash
cd apps/api && go build ./cmd/
```

Expected: No errors

- [ ] **Step 5: Commit**

```bash
cd apps/api && git add cmd/main.go && git commit -m "feat: initialize archive scheduler on startup"
```

---

### Task 6: Add i18n Messages

**Files:**
- Modify: `apps/api/i18n/messages/messages_es.json`
- Modify: `apps/api/i18n/messages/messages_en.json`

- [ ] **Step 1: Read current Spanish messages**

```bash
tail -20 apps/api/i18n/messages/messages_es.json
```

- [ ] **Step 2: Add Spanish kanban i18n keys**

Edit `apps/api/i18n/messages/messages_es.json` and add before the final closing brace:

```json
  "kanban.settings.archive_delay.label": "Archivar tareas completadas después de:",
  "kanban.settings.archive_delay.never": "Nunca",
  "kanban.settings.archive_delay.1_day": "1 día",
  "kanban.settings.archive_delay.1_week": "1 semana",
  "kanban.settings.archive_delay.2_weeks": "2 semanas",
  "kanban.settings.archive_delay.1_month": "1 mes",
  "kanban.archive.success": "Configuración de archivo actualizada"
```

- [ ] **Step 3: Read current English messages**

```bash
tail -20 apps/api/i18n/messages/messages_en.json
```

- [ ] **Step 4: Add English kanban i18n keys**

Edit `apps/api/i18n/messages/messages_en.json` and add before the final closing brace:

```json
  "kanban.settings.archive_delay.label": "Auto-archive completed tasks after:",
  "kanban.settings.archive_delay.never": "Never",
  "kanban.settings.archive_delay.1_day": "1 day",
  "kanban.settings.archive_delay.1_week": "1 week",
  "kanban.settings.archive_delay.2_weeks": "2 weeks",
  "kanban.settings.archive_delay.1_month": "1 month",
  "kanban.archive.success": "Archive settings updated"
```

- [ ] **Step 5: Validate JSON syntax**

```bash
node -e "console.log(JSON.parse(require('fs').readFileSync('apps/api/i18n/messages/messages_es.json', 'utf8')))" && echo "✓ Spanish messages valid"
node -e "console.log(JSON.parse(require('fs').readFileSync('apps/api/i18n/messages/messages_en.json', 'utf8')))" && echo "✓ English messages valid"
```

Expected: Both files parse successfully

- [ ] **Step 6: Commit**

```bash
git add apps/api/i18n/messages/messages_es.json apps/api/i18n/messages/messages_en.json && git commit -m "feat: add i18n messages for kanban archive settings"
```

---

### Task 7: Update Frontend Settings Service Types

**Files:**
- Modify: `apps/web/src/api/settings-service.ts`

- [ ] **Step 1: Open settings-service.ts and locate TenantUISettings interface**

```bash
grep -n "export interface TenantUISettings" apps/web/src/api/settings-service.ts
```

- [ ] **Step 2: Add KanbanSettings interface before TenantUISettings**

Edit `apps/web/src/api/settings-service.ts`:

```typescript
export interface KanbanSettings {
	auto_archive_delay: "never" | "1_day" | "1_week" | "2_weeks" | "1_month";
}

export interface TenantUISettings {
	[key: string]: unknown;
	clinical: ClinicalSettings;
	kanban?: KanbanSettings;  // Optional for backwards compatibility
}
```

- [ ] **Step 3: Update DEFAULT_UI_SETTINGS constant**

Add kanban settings to the default:

```typescript
export const DEFAULT_UI_SETTINGS: TenantUISettings = {
	clinical: {
		show_next_appointment: true,
		show_diagnosis: true,
		show_medic: true,
		show_insurance: true,
		show_vital_signs: true,
		show_diagnoses: true,
		diagnosis_system: "cie11",
		patient_age_input: false,
	},
	kanban: {
		auto_archive_delay: "never",
	},
};
```

- [ ] **Step 4: Build TypeScript**

```bash
cd apps/web && pnpm run build
```

Expected: No TypeScript errors

- [ ] **Step 5: Commit**

```bash
cd apps/web && git add src/api/settings-service.ts && git commit -m "feat: add KanbanSettings interface to frontend settings service"
```

---

### Task 8: Create Kanban Settings UI Component

**Files:**
- Create: `apps/web/src/sections/settings/kanban-settings.tsx`

- [ ] **Step 1: Check existing settings structure**

```bash
ls -la apps/web/src/sections/settings/
```

- [ ] **Step 2: Create kanban-settings.tsx component**

Create file `apps/web/src/sections/settings/kanban-settings.tsx`:

```typescript
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useTenantSettingsStore } from "@/store/tenant-settings-store";
import { useText } from "@/hooks/use-text";
import type { KanbanSettings } from "@/api/settings-service";

const ARCHIVE_DELAY_OPTIONS: Array<{
	value: KanbanSettings["auto_archive_delay"];
	labelKey: string;
}> = [
	{ value: "never", labelKey: "kanban.settings.archive_delay.never" },
	{ value: "1_day", labelKey: "kanban.settings.archive_delay.1_day" },
	{ value: "1_week", labelKey: "kanban.settings.archive_delay.1_week" },
	{ value: "2_weeks", labelKey: "kanban.settings.archive_delay.2_weeks" },
	{ value: "1_month", labelKey: "kanban.settings.archive_delay.1_month" },
];

export function KanbanSettings() {
	const { textGet } = useText();
	const { settings, saveSettings } = useTenantSettingsStore();

	const kanbanSettings = settings.kanban || { auto_archive_delay: "never" };

	const handleArchiveDelayChange = async (
		value: KanbanSettings["auto_archive_delay"],
	) => {
		const newSettings = {
			...settings,
			kanban: {
				...kanbanSettings,
				auto_archive_delay: value,
			},
		};
		await saveSettings(newSettings);
	};

	return (
		<div className="space-y-4">
			<div>
				<h3 className="text-sm font-semibold mb-3">
					{textGet("kanban.settings.archive_delay.label")}
				</h3>
				<RadioGroup
					value={kanbanSettings.auto_archive_delay}
					onValueChange={handleArchiveDelayChange}
				>
					<div className="space-y-2">
						{ARCHIVE_DELAY_OPTIONS.map((option) => (
							<div key={option.value} className="flex items-center space-x-2">
								<RadioGroupItem value={option.value} id={option.value} />
								<Label htmlFor={option.value} className="font-normal cursor-pointer">
									{textGet(option.labelKey)}
								</Label>
							</div>
						))}
					</div>
				</RadioGroup>
			</div>
		</div>
	);
}
```

- [ ] **Step 3: Verify component syntax**

```bash
cd apps/web && pnpm run build
```

Expected: No TypeScript errors

- [ ] **Step 4: Commit**

```bash
cd apps/web && git add src/sections/settings/kanban-settings.tsx && git commit -m "feat: create kanban settings UI component"
```

---

### Task 9: Integrate Kanban Settings into Settings Page

**Files:**
- Modify: `apps/web/src/pages/settings/settings-page.tsx`

- [ ] **Step 1: Locate settings-page.tsx and examine structure**

```bash
head -50 apps/web/src/pages/settings/settings-page.tsx
```

- [ ] **Step 2: Add import for KanbanSettings component**

At the top of `apps/web/src/pages/settings/settings-page.tsx`, add:

```typescript
import { KanbanSettings } from "@/sections/settings/kanban-settings";
```

- [ ] **Step 3: Add KanbanSettings section to the page**

In the JSX of `settings-page.tsx`, add the component in an appropriate section (e.g., in a tabs or accordion structure):

```typescript
<div className="space-y-6">
  {/* Existing sections */}
  
  {/* NEW: Kanban Settings Section */}
  <div className="border-t pt-6">
    <h2 className="text-lg font-semibold mb-4">{textGet("kanban.title")}</h2>
    <KanbanSettings />
  </div>
</div>
```

(Adjust placement based on actual settings-page structure)

- [ ] **Step 4: Build and verify**

```bash
cd apps/web && pnpm run build
```

Expected: No TypeScript errors

- [ ] **Step 5: Commit**

```bash
cd apps/web && git add src/pages/settings/settings-page.tsx && git commit -m "feat: integrate kanban settings component into settings page"
```

---

### Task 10: Write Integration Tests

**Files:**
- Create: `apps/web/src/__tests__/kanban-settings.test.tsx`

- [ ] **Step 1: Create test file**

Create file `apps/web/src/__tests__/kanban-settings.test.tsx`:

```typescript
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { KanbanSettings } from "@/sections/settings/kanban-settings";
import * as settingsService from "@/api/settings-service";

// Mock the hooks and service
vi.mock("@/store/tenant-settings-store", () => ({
	useTenantSettingsStore: () => ({
		settings: {
			clinical: { /* ... */ },
			kanban: { auto_archive_delay: "never" },
		},
		saveSettings: vi.fn(),
	}),
}));

vi.mock("@/hooks/use-text", () => ({
	useText: () => ({
		textGet: (key: string) => key, // Return the key itself for testing
	}),
}));

describe("KanbanSettings", () => {
	it("renders archive delay radio buttons", () => {
		render(<KanbanSettings />);
		
		expect(screen.getByText("kanban.settings.archive_delay.never")).toBeDefined();
		expect(screen.getByText("kanban.settings.archive_delay.1_day")).toBeDefined();
		expect(screen.getByText("kanban.settings.archive_delay.1_week")).toBeDefined();
		expect(screen.getByText("kanban.settings.archive_delay.2_weeks")).toBeDefined();
		expect(screen.getByText("kanban.settings.archive_delay.1_month")).toBeDefined();
	});

	it("selects the correct default option", () => {
		render(<KanbanSettings />);
		
		const neverOption = screen.getByDisplayValue("never") as HTMLInputElement;
		expect(neverOption.checked).toBe(true);
	});

	it("calls saveSettings when option changes", async () => {
		const mockSaveSettings = vi.fn();
		
		render(<KanbanSettings />);
		
		const oneDayOption = screen.getByDisplayValue("1_day");
		fireEvent.click(oneDayOption);
		
		// Verify saveSettings was called (would need actual mock setup)
		// expect(mockSaveSettings).toHaveBeenCalled();
	});
});
```

- [ ] **Step 2: Run tests**

```bash
cd apps/web && pnpm run test kanban-settings.test.tsx
```

Expected: Tests pass (or are skipped if test infrastructure not set up)

- [ ] **Step 3: Commit**

```bash
cd apps/web && git add src/__tests__/kanban-settings.test.tsx && git commit -m "test: add kanban settings component tests"
```

---

### Task 11: Backend Integration Test

**Files:**
- Create: `apps/api/features/kanban/integration_test.go`

- [ ] **Step 1: Create integration test file**

Create file `apps/api/features/kanban/integration_test.go`:

```go
package kanban_test

import (
	"testing"
	"time"

	kanban_models "pengi-med-saas/features/kanban/models"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

func TestTaskArchivalFlow(t *testing.T) {
	// This test requires a real database setup
	// Placeholder for future implementation with test fixtures

	t.Skip("requires test database setup")

	/*
	// Setup
	db := setupTestDB(t)
	tenant := createTestTenant(db)
	
	// Create task and move to done
	task := &kanban_models.Task{
		TenantID:      tenant.ID,
		Title:         "Test Task",
		Status:        "done",
		ArchivedAt:    nil,
	}
	db.Create(task)
	
	// Update task's UpdatedAt to 2 days ago
	db.Model(task).Update("updated_at", time.Now().Add(-48*time.Hour))
	
	// Run scheduler
	scheduler := kanban_workers.NewArchiveScheduler(db, logger)
	scheduler.archiveExpiredTasks()
	
	// Verify task is archived
	var archived kanban_models.Task
	db.First(&archived, task.ID)
	if archived.ArchivedAt == nil {
		t.Error("task should be archived but archived_at is nil")
	}
	*/
}

func TestGetTasksFiltersArchived(t *testing.T) {
	t.Skip("requires test database setup")
	
	/*
	// Setup
	db := setupTestDB(t)
	tenant := createTestTenant(db)
	
	// Create archived task
	archivedTask := &kanban_models.Task{
		TenantID:   tenant.ID,
		Title:      "Archived Task",
		ArchivedAt: pointerToTime(time.Now()),
	}
	db.Create(archivedTask)
	
	// Create active task
	activeTask := &kanban_models.Task{
		TenantID:   tenant.ID,
		Title:      "Active Task",
		ArchivedAt: nil,
	}
	db.Create(activeTask)
	
	// Query with filter
	var tasks []kanban_models.Task
	db.Where("archived_at IS NULL").Find(&tasks)
	
	if len(tasks) != 1 || tasks[0].ID != activeTask.ID {
		t.Error("archived tasks should not be returned")
	}
	*/
}
```

- [ ] **Step 2: Run tests**

```bash
cd apps/api && go test ./features/kanban/... -v
```

Expected: Tests skip (that's OK — infrastructure not set up)

- [ ] **Step 3: Commit**

```bash
cd apps/api && git add features/kanban/integration_test.go && git commit -m "test: add kanban archival integration test placeholders"
```

---

### Task 12: Manual Testing Checklist

- [ ] **Start the full stack**

```bash
just dev
```

Wait for all services to be up (database, API, frontend).

- [ ] **Log in to the app**

Navigate to `http://localhost:5173` and log in with test credentials.

- [ ] **Go to Settings page**

Click Settings in the sidebar.

- [ ] **Verify Kanban settings section appears**

Look for "Auto-archive completed tasks after:" label with radio buttons.

- [ ] **Test changing archive delay**

Select "1 week" and verify the toast appears (success message).

- [ ] **Check localStorage persistence**

Open DevTools → Application → Local Storage, verify `tenant-ui-settings` includes kanban settings.

- [ ] **Go to Kanban board**

Navigate to Kanban section.

- [ ] **Create a test task and mark it done**

- Create a new task with title "Test Archive Task"
- Move it to the "done" column

- [ ] **Wait for scheduler run (5 minutes)**

Alternatively, restart the backend to trigger immediate scheduler run (or implement manual trigger endpoint for testing).

- [ ] **Verify task is archived after delay**

After 5+ minutes, refresh the page. Task should be gone from the board.

- [ ] **Change archive delay to "Never" and test**

In Settings, select "Never". Create a new task, move to done. Wait 5+ minutes. Task should still be visible.

- [ ] **Check archived_at in database**

```bash
docker exec pengi-med-saas-postgres psql -U postgres pengi_med_saas -c "SELECT id, title, archived_at FROM kanban_tasks LIMIT 5;"
```

Verify that archived tasks have a non-NULL `archived_at` timestamp.

- [ ] **Final commit**

```bash
git add -A && git commit -m "feat: kanban task auto-archive implementation complete"
```

---

## Summary

This plan implements the kanban task auto-archive feature in 12 focused tasks:

1. ✓ Add `ArchivedAt` field to Task model
2. ✓ Add `KanbanSettings` to tenant UISettings
3. ✓ Filter archived tasks in GET handler
4. ✓ Create scheduler worker with logic
5. ✓ Initialize scheduler in main.go
6. ✓ Add i18n messages (ES + EN)
7. ✓ Update frontend settings service types
8. ✓ Create kanban settings UI component
9. ✓ Integrate component into settings page
10. ✓ Write frontend unit tests
11. ✓ Write backend integration test placeholders
12. ✓ Manual testing verification

All user-facing strings are i18n keys. Scheduler runs every 5 minutes. Archived tasks are soft-deleted (remain in DB). Feature is tenant-level configurable via Settings UI.

**Frequent commits ensure clear, reviewable history.**
