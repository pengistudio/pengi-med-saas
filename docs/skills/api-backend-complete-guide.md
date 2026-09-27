---
name: API Backend — Complete Guide
description: Guía completa del backend Go/Gin/GORM incluyendo arquitectura, patrones, y cómo implementar nuevas features
---

# API Backend — Complete Guide

Guía de la arquitectura y los patrones del backend `apps/api` (Go + Gin + GORM + PostgreSQL).

> Para crear o extender una feature de punta a punta (permisos, plan, frontend,
> tests), la receta es la skill `create-feature` (`.claude/skills/create-feature/`).
> Esta guía es la referencia de arquitectura y patrones del backend.
> Dominio de referencia para copiar: `features/kanban/` + `routes/kanban-routes.go`.

## 📐 Arquitectura General

### Stack Técnico

- **Lenguaje:** Go (módulo `pengi-med-saas`)
- **Framework HTTP:** Gin
- **ORM:** GORM con PostgreSQL
- **Logging:** Zap (`core/logger`, singleton `logger.Log`)
- **Messaging:** RabbitMQ (comprobantes SRI)
- **PDFs:** Gotenberg, siempre vía `core/pdfrender`

### Estructura de Directorios

```
apps/api/
├── cmd/main.go                  # Entrypoint: conexión, migraciones, workers, router
├── core/
│   ├── audit/                   # Plugin de auditoría (IsAuditable) + RecordAccess
│   ├── auth/                    # JWT, password hashing
│   ├── brokers/rabbitmq/        # Colas, publish, consumers
│   ├── config/                  # Configuración
│   ├── database/                # Conexión GORM, GlobalDBMap de code-migrations
│   ├── envelope/                # Response wrapper (Handle, Success, Error)
│   ├── errors/                  # AppError + error codes
│   ├── logger/                  # Zap singleton
│   ├── mailer/                  # Emails (Resend), con adjuntos
│   ├── middleware/              # Middleware global (rate limiter)
│   ├── pdfrender/               # HTML → PDF con override por tenant
│   ├── tenantdb/                # Aislamiento por tenant en la capa de datos
│   ├── tenantfiles/             # Archivos por tenant (P12, logo, XML, plantillas)
│   └── utils/
├── features/
│   └── <dominio>/
│       ├── handlers/            # Un archivo por recurso + *_test.go
│       ├── models/              # Modelos GORM
│       ├── dto/                 # Request DTOs
│       ├── services/            # Opcional: lógica compartida entre handlers
│       ├── workers/             # Opcional: schedulers / consumers
│       ├── templates/           # Opcional: HTML de PDFs + embed.go
│       ├── data/                # Opcional: catálogos estáticos (permissions, users)
│       └── middleware/          # Opcional: middleware del dominio
├── i18n/messages/               # messages_es.json, messages_en.json (+ embed.go)
├── migrations/
│   ├── migrate.go               # RunMigrations (AutoMigrate) + RunAllMigrations
│   └── code-migrations/2026/    # Migraciones de datos (package y2026)
├── routes/
│   ├── index.go                 # RegisterRoutes(): llama a cada Register<X>Routes
│   ├── documents.go             # tenantFiles + documentRenderer() compartidos
│   └── <dominio>-routes.go      # Rutas + middleware del dominio
└── testutils/                   # SetupTestDB, NewGinContext, Ptr
```

### Dominios Existentes

| Dominio | Propósito | Categoría de permisos |
|---------|-----------|----------|
| `audit` | Consulta del log de auditoría | `AUDIT` |
| `backoffice` | API del panel de administración de plataforma | — (auth propia de backoffice) |
| `billing` | Facturación electrónica SRI | `BILLING` |
| `clinical` | Pacientes, citas, historias, documentos médicos | `CLINICAL` |
| `companies` | Empresas, planes, features, suscripciones, equipo | `TEAM` |
| `contact` | Formulario de contacto público | — |
| `health` | Health check | — |
| `integrations` | Integraciones por tenant (p. ej. calendario) | — |
| `kanban` | Tareas | `KANBAN` |
| `notifications` | Notificaciones y anuncios | — |
| `permissions` | Catálogo de permisos (RBAC) | — |
| `settings` | Configuración | — |
| `tenants` | Multi-tenancy, ajustes del tenant, logo | — |
| `users` | Auth, JWT, environments, roles | — |

Los IDs de permiso son `<ACCION>_<RECURSO>` (`READ_KANBAN`, `CREATE_PATIENT`,
`MANAGE_SRI_SETTINGS`) y viven en `features/permissions/data/permission-data.go`.
Detalle en [`docs/backend/permissions-system.md`](../backend/permissions-system.md).

