# Docs desactualizadas

Las guías de `docs/` enseñan patrones que el código ya abandonó. Cuando una
contradiga a esta skill o al código, gana el código.

| Doc | Dice | Realidad |
|---|---|---|
| CLAUDE.md, `docs/skills/api-backend-complete-guide.md` | `h.db.Scopes(tenant_middleware.TenantScope(c))` | `tenantdb.For(c, h.db)` (ADR 0002) |
| `docs/backend/permissions-system.md` | `permission_middleware.RequirePermission`, `applyPlanFeaturesToTenant` | `subscription_middleware.RequirePermission`; flags calculados en `enabled-features-service.go` |
| `docs/backend/api-code-migration.md` | `BaseStringID: "X"` | `database.BaseStringID{ID: "X"}` |
| `docs/skills/web-frontend-complete-guide.md` | `services/`, `api/fetch.ts`, `hooks/use-text.tsx`, `components/access-control`, `pages/<d>/page.tsx`, i18n JSON anidado, `item.id`, `createNavItems(textGet, enabledFeatures)` | `src/api/<d>-service.ts`, `@pengi/shared`, `components/custom/check-permission` (default export), `pages/<d>/<kebab>.tsx`, JSON plano `[{key,value}]`, `ID`, `createNavItems(textGet)` |
| `docs/skills/form-creation-standard.md` | `Form` en `@/components/forms/form`; `FormInputPassword`; `FormCalendar` en backoffice | `Form` en `@pengi/ui`; `FormPasswordInput`; `FormCalendar` solo en web |
| `docs/frontend/README.md` | Vite 5, Zustand 4, React Router 6; archivos de ejemplo | versiones actuales en `package.json`; los ejemplos no existen |
| `docs/backoffice/*.md` | `store/`, `types/`, `pages/<x>/list.tsx` | `src/api/*` + `src/lib/resource` |
