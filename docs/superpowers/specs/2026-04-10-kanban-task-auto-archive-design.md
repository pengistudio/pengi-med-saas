# Kanban Task Auto-Archive Feature Design

**Date:** 2026-04-10  
**Feature:** Automatic archival of completed tasks after configurable delays  
**Scope:** Backend scheduler + UI configuration in tenant settings

---

## Overview

This feature allows users to configure automatic archival of tasks marked as "done". Once a task reaches "done" status, it will automatically be archived after a user-configured delay (1 day, 1 week, 2 weeks, 1 month, or never). Archived tasks are soft-deleted—they remain in the database but are hidden from the kanban board view.

---

## Requirements

### Functional
- Tasks can only be archived when in "done" status
- Archival is automatic based on a timer configured per-tenant
- Auto-archive delay options: "never", "1 day", "1 week", "2 weeks", "1 month"
- Configuration is stored in tenant settings and applies to all users in the tenant
- Users configure this delay in the app's Settings section (not backoffice)
- Archived tasks are completely hidden from the kanban board
- Archived tasks remain in the database for audit purposes (soft-delete via `archived_at` timestamp)

### Non-Functional
- Scheduler runs every 5 minutes (balance between responsiveness and database load)
- No manual archive/unarchive in this version (audit view comes later)
- Tasks moved back from "done" to another status cancel the archival timer (no special logic needed—just don't archive if status != "done")

---

## Architecture

### 1. Database Schema Changes

#### Task Model (`apps/api/features/kanban/models/task-model.go`)
```go
type Task struct {
    gorm.Model
    TenantID      uint
    Title         string
    Description   string
    Status        string     // "todo", "in_progress", "done"
    Position      int
    DueDate       *time.Time
    CreatedByName string
    ArchivedAt    *time.Time // NEW: nil = not archived, NOT NULL = archived timestamp
}
```

No database migration needed—GORM will auto-migrate the new field on startup.

#### UISettings Struct (`apps/api/features/tenants/models/tenant-model.go`)
```go
type UISettings struct {
    Clinical ClinicalSettings `json:"clinical"`
    Kanban   KanbanSettings   `json:"kanban"`  // NEW
}

type KanbanSettings struct {
    AutoArchiveDelay string `json:"auto_archive_delay"` 
    // Allowed values: "never", "1_day", "1_week", "2_weeks", "1_month"
}

func DefaultUISettings() UISettings {
    return UISettings{
        Clinical: ClinicalSettings{ /* ... */ },
        Kanban: KanbanSettings{
            AutoArchiveDelay: "never",  // Default: no auto-archive
        },
    }
}
```

---

### 2. Backend Implementation

#### Scheduler Job (`apps/api/features/kanban/workers/archive-scheduler.go` — NEW)
```
Purpose: Background job that runs every 5 minutes and archives tasks whose timer has expired.

Pseudocode:
- Query all tasks WHERE status = "done" AND archived_at IS NULL
- For each task:
  - Load tenant settings → get auto_archive_delay
  - If auto_archive_delay == "never", skip
  - Calculate archive_time = task.updated_at + delay_duration
  - If now >= archive_time, set task.archived_at = now
- Log results (count archived, errors)
```

**Integration:** Initialize the scheduler in `apps/api/cmd/main.go` on startup (similar to other background workers).

#### API Changes
**GET `/kanban/tasks`** in `apps/api/features/kanban/handlers/kanban-handler.go`:
- Add filter clause: `WHERE archived_at IS NULL`
- No other changes to the handler

**Existing handlers remain unchanged:**
- `UpdateUISettings()` already saves the kanban settings as part of the overall UISettings
- No need for new endpoints

---

### 3. Frontend Implementation

#### Types (`apps/web/src/types/kanban-type.ts`)
No changes needed—`archived_at` is not exposed in the API response (filtered out in GET).

#### Settings Service (`apps/web/src/api/settings-service.ts`)
Update the `TenantUISettings` interface:
```typescript
export interface KanbanSettings {
    auto_archive_delay: "never" | "1_day" | "1_week" | "2_weeks" | "1_month";
}

export interface TenantUISettings {
    clinical: ClinicalSettings;
    kanban?: KanbanSettings;  // Optional for backwards compatibility
}

export const DEFAULT_UI_SETTINGS: TenantUISettings = {
    clinical: { /* ... */ },
    kanban: {
        auto_archive_delay: "never",
    },
};
```

The existing `updateUISettings()` function already handles saving the entire settings object.

#### Settings Store (`apps/web/src/store/tenant-settings-store.ts`)
No changes needed—the store already handles generic `TenantUISettings`.

#### Settings UI Component (`apps/web/src/pages/settings/settings-page.tsx` or new kanban settings section)
New section:
- Label: "Auto-archive completed tasks after:"
- Radio buttons:
  - Never
  - 1 day
  - 1 week
  - 2 weeks
  - 1 month
- On change: call `saveSettings()` with updated kanban settings
- Optimistic UI update via Zustand store
- Toast from service layer on success/error

#### Kanban Board (`apps/web/src/pages/kanban/kanban-page.tsx`)
No changes—GET `/kanban/tasks` already returns filtered results.

---

## Data Flow

```
1. User marks task as "done" in kanban board
   → Task.status = "done"
   → Task.updated_at = now (GORM sets automatically)

2. Scheduler runs every 5 minutes
   → Query: tasks WHERE status = "done" AND archived_at IS NULL
   → For each task:
     → Load tenant settings → auto_archive_delay
     → Calculate: archive_time = updated_at + delay_duration
     → If now >= archive_time AND auto_archive_delay != "never"
       → Set archived_at = now

3. User loads kanban board
   → GET /kanban/tasks
   → Returns only tasks WHERE archived_at IS NULL
   → Archived tasks never appear in UI

4. User configures archive delay
   → Updates settings in Settings page
   → PUT /tenants/settings { kanban: { auto_archive_delay: "1_week" } }
   → Settings persisted in tenant.ui_settings (JSON)
```

---

## Error Handling & Edge Cases

| Case | Behavior |
|------|----------|
| Task moved back from "done" to another status | Archival timer is not triggered (scheduler only archives "done" tasks). If moved back to "done" later, timer restarts from the new `updated_at`. |
| Tenant has no kanban settings (legacy data) | Use default `auto_archive_delay = "never"`. Frontend defaults to "never" on first load. |
| Scheduler fails to update a task | Log error and continue; next run will retry (idempotent). |
| User changes auto-archive delay | Only affects tasks set to "done" *after* the change. Already-started timers are not recalculated. |
| Archived task is restored (future feature) | Unarchive by setting `archived_at = NULL` and resetting `updated_at` if needed. |

---

## Testing Strategy

### Backend Unit Tests
- `TestSchedulerArchivesExpiredTasks` — Verify tasks past delay are archived
- `TestSchedulerSkipsNeverArchive` — Verify "never" setting blocks archival
- `TestSchedulerSkipsNonDoneTasks` — Verify only "done" tasks are archived
- `TestGetTasksFiltersArchived` — Verify GET /kanban/tasks filters `archived_at IS NULL`

### Frontend Unit Tests
- `TestKanbanSettingsToggle` — Verify radio button changes update store
- `TestSettingsSaveCallsUpdateUISettings` — Verify service call includes kanban settings
- `TestDefaultKanbanSettings` — Verify "never" is the default

### Integration Tests
- Task created → set to done → scheduler runs → task archived → GET returns nothing
- Change tenant setting → create new task → set to done → wait → archived after new delay

---

## Implementation Order

1. **Backend Model** — Add `ArchivedAt` to Task, add KanbanSettings to UISettings
2. **Backend Handler** — Update GET `/kanban/tasks` with `archived_at IS NULL` filter
3. **Backend Scheduler** — Create archive-scheduler job, initialize in main.go
4. **Frontend Types** — Update `TenantUISettings` with KanbanSettings
5. **Frontend UI** — Add kanban settings section in Settings page
6. **Integration Tests** — Write tests for full flow
7. **Manual Testing** — Test in browser (settings, archival delay, scheduler)

---

## Migration & Rollout

- No data migration needed (new nullable column, defaults to NULL)
- Scheduler starts on app startup; no manual trigger needed
- Existing tasks are unaffected (default auto-archive = "never")
- Feature is self-service via Settings page

---

## Future Enhancements (Out of Scope)

- Archive view showing archived tasks with restore option
- Per-task archive override
- Bulk archive/restore
- Archive retention policy (permanently delete after X days)
- Audit log of archival events