---

## 🏢 Multi-Tenancy Model

```
User
  ├─ Environment[0] → {CompanyID, TenantID, RoleID}
  └─ Environment[1] → ...

Company ── Subscription → Plan → Features → Permissions
  └─ Tenant[0..n]

Modelos de dominio: TenantID uint
```

`TenantMiddleware(db)` lee el header `X-Tenant-Slug`, verifica que el usuario
pertenece al tenant y deja `tenant_id`, `company_id` y `environment_id` en el
contexto.

### Aislamiento — `core/tenantdb`

ADR: [`docs/adr/0002-aislamiento-por-tenant-en-la-capa-de-datos.md`](../adr/0002-aislamiento-por-tenant-en-la-capa-de-datos.md).
Un plugin GORM agrega `tenant_id = ?` a cada consulta y estampa `TenantID` al
crear, sobre los handles obtenidos así:

| Contexto | Handle |
|---|---|
| Handler HTTP | `db := tenantdb.For(c, h.db)` |
| Trabajo de fondo sobre un tenant | `tenantdb.ForTenant(db, tenantID)` |
| Trabajo que cruza tenants (workers, migraciones, backoffice) | `tenantdb.System(db)` |

- `tenantdb.TenantID(c)` devuelve el tenant de la request.
- `TENANTDB_MODE` = `permissive` | `warn` (default) | `strict`. En `strict` una
  consulta sobre una tabla con `tenant_id` sin handle de tenant falla.
- `tenantdb.For` también propaga el usuario para la auditoría.
- `tenant_middleware.TenantScope` / `AuditScope` son legado: el código nuevo usa `tenantdb`.

---

## 🔄 Flujo de Request

```
HTTP Request
    ↓
RateLimiter (rutas de auth)
    ↓
AuthMiddleware (JWT → user_id)
    ↓
TenantMiddleware (X-Tenant-Slug → tenant_id)
    ↓
SubscriptionMiddleware (suscripción activa → permisos permitidos por el plan)
    ↓
RequirePermission / RequireRolePermission (por ruta)
    ↓
envelope.Handle(handler.Method)
    ├─ handler.Method(c) envelope.Response
    └─ traduce Message (i18n key) y el error code
    ↓
HTTP Response (JSON)
```

---

## 📦 Patrones Core

### Handler Pattern

```go
type KanbanHandler struct {
    db     *gorm.DB
    logger *zap.Logger
}

func NewKanbanHandler(db *gorm.DB, logger *zap.Logger) *KanbanHandler {
    return &KanbanHandler{db: db, logger: logger}
}

func (h *KanbanHandler) GetTasks(c *gin.Context) envelope.Response {
    db := tenantdb.For(c, h.db)
    var tasks []kanban_models.Task
    if err := db.Order("position ASC").Find(&tasks).Error; err != nil {
        h.logger.Error("failed to fetch tasks", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "kanban.tasks.fetch.error", core_errors.ErrInternal)
    }
    return envelope.SuccessResponse(tasks, "kanban.tasks.fetch.success")
}
```

**Reglas:**
- Struct con `db` + `logger`; dependencias extra (`*mailer.Mailer`,
  `*pdfrender.Renderer`, `tenantfiles.Store`) como argumentos del constructor.
- Constructor `New<X>Handler`.
- Métodos retornan `envelope.Response` (excepción: descargas de binarios, ver abajo).
- Toda consulta de datos de tenant sale de `tenantdb.For(c, h.db)`.
- Logs con `zap`.

### DTO Pattern (Validación)

```go
// Create: campos planos con binding
type CreateItemRequest struct {
    Name   string `json:"name" binding:"required,min=1"`
    Email  string `json:"email" binding:"required,email"`
    Status string `json:"status" binding:"required,oneof=active inactive"`
}

// Update: todos los campos como punteros (partial updates)
type UpdateItemRequest struct {
    Name   *string `json:"name"`
    Email  *string `json:"email" binding:"omitempty,email"`
    Status *string `json:"status" binding:"omitempty,oneof=active inactive"`
}
```

### Modelo Pattern (GORM)

```go
type Item struct {
    gorm.Model
    TenantID    uint   `gorm:"not null;index" json:"tenant_id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

