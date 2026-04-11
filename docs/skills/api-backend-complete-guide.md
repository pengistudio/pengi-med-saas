---
name: API Backend — Complete Guide
description: Guía completa del backend Go/Gin/GORM incluyendo arquitectura, patrones, y cómo implementar nuevas features
---

# API Backend — Complete Guide

Guía completa de la arquitectura y desarrollo del backend `apps/api` (Go + Gin + GORM + PostgreSQL).

## 📐 Arquitectura General

### Stack Técnico

- **Lenguaje:** Go (`pengi-med-saas`)
- **Framework HTTP:** Gin
- **ORM:** GORM con PostgreSQL
- **Logging:** Zap (structured logging)
- **Messaging:** RabbitMQ (async tasks)
- **Database:** PostgreSQL 13+

### Estructura de Directorios

```
apps/api/
├── cmd/                          # Entrypoint
├── configuration/                # Env variables
├── core/
│   ├── audit/                   # Auditoría de cambios
│   ├── auth/                    # JWT, password hashing
│   ├── brokers/                 # RabbitMQ/messaging
│   ├── database/                # Setup GORM, migrations map
│   ├── envelope/                # Response wrapper (Handle, Success, Error)
│   ├── errors/                  # AppError type + error codes
│   ├── logger/                  # Zap singleton
│   ├── middleware/              # Global middleware (rate limiter)
│   └── utils/                   # General utilities
├── features/
│   ├── [dominio]/
│   │   ├── handlers/            # Business logic (CRUD operations)
│   │   ├── models/              # GORM database models
│   │   ├── dto/                 # Data Transfer Objects
│   │   ├── workers/             # Async workers (optional)
│   │   └── middleware/          # Domain-specific middleware
│   ├── backoffice/
│   ├── billing/
│   ├── clinical/
│   ├── companies/               # Plans, subscriptions
│   ├── health/
│   ├── kanban/
│   ├── permissions/
│   ├── tenants/
│   └── users/
├── i18n/
│   └── messages/
│       ├── messages_es.json
│       └── messages_en.json
├── migrations/
│   ├── migrate.go               # AutoMigrate + RunAllMigrations
│   └── code-migrations/
│       └── [año]/               # Año-basado migrations
└── routes/
    ├── index.go                 # RegisterRoutes() master
    └── [dominio]_routes.go      # Domain-specific routes
```

### Dominios Existentes

| Dominio | Propósito | Permisos |
|---------|-----------|----------|
| `backoffice` | Panel de administración | `PERM_BO_*` |
| `billing` | Facturación, invoices | `PERM_BILL_*` |
| `clinical` | Pacientes, citas, records | `PERM_CLIN_*` |
| `companies` | Empresas, planes, suscripciones | `PERM_COMP_*` |
| `health` | Health check / status | — |
| `kanban` | Task management | `PERM_KANB_*` |
| `permissions` | RBAC granular | — |
| `tenants` | Multi-tenancy | — |
| `users` | Auth, JWT, environments | `PERM_USR_*` |

---

## 🏢 Multi-Tenancy Model

El sistema es **multi-tenant por defecto**:

```
User
  ├─ Environment[0] → {CompanyID, RoleID, Name}
  ├─ Environment[1] → {CompanyID, RoleID, Name}
  └─ Environment[2] → {CompanyID, RoleID, Name}

Company
  ├─ Tenant[0] → {subscription, SriP12Path, ...}
  └─ Tenant[1]

Todos los modelos de dominio:
  └─ TenantID uint (filtro obligatorio en queries)
```

**Clave:** Cada request inyecta `tenant_id` vía `TenantMiddleware`. **TODAS** las queries deben usar `tenant_middleware.TenantScope(c)`.

---

## 🔄 Flujo de Request

```
HTTP Request
    ↓
RateLimiter (opcional)
    ↓
AuthMiddleware (JWT → user_id)
    ↓
TenantMiddleware (tenant_id from Environment)
    ↓
SubscriptionMiddleware (validate subscription)
    ↓
RequirePermission (optional, granular RBAC)
    ↓
envelope.Handle(handler.Method)
    ├─ handler.Method(c *gin.Context) envelope.Response
    ├─ Procesa lógica
    └─ Retorna envelope.Response
    ↓
envelope.Handle() traduce i18n
    ↓
HTTP Response (JSON)
```

---

## 📦 Patrones Core

### Handler Pattern

```go
type [Nombre]Handler struct {
    db     *gorm.DB
    logger *zap.Logger
}

func New[Nombre]Handler(db *gorm.DB, logger *zap.Logger) *[Nombre]Handler {
    return &[Nombre]Handler{db: db, logger: logger}
}

// Método siempre retorna envelope.Response, NO gin.HandlerFunc
func (h *[Nombre]Handler) GetAll(c *gin.Context) envelope.Response {
    var items []models.[Nombre]
    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Find(&items).Error; err != nil {
        h.logger.Error("Failed to fetch", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrXxx)
    }
    return envelope.SuccessResponse(items, "i18n.key")
}
```

