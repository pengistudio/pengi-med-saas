# Skill: Crear un Nuevo Feature en el Backend (Go / Gin / GORM)

Este skill describe paso a paso cómo crear un feature completo en el backend del monorepo `apps/api`.

## Contexto del Proyecto

- **Módulo Go:** `pengi-med-saas`
- **Framework:** Gin + GORM + PostgreSQL
- **Estructura de features:** `features/[dominio]/` → `handlers/`, `models/`, `dto/`
- **Rutas:** `routes/[dominio]_routes.go`, registradas en `routes/index.go`

---

## Checklist de Creación de Feature

### 1. Modelo (`features/[dominio]/models/[nombre].go`)

```go
package [dominio]_models

import "gorm.io/gorm"

type [Nombre] struct {
    gorm.Model
    TenantID  uint   `json:"tenant_id"` // OBLIGATORIO si es tenant-aware
    // ... campos del modelo con tags JSON en snake_case
}

// Si el modelo debe ser auditado:
func ([Nombre]) IsAuditable() bool { return true }
```

**Reglas:**
- Siempre heredar de `gorm.Model`
- Si es multi-tenant, incluir `TenantID uint`
- Tags JSON en `snake_case`
- Relaciones con `gorm:"foreignKey:...; constraint:OnDelete:..."`

---

### 2. DTOs (`features/[dominio]/dto/[nombre]-dto.go`)

```go
package dto

type Create[Nombre]DTO struct {
    Campo1  string  `json:"campo_1" binding:"required"`
    Campo2  string  `json:"campo_2"`                    // opcional
    Campo3  *string `json:"campo_3"`                    // opcional como puntero
}

type Update[Nombre]DTO struct {
    Campo1  *string `json:"campo_1"` // TODO como puntero para actualizaciones parciales
    Campo2  *string `json:"campo_2"`
}
```

**Reglas:**
- `Create[X]DTO`: campos required con `binding:"required"`, opcionales como punteros
- `Update[X]DTO`: **TODOS** los campos como punteros (`*tipo`)

---

### 3. Handler (`features/[dominio]/handlers/[nombre]-handler.go`)

