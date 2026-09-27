# Docs desactualizadas

Estas docs enseñan patrones que el código ya abandonó. Cuando una contradiga a
esta skill o al código, gana el código. (`docs/skills/*` está al día.)

| Doc | Dice | Realidad |
|---|---|---|
| CLAUDE.md (§ Multi-tenancy, patrón #1) | `h.db.Scopes(tenant_middleware.TenantScope(c))` | `tenantdb.For(c, h.db)` (ADR 0002) |
| CLAUDE.md (§ Infrastructure dependencies) | `up pengi-db-dev pengi-rabbitmq-dev ...` | nombres de servicio: `up db rabbitmq gotenberg sri-xml-signer` |
| `docs/backend/permissions-system.md` | `permission_middleware.RequirePermission`, `applyPlanFeaturesToTenant` | `subscription_middleware.RequirePermission`; flags calculados en `enabled-features-service.go` |
| `docs/backend/api-code-migration.md` | `BaseStringID: "X"` | `database.BaseStringID{ID: "X"}` |
| `docs/frontend/README.md` | Vite 5, Zustand 4, React Router 6; archivos de ejemplo | versiones actuales en `package.json`; los ejemplos no existen |
| `docs/backoffice/*.md` | `store/`, `types/`, `pages/<x>/list.tsx` | `src/api/*` + `src/lib/resource` |
