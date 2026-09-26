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
- **Si estás extendiendo un dominio existente** (ej. agregar una acción nueva
  a `clinical`), lo más probable es que YA exista un `Feature` con ese
  `Code` asociado al/los `Plan`(s) — no necesitas crear uno nuevo, solo
  agregar tus permisos nuevos a ese `Feature` existente (ver paso 7). Revisa
  qué features existen en dev antes de asumir nada:
  ```bash
  docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo \
    -c "select id, code, name from features;"
  ```

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

### Excepción: handlers que devuelven un binario (PDF, archivo)

Un handler que hace *stream* de un PDF u otro binario **no** devuelve
`envelope.Response` — escribe directo con `c.Data(...)`/`c.JSON(...)` y se
registra en las rutas **sin** `envelope.Handle(...)` (llamada directa al
método). Patrón existente: `DownloadPrescription`/`DownloadMedicalReport` en
`apps/api/features/clinical/handlers/*.go`. No fuerces estos handlers al
patrón `envelope.Response` — no aplica.

### Generar PDFs (Gotenberg) — si el dominio necesita documentos/reportes

No uses una librería de PDF nueva. El patrón establecido es HTML → Gotenberg:

```go
tmpl, _ := template.ParseFiles(tmplPath) // features/<domain>/templates/<name>.html
var buf bytes.Buffer
tmpl.Execute(&buf, data)
client := utils.NewGotenbergClient(os.Getenv("GOTENBERG_URL")) // default http://gotenberg:3000
pdfBytes, err := client.GeneratePDFFromHTMLWithOptions(buf.String(), utils.A4Portrait) // o A5Landscape
```

Las plantillas soportan **override por tenant**: antes de usar la plantilla
default (`features/<domain>/templates/<name>.html`), revisa si existe
`storage/tenants/{tenantID}/<name>.html` y úsala en su lugar (ver
`generatePrescriptionPDF` en `download-record-handler.go` para el patrón
completo, incluyendo fallback).

### Enviar un documento por email

`core/mailer.Mailer` ya soporta adjuntos vía
`SendMedicalDocumentEmail(toEmail, subject, title, filename, pdfBytes)` (base64
+ Resend API). Si necesitas un nuevo tipo de email con adjunto, sigue ese
patrón (`sendWithAttachments`) en vez de reinventarlo. Un error
`resend API error: status 4xx` casi siempre es una restricción del API key de
Resend en modo sandbox (solo permite enviar a la dirección verificada del
dueño de la cuenta) — no asumas que es un bug antes de revisar eso.

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

Retoma la decisión del paso 0. **No te lo saltes ni cuando extiendes un
dominio existente** — es el paso que más fácil se olvida, y el síntoma solo
aparece probando en el navegador (el linter/build no lo detecta):

> **Síntoma si te lo saltas:** en el navegador ves un toast
> `"Your plan does not include this feature"` (403) al llamar el endpoint
> nuevo, **aunque** el usuario/rol sí tenga el permiso asignado (paso 3). Eso
> es porque `RequirePermission` chequea dos cosas por separado — el rol del
> usuario Y que el `Plan` de la suscripción incluya el permiso vía `Feature`
> (`apps/api/features/companies/middleware/subscription-middleware.go`,
> `IsPermissionAllowed`) — y solo wireaste la primera en el paso 3.

- **Si es plan-gated y el `Feature` YA existe** (caso más común — ej. agregas
  una acción a `clinical`, que ya tiene `Feature{Code: "CLINICAL"}` asociado
  al plan PRO): agrega una migración nueva que asocie tus permisos nuevos a
  ese `Feature` existente, igual que el paso 3 los asocia al rol admin pero
  sobre `company_models.Feature` en vez de `user_models.Role`:
  ```go
  var feature company_models.Feature
  if err := db.Where(company_models.Feature{Code: "<DOMAIN>"}).First(&feature).Error; err != nil {
      if err == gorm.ErrRecordNotFound {
          fmt.Println("⚠️  <DOMAIN> feature not found, skipping.")
          return nil // no rompas el arranque en un ambiente que gestiona Features distinto
      }
      return fmt.Errorf("failed to find <DOMAIN> feature: %w", err)
  }
  for _, id := range []string{"CREATE_<X>", "..."} {
      var perm permission_models.Permission
      db.Where(permission_models.Permission{BaseStringID: database.BaseStringID{ID: id}}).First(&perm)
      db.Model(&feature).Association("Permissions").Append(&perm)
  }
  ```
  Ponla en su **propia key** de `GlobalDBMap` (no reuses la del paso 3) para
  que corra después de que los permisos existan. `Association(...).Append(...)`
  es idempotente (GORM no duplica la fila en `feature_permissions` si ya
  existe), así que es seguro si la corres más de una vez.