```go
package [dominio]_handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "gorm.io/gorm"

    "pengi-med-saas/core/envelope"
    core_errors "pengi-med-saas/core/errors"
    [dominio]_dto "pengi-med-saas/features/[dominio]/dto"
    [dominio]_models "pengi-med-saas/features/[dominio]/models"
    tenant_middleware "pengi-med-saas/features/tenants/middleware"
)

type [Nombre]Handler struct {
    db     *gorm.DB
    logger *zap.Logger
}

func New[Nombre]Handler(db *gorm.DB, logger *zap.Logger) *[Nombre]Handler {
    return &[Nombre]Handler{db: db, logger: logger}
}

// GET /[dominio]/[recursos]
func (h *[Nombre]Handler) GetAll(c *gin.Context) envelope.Response {
    var items [][dominio]_models.[Nombre]
    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Find(&items).Error; err != nil {
        h.logger.Error("Failed to fetch [recursos]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.Err[Dominio]NotFound)
    }
    return envelope.SuccessResponse(items, "[dominio].[recurso].list.success")
}

// GET /[dominio]/[recursos]/:id
func (h *[Nombre]Handler) GetByID(c *gin.Context) envelope.Response {
    idParam := c.Param("id")
    id, err := strconv.ParseUint(idParam, 10, 32)
    if err != nil {
        h.logger.Error("Invalid [recurso] ID", zap.Error(err))
        return envelope.ErrorResponse(http.StatusBadRequest, "Invalid ID format", core_errors.Err[Dominio]InvalidRequest)
    }

    var item [dominio]_models.[Nombre]
    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&item, id).Error; err != nil {
        h.logger.Error("Failed to find [recurso]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.Err[Dominio]NotFound)
    }

    return envelope.SuccessResponse(item, "[dominio].[recurso].found")
}

// POST /[dominio]/[recursos]
func (h *[Nombre]Handler) Create(c *gin.Context) envelope.Response {
    var dto [dominio]_dto.Create[Nombre]DTO
    if err := c.ShouldBind(&dto); err != nil {
        h.logger.Error("Invalid create [recurso] request", zap.Error(err))
        return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.Err[Dominio]InvalidRequest)
    }

    tenantID, _ := c.Get("tenant_id")
    item := &[dominio]_models.[Nombre]{
        // Mapear campos desde DTO
    }
    item.TenantID = tenantID.(uint)

    if err := h.db.Scopes(tenant_middleware.AuditScope(c)).Create(item).Error; err != nil {
        h.logger.Error("Failed to create [recurso]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.Err[Dominio]CreateError)
    }

    h.logger.Info("[Nombre] created successfully", zap.Uint("id", item.ID))
    return envelope.SuccessResponse(item, "[dominio].[recurso].create.success")
}

// PUT /[dominio]/[recursos]/:id
func (h *[Nombre]Handler) Update(c *gin.Context) envelope.Response {
    idParam := c.Param("id")
    id, err := strconv.ParseUint(idParam, 10, 32)
    if err != nil {
        h.logger.Error("Invalid [recurso] ID", zap.Error(err))
        return envelope.ErrorResponse(http.StatusBadRequest, "Invalid ID format", core_errors.Err[Dominio]InvalidRequest)
    }

    var item [dominio]_models.[Nombre]
    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).First(&item, id).Error; err != nil {
        h.logger.Error("Failed to find [recurso]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.Err[Dominio]NotFound)
    }

    var updateData [dominio]_dto.Update[Nombre]DTO
    if err := c.ShouldBind(&updateData); err != nil {
        h.logger.Error("Invalid update [recurso] request", zap.Error(err))
        return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.Err[Dominio]InvalidRequest)
    }

    updates := make(map[string]interface{})
    // if updateData.Campo != nil { updates["campo"] = *updateData.Campo }

    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&item).Updates(updates).Error; err != nil {
        h.logger.Error("Failed to update [recurso]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.Err[Dominio]UpdateError)
    }

    // Reload
    h.db.Scopes(tenant_middleware.TenantScope(c)).First(&item, id)

    h.logger.Info("[Nombre] updated successfully", zap.Uint("id", item.ID))
    return envelope.SuccessResponse(item, "[dominio].[recurso].update.success")
}

// DELETE /[dominio]/[recursos]/:id
func (h *[Nombre]Handler) Delete(c *gin.Context) envelope.Response {
    idParam := c.Param("id")
    id, err := strconv.ParseUint(idParam, 10, 32)
    if err != nil {
        h.logger.Error("Invalid [recurso] ID", zap.Error(err))
        return envelope.ErrorResponse(http.StatusBadRequest, "Invalid ID format", core_errors.Err[Dominio]InvalidRequest)
    }

    if err := h.db.Scopes(tenant_middleware.TenantScope(c)).Where("id = ?", id).Delete(&[dominio]_models.[Nombre]{}).Error; err != nil {
        h.logger.Error("Failed to delete [recurso]", zap.Uint64("id", id), zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.Err[Dominio]DeleteError)
    }

    return envelope.SuccessResponse(nil, "[dominio].[recurso].delete.success")
}
```

**Reglas:**
- El handler es un struct con `db *gorm.DB` y `logger *zap.Logger`
- Constructor `New[X]Handler(db, logger)`
- Los métodos retornan `envelope.Response`, NO son `gin.HandlerFunc`
- Usar `tenant_middleware.TenantScope(c)` en TODAS las queries
- Usar `tenant_middleware.AuditScope(c)` en Create/Update si el modelo es Auditable
- Logs con `zap`, NO `fmt.Println`
- Los errores se retornan con `envelope.ErrorResponse(httpCode, err.Error(), core_errors.ErrXxx)`

---

### 4. Errores (`core/errors/codes.go`)

Agregar las constantes de error para el nuevo dominio:

```go
// [Dominio] Errors
Err[Dominio]InvalidRequest AppError = NewAppError("E-[DOM]-001", "Invalid [dominio] request.")
Err[Dominio]NotFound       AppError = NewAppError("E-[DOM]-002", "[Nombre] not found.")
Err[Dominio]CreateError    AppError = NewAppError("E-[DOM]-003", "Error creating [nombre].")
Err[Dominio]UpdateError    AppError = NewAppError("E-[DOM]-004", "Error updating [nombre].")
Err[Dominio]DeleteError    AppError = NewAppError("E-[DOM]-005", "Error deleting [nombre].")
```

**Convención de códigos:** `E-[PREFIJO]-[NNN]` donde PREFIJO es 2-5 letras mayúsculas del dominio.
Prefijos existentes: `INT`, `MES`, `COMP`, `TEN`, `USR`, `AUTH`, `CLIN`, `PERM`, `BO`, `PLAN`, `BILL`.

---