// Opcional: auditar create/update/delete automáticamente
func (Item) IsAuditable() bool { return true }
```

Lecturas sensibles (datos de paciente) se registran explícitamente con
`audit.RecordAccess(db, c, entityType, id, patientID)`.

### Response Envelope Pattern

```go
envelope.SuccessResponse(data, "domain.resource.action.success")
envelope.PagedSuccessResponse(items, int(total), page, limit, "domain.resource.list.success")
envelope.ErrorResponse(http.StatusBadRequest, "domain.resource.invalid.request", core_errors.ErrInvalidRequest)
```

El segundo argumento es siempre una **key i18n**, nunca `err.Error()`. El
error real va al log. `envelope.Handle` traduce la key y también reemplaza el
mensaje del `AppError` por la traducción de su código.

### Descargas de binarios (PDF, archivos)

Escriben directo con `c.Data(...)` y se registran **sin** `envelope.Handle`:

```go
recordGroup.GET("/:id/prescription/download", rp(db, "UPDATE_PRESCRIPTION"), downloadHandler.DownloadPrescription)
```

PDFs: `renderer.Render(tenantdb.TenantID(c), "<name>.html", data, utils.A4Portrait)`
de `core/pdfrender`; la plantilla default va en `features/<dominio>/templates/`
con su `embed.go`, sumada a `documentRenderer()` en `routes/documents.go`.
Archivos del tenant: `core/tenantfiles`. Nunca construyas rutas
`storage/tenants/...` ni llames a Gotenberg a mano.

### Error Codes Pattern

```go
// core/errors/codes.go
var (
    // Item Errors
    ErrItemNotFound AppError = NewAppError("E-ITEM-001", "Item not found.")
)
```

**Formato:** `E-<PREFIJO>-<NNN>`, numeración secuencial por prefijo. Cada código
necesita su key en **ambos** JSON de i18n (`{"key": "E-ITEM-001", "value": "..."}`);
si falta, el cliente ve el código literal.

Prefijos en uso: `INT` (interno/genérico e integraciones), `AUTH`, `USR`, `COMP`,
`TEN`, `TEAM`, `PLAN`, `PERM`, `CLIN`, `BILL`, `BO` (backoffice), `ANN`
(anuncios), `NOTIF`, `AUDIT`, `MES`. Para errores genéricos reutiliza
`ErrInternal` / `ErrInvalidRequest`.

### Middleware Pattern

El orden de middleware se declara en cada archivo de rutas:

```go
func RegisterKanbanRoutes(router *gin.RouterGroup, db *gorm.DB) {
    kanbanHandler := kanban_handlers.NewKanbanHandler(db, logger.Log)

    kanbanGroup := router.Group("/kanban",
        auth_middleware.AuthMiddleware(),
        tenant_middleware.TenantMiddleware(db),
        subscription_middleware.SubscriptionMiddleware(db),
    )

    rp := subscription_middleware.RequirePermission

    kanbanGroup.GET("/tasks", rp(db, "READ_KANBAN"), envelope.Handle(kanbanHandler.GetTasks))
    kanbanGroup.POST("/tasks", rp(db, "CREATE_KANBAN"), envelope.Handle(kanbanHandler.CreateTask))
}
```

- `RequirePermission(db, id)`: el plan de la suscripción incluye el permiso **y** el rol lo tiene.
- `RequireRolePermission(db, id)`: solo el rol (equipo, `routes/company_routes.go`).
- Sin permisos: el grupo lleva solo auth + tenant (`routes/notification-routes.go`).

Import: `subscription_middleware "pengi-med-saas/features/companies/middleware"`.

---

## 🛠️ Cómo Implementar un Feature Nuevo

La receta completa (incluidos permisos, plan, frontend y tests) es la skill
`create-feature`. Los pasos de backend:

### Paso 1: Modelo

`features/<dominio>/models/<nombre>.go` (package `<dominio>_models`), con
`gorm.Model` + `TenantID`, como en el patrón de arriba.

### Paso 2: DTOs

`features/<dominio>/dto/<nombre>-dto.go` (package `<dominio>_dto`):
`Create<X>Request` / `Update<X>Request`.

### Paso 3: Handler

`features/<dominio>/handlers/<nombre>-handler.go` (package `<dominio>_handlers`):

```go
func (h *ItemHandler) Create(c *gin.Context) envelope.Response {
    var req item_dto.CreateItemRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        return envelope.ErrorResponse(http.StatusBadRequest, "item.invalid.request", core_errors.ErrInvalidRequest)
    }

    db := tenantdb.For(c, h.db)
    item := item_models.Item{
        TenantID:    tenantdb.TenantID(c),
        Name:        req.Name,
        Description: req.Description,
    }
    if err := db.Create(&item).Error; err != nil {
        h.logger.Error("failed to create item", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "item.create.error", core_errors.ErrInternal)
    }
    return envelope.SuccessResponse(item, "item.create.success")
}

