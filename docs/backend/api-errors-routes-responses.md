# Skill: Agregar Errores, Rutas y Respuestas en el Backend

## 1. Agregar Error Code (`core/errors/codes.go`)

Los errores de aplicación son constantes tipadas `AppError` definidas en `apps/api/core/errors/codes.go`.

### Patrón

```go
// [Dominio] Errors
ErrXxxInvalidRequest AppError = NewAppError("E-[PRE]-001", "Invalid [dominio] request.")
ErrXxxNotFound       AppError = NewAppError("E-[PRE]-002", "[Recurso] not found.")
ErrXxxCreateError    AppError = NewAppError("E-[PRE]-003", "Error creating [recurso].")
ErrXxxUpdateError    AppError = NewAppError("E-[PRE]-004", "Error updating [recurso].")
ErrXxxDeleteError    AppError = NewAppError("E-[PRE]-005", "Error deleting [recurso].")
```

### Prefijos existentes (NO reutilizar)

| Prefijo | Dominio |
|---------|---------|
| `INT` | Errores internos generales |
| `MES` | Mensajes i18n |
| `COMP` | Companies |
| `TEN` | Tenants |
| `USR` | Users |
| `AUTH` | Auth |
| `CLIN` | Clinical |
| `PERM` | Permissions |
| `BO` | Backoffice |
| `PLAN` | Plan limits |
| `BILL` | Billing |

### Uso en handlers

```go
// Error de request inválido
return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrXxxInvalidRequest)

// Not found
return envelope.ErrorResponse(http.StatusNotFound, err.Error(), core_errors.ErrXxxNotFound)

// Error de servidor
return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrXxxCreateError)

// Forbidden (plan limit, permisos)
return envelope.ErrorResponse(http.StatusForbidden, "reason", core_errors.ErrXxxForbidden)
```

---

## 2. Agregar Ruta

### Estructura de una función de rutas

```go
package routes

import (
    "pengi-med-saas/core/envelope"
    "pengi-med-saas/core/logger"
    [dom]_handlers "pengi-med-saas/features/[dominio]/handlers"
    tenant_middleware "pengi-med-saas/features/tenants/middleware"
    auth_middleware "pengi-med-saas/features/users/middleware"
    // subscription_middleware "pengi-med-saas/features/companies/middleware"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func Register[Dominio]Routes(router *gin.RouterGroup, db *gorm.DB) {
    handler := [dom]_handlers.New[Nombre]Handler(db, logger.Log)

    // Grupo con middlewares comunes
    group := router.Group("/[ruta-base]",
        auth_middleware.AuthMiddleware(),
        tenant_middleware.TenantMiddleware(db),
        // subscription_middleware.SubscriptionMiddleware(db), // si aplica
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

### Agregar en `routes/index.go`

```go
func RegisterRoutes(router *gin.RouterGroup, db *gorm.DB) {
    RegisterI18nRoutes(router, db)
    RegisterCompanyRoutes(router, db)
    RegisterUserRoutes(router, db)
    RegisterClinicalRoutes(router, db)
    RegisterPermissionRoutes(router, db)
    RegisterBackofficeRoutes(router, db)
    RegisterBillingRoutes(router, db)
    RegisterTenantRoutes(router, db)
    RegisterIntegrationRoutes(router, db)
    Register[Dominio]Routes(router, db) // ← Agregar aquí
}
```

### Rutas con permiso granular

```go
rp := subscription_middleware.RequirePermission

group.POST("", rp(db, "CREATE_[RECURSO]"), envelope.Handle(handler.Create))
group.GET("", rp(db, "READ_[RECURSO]"), envelope.Handle(handler.GetAll))
group.PUT("/:id", rp(db, "UPDATE_[RECURSO]"), envelope.Handle(handler.Update))
group.DELETE("/:id", rp(db, "DELETE_[RECURSO]"), envelope.Handle(handler.Delete))
```

---

## 3. Respuestas Envelope

### SuccessResponse — respuesta simple exitosa

```go
return envelope.SuccessResponse(data, "i18n.key.success")
```

### PagedSuccessResponse — lista paginada

```go
return envelope.PagedSuccessResponse(items, int(total), page, limit, "i18n.key.list.success")
// Retorna: { items, total, page, limit, total_pages }
```

### ErrorResponse — respuesta de error

```go
return envelope.ErrorResponse(httpStatusCode, errorMessage, core_errors.ErrXxx)
// El errorMessage puede ser err.Error() o una clave i18n
```

---

## 4. Agregar mensajes i18n

Cada clave usada en `SuccessResponse` o `ErrorResponse` DEBE existir en ambos archivos:

**`apps/api/i18n/messages/messages_es.json`**
```json
{
  "[dominio].[recurso].create.success": "Recurso creado exitosamente.",
  "[dominio].[recurso].update.success": "Recurso actualizado exitosamente.",
  "[dominio].[recurso].delete.success": "Recurso eliminado exitosamente.",
  "[dominio].[recurso].list.success": "Recursos obtenidos exitosamente.",
  "[dominio].[recurso].found": "Recurso encontrado.",
  "[dominio].[recurso].not_found": "Recurso no encontrado."
}
```

**`apps/api/i18n/messages/messages_en.json`**
```json
{
  "[dominio].[recurso].create.success": "Resource created successfully.",
  "[dominio].[recurso].update.success": "Resource updated successfully.",
  "[dominio].[recurso].delete.success": "Resource deleted successfully.",
  "[dominio].[recurso].list.success": "Resources retrieved successfully.",
  "[dominio].[recurso].found": "Resource found.",
  "[dominio].[recurso].not_found": "Resource not found."
}
```

---

## 5. Middleware de Tenant — TenantScope vs AuditScope

| Scope | Uso | Efecto |
|-------|-----|--------|
| `tenant_middleware.TenantScope(c)` | Queries de lectura y update/delete | Filtra por `tenant_id` automáticamente |
| `tenant_middleware.AuditScope(c)` | Create (y updates que deben auditarse) | Filtra + registra auditoría del cambio |

```go
// Lectura
h.db.Scopes(tenant_middleware.TenantScope(c)).Find(&items)

// Creación con auditoría
h.db.Scopes(tenant_middleware.AuditScope(c)).Create(&item)

// Actualización normal con scope de tenant
h.db.Scopes(tenant_middleware.TenantScope(c)).Model(&item).Updates(updates)
```

**Importante:** Si el modelo NO implementa `IsAuditable() bool { return true }`, `AuditScope` se comporta igual que `TenantScope`.