- **Si es plan-gated y el `Feature` NO existe todavía:** créalo (vía
  backoffice UI `apps/backoffice/src/pages/features/*` o seed) con
  `Feature{Code, Name, Permissions}`, y asócialo a el/los `Plan`(s)
  relevantes (`apps/backoffice/src/pages/plans/*`).
- Para depurar en dev, inspecciona la tabla directamente (no asumas por logs):
  ```bash
  docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo \
    -c "select feature_id, permission_id from feature_permissions where feature_id = <id>;"
  ```
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

### ¿Diálogo (modal) o página independiente?

Un `Dialog` (patrón `edit-prescription-dialog.tsx`) está bien para una
edición rápida de 1-2 campos disparada desde una fila/acción puntual. Si el
flujo genera un registro que el usuario querrá **volver a ver, imprimir o
reenviar más tarde** (informes, documentos, cualquier cosa "generada"), usa
una página dedicada + una página de listado, no un modal — un modal no tiene
URL propia ni forma natural de listar lo ya guardado. Regla práctica: si te
preguntas "¿y cómo veo los que ya generé?", es una página, no un diálogo.
Cuidado además con grids de N columnas dentro de un `Dialog` angosto para
mostrar fechas largas (`FormCalendar` con `format(date, "PPP")` en español
puede desbordar una columna de 3 en un modal de 600px) — en una página con
más ancho este problema desaparece solo.

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

**Gotcha de caché en dev:** el frontend guarda los mensajes en
`localStorage["messages"]` (`src/store/message-store.ts`) y solo los vuelve a
pedir al backend si cambia `__APP_VERSION__`. Si agregaste keys nuevas y las
ves como `*key.nueva*` en el navegador aunque ya estén en el JSON y sembradas
en la BD (verificable con
`docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo -c "select value from messages where key='...';"`),
no es un bug — es el caché del navegador. Limpialo antes de dar el smoke test
por fallido:
```js
localStorage.removeItem("messages"); // luego recarga la página
```

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

- **Entorno docker dev:** `apps/api` corre bajo `air` (`docker-compose.dev.yaml`)
  con el código montado como volumen — cada guardado dispara un rebuild
  automático (`docker logs pengi-api` muestra `building... / running...`).
  No hace falta reiniciar el contenedor a mano. `RunAllMigrations` (incluidas
  tus code-migrations nuevas) corre en cada arranque del binario, así que
  una migración nueva se aplica sola en el siguiente rebuild — solo
  confirma en los logs (`docker logs pengi-api | grep <tu-key>`) que
  imprimió éxito y no quedó en loop de error.
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
- [ ] Si el Feature de plan ya existía (dominio extendido), permisos nuevos asociados
      a él vía migración propia (no solo al rol admin) — probado en navegador,
      no solo por build/lint
- [ ] Tipos/servicio/store frontend creados
- [ ] Página/componentes/formularios creados (Zod + Form components)
- [ ] Si genera documentos: reutiliza Gotenberg (no libs de PDF nuevas) y el
      handler de descarga NO usa envelope.Handle
- [ ] Si el flujo produce registros para revisar después: página + listado,
      no un Dialog
- [ ] Tests backend escritos y pasando (just tests-api, go vet ./...)
- [ ] Tests frontend pasando (just tests-web)
- [ ] /doctor (react-doctor) corrido sin issues nuevos
- [ ] Smoke test manual de la pantalla nueva (sin *key* renderizado — limpiar
      localStorage["messages"] si agregaste i18n keys nuevas —, sin errores
      de consola, y probando en navegador que el plan/feature-gate no bloquea)
```

## Referencias

- [`docs/skills/api-backend-complete-guide.md`](../../../docs/skills/api-backend-complete-guide.md)
- [`docs/skills/web-frontend-complete-guide.md`](../../../docs/skills/web-frontend-complete-guide.md)
- [`docs/skills/form-creation-standard.md`](../../../docs/skills/form-creation-standard.md)
- `react-doctor` skill (`/doctor`) — `apps/web/.claude/skills`, `apps/backoffice/.claude/skills`
