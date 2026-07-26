---
name: create-feature
description: Use when asked to create a new feature, add a feature, build a new domain/module, or extend an existing domain in pengi-med-saas (apps/api Go backend, apps/web React frontend, apps/backoffice React admin). End-to-end checklist covering backend scaffolding, permissions, migrations, error codes, i18n, feature-flag/plan wiring, frontend routes/nav, and tests — so no step is missed. Triggers on "crear feature", "nuevo feature", "add new domain", "nueva funcionalidad", "implement X feature".
---

# Crear una feature nueva en pengi-med-saas

Checklist de extremo a extremo. Sigue los pasos en orden — cada uno referencia
el paso anterior cuando corresponde. Los identificadores de código (rutas,
structs, funciones, constantes) están en inglés tal como aparecen en el repo.

Esta skill **orquesta**; para el detalle de "cómo se ve el código" de modelo →
DTO → handler → página → componente, remite a las guías completas que ya
existen en `docs/skills/`. Lo que agrega esta skill son los pasos que esas
guías **no** cubren: permisos, feature-flags/plan, tests y pre-flight.

## 0. Antes de empezar — investigar primero

Responde esto antes de escribir código:

- ¿Es un dominio 100% nuevo (`apps/api/features/<domain>/` no existe) o una
  extensión de uno existente? Si ya existe, lee sus archivos actuales antes
  de tocar nada.
- ¿Necesita un modelo GORM nuevo, un worker en background (`workers/`), o
  middleware propio (`middleware/`)?
- Enumera los permisos nuevos que necesita (verbo + recurso, ej.
  `READ_X`, `CREATE_X`, `UPDATE_X`, `DELETE_X`).
- **¿Debe estar detrás de un plan/feature-flag, o debe estar disponible por
  defecto?** Hay dos mecanismos distintos en este repo, no los confundas:
  - **Enforcement real** (bloquea acceso a datos/endpoints): `Feature` ↔
    `Plan` ↔ `Permission`, ver `apps/api/features/companies/models/feature-model.go`.
    Si la feature debe limitarse por plan de suscripción, hay que crear un
    `Feature` con un `Code` explícito y asociarlo a los `Plan`(s)
    correspondientes (ver paso 7).
  - **Cosmético** (solo oculta/muestra el ítem del menú): `Tenant.EnabledFeatures`
    + `nav-config.ts` `item.feature`. La semántica es **opt-out**: un
    `feature` string nuevo que no exista en `EnabledFeatures` se considera
    HABILITADO por defecto. Si quieres que la feature nueva esté oculta hasta
    activarse manualmente, el default debe ser `false` explícito (ver paso 7).
  - Decide explícitamente cuál aplica (o ninguno) y anótalo — no lo dejes
    implícito.
- **Corrección de nombre:** el paquete base compartido es
  `apps/api/features/companies/` (plural) — ahí viven `Company`, `Plan`,
  `Feature`, `Subscription`, `SubscriptionMiddleware` y `RequirePermission`.
  No existe `features/company` (singular).

## 1. Backend — scaffolding del dominio

Sigue los pasos 1-3 de
[`docs/skills/api-backend-complete-guide.md`](../../../docs/skills/api-backend-complete-guide.md)
(modelo → DTOs → handler). Recordatorio rápido de la convención:

```
apps/api/features/<domain>/
  models/       # gorm.Model + TenantID
  dto/          # Create (campos planos) / Update (todos punteros)
  handlers/     # struct { db *gorm.DB; logger *zap.Logger } + New<Domain>Handler
  workers/      # opcional — consumers de RabbitMQ
  middleware/   # opcional — solo si el dominio necesita su propio gate
```

Los handlers **siempre** devuelven `envelope.Response`, nunca escriben
directo a `gin.Context`.

## 2. Backend — Error codes

En `apps/api/core/errors/codes.go`, agrupados por dominio (bloque de
comentario + numeración secuencial desde 001):

```go
// --- <DOMAIN> ---
Err<Domain><Detail> = NewAppError("E-<DOMAIN>-001", "Mensaje.")
```

## 3. Backend — Permisos (paso clave, no te lo saltes)

1. Agrega un slice nuevo en
   `apps/api/features/permissions/data/permission-data.go`:
   ```go
   var <Domain>Permissions = []permission_models.Permission{
       {
           BaseStringID: database.BaseStringID{ID: "READ_<X>"},
           Name:         "Read <X>",
           Category:     "<DOMAIN>",
           Description:  "...",
       },
       // CREATE_<X>, UPDATE_<X>, DELETE_<X>, etc.
   }
   ```