func (h *ItemHandler) GetByID(c *gin.Context) envelope.Response {
    db := tenantdb.For(c, h.db)
    var item item_models.Item
    if err := db.First(&item, c.Param("id")).Error; err != nil {
        return envelope.ErrorResponse(http.StatusNotFound, "item.not_found", core_errors.ErrItemNotFound)
    }
    return envelope.SuccessResponse(item, "item.found")
}
```

`First` sobre un handle de tenant devuelve not found si el registro es de otro
tenant: no hace falta filtrar a mano.

### Paso 4: Error Codes

En `core/errors/codes.go` + su key en ambos JSON de i18n.

### Paso 5: Rutas

`routes/<dominio>-routes.go` con `Register<Dominio>Routes`, siguiendo el
patrón de middleware de arriba.

### Paso 6: Registrar en Index

```go
// routes/index.go → RegisterRoutes
RegisterItemRoutes(router, db)
```

### Paso 7: Migración de esquema

```go
// migrations/migrate.go → RunMigrations, en la lista de modelos
&item_models.Item{},
```

Las migraciones de **datos** (seeds, permisos) son code-migrations: archivo
nuevo en `migrations/code-migrations/2026/` con key `DB<YYYYMMDD>_<n>` nueva.
Una migración ya presente en `origin/main` es inmutable. Reglas en
[`docs/backend/api-code-migration.md`](../backend/api-code-migration.md).

### Paso 8: i18n Keys

`apps/api/i18n/messages/messages_es.json` y `messages_en.json` son un array plano:

```json
[
  { "key": "item.list.success", "value": "Lista de ítems obtenida" },
  { "key": "item.create.success", "value": "Ítem creado exitosamente" },
  { "key": "E-ITEM-001", "value": "Ítem no encontrado." }
]
```

Se siembran en la BD en cada arranque. Agrega cada key en los dos archivos.

### Paso 9: Tests

`features/<dominio>/handlers/<nombre>_test.go`:

```go
func TestCreateItem(t *testing.T) {
    db := testutils.SetupTestDB(t, &item_models.Item{})
    h := NewItemHandler(db, zap.NewNop())

    c, _ := testutils.NewGinContext(1, 1)
    body, _ := json.Marshal(item_dto.CreateItemRequest{Name: "x"})
    c.Request = httptest.NewRequest(http.MethodPost, "/items", bytes.NewReader(body))
    c.Request.Header.Set("Content-Type", "application/json")

    res := h.Create(c)
    if res.Code != http.StatusOK { t.Fatalf("got %d", res.Code) }
}
```

- `SetupTestDB` usa el Postgres del contenedor si está disponible y sqlite en
  memoria en CI; migra solo los modelos pasados.
- Aislamiento entre tenants: patrón `features/clinical/handlers/tenant_isolation_test.go`.
- Correr: `just tests-api` (dentro del contenedor) o `go test ./...` en `apps/api`.

---

## ⚙️ Trabajo en segundo plano

- **Schedulers:** `features/<dominio>/workers/<x>-scheduler.go` con `Start()`,
  arrancado en `cmd/main.go` (`go scheduler.Start()`).
- **Consumers RabbitMQ:** `core/brokers/rabbitmq` (`DeclareQueueWithRetry`,
  `PublishMessage`, `StartConsumer`), lanzados con `rabbitmq.Run(...)` en `cmd/main.go`.
- Datos: `tenantdb.System(db)` para recorrer tenants, `tenantdb.ForTenant(db, id)` para uno.
- Comprobantes SRI: un tipo nuevo es un `Kind` nuevo en
  `features/billing/sri-document/kinds.go` (ver ADR 0001).

---

## 📝 Logging con Zap

```go
h.logger.Error("failed to create item", zap.Error(err))
h.logger.Info("item created", zap.Uint("id", item.ID))
h.logger.Warn("unexpected state", zap.String("field", value))
```

Excepción: las code-migrations imprimen progreso con `fmt.Printf("✅ ...")`.

---

## ✅ Checklist para Feature Nuevo (backend)

- [ ] Modelo con `gorm.Model` + `TenantID`, agregado a `RunMigrations`
- [ ] DTOs (Create plano, Update con punteros)
- [ ] Handler con struct + constructor + métodos `envelope.Response`
- [ ] Todas las consultas vía `tenantdb.For(c, h.db)` (o `ForTenant`/`System` fuera de requests)
- [ ] Error codes en `core/errors/codes.go`, cada uno con key i18n
- [ ] Rutas con middleware + `RequirePermission` + `envelope.Handle()`
- [ ] Rutas registradas en `routes/index.go`
- [ ] i18n keys en ambos JSON
- [ ] Tests `*_test.go` del handler
- [ ] `go build ./... && go vet ./... && go test ./...` en `apps/api`
