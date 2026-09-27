# Backend (apps/api)

Módulo Go `pengi-med-saas`. Dominio canónico: `features/kanban/` + `routes/kanban-routes.go`.

## Estructura del dominio

```
features/<domain>/
  models/      # structs GORM con TenantID uint
  dto/         # Create<X>Request (campos planos) / Update<X>Request (punteros)
  handlers/    # un archivo por recurso + *_test.go al lado
  services/    # opcional — lógica reutilizada por varios handlers (companies, notifications)
  workers/     # opcional — schedulers o consumers (ver Segundo plano)
  templates/   # opcional — HTML de PDFs + embed.go (ver documentos.md)
```

Alias de import por capa: `<domain>_models`, `<domain>_dto`, `<domain>_handlers`.

## Handler

```go
type <X>Handler struct {
    db     *gorm.DB
    logger *zap.Logger
}

func New<X>Handler(db *gorm.DB, logger *zap.Logger) *<X>Handler {
    return &<X>Handler{db: db, logger: logger}
}

func (h *<X>Handler) Create<X>(c *gin.Context) envelope.Response {
    var req <domain>_dto.Create<X>Request
    if err := c.ShouldBindJSON(&req); err != nil {
        return envelope.ErrorResponse(http.StatusBadRequest, "<domain>.<x>.invalid.request", core_errors.ErrInvalidRequest)
    }
    db := tenantdb.For(c, h.db)
    item := <domain>_models.<X>{TenantID: tenantdb.TenantID(c) /* ... */}
    if err := db.Create(&item).Error; err != nil {
        h.logger.Error("failed to create <x>", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "<domain>.<x>.create.error", core_errors.ErrInternal)
    }
    return envelope.SuccessResponse(item, "<domain>.<x>.create.success")
}
```

Dependencias extra (mailer, `*pdfrender.Renderer`, `tenantfiles.Store`,
`*sri_document.Lifecycle`) se inyectan como argumentos del constructor y se
instancian en el archivo de rutas.

### Aislamiento por tenant — `tenantdb`

ADR: `docs/adr/0002-aislamiento-por-tenant-en-la-capa-de-datos.md`. Un plugin
GORM filtra por `tenant_id` y estampa `TenantID` al crear, **solo** sobre
handles obtenidos así:

| Contexto | Handle |
|---|---|
| Handler HTTP | `tenantdb.For(c, h.db)` |
| Trabajo de fondo sobre un tenant | `tenantdb.ForTenant(db, tenantID)` |
| Trabajo que cruza tenants (workers, migraciones) | `tenantdb.System(db)` |

`tenantdb.TenantID(c)` da el tenant de la request. `TENANTDB_MODE`
(`permissive` / `warn` por defecto / `strict`) decide qué pasa con una query
sin handle: en `strict` falla. `tenant_middleware.TenantScope` es legado.

`tenantdb.For` también lleva los metadatos de auditoría: los modelos que
implementan `IsAuditable() bool` quedan auditados en create/update/delete
automáticamente; lecturas sensibles (datos de paciente) llaman
`audit.RecordAccess(db, c, entityType, id, patientID)` explícitamente.

### Respuestas

- `envelope.SuccessResponse(data, "<i18n.key>")` /
  `envelope.ErrorResponse(status, "<i18n.key>", core_errors.Err<X>)`.
- `envelope.Handle` traduce `Message` y reemplaza el mensaje del `AppError` por
  la traducción de su **código** — por eso cada error code necesita key i18n.
- Handlers que devuelven binarios: ver `documentos.md`.

## Cableado

### Rutas — `routes/<domain>-routes.go`

```go
func Register<Domain>Routes(router *gin.RouterGroup, db *gorm.DB) {
    h := <domain>_handlers.New<X>Handler(db, logger.Log)

    group := router.Group("/<domain>",
        auth_middleware.AuthMiddleware(),
        tenant_middleware.TenantMiddleware(db),
        subscription_middleware.SubscriptionMiddleware(db),
    )
    rp := subscription_middleware.RequirePermission

    group.GET("/<xs>", rp(db, "READ_<X>"), envelope.Handle(h.Get<X>s))
}
```

El orden de middleware vive en cada archivo de rutas; `routes/index.go` →
`RegisterRoutes` solo llama `Register<Domain>Routes(...)` — agrégala ahí.
Con decisión **sin permisos**, omite `SubscriptionMiddleware` y `rp`
(patrón `notification-routes.go`).

### Modelos — `migrations/migrate.go`

Agrega cada modelo nuevo a la lista de `RunMigrations` (AutoMigrate), con import
`<domain>_models`.

### Error codes — `core/errors/codes.go`

```go
// <Domain> Errors
Err<Domain><Detail> = NewAppError("E-<DOM>-001", "English message.")
```

Numera secuencialmente dentro del prefijo. Agrega `{"key": "E-<DOM>-001", "value": ...}`
en ambos JSON de i18n. Hay ~29 códigos históricos sin key; no sumes más.

### i18n — `apps/api/i18n/messages/messages_{es,en}.json`

Array plano `[{"key": "...", "value": "..."}]`, keys `{domain}.{resource}.{action}`.
Se siembran en la BD en cada arranque (`MigrateMessages`). No hay script de
paridad: verifica con

```bash
for k in <key1> <key2>; do grep -c "\"$k\"" apps/api/i18n/messages/messages_es.json apps/api/i18n/messages/messages_en.json; done
```

Evita duplicar keys existentes (`grep` antes de agregar).

### Code-migrations — `migrations/code-migrations/2026/`

- Package `y2026`, archivo `snake_case` (`add_<domain>_permissions.go`),
  registro en `init()`: `database.GlobalDBMap["DB<YYYYMMDD>_<n>"]`.
- Key nueva, mayor que la última registrada:
  `grep -rho 'GlobalDBMap\["[^"]*"\]' apps/api/migrations/code-migrations/2026/*.go | sort | tail -3`
- Una migración ya presente en `origin/main` es inmutable (el hook
  `.claude/hooks/guard-migrations.mjs` bloquea la edición): para cambiar algo,
  archivo y key nuevos.
- Idempotente (`FirstOrCreate`, `Association().Append`), errores con
  `fmt.Errorf("...: %w", err)`, progreso con `fmt.Printf("✅ ...")`.
- Reglas completas y orden de ejecución: `docs/backend/api-code-migration.md`.

## Segundo plano

- Schedulers: `features/<domain>/workers/<x>-scheduler.go` con método `Start()`;
  arráncalo en `cmd/main.go` (`go scheduler.Start()`, junto a
  `archiveScheduler`/`staleDraftScheduler`). No hay registro automático.
- Consumers RabbitMQ: helpers en `core/brokers/rabbitmq/rabbitmq.go`
  (`DeclareQueueWithRetry`, `PublishMessage`, `StartConsumer`); se lanzan con
  `rabbitmq.Run(...)` en `cmd/main.go`.
- Acceso a datos: `tenantdb.System(db)` para barrer todos los tenants,
  `tenantdb.ForTenant(db, id)` al procesar uno.
- Comprobantes SRI: no crees un worker nuevo; agrega un `Kind` en
  `features/billing/sri-document/kinds.go` (ver CLAUDE.md, ADR 0001).

## Entorno dev

`apps/api` corre bajo `air` en el contenedor `pengi-api`: cada guardado
(`.go`, `.html`, `.json`) recompila y re-ejecuta las migraciones. Confirma con
`docker logs pengi-api --tail 100`. BD:
`docker exec -i pengi-db-dev psql -U postgres -d pengi_gentoo -c "<sql>"`.
