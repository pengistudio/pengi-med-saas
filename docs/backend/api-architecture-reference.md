# Skill: Arquitectura y Patrones del Backend API

Referencia rápida de la arquitectura del backend `apps/api` para contextualizar cualquier trabajo.

## Stack Técnico

- **Lenguaje:** Go
- **Framework HTTP:** Gin (`github.com/gin-gonic/gin`)
- **ORM:** GORM (`gorm.io/gorm`) con PostgreSQL (`gorm.io/driver/postgres`)
- **Logging:** Zap (`go.uber.org/zap`)
- **Módulo Go:** `pengi-med-saas`

## Estructura de Directorios

```
apps/api/
├── cmd/                    # Entrypoint de la aplicación
├── configuration/          # Config de entorno (env vars)
├── core/
│   ├── audit/             # Lógica de auditoría
│   ├── auth/              # JWT, hashing de passwords
│   ├── brokers/           # RabbitMQ/messaging
│   ├── database/          # Setup GORM, migrations map
│   ├── envelope/          # Response wrapper (Response, Handle, SuccessResponse, ErrorResponse)
│   ├── errors/            # AppError type + códigos de error
│   ├── logger/            # Singleton del logger zap
│   ├── middleware/         # Rate limiter
│   └── utils/             # Utilidades generales
├── features/
│   ├── [dominio]/
│   │   ├── handlers/      # Controladores (structs con métodos)
│   │   ├── models/        # Modelos GORM
│   │   ├── dto/           # Data Transfer Objects
│   │   └── middleware/    # Middlewares específicos del dominio (opt.)
│   ├── backoffice/
│   ├── billing/
│   ├── clinical/
│   ├── companies/         # Plans, subscriptions, features
│   ├── health/
│   ├── integrations/
│   ├── permissions/
│   ├── tenants/
│   └── users/
├── i18n/
│   ├── messages/
│   │   ├── messages_es.json
│   │   └── messages_en.json
│   ├── handlers/
│   └── models/
├── migrations/
│   ├── migrate.go         # AutoMigrate + RunAllMigrations
│   └── code-migrations/
│       └── [año]/         # Package y[año], migrations con ID único
└── routes/
    ├── index.go           # RegisterRoutes() — registra todos los dominios
    └── [dominio]_routes.go
```

## Dominios Existentes

| Dominio | Path | Descripción |
|---------|------|-------------|
| `backoffice` | `features/backoffice/` | Panel de administración interna |
| `billing` | `features/billing/` | Facturación, facturas, catálogo |
| `clinical` | `features/clinical/` | Pacientes, registros médicos, citas |
| `companies` | `features/companies/` | Empresas, planes, suscripciones |
| `health` | `features/health/` | Health check |
| `integrations` | `features/integrations/` | Integraciones externas |
| `permissions` | `features/permissions/` | Permisos granulares por rol |
| `tenants` | `features/tenants/` | Multi-tenancy |
| `users` | `features/users/` | Usuarios, auth, environments |

## Multi-tenancy

El sistema es multi-tenant. Cada tenant puede tener múltiples empresas (`Company`), y cada empresa tiene su propia suscripción y usuarios mediante `Environment`.

- Un `User` puede pertenecer a múltiples empresas a través de `Environment`
- `Environment` = {UserID, CompanyID, RoleID, Name}
- Todos los modelos de dominio tienen `TenantID uint` para el scope de datos

## Flujo de una Request

```
HTTP Request
    → Rate Limiter (opcional)
    → AuthMiddleware (valida JWT, inyecta user_id)
    → TenantMiddleware (inyecta tenant_id desde el env del usuario)
    → SubscriptionMiddleware (valida suscripción activa)
    → RequirePermission (valida permiso granular, opcional)
    → envelope.Handle(handler.Metodo)
        → handler.Metodo(c *gin.Context) envelope.Response
        → envelope.Handle traduce i18n y emite c.JSON(code, response)
```

## Convenciones de Packages

| Tipo | Nombre del package |
|------|-------------------|
| Modelo | `[dominio]_models` |
| DTO | `dto` (sin prefijo dentro del package) |
| Handler | `[dominio]_handlers` |
| Middleware | `[dominio]_middleware` o `[sub]_middleware` |
| Rutas | `routes` |

## Convenciones de Imports

```go
import (
    // stdlib
    "net/http"
    "strconv"

    // dependencias externas
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "gorm.io/gorm"

    // internal core
    "pengi-med-saas/core/envelope"
    core_errors "pengi-med-saas/core/errors"

    // internal features — siempre con alias
    clinical_dto "pengi-med-saas/features/clinical/dto"
    clinical_models "pengi-med-saas/features/clinical/models"
    tenant_middleware "pengi-med-saas/features/tenants/middleware"
)
```

**Regla:** Siempre usar alias para imports de features para evitar colisiones de nombres.

## Logs — Uso de Zap

```go
// Error con contexto
h.logger.Error("Failed to create [recurso]", zap.Error(err))

// Info con campos
h.logger.Info("[Recurso] created", zap.Uint("id", item.ID))

// Warning
h.logger.Warn("Unexpected state", zap.String("field", value))

// Campos comunes
zap.Error(err)
zap.Uint("id", id)
zap.Uint64("id", id)
zap.String("name", name)
zap.Int("count", count)
```

**NUNCA usar `fmt.Println` en handlers o lógica de negocio** (solo en migrations).

## Sistema de Permisos

Los permisos son strings tipo `"CREATE_PATIENT"`, `"READ_MEDICAL_RECORD"`, etc. Se asignan a roles y se verifican por `subscription_middleware.RequirePermission(db, "PERM_CODE")`.

Los permisos se definen en `features/permissions/data/` y se siembran en code migrations.