**Reglas:**
- Struct con `db` + `logger`
- Constructor `New[X]Handler`
- Métodos retornan `envelope.Response`
- **NUNCA** escribir a `c` directamente
- Log con `zap`, nunca `fmt.Println`

### DTO Pattern (Validación)

```go
// Create: campos required, opcionales sin puntero
type CreateItemDTO struct {
    Name     string `json:"name" binding:"required,min=1"`
    Email    string `json:"email" binding:"required,email"`
    Status   string `json:"status" binding:"required,oneof=active inactive"`
}

// Update: **TODOS** campos como punteros (partial updates)
type UpdateItemDTO struct {
    Name     *string `json:"name"`
    Email    *string `json:"email" binding:"omitempty,email"`
    Status   *string `json:"status" binding:"omitempty,oneof=active inactive"`
}
```

**Binding tags:** `required`, `min`, `max`, `email`, `oneof`, `omitempty`, etc.

### Modelo Pattern (GORM)

```go
type Item struct {
    gorm.Model
    TenantID    uint   `json:"tenant_id"` // OBLIGATORIO
    Name        string `json:"name"`
    Description string `json:"description"`
    CompanyID   uint   `json:"company_id"`
    Company     *Company `gorm:"foreignKey:CompanyID; constraint:OnDelete:CASCADE"`
}

func (Item) TableName() string { return "items" }
func (Item) IsAuditable() bool { return true }
```

### Response Envelope Pattern

```go
// Success simple
envelope.SuccessResponse(data, "i18n.key")

// Success paginado
envelope.PagedSuccessResponse(items, total, page, limit, "i18n.key")

// Error
envelope.ErrorResponse(http.StatusBadRequest, msg, core_errors.ErrXxx)
```

### Error Codes Pattern

```go
// core/errors/codes.go
var (
    Err[Domain]InvalidRequest AppError = NewAppError("E-[DOM]-001", "Invalid request")
    Err[Domain]NotFound       AppError = NewAppError("E-[Domain]-002", "Not found")
    Err[Domain]CreateError    AppError = NewAppError("E-[DOM]-003", "Error creating")
    Err[Domain]UpdateError    AppError = NewAppError("E-[DOM]-004", "Error updating")
    Err[Domain]DeleteError    AppError = NewAppError("E-[DOM]-005", "Error deleting")
)
```

**Formato código:** `E-[PREFIJO]-[NNN]`

Prefijos existentes:
- `INT` = Internal
- `AUTH` = Authentication
- `USR` = User
- `TEN` = Tenant
- `CLIN` = Clinical
- `BILL` = Billing
- `KANB` = Kanban
- `PERM` = Permissions

### Middleware Pattern

```go
// Middleware global (en routes/index.go)
group := router.Group(
    "/api/v1",
    auth_middleware.AuthMiddleware(),
    tenant_middleware.TenantMiddleware(db),
    subscription_middleware.SubscriptionMiddleware(db),
)

// Middleware específico por ruta
group.POST("/items", 
    subscription_middleware.RequirePermission(db, "PERM_ITEM_CREATE"),
    envelope.Handle(handler.Create),
)
```

---

## 🛠️ Cómo Implementar un Feature Nuevo

### Paso 1: Crear Modelo

**Archivo:** `features/[dominio]/models/[nombre].go`

```go
package [dominio]_models

import "gorm.io/gorm"

type Item struct {
    gorm.Model
    TenantID    uint   `json:"tenant_id"`
    Name        string `json:"name"`
    Description string `json:"description"`
}

func (Item) IsAuditable() bool { return true }
```

### Paso 2: Crear DTOs

**Archivo:** `features/[dominio]/dto/[nombre]-dto.go`

```go
package dto

type CreateItemDTO struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
}

type UpdateItemDTO struct {
    Name        *string `json:"name"`
    Description *string `json:"description"`
}
```

### Paso 3: Crear Handler

**Archivo:** `features/[dominio]/handlers/[nombre]-handler.go`

```go
package [dominio]_handlers

import (
    "net/http"
    "pengi-med-saas/core/envelope"
    core_errors "pengi-med-saas/core/errors"
    [dominio]_dto "pengi-med-saas/features/[dominio]/dto"
    [dominio]_models "pengi-med-saas/features/[dominio]/models"
    tenant_middleware "pengi-med-saas/features/tenants/middleware"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

type ItemHandler struct {
    db     *gorm.DB
    logger *zap.Logger
}

func NewItemHandler(db *gorm.DB, logger *zap.Logger) *ItemHandler {
    return &ItemHandler{db: db, logger: logger}
}

func (h *ItemHandler) GetAll(c *gin.Context) envelope.Response {
    var items [][dominio]_models.Item
    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Find(&items).Error; err != nil {
        h.logger.Error("Failed to fetch items", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrItemInvalidRequest)
    }
    return envelope.SuccessResponse(items, "item.list.success")
}

func (h *ItemHandler) Create(c *gin.Context) envelope.Response {
    var dto [dominio]_dto.CreateItemDTO
    if err := c.ShouldBindJSON(&dto); err != nil {
        return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrItemInvalidRequest)
    }

    tenantID, _ := c.Get("tenant_id")
    item := &[dominio]_models.Item{
        TenantID:    tenantID.(uint),
        Name:        dto.Name,
        Description: dto.Description,
    }

    if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Create(item).Error; err != nil {
        h.logger.Error("Failed to create item", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrItemCreateError)
    }

    return envelope.SuccessResponse(item, "item.create.success")
}
```

