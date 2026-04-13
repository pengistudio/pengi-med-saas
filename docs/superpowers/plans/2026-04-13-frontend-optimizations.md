# Frontend Optimizations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce bundle size and eliminate unnecessary Zustand re-renders across the React frontend.

**Architecture:** Three independent changes — Tailwind content config (prevents unused CSS in prod), Vite manual chunking (vendor cache separation), and Zustand granular selectors (stops full-store re-renders).

**Tech Stack:** React 19 + Vite 6 + TailwindCSS v4 (@tailwindcss/vite) + Zustand + TypeScript

**Spec:** `docs/superpowers/specs/2026-04-13-frontend-optimizations-design.md`

**Note — already done:** `form-icd11-select.tsx` already debounces ICD search with `useRef + setTimeout` at 350ms. No action needed. KanbanCard memoization skipped — component has local state + DnD dynamic values that legitimately trigger re-renders.

---

## File Map

| File | Action | Why |
|------|--------|-----|
| `apps/web/tailwind.config.ts` | Create | Enables CSS purging — without it, prod CSS may include unused utilities |
| `apps/web/vite.config.ts` | Modify | Add `manualChunks` to separate vendor libs for browser cache |
| `apps/web/src/store/session-store.ts` | Modify | Export granular selectors |
| `apps/web/src/store/patient-store.ts` | Modify | Export granular selectors |
| `apps/web/src/sections/template/dashboard-template.tsx` | Modify | Use `useEnvironment`, `useSubscriptionExpired` selectors |
| `apps/web/src/hooks/use-permission.ts` | Modify | Use `usePermissions` selector |
| `apps/web/src/pages/login/login-environments.tsx` | Modify | Use session selectors |
| `apps/web/src/pages/team/team-page.tsx` | Modify | Use session selectors |
| `apps/web/src/pages/profile/profile.tsx` | Modify | Use session selectors |
| `apps/web/src/sections/forms/clinical/medical-record-create-form.tsx` | Modify | Use `usePatient` selector |
| `apps/web/src/sections/forms/clinical/patient-update-form.tsx` | Modify | Use `usePatient` selector |
| `apps/web/src/sections/columns/clinical/patient-columns.tsx` | Modify | Use patient selectors |
| `apps/web/src/sections/kanban/task-card-content.tsx` | Modify | Use patient selectors |
| `apps/web/src/sections/kanban/task-form-dialog.tsx` | Modify | Use patient selectors |
| `apps/web/src/sections/kanban/task-details-panel.tsx` | Modify | Use patient selectors |
| `apps/web/src/pages/clincal/patient/medical-record-list.tsx` | Modify | Use patient selectors |

---

## Task 1: Create Tailwind Content Config

**Files:**
- Create: `apps/web/tailwind.config.ts`

- [ ] **Step 1: Create the config file**

```typescript
import type { Config } from "tailwindcss";

export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
} satisfies Config;
```

- [ ] **Step 2: Verify build works**

```bash
cd apps/web && pnpm run build 2>&1 | tail -20
```

Expected: build completes without errors. No "Config file not found" warnings.

- [ ] **Step 3: Check CSS output size**

```bash
ls -lh apps/web/dist/assets/*.css
```

Note the file size(s). If size decreased vs. a prior build, the purging is working.

- [ ] **Step 4: Commit**

```bash
git add apps/web/tailwind.config.ts
git commit -m "perf(web): add tailwind content config to enable CSS purging"
```

---

## Task 2: Add Vite Manual Chunks

**Files:**
- Modify: `apps/web/vite.config.ts`

Current file (`apps/web/vite.config.ts`):
```typescript
import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [react(), tailwindcss()],
	define: {
		__APP_VERSION__: JSON.stringify(Date.now().toString()),
	},
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
});
```

- [ ] **Step 1: Add build config with manual chunks**

Replace the content of `apps/web/vite.config.ts` with:

```typescript
import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [react(), tailwindcss()],
	define: {
		__APP_VERSION__: JSON.stringify(Date.now().toString()),
	},
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks: {
					"vendor-react": ["react", "react-dom", "react-router"],
					"vendor-ui": ["@base-ui/react", "lucide-react", "recharts"],
					"vendor-forms": ["@hookform/resolvers", "react-hook-form", "zod"],
					"vendor-dnd": ["@dnd-kit/core", "@dnd-kit/sortable"],
				},
			},
		},
	},
});
```

- [ ] **Step 2: Run build**

```bash
cd apps/web && pnpm run build 2>&1 | tail -30
```

Expected: build succeeds. Output shows separate chunk files: `vendor-react-[hash].js`, `vendor-ui-[hash].js`, `vendor-forms-[hash].js`, `vendor-dnd-[hash].js`.

- [ ] **Step 3: Verify chunks exist**

