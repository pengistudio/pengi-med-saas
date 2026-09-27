# Frontend Development Guide

Guías de desarrollo para el frontend SaaS (`apps/web`). El panel de
administración (`apps/backoffice`) tiene sus propias guías en
[`../backoffice/`](../backoffice/README.md).

## 📚 Documentación

- **[Web Frontend — Complete Guide](../skills/web-frontend-complete-guide.md)** —
  estructura, instancias axios, service layer, tipos, Zustand, i18n, listados
  con `DataTable`, rutas y navegación, paso a paso.
- **[Form Creation Standard](../skills/form-creation-standard.md)** — Zod +
  `Form` y componentes `Form*` de `@pengi/ui`.
- **Feature de punta a punta** (backend + permisos + plan + frontend + tests):
  skill `create-feature` (`.claude/skills/create-feature/`).
- **Código compartido entre apps:** [`packages/shared/README.md`](../../packages/shared/README.md)
  (`@pengi/shared`, no visual) y `packages/ui` (`@pengi/ui`, visual).

## 🔑 Key Concepts

### API Requests
Siempre a través de un servicio en `src/api/<domain>-service.ts`:

```typescript
// ❌
const res = await apiWithTenant.get("/clinical/patients");

// ✅
const res = await getPatients();
```

### i18n
Todas las strings visibles pasan por `useText()` de `@pengi/shared`:

```typescript
const { textGet } = useText();
<h1>{textGet("dashboard.title")}</h1>
```

### State Management
Zustand solo para estado compartido entre componentes (`src/store/`); estado
local con `useState`.

### Permissions
Rutas y acciones se protegen con los permisos **del rol**:

```typescript
import CheckPermission from "@/components/custom/check-permission";

<CheckPermission permissions={[PERMISSIONS.TEAM.PERMISSION_READ_TEAM]}>
    <TeamPage />
</CheckPermission>

const { checkPermission } = usePermission();
```

### Features habilitados
El sidebar oculta ítems según `permission` (rol) y `feature` (flags que el
backend calcula desde el plan). Ver
[`../backend/permissions-system.md`](../backend/permissions-system.md).

## 📋 Stack

React 19, TypeScript, Vite, TailwindCSS v4, componentes sobre Base UI
(`@base-ui/react`, estilo shadcn) en `@pengi/ui`, Zustand, React Router,
React Hook Form + Zod, TanStack Table, Vitest, Playwright. Las versiones viven
en el `catalog` de `pnpm-workspace.yaml`.

## 🛠️ Common Patterns

| Situación | Dónde copiarlo |
|-----------|---|
| Servicio + tipos de API | `src/api/billing-service.ts` |
| Listado con tabla | `src/pages/billing/catalog-item-list.tsx` + `src/sections/columns/billing/catalog-item-columns.tsx` |
| Formulario crear/editar | `src/sections/forms/billing/catalog-item-form.tsx` |
| Páginas crear/editar | `src/pages/billing/create-catalog-item.tsx`, `edit-catalog-item.tsx` |
| Diálogo de edición rápida | `src/components/features/patient/edit-prescription-dialog.tsx` |
| Store | `src/store/kanban-store.ts` (sin persistencia), `src/store/billing-store.ts` (sessionStorage) |
| Acciones según permiso | `src/pages/billing/catalog-item-list.tsx` (`usePermission`) |
| Test de componente | `src/__tests__/kanban-settings.test.tsx` |

## ⚡ Comandos

```bash
cd apps/web
pnpm run dev          # Vite
pnpm run typecheck    # tsc
pnpm run test:run     # Vitest
pnpm exec playwright test   # e2e (stack arriba), o `just tests-e2e`
just check            # Biome desde la raíz
```

## 📞 Debugging

- **i18n:** una key que aparece como `*key*` recién agregada suele ser la
  caché: `localStorage.removeItem("messages")` y recarga.
- **403 al llamar un endpoint con el permiso del rol:** el plan no lo incluye
  (ver `permissions-system.md`).
- **Network / React DevTools** para requests y props.
