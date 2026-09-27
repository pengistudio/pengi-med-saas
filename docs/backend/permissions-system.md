# Sistema de Permisos, Roles y Planes

Cómo se decide si un usuario puede usar un endpoint o ver una pantalla en
Pengi Med SaaS. La receta para agregar permisos a una feature está en la skill
`create-feature` (`.claude/skills/create-feature/references/permisos-y-plan.md`).

## 🎯 Visión General

Hay **dos llaves** independientes, y un endpoint normal exige las dos:

1. **Rol:** el rol del usuario en la empresa (`Environment.Role`) tiene el permiso.
2. **Plan:** el plan de la suscripción activa de la empresa incluye el permiso,
   a través de sus `Feature`s.

Los permisos son granulares (`READ_PATIENT`, `CREATE_BILLING`,
`MANAGE_SRI_SETTINGS`) y cada uno pertenece a una **categoría** (`CLINICAL`,
`BILLING`, `TEAM`, `KANBAN`, `AUDIT`). La categoría decide qué ítems del menú
se muestran según el plan.

## 🏗️ Modelo de datos

```
Permission                                 features/permissions/models/permission-model.go
├── ID string (BaseStringID)               "READ_PATIENT"
├── Name, Description
└── Category                               "CLINICAL"

Role ── many2many ── Permission            features/users/models/user-model.go
Environment {UserID, CompanyID, TenantID, RoleID}   = usuario + empresa + rol

Feature {Code, Name} ── many2many feature_permissions ── Permission
Plan {Code, ...} ── many2many plan_features ── Feature       features/companies/models/
Subscription {CompanyID, PlanCode, Status, ExpiresAt}
```

- **Roles canónicos:** `admin`, `doctor`, `recepcionista`, `contador`. `admin`
  recibe todos los permisos; los demás, los de
  `role_data.RolePermissionMatrix` (`features/users/data/role-data.go`),
  sembrados por `seed_canonical_roles.go`.
- **Features y planes** no se siembran en código (salvo el plan `TRIAL`): se
  crean en el backoffice (`apps/backoffice/src/pages/features`, `pages/plans`)
  y existen solo en la BD de cada ambiente.

## 🔐 Validación en el backend

Middleware en `features/companies/middleware/subscription-middleware.go`
(import `subscription_middleware "pengi-med-saas/features/companies/middleware"`):

| Middleware | Qué hace |
|---|---|
| `SubscriptionMiddleware(db)` | En el grupo de rutas, tras `TenantMiddleware`. Exige una suscripción activa (con 3 días de gracia tras el vencimiento) y carga en el contexto los permisos que permite su plan. Sin suscripción → 403. |
| `RequirePermission(db, id)` | Por ruta. El plan incluye el permiso (si no: 403 *"Your plan does not include this feature"*) **y** el rol lo tiene. |
| `RequireRolePermission(db, id)` | Por ruta. Solo el rol; para administración básica que no depende del plan (equipo). |

```go
teamGroup := router.Group("/team",
    auth_middleware.AuthMiddleware(),
    tenant_middleware.TenantMiddleware(db),
)
rp := subscription_middleware.RequireRolePermission
teamGroup.GET("", rp(db, "READ_TEAM"), envelope.Handle(companyHandler.GetTeamMembers))

kanbanGroup := router.Group("/kanban",
    auth_middleware.AuthMiddleware(),
    tenant_middleware.TenantMiddleware(db),
    subscription_middleware.SubscriptionMiddleware(db),
)
rp := subscription_middleware.RequirePermission
kanbanGroup.GET("/tasks", rp(db, "READ_KANBAN"), envelope.Handle(kanbanHandler.GetTasks))
```

Límites numéricos por plan (`Plan.Properties`) se consultan con
`GetPlanLimitForCompany` / `ExceedsPlanLimit` (`features/companies/middleware/plan-limit.go`).

## 🖥️ Validación en el frontend

- El environment de la sesión trae `permissions` = los permisos **del rol**
  (sin cruzar con el plan). `usePermission().checkPermission([...])` exige
  todos los listados.
- Rutas: `<CheckPermission permissions={[...]}>` (default export de
  `@/components/custom/check-permission`) redirige a `/` si falta alguno.