2. Crea un code-migration nuevo en
   `apps/api/migrations/code-migrations/2026/<domain>_permissions.go`
   (package `y2026`), siguiendo el patrón exacto de
   `add_kanban_permissions.go`:
   ```go
   func init() {
       database.GlobalDBMap["DB<YYYYMMDD>_<n>"] = database.DBExecute{
           ID: "DB<YYYYMMDD>_<n>",
           Execute: func(db *gorm.DB) error {
               var adminRole user_models.Role
               if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
                   return fmt.Errorf("failed to find admin role: %w", err)
               }
               for _, perm := range permission_data.<Domain>Permissions {
                   if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
                       return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
                   }
                   if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
                       return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
                   }
               }
               return nil
           },
       }
   }
   ```
   **Revisa las keys ya usadas antes de elegir una** (`grep -rho
   'GlobalDBMap\["[^"]*"\]' apps/api/migrations/code-migrations/2026/*.go`)
   — no colisiones con una fecha/índice ya registrado.
3. En `apps/api/routes/<domain>-routes.go`, envuelve cada endpoint con
   `subscription_middleware.RequirePermission(db, "ACTION_RESOURCE")` usando
   el mismo string ID exacto del paso 1.
4. Espeja en el frontend, `apps/web/src/lib/constants.ts`, dentro de
   `PERMISSIONS`:
   ```ts
   <DOMAIN>: {
     PERMISSION_READ_<X>: "READ_<X>",
     PERMISSION_CREATE_<X>: "CREATE_<X>",
     // ...
   },
   ```
   El string value debe ser **idéntico byte a byte** al ID del backend, no
   solo el nombre de la constante.
5. En `apps/web/src/routes/routes.tsx`, envuelve el grupo de rutas y cada
   ruta individual en `<CheckPermission permissions={[PERMISSIONS.<DOMAIN>.PERMISSION_X]}>`.
6. En `apps/web/src/config/nav-config.ts`, el nav item del dominio debe tener
   `permission: PERMISSIONS.<DOMAIN>.PERMISSION_READ_<X>`.

## 4. Backend — Migraciones de esquema

Agrega el/los modelo(s) nuevo(s) a la lista explícita de
`apps/api/migrations/migrate.go` → `RunMigrations` (GORM `AutoMigrate`). No
confundir con el code-migration de permisos del paso 3 — son mecanismos
distintos (schema vs. data/seed).

## 5. Backend — i18n

Agrega las keys en **ambos** `apps/api/i18n/messages/messages_es.json` y
`messages_en.json` (array plano `{"key": ..., "value": ...}`). Si la key
corresponde a un error code, debe ser exactamente igual (`E-<DOMAIN>-<NNN>`).
Verifica con grep que la key existe en los dos archivos antes de continuar.

## 6. Backend — Registro final de rutas

Agrega la llamada `Register<Domain>Routes(...)` dentro de
`apps/api/routes/index.go` → `RegisterRoutes`. Orden de middleware de grupo:

```
auth_middleware.AuthMiddleware()
  → tenant_middleware.TenantMiddleware(db)
  → subscription_middleware.SubscriptionMiddleware(db)
```

## 7. Feature-flag / Plan wiring

Retoma la decisión del paso 0:

- **Si es plan-gated:** crea el `Feature{Code, Name, Permissions}` (vía
  backoffice UI `apps/backoffice/src/pages/features/*` o seed), asocia los
  `Permission`s del paso 3, y asócialo a el/los `Plan`(s) relevantes
  (`apps/backoffice/src/pages/plans/*`). Confirma que `SubscriptionMiddleware`
  calcula bien `allowed_permissions` para un tenant en ese plan.
- **Si es toggle cosmético:** agrega el bool field en
  `apps/api/features/tenants/models/tenant-model.go` → `EnabledFeatures`,
  actualiza `GetEnabledFeatures`/`UpdateEnabledFeatures` si aplica, y usa
  exactamente el mismo string en `nav-config.ts` → `item.feature`. Recuerda
  la semántica opt-out — si debe iniciar oculta, el default debe ser `false`
  explícito, no un zero-value implícito sin revisar.
- **Si no aplica ninguno:** déjalo explícito en la descripción del PR/commit
  para que quede claro que fue una decisión y no un olvido.

## 8. Frontend — Tipos, servicio, store

Sigue los pasos 1-3 de
[`docs/skills/web-frontend-complete-guide.md`](../../../docs/skills/web-frontend-complete-guide.md).
Recordatorios rápidos:

- Tipos extienden `BaseModel` (`ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`,
  PascalCase) + campos de negocio en snake_case.