```bash
ls apps/web/dist/assets/vendor-*.js
```

Expected: 4 files (`vendor-react-*.js`, `vendor-ui-*.js`, `vendor-forms-*.js`, `vendor-dnd-*.js`).

- [ ] **Step 4: Commit**

```bash
git add apps/web/vite.config.ts
git commit -m "perf(web): add vite manual chunks to separate vendor libs for browser caching"
```

---

## Task 3: Add Granular Zustand Selectors

**Files:**
- Modify: `apps/web/src/store/session-store.ts`
- Modify: `apps/web/src/store/patient-store.ts`

### Why this matters

Currently components do:
```typescript
const { environment, subscriptionExpired } = useSessionStore();
```
This subscribes the component to ALL changes in the store. When `subscriptionExpired` flips, every component that uses `environment` re-renders even if `environment` didn't change. Selectors fix this.

- [ ] **Step 1: Add selectors to session-store.ts**

Append to the end of `apps/web/src/store/session-store.ts` (after the `export const useSessionStore` line):

```typescript
// Granular selectors — use these in components instead of useSessionStore()
export const useEnvironment = () =>
	useSessionStore((s) => s.environment);
export const useSubscriptionExpired = () =>
	useSessionStore((s) => s.subscriptionExpired);
export const useTenantSlug = () =>
	useSessionStore((s) => s.environment?.tenant_slug);
export const usePermissions = () =>
	useSessionStore((s) => s.environment?.permissions ?? []);
export const useEnabledFeatures = () =>
	useSessionStore((s) => s.environment?.enabled_features);
export const useSetSubscriptionExpired = () =>
	useSessionStore((s) => s.setSubscriptionExpired);
export const useCleanSession = () =>
	useSessionStore((s) => s.clean);
export const useSetEnvironment = () =>
	useSessionStore((s) => s.setEnvironment);
```

- [ ] **Step 2: Add selectors to patient-store.ts**

Append to the end of `apps/web/src/store/patient-store.ts` (after the `export const usePatientStore` line):

```typescript
// Granular selectors — use these in components instead of usePatientStore()
export const usePatient = () =>
	usePatientStore((s) => s.patient);
export const usePatientList = () =>
	usePatientStore((s) => s.patientList);
export const useSetPatient = () =>
	usePatientStore((s) => s.setPatient);
export const useSetPatientList = () =>
	usePatientStore((s) => s.setPatientList);
export const useCleanPatient = () =>
	usePatientStore((s) => s.cleanPatient);
```

- [ ] **Step 3: TypeScript check (verify selectors compile)**

```bash
cd apps/web && npx tsc --noEmit 2>&1 | head -30
```

Expected: no errors from the store files.

- [ ] **Step 4: Commit stores**

```bash
git add apps/web/src/store/session-store.ts apps/web/src/store/patient-store.ts
git commit -m "perf(web): add granular zustand selectors to session and patient stores"
```

---

## Task 4: Migrate Session Store Call Sites

**Files to update** (React hooks/components that call `useSessionStore()` as a hook):
- `apps/web/src/sections/template/dashboard-template.tsx`
- `apps/web/src/hooks/use-permission.ts`
- `apps/web/src/pages/login/login-environments.tsx`
- `apps/web/src/pages/team/team-page.tsx`
- `apps/web/src/pages/profile/profile.tsx`

> **Note:** `apps/web/src/api/index.ts` uses `useSessionStore.getState()` (outside React) — leave it unchanged. That pattern is already optimal (no subscription).

For each file:
1. Find the `useSessionStore` import
2. Replace it with the specific selector imports needed
3. Replace destructured calls with selector calls

### dashboard-template.tsx

- [ ] **Step 1: Update import**

In `apps/web/src/sections/template/dashboard-template.tsx`, find:
```typescript
import { useSessionStore } from "@/store/session-store";
```
Replace with:
```typescript
import { useEnvironment, useSubscriptionExpired } from "@/store/session-store";
```

- [ ] **Step 2: Update usage**

Find:
```typescript
const { environment, subscriptionExpired } = useSessionStore();
```
Replace with:
```typescript
const environment = useEnvironment();
const subscriptionExpired = useSubscriptionExpired();
```

### use-permission.ts

- [ ] **Step 3: Update use-permission.ts**

Read the file first to see current usage, then replace `useSessionStore` import with `usePermissions` and update the call accordingly:

```bash
grep -n "useSessionStore\|usePatientStore" apps/web/src/hooks/use-permission.ts
```

Replace the store import with:
```typescript
import { usePermissions } from "@/store/session-store";
```
And replace any `useSessionStore()` destructure with:
```typescript
const permissions = usePermissions();
```

### Remaining session store files

- [ ] **Step 4: Update login-environments.tsx, team-page.tsx, profile.tsx**

