# Permisos, plan y flag de navegación

Contexto completo del sistema: `docs/backend/permissions-system.md` (ojo: usa el
alias `permission_middleware`, el real es `subscription_middleware`).

## Las tres decisiones de gating

| Decisión | Middleware en la ruta | Qué chequea | Ejemplo |
|---|---|---|---|
| **Plan + rol** | `subscription_middleware.RequirePermission(db, id)` | el `Plan` de la suscripción incluye el permiso (vía `Feature`) **y** el rol del usuario lo tiene | clinical, billing, kanban |
| **Solo rol** | `subscription_middleware.RequireRolePermission(db, id)` | solo el rol | team (`routes/company_routes.go`) |
| **Sin permisos** | ninguno; el grupo solo lleva auth + tenant | pertenencia al tenant | notifications |

Import: `subscription_middleware "pengi-med-saas/features/companies/middleware"`.
Alias usual en el archivo de rutas: `rp := subscription_middleware.RequirePermission`.

## Permisos

1. **Catálogo** — `apps/api/features/permissions/data/permission-data.go`:
   ```go
   var <Domain>Permissions = []permission_models.Permission{
       {
           BaseStringID: database.BaseStringID{ID: "READ_<X>"},
           Name:         "Read <X>",
           Category:     "<CATEGORY>", // decide el flag de nav, ver abajo
           Description:  "...",
       },
   }
   ```
   Si extiendes un dominio, agrega entradas al slice existente con la misma
   `Category` del dominio (`CLINICAL`, `BILLING`, `TEAM`, `KANBAN`, `AUDIT`).
2. **Migración al rol admin** — archivo nuevo, patrón exacto de
   `migrations/code-migrations/2026/add_kanban_permissions.go`: busca
   `user_models.Role{Role: "admin"}`, `FirstOrCreate` de cada permiso y
   `Association("Permissions").Append`. Si extiendes un slice existente, itera
   solo tus IDs nuevos.
3. **Roles canónicos** — doctor, recepcionista y contador reciben permisos de
   `role_data.RolePermissionMatrix` (`features/users/data/role-data.go`), que
   solo se siembra en `seed_canonical_roles.go` (ya corrida). Decide para cada
   rol si recibe cada permiso nuevo; para los que sí:
   - agrega los IDs a la matriz (tenants futuros), y
   - en tu migración, además del admin, `Append` a los roles existentes con
     `Role: "<rol>"` (tenants actuales).
   Anota la decisión aunque sea "solo admin".
4. **Rutas** — un `rp(db, "<ID>")` por endpoint.
5. **Frontend** — `apps/web/src/lib/constants.ts` → `PERMISSIONS`:
   ```ts
   <GROUP>: {
     PERMISSION_READ_<X>: "READ_<X>",
   },
   ```
   Si el grupo ya existe (p. ej. `MEDICAL_RECORD` para clinical), agrega ahí.

## Plan

Solo con **plan + rol**. `RequirePermission` responde 403 *"Your plan does not
include this feature"* si ningún `Feature` del plan activo contiene el permiso.
Los `Feature` y `Plan` no se siembran en código: se crean en el backoffice
(`apps/backoffice/src/pages/features`, `pages/plans`) y existen solo en la BD
de cada ambiente.

- **El `Feature` del dominio ya existe** (extensión — el caso común): migración
  propia, con key distinta a la del paso de permisos, que asocia tus permisos
  a él. Patrón exacto:
  `migrations/code-migrations/2026/add_medical_document_feature_permissions.go`
  (hace `return nil` con aviso si el `Feature` no existe en ese ambiente).
- **Dominio nuevo sin `Feature`:** créalo en el backoffice con su `Code` y
  permisos, y asígnalo a los planes que correspondan. El picker de permisos
  agrupa por `Category` solo; no requiere código. Deja el paso anotado en el PR
  para cada ambiente (dev, prod).

Depurar en dev:
```bash
docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo -c "select id, code from features;"
docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo -c "select feature_id, permission_id from feature_permissions where feature_id = <id>;"
```

## Flag de navegación

`Tenant.EnabledFeatures` no es un toggle: el backend lo **calcula** en cada
request a partir de las `Category` de los permisos de los `Feature` del plan
(`company_services.CalculateEnabledFeaturesFromFeatures`). Sin suscripción,
todo queda habilitado. En el frontend un ítem de nav con `feature: "<key>"` se
oculta solo si `enabledFeatures[<key>] === false`.

- **Dominio extendido o categoría existente:** reutiliza la `feature` del
  dominio en `nav-config.ts` (`"clinical"`, `"billing"`, `"team"`, `"kanban"`).
  Nada más que hacer.
- **Categoría nueva que debe ocultarse fuera del plan** — todos juntos, o el
  ítem queda siempre oculto (campo Go sin mapeo → `false`) o siempre visible
  (key TS sin campo Go):
  1. `features/tenants/models/tenant-model.go`: campo en `EnabledFeatures`
     (`json:"<key>"`) y `true` en `DefaultEnabledFeatures`.
  2. `features/companies/services/enabled-features-service.go`:
     `<Key>: categoriesFound["<CATEGORY>"]`.
  3. Mapas hardcodeados: `features/companies/handlers/dashboard-handler.go`
     (`subscriptionInfo.EnabledFeatures`) y
     `features/backoffice/handlers/backoffice-plan-handler.go`
     (`calculateEnabledFeatures`, ambos returns).
  4. Web: interfaz `EnabledFeatures` en `src/config/nav-config.ts`, fallback en
     `src/sections/template/dashboard-template.tsx`, `FEATURE_KEYS` en
     `src/pages/subscription/my-subscription-page.tsx` + key i18n
     `subscription.plans.feature.<key>`.
  5. `feature: "<key>"` en el ítem de `nav-config.ts`.

El flag solo oculta el ítem; la URL sigue cargando, y lo que protege los datos
es `RequirePermission` en el backend.
