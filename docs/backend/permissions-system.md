# Skill: Sistema de Permisos Granulares

Guía completa del sistema de permisos basado en roles y categorías en Pengi Med SaaS.

## 🎯 Visión General

El sistema de permisos es **granular, basado en categorías y dinámico**:

- **Granular**: Permisos específicos por acción (READ_PATIENT, CREATE_BILLING, etc.)
- **Basado en categorías**: Permisos agrupados en CLINICAL, BILLING, TEAM, etc.
- **Dinámico**: Los features habilitados se calculan automáticamente según los permisos asignados

## 🏗️ Arquitectura

```
Permission (modelo GORM)
├── ID: string (ej: "READ_PATIENT")
├── Name: string (ej: "Read Patient")
├── Category: string (ej: "CLINICAL")
├── Description: string
└── [Association] Roles

Role (modelo GORM)
├── ID: uint
├── Role: string (ej: "admin", "user", "moderator")
└── [Association] Permissions ← many-to-many

Environment (usuario + empresa + rol)
├── UserID
├── CompanyID
├── RoleID ← determina permisos disponibles
└── Tenant.EnabledFeatures ← qué features están activos
```

## 📍 Ubicaciones Clave

### Backend

| Ubicación | Responsabilidad |
|-----------|-----------------|
| `features/permissions/models/permission-model.go` | Modelo Permission |
| `features/permissions/data/permission-data.go` | Definición de permisos por categoría |
| `features/users/models/user-model.go` | Relación Role ↔ Permission |
| `features/backoffice/handlers/backoffice-plan-handler.go` | Cálculo automático de enabled_features |
| `features/backoffice/handlers/backoffice-subscription-handler.go` | Asigna features al tenant |

### Frontend

| Ubicación | Responsabilidad |
|-----------|-----------------|
| `apps/web/src/lib/constants.ts` | Constantes PERMISSIONS por categoría |
| `apps/web/src/hooks/use-permission.tsx` | Hook usePermission() |
| `apps/web/src/routes/routes.tsx` | Rutas protegidas con CheckPermission |
| `apps/web/src/store/session-store.ts` | Almacena permisos en sesión |

---

## ➕ Cómo Crear Nuevos Permisos

### 1. Backend: permission-data.go

```go
// features/permissions/data/permission-data.go
var TeamPermissions = []permission_models.Permission{
    {
        BaseStringID: database.BaseStringID{ID: "READ_TEAM"},
        Name:         "Read Team",
        Category:     "TEAM",
        Description:  "View teams",
    },
    {
        BaseStringID: database.BaseStringID{ID: "CREATE_TEAM"},
        Name:         "Create Team",
        Category:     "TEAM",
        Description:  "Create new teams",
    },
}
```

**Reglas:**
- ID: UPPERCASE_SNAKE_CASE, único globalmente
- Category: CLINICAL, BILLING, TEAM (debe existir en tenant_models.EnabledFeatures)
- Documentar bien la Description

### 2. Frontend: constants.ts

```typescript
// apps/web/src/lib/constants.ts
export const PERMISSIONS = {
    TEAM: {
        PERMISSION_READ_TEAM: "READ_TEAM",
        PERMISSION_CREATE_TEAM: "CREATE_TEAM",
        PERMISSION_UPDATE_TEAM: "UPDATE_TEAM",
        PERMISSION_DELETE_TEAM: "DELETE_TEAM",
        PERMISSION_MANAGE_TEAM_MEMBERS: "MANAGE_TEAM_MEMBERS",
    },
};
```

**Convención**: `PERMISSION_[ACCIÓN]_[RECURSO]`

### 3. Migración: Crear e asignar

```go
// migrations/code-migrations/2026/add_team_permissions.go
func init() {
    database.GlobalDBMap["DB20260409_2"] = database.DBExecute{
        ID: "DB20260409_2",
        Execute: func(db *gorm.DB) error {
            var adminRole user_models.Role
            if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
                return err
            }

            for _, perm := range permission_data.TeamPermissions {
                if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
                    return err
                }
                if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
                    return err
                }
            }
            return nil
        },
    }
}
```

**ID formato**: `DByyyymmdd_N` — nunca reutilizar

---

## 🔐 Validar Permisos

### Backend (Go)

```go
// En handlers protegidos por RequirePermission middleware
func (h *Handler) CreateTeam(c *gin.Context) envelope.Response {
    userID, _ := c.Get("user_id")
    tenantID, _ := c.Get("tenant_id")
    // El permiso ya fue validado por el middleware
    // ...
}

// En routes:
router.POST("/team",
    permission_middleware.RequirePermission(h.db, "CREATE_TEAM"),
    envelope.Handle(h.CreateTeam))
```

### Frontend (React)

```typescript
// Proteger componentes
import { CheckPermission } from "@/components/access-control/check-permission";

<CheckPermission permissions={[PERMISSIONS.TEAM.PERMISSION_READ_TEAM]}>
    <TeamContent />
</CheckPermission>

// Hook para lógica condicional
const { checkPermission } = usePermission();

{checkPermission([PERMISSIONS.TEAM.PERMISSION_DELETE_TEAM]) && (
    <Button onClick={deleteTeam}>Delete</Button>
)}
```

---

## 🎨 Features Habilitados (Automático)

Los **features** se habilitan automáticamente según categorías de permisos:

**Backend:**
- Si plan tiene ≥1 permiso CLINICAL → `clinical: true`
- Si plan tiene ≥1 permiso BILLING → `billing: true`
- Si plan tiene ≥1 permiso TEAM → `team: true`

Se calcula en:
- `backoffice-plan-handler.go` → `calculateEnabledFeatures()`
- `backoffice-subscription-handler.go` → `applyPlanFeaturesToTenant()`

**Frontend:**
- Lee `environment.enabled_features`
- Filtra menú de navegación automáticamente
- Oculta items cuyo `feature` no está habilitado

---

## 📋 Ejemplo Real: Team Permissions

Lo que hicimos hoy:

1. **Backend**: Agregamos `TeamPermissions` en `permission-data.go`
2. **Frontend**: Agregamos `PERMISSIONS.TEAM` en `constants.ts`
3. **Migración**: Creamos `add_team_permissions.go` que asigna al admin
4. **Automático**: Features se habilitan según permisos en el plan
5. **Frontend**: Si `team: false` en `enabled_features`, se oculta el item del menú

---

## ⚠️ Reglas Importantes

### Nunca

- ❌ Hardcodear strings de permisos: usar `PERMISSIONS.TEAM.PERMISSION_READ_TEAM`
- ❌ Reutilizar IDs de migración
- ❌ Crear categorías sin actualizar struct `EnabledFeatures`
- ❌ Validar solo en frontend (siempre validar en backend también)

### Siempre

- ✅ Usar categorías estándar: CLINICAL, BILLING, TEAM
- ✅ Documentar qué hace cada permiso
- ✅ Sincronizar backend ↔ frontend
- ✅ Crear migración para cada nuevo permiso
- ✅ Asignar al rol admin en la migración

---

## 📚 Referencias

- Modelos: `features/permissions/models/permission-model.go`
- Datos: `features/permissions/data/permission-data.go`
- Usuarios: `features/users/models/user-model.go`
- Plan Handler: `features/backoffice/handlers/backoffice-plan-handler.go`
- Subscription Handler: `features/backoffice/handlers/backoffice-subscription-handler.go`
- Frontend Hook: `apps/web/src/hooks/use-permission.tsx`
- Frontend Constants: `apps/web/src/lib/constants.ts`