### Paso 4: Agregar Error Codes

**Archivo:** `core/errors/codes.go`

```go
var (
    ErrItemInvalidRequest AppError = NewAppError("E-ITEM-001", "Invalid item request")
    ErrItemNotFound       AppError = NewAppError("E-ITEM-002", "Item not found")
    ErrItemCreateError    AppError = NewAppError("E-ITEM-003", "Error creating item")
)
```

### Paso 5: Crear Rutas

**Archivo:** `routes/item_routes.go`

```go
package routes

import (
    "pengi-med-saas/core/envelope"
    "pengi-med-saas/core/logger"
    item_handlers "pengi-med-saas/features/item/handlers"
    tenant_middleware "pengi-med-saas/features/tenants/middleware"
    auth_middleware "pengi-med-saas/features/users/middleware"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func RegisterItemRoutes(router *gin.RouterGroup, db *gorm.DB) {
    handler := item_handlers.NewItemHandler(db, logger.Log)
    
    group := router.Group(
        "/items",
        auth_middleware.AuthMiddleware(),
        tenant_middleware.TenantMiddleware(db),
    )
    {
        group.GET("", envelope.Handle(handler.GetAll))
        group.GET("/:id", envelope.Handle(handler.GetByID))
        group.POST("", envelope.Handle(handler.Create))
        group.PUT("/:id", envelope.Handle(handler.Update))
        group.DELETE("/:id", envelope.Handle(handler.Delete))
    }
}
```

### Paso 6: Registrar en Index

**Archivo:** `routes/index.go`

```go
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    // ... otras rutas
    RegisterItemRoutes(router, db)  // ← Agregar
}
```

### Paso 7: Agregar Migración

**Archivo:** `migrations/migrate.go`

```go
import item_models "pengi-med-saas/features/item/models"

// En RunMigrations():
err := db.AutoMigrate(
    // ... otros
    &item_models.Item{},
)
```

### Paso 8: Agregar i18n Keys

**Archivo:** `apps/api/i18n/messages/messages_es.json`

```json
{
  "item": {
    "list": { "success": "Lista de ítems obtenida" },
    "found": "Ítem encontrado",
    "create": { "success": "Ítem creado exitosamente" },
    "update": { "success": "Ítem actualizado" },
    "delete": { "success": "Ítem eliminado" }
  }
}
```

Y en `messages_en.json` con equivalentes en inglés.

---

## 📝 Logging con Zap

```go
// Error con contexto
h.logger.Error("Failed to create item", zap.Error(err))

// Info con campos
h.logger.Info("Item created successfully", zap.Uint("id", item.ID))

// Warning
h.logger.Warn("Unexpected behavior", zap.String("field", value))

// Campos comunes
zap.Error(err)
zap.Uint("id", id)
zap.String("name", name)
zap.Bool("active", active)
```

**Regla:** Nunca `fmt.Println`, `log.Fatal`, `panic`. Usar zap para TODO.

---

## 🔐 Multi-Tenancy Rules

1. **Modelo:** Siempre incluir `TenantID uint`
2. **Query:** **SIEMPRE** usar `tenant_middleware.TenantScope(c)`
3. **Create:** Inyectar `tenantID` desde context: `tenantID, _ := c.Get("tenant_id")`
4. **Audit:** Usar `AuditScope(c)` en Create/Update si `IsAuditable()`

```go
// ❌ MALO
db.Find(&items)  // Sin tenant scope = datos de otros tenants

// ✅ BUENO
db.Scopes(tenant_middleware.TenantScope(c)).Find(&items)  // Filtrado por tenant
```

---

## ✅ Checklist para Feature Nuevo

- [ ] Modelo creado con `gorm.Model` + `TenantID`
- [ ] DTOs creados (Create normal, Update con punteros)
- [ ] Handler con struct + constructor + métodos `envelope.Response`
- [ ] Error codes agregados en `core/errors/codes.go`
- [ ] Rutas creadas con `envelope.Handle()`
- [ ] Rutas registradas en `routes/index.go`
- [ ] Modelo en `migrations/migrate.go`
- [ ] i18n keys en ambos JSON
- [ ] Todos los logs con `zap`
- [ ] Todas las queries con `TenantScope(c)`
- [ ] `go vet ./...` sin errores
- [ ] Siguiendo imports pattern con aliases

