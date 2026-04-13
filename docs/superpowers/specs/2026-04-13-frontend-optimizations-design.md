# Frontend Optimizations Design

**Date:** 2026-04-13  
**Scope:** `apps/web/` — React 19 + Vite + Zustand + TailwindCSS v4  
**Goal:** Reduce bundle size, eliminate unnecessary re-renders, and cut redundant API requests.

---

## Context

Analysis of `apps/web/` revealed 4 high-impact optimization areas:
- Vite produces a single large chunk with no vendor separation (no browser cache leverage between deploys)
- Tailwind v4 is configured without a content config file, risking unoptimized CSS output
- Zustand stores return full objects, causing all subscribers to re-render on any state change
- ICD diagnostic search fires one HTTP request per keystroke with no debouncing

---

## 1. Bundle Optimization

### 1a. Vite Manual Chunks

**File:** `apps/web/vite.config.ts`

Add `build.rollupOptions.output.manualChunks` to separate stable vendor libraries into their own chunks. Browsers cache these chunks independently — a code change in app logic won't bust the React or UI vendor cache.

```typescript
build: {
  rollupOptions: {
    output: {
      manualChunks: {
        'vendor-react': ['react', 'react-dom', 'react-router'],
        'vendor-ui': ['@base-ui/react', 'lucide-react', 'recharts'],
        'vendor-forms': ['@hookform/resolvers', 'react-hook-form', 'zod'],
        'vendor-dnd': ['@dnd-kit/core', '@dnd-kit/sortable'],
      },
    },
  },
},
```

**Chunk grouping rationale:**
- `vendor-react` — core framework, changes only on major upgrades
- `vendor-ui` — UI primitives, stable between feature releases
- `vendor-forms` — form stack, changes only when validation logic changes
- `vendor-dnd` — drag-and-drop, isolated feature

### 1b. Tailwind Content Config

**File:** `apps/web/tailwind.config.ts` (create)

Tailwind v4 with `@tailwindcss/vite` needs explicit content scanning to purge unused CSS classes in production. Without it, the generated CSS may include unused utilities.

```typescript
import type { Config } from "tailwindcss";

export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
} satisfies Config;
```

---

## 2. Zustand Granular Selectors

**Files:** `apps/web/src/store/session-store.ts`, `apps/web/src/store/patient-store.ts`

### Problem

Components destructure the full store object:
```typescript
const { environment, subscriptionExpired } = useSessionStore();
```
When `subscriptionExpired` changes, every component subscribed to `useSessionStore()` re-renders — even if they only use `environment`.

### Solution

Export slice selectors from each store file. Components use the specific selector they need.

**session-store.ts additions:**
```typescript
export const useEnvironment = () => useSessionStore((s) => s.environment);
export const useSubscriptionExpired = () => useSessionStore((s) => s.subscriptionExpired);
export const useTenantSlug = () => useSessionStore((s) => s.environment?.slug);
export const usePermissions = () => useSessionStore((s) => s.environment?.permissions);
```

**patient-store.ts additions:**
```typescript
export const usePatient = () => usePatientStore((s) => s.patient);
export const usePatientList = () => usePatientStore((s) => s.patientList);
```

**Migration:** Update call sites to use the new selectors. No store internals change — only the public API surface grows.

---

## 3. Component Memoization

Only two components identified with concrete re-render impact:

### 3a. PatientList DataTable

**File:** `apps/web/src/pages/clinical/patient/patient-list.tsx`

The `DataTable` re-renders when the parent re-renders, even if `patients` array hasn't changed. Wrap with `React.memo` comparing by array reference.

```typescript
const PatientDataTable = React.memo(({ patients, columns, ...props }) => (
  <DataTable data={patients} columns={columns} {...props} />
));
```

### 3b. Kanban Cards

**File:** `apps/web/src/pages/kanban/kanban-page.tsx`

DnD-Kit triggers re-renders on all cards during any drag event. Each `KanbanCard` should only re-render when its own task data changes.

```typescript
const KanbanCard = React.memo(({ task }) => (
  <TaskCard task={task} />
), (prev, next) => prev.task.id === next.task.id && prev.task.status === next.task.status);
```

**Constraint:** Do not add `useMemo`/`useCallback` speculatively elsewhere. Only these two cases have identified impact.

---

## 4. ICD Search Debounce

**File:** Where the ICD diagnostic search component calls `searchICD11` / `searchICD10`

### Problem

`searchICD11(query)` fires on every character change. "dia" typed quickly = 3 simultaneous in-flight requests for essentially the same search.

### Solution

Add `useDebouncedValue` hook (or inline `useEffect` with `setTimeout`) at the call site — 300ms delay. No changes to `clinical-service.ts` or `fetch.ts`.

```typescript
// In the search component
const debouncedQuery = useDebouncedValue(query, 300);

useEffect(() => {
  if (debouncedQuery.length >= 2) {
    searchICD11(debouncedQuery).then(setResults);
  }
}, [debouncedQuery]);
```

If a `useDebouncedValue` hook doesn't exist in `src/hooks/`, create it:
```typescript
export function useDebouncedValue<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}
```

---

## Implementation Order

Priority by impact-to-effort ratio:

1. **Tailwind config** — 5 min, no risk, potentially large CSS savings
2. **Vite manual chunks** — 15 min, no logic changes, better browser caching
3. **Zustand selectors** — 30-60 min, requires updating call sites across components
4. **ICD debounce** — 15 min, isolated to one search component
5. **Component memoization** — 30 min, requires verifying render behavior

---

## Verification

- **Bundle**: Run `pnpm run build` and check `dist/assets/` for separate `vendor-*` chunks
- **CSS**: Confirm `dist/assets/*.css` size decreased vs. baseline
- **Zustand**: Use React DevTools Profiler — confirm components only re-render when their specific slice changes
- **ICD debounce**: Open Network tab, type quickly in diagnosis search — confirm only 1 request fires per 300ms window
- **Kanban memo**: Open React DevTools, drag a card — confirm non-dragged cards don't flash (re-render)