For each file, run:
```bash
grep -n "useSessionStore" apps/web/src/pages/login/login-environments.tsx
grep -n "useSessionStore" apps/web/src/pages/team/team-page.tsx
grep -n "useSessionStore" apps/web/src/pages/profile/profile.tsx
```

For each: replace `useSessionStore` import with the specific selectors used (e.g., `useSetEnvironment`, `useCleanSession`, `useEnvironment`, etc.), then update each destructure to a separate selector call.

Pattern for every migration:
```typescript
// Before
import { useSessionStore } from "@/store/session-store";
const { environment, clean } = useSessionStore();

// After
import { useEnvironment, useCleanSession } from "@/store/session-store";
const environment = useEnvironment();
const clean = useCleanSession();
```

- [ ] **Step 5: TypeScript check**

```bash
cd apps/web && npx tsc --noEmit 2>&1 | head -40
```

Expected: no type errors.

- [ ] **Step 6: Lint check**

```bash
cd /Users/wsantacruz/Projects/pengi-med-saas && just check 2>&1 | tail -20
```

Expected: no lint errors.

- [ ] **Step 7: Commit**

```bash
git add apps/web/src/sections/template/dashboard-template.tsx \
        apps/web/src/hooks/use-permission.ts \
        apps/web/src/pages/login/login-environments.tsx \
        apps/web/src/pages/team/team-page.tsx \
        apps/web/src/pages/profile/profile.tsx
git commit -m "perf(web): migrate session store call sites to granular selectors"
```

---

## Task 5: Migrate Patient Store Call Sites

**Files to update:**
- `apps/web/src/sections/forms/clinical/medical-record-create-form.tsx`
- `apps/web/src/sections/forms/clinical/patient-update-form.tsx`
- `apps/web/src/sections/columns/clinical/patient-columns.tsx`
- `apps/web/src/sections/kanban/task-card-content.tsx`
- `apps/web/src/sections/kanban/task-form-dialog.tsx`
- `apps/web/src/sections/kanban/task-details-panel.tsx`
- `apps/web/src/pages/clincal/patient/medical-record-list.tsx`

- [ ] **Step 1: Check current usage in each file**

```bash
grep -n "usePatientStore\|useSessionStore" \
  apps/web/src/sections/forms/clinical/medical-record-create-form.tsx \
  apps/web/src/sections/forms/clinical/patient-update-form.tsx \
  apps/web/src/sections/columns/clinical/patient-columns.tsx \
  apps/web/src/sections/kanban/task-card-content.tsx \
  apps/web/src/sections/kanban/task-form-dialog.tsx \
  apps/web/src/sections/kanban/task-details-panel.tsx \
  apps/web/src/pages/clincal/patient/medical-record-list.tsx
```

- [ ] **Step 2: Migrate each file**

For each file from step 1, apply the migration pattern:

```typescript
// Before
import { usePatientStore } from "@/store/patient-store";
const { patient, setPatient } = usePatientStore();

// After
import { usePatient, useSetPatient } from "@/store/patient-store";
const patient = usePatient();
const setPatient = useSetPatient();
```

Use the grep output from step 1 to determine which selectors each file needs.

- [ ] **Step 3: TypeScript check**

```bash
cd apps/web && npx tsc --noEmit 2>&1 | head -40
```

Expected: no type errors.

- [ ] **Step 4: Lint check**

```bash
cd /Users/wsantacruz/Projects/pengi-med-saas && just check 2>&1 | tail -20
```

Expected: no lint errors.

- [ ] **Step 5: Commit**

```bash
git add \
  apps/web/src/sections/forms/clinical/medical-record-create-form.tsx \
  apps/web/src/sections/forms/clinical/patient-update-form.tsx \
  apps/web/src/sections/columns/clinical/patient-columns.tsx \
  apps/web/src/sections/kanban/task-card-content.tsx \
  apps/web/src/sections/kanban/task-form-dialog.tsx \
  apps/web/src/sections/kanban/task-details-panel.tsx \
  apps/web/src/pages/clincal/patient/medical-record-list.tsx
git commit -m "perf(web): migrate patient store call sites to granular selectors"
```

---

## Final Verification

- [ ] **Full build passes**

```bash
cd apps/web && pnpm run build 2>&1 | tail -20
```

Expected: no errors, output lists `vendor-react`, `vendor-ui`, `vendor-forms`, `vendor-dnd` chunks.

- [ ] **Dev server runs**

```bash
cd apps/web && pnpm run dev
```

Open browser, navigate through: dashboard → patient list → kanban. No console errors. Pages render correctly.

- [ ] **Re-render check (optional, manual)**

Open React DevTools → Profiler → Record. On the dashboard, trigger a change that updates only `subscriptionExpired` (or navigate to a route). Confirm that components which only use `environment` do NOT highlight as re-rendered.