- Constantes: `apps/web/src/lib/constants.ts` → `PERMISSIONS.<GRUPO>.PERMISSION_<ACCION>_<RECURSO>`,
  con el string **idéntico** al ID del backend. Los grupos no siempre coinciden
  con la categoría (los permisos `CLINICAL` están en el grupo `MEDICAL_RECORD`).

Como el frontend solo mira el rol, un usuario puede entrar a una pantalla cuyo
endpoint luego responde 403 por plan. El backend es la única barrera real.

## 🎨 Features habilitados (menú)

`EnabledFeatures {clinical, billing, team, kanban}` se **calcula** en cada
request, no se guarda ni se edita:

- `company_services.CalculateEnabledFeaturesFromFeatures`
  (`features/companies/services/enabled-features-service.go`) marca
  `clinical: true` si algún `Feature` del plan tiene un permiso con
  `Category == "CLINICAL"`, y así con cada categoría.
- Sin suscripción, todos quedan en `true` (`DefaultEnabledFeatures`).
- Se entrega en `environment.enabled_features` (y en `GET /tenants/features`).
- El menú (`nav-config.ts`, `dashboard-template.tsx`) oculta un ítem con
  `feature: "<key>"` solo si ese flag es `false`.

Agregar una categoría nueva que afecte el menú requiere tocar: el struct
`EnabledFeatures` y `DefaultEnabledFeatures` (`features/tenants/models/tenant-model.go`),
`CalculateEnabledFeaturesFromFeatures`, los mapas de
`dashboard-handler.go` y `backoffice-plan-handler.go`, y en web la interfaz
`EnabledFeatures`, el fallback de `dashboard-template.tsx` y `FEATURE_KEYS` de
`my-subscription-page.tsx`. Detalle en la skill `create-feature`.

## ➕ Cómo crear nuevos permisos

1. **Catálogo** — `features/permissions/data/permission-data.go`:
   ```go
   var KanbanPermissions = []permission_models.Permission{
       {
           BaseStringID: database.BaseStringID{ID: "READ_KANBAN"},
           Name:         "Read Kanban",
           Category:     "KANBAN",
           Description:  "View kanban tasks",
       },
   }
   ```
2. **Migración nueva** que los crea y los asigna al rol `admin` (patrón
   `migrations/code-migrations/2026/add_kanban_permissions.go`; reglas en
   [`api-code-migration.md`](api-code-migration.md)).
3. **Roles no-admin:** decide si doctor, recepcionista o contador los reciben.
   Si sí, agrégalos a `RolePermissionMatrix` (tenants nuevos) y asígnalos en la
   migración a los roles existentes (tenants actuales).
4. **Plan:** si la ruta usa `RequirePermission`, asocia los permisos al
   `Feature` del plan con una migración propia (patrón
   `add_medical_document_feature_permissions.go`) o, si el `Feature` no existe,
   créalo en el backoffice y agrégalo a los planes. Sin esto, el endpoint
   responde 403 aunque el rol tenga el permiso.
5. **Rutas:** `rp(db, "<ID>")` en cada endpoint.
6. **Frontend:** `PERMISSIONS` en `constants.ts`, `CheckPermission` en
   `routes.tsx`, `permission` en el ítem de `nav-config.ts`.

## 🐛 Depurar en dev

```bash
# Features y sus permisos
docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo -c "select id, code from features;"
docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo -c "select feature_id, permission_id from feature_permissions where feature_id = <id>;"
```

| Síntoma | Causa probable |
|---|---|
| 403 *"Your plan does not include this feature"* | El permiso no está en ningún `Feature` del plan (paso 4) |
| 403 *"Insufficient permissions"* o redirect a `/` | El rol no tiene el permiso (pasos 2-3) |
| 403 *"No active subscription found"* | La empresa no tiene suscripción activa |
| Ítem de menú oculto | La categoría no está en el plan, o falta el mapeo del flag |

## 📚 Referencias

- Catálogo: `features/permissions/data/permission-data.go`
- Roles: `features/users/data/role-data.go`, `features/users/models/user-model.go`
- Middleware: `features/companies/middleware/subscription-middleware.go`
- Flags de menú: `features/companies/services/enabled-features-service.go`
- Frontend: `apps/web/src/hooks/use-permission.ts`, `src/lib/constants.ts`,
  `src/components/custom/check-permission.tsx`, `src/config/nav-config.ts`