### 5. Rutas (`routes/[dominio]_routes.go`)

```go
package routes

import (
    "pengi-med-saas/core/envelope"
    "pengi-med-saas/core/logger"
    [dominio]_handlers "pengi-med-saas/features/[dominio]/handlers"
    tenant_middleware "pengi-med-saas/features/tenants/middleware"
    auth_middleware "pengi-med-saas/features/users/middleware"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func Register[Dominio]Routes(router *gin.RouterGroup, db *gorm.DB) {
    [nombre]Handler := [dominio]_handlers.New[Nombre]Handler(db, logger.Log)

    group := router.Group("/[ruta]", auth_middleware.AuthMiddleware(), tenant_middleware.TenantMiddleware(db))
    {
        group.GET("", envelope.Handle([nombre]Handler.GetAll))
        group.GET("/:id", envelope.Handle([nombre]Handler.GetByID))
        group.POST("", envelope.Handle([nombre]Handler.Create))
        group.PUT("/:id", envelope.Handle([nombre]Handler.Update))
        group.DELETE("/:id", envelope.Handle([nombre]Handler.Delete))
    }
}
```

**Reglas:**
- Usar `envelope.Handle(handler.Metodo)` para TODAS las rutas
- Los middlewares van en el `Group(...)`, no en cada ruta individual
- Si se requiere permiso granular: `subscription_middleware.RequirePermission(db, "PERMISO_CODE")`

---

### 6. Registrar la ruta (`routes/index.go`)

```go
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    // ... rutas existentes
    Register[Dominio]Routes(router, db) // Agregar aquí
}
```

---

### 7. Migración del modelo (`migrations/migrate.go`)

Agregar el modelo nuevo en la lista de `RunMigrations`:

```go
import [dominio]_models "pengi-med-saas/features/[dominio]/models"

// Dentro de RunMigrations, en database.MigrateDB():
[dominio]_models.[Nombre]{},
```

---

### 8. Paginación (cuando aplique)

Patrón estándar para endpoints de lista paginada:

```go
func (h *[Nombre]Handler) GetAllPaged(c *gin.Context) envelope.Response {
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
    if page < 1 { page = 1 }
    if limit < 1 || limit > 100 { limit = 20 }
    offset := (page - 1) * limit

    baseQuery := h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&[dominio]_models.[Nombre]{})

    var total int64
    if err := baseQuery.Count(&total).Error; err != nil {
        h.logger.Error("Failed to count [recursos]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.Err[Dominio]NotFound)
    }

    var items [][dominio]_models.[Nombre]
    if err := baseQuery.Order("created_at desc").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
        h.logger.Error("Failed to fetch [recursos]", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.Err[Dominio]NotFound)
    }

    return envelope.PagedSuccessResponse(items, int(total), page, limit, "[dominio].[recurso].list.success")
}
```

---

### 9. i18n — Mensajes

**SIEMPRE** que crees claves de mensaje (las que van en `envelope.SuccessResponse` o `ErrorResponse`), agregarlas en ambos archivos:
- `apps/api/i18n/messages/messages_es.json`
- `apps/api/i18n/messages/messages_en.json`

Ejemplo de clave: `"[dominio].[recurso].create.success": "Recurso creado exitosamente."`

---

## Middlewares disponibles

| Middleware | Import path | Uso |
|---|---|---|
| `auth_middleware.AuthMiddleware()` | `pengi-med-saas/features/users/middleware` | Valida JWT, inyecta `user_id` |
| `tenant_middleware.TenantMiddleware(db)` | `pengi-med-saas/features/tenants/middleware` | Inyecta `tenant_id` |
| `tenant_middleware.TenantScope(c)` | mismo | Scope GORM para filtrar por tenant |
| `tenant_middleware.AuditScope(c)` | mismo | Scope GORM para crear con auditoría |
| `subscription_middleware.SubscriptionMiddleware(db)` | `pengi-med-saas/features/companies/middleware` | Valida suscripción activa |
| `subscription_middleware.RequirePermission(db, "PERM")` | mismo | Aplica permiso granular por código |

## Respuestas disponibles (`core/envelope`)

| Función | Cuándo usar |
|---|---|
| `envelope.SuccessResponse(data, "i18n.key")` | Respuesta exitosa simple |
| `envelope.PagedSuccessResponse(items, total, page, limit, "i18n.key")` | Lista paginada |
| `envelope.ErrorResponse(httpCode, msg, core_errors.ErrXxx)` | Error con código de error |