- Servicio vía `createHttpService(apiWithTenant | api | noAuthApi)`. GET =
  `{ notifyError: true }`; writes = `{ notifySuccess: true, notifyError: true }`.
- Store Zustand solo si el estado es compartido entre componentes:
  `persist((set) => ({...}), { name: "<feature>-storage", storage: createJSONStorage(() => sessionStorage) })`.

## 9. Frontend — Página, componentes, formularios

Sigue la guía frontend para página/componentes. Para cualquier formulario,
sigue [`docs/skills/form-creation-standard.md`](../../../docs/skills/form-creation-standard.md):
Zod schema + `<Form schema={} onSubmit={}>` +
`FormInput/FormSelect/FormTextArea/FormRadioGroup` (de `@pengi/ui`) +
`FormCalendar/FormTagInput` (locales a cada app; `FormTagInput` solo existe
en `apps/web`) — nunca inputs HTML crudos. `FormCheckbox` ya no existe.

## 10. Frontend — Rutas y navegación

Ya cubierto en el paso 3 (`CheckPermission` en `routes.tsx`, `permission` en
`nav-config.ts`) y paso 7 (`feature` en `nav-config.ts` si aplica gating
cosmético). No dupliques código, solo confirma que ambos campos están
correctamente seteados en el nav item.

## 11. Frontend — i18n

Las keys son backend-owned — no hay JSON local en `apps/web`. Confirma que
las keys usadas en `textGet(key)` existen en `messages_es.json`/`messages_en.json`
(paso 5). `textGet` falla en silencio devolviendo `*key*` — smoke-testea
visualmente la pantalla nueva para detectar cualquier `*key*` renderizado.

## 12. Tests

- **Backend:** escribe/actualiza `*_test.go` junto al handler nuevo, usando
  `testutils.SetupTestDB` + `testutils.NewGinContext`, invocando el handler
  directamente y asertando sobre `envelope.Response` (patrón de
  `invoice_handler_test.go`).
- Corre `just tests-api` y `go vet ./...` localmente — el pre-commit hook
  (`.githooks/pre-commit`) **no** corre esto, solo Biome + `pnpm run typecheck`
  en `apps/web`/`apps/backoffice`.
- **Frontend:** agrega tests si el patrón del feature similar los tiene;
  corre `just tests-web`.
- Corre `just tests-e2e` para flujos críticos si aplica.

## 13. Pre-flight / Doctor

- **Frontend:** corre `/doctor` (skill `react-doctor`, ya existe en
  `apps/web/.claude/skills` y `apps/backoffice/.claude/skills`) antes de dar
  por terminado el trabajo frontend — no reinventes lint/a11y/bundle-size
  checks a mano.
- **Backend:** `go vet ./...`.
- Confirma que `pnpm run typecheck` pasa (el hook ya lo corre, pero verifica
  a mano si hiciste cambios post-commit).

## 14. Definition of Done

```
- [ ] Modelo(s)/DTOs/handler creados y compilando
- [ ] Error codes agregados en codes.go
- [ ] Permisos agregados en permission-data.go
- [ ] Code-migration de permisos creado y registrado en GlobalDBMap (key sin colisión)
- [ ] RequirePermission agregado en las rutas correspondientes
- [ ] PERMISSIONS espejado en apps/web/src/lib/constants.ts (strings idénticos)
- [ ] CheckPermission envolviendo rutas en routes.tsx
- [ ] permission asignado en nav-config.ts
- [ ] Modelo(s) agregados a RunMigrations (AutoMigrate)
- [ ] Rutas registradas en routes/index.go
- [ ] i18n keys agregadas en messages_es.json Y messages_en.json
- [ ] Decisión de feature-flag/plan tomada y documentada; wiring hecho si aplica
- [ ] Tipos/servicio/store frontend creados
- [ ] Página/componentes/formularios creados (Zod + Form components)
- [ ] Tests backend escritos y pasando (just tests-api, go vet ./...)
- [ ] Tests frontend pasando (just tests-web)
- [ ] /doctor (react-doctor) corrido sin issues nuevos
- [ ] Smoke test manual de la pantalla nueva (sin *key* renderizado, sin errores de consola)
```

## Referencias

- [`docs/skills/api-backend-complete-guide.md`](../../../docs/skills/api-backend-complete-guide.md)
- [`docs/skills/web-frontend-complete-guide.md`](../../../docs/skills/web-frontend-complete-guide.md)
- [`docs/skills/form-creation-standard.md`](../../../docs/skills/form-creation-standard.md)
- `react-doctor` skill (`/doctor`) — `apps/web/.claude/skills`, `apps/backoffice/.claude/skills`
