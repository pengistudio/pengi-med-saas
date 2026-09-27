# Skill: Crear una Code Migration en el Backend

Las code migrations son scripts Go que se ejecutan una vez al iniciar la aplicación para realizar cambios de datos o esquema en la base de datos. Son idempotentes y se identifican con un ID único.

## Ubicación

```
apps/api/migrations/code-migrations/[año]/
```

Los archivos del año se auto-registran via `init()` en un package `y[año]`.

El archivo `migrations/migrate.go` importa el package del año:
```go
import _ "pengi-med-saas/migrations/code-migrations/2026"
```

---

## Cuándo crear una migration vs AutoMigrate

- **AutoMigrate** (en `migrate.go`): Para **modelos nuevos** o campos nuevos detectables automáticamente por GORM.
- **Code Migration** (en `code-migrations/`): Para:
  - Seeds de datos iniciales (roles, permisos, registros por defecto)
  - Migraciones condicionales de columnas (`HasColumn`, `AddColumn`)
  - Transformaciones de datos existentes
  - Cambios que no puede inferir GORM automáticamente

---

## Patrón de Code Migration

### Archivo nuevo para un cambio específico: `[descripcion].go`

```go
package y[año]

import (
    "fmt"
    "pengi-med-saas/core/database"
    // Importar los models/packages que necesites

    "gorm.io/gorm"
)

func init() {
    database.GlobalDBMap["DB[YYYYMMDD]_[N]"] = database.DBExecute{
        ID: "DB[YYYYMMDD]_[N]",
        Execute: func(db *gorm.DB) error {
            // Lógica idempotente aquí
            // SIEMPRE verificar antes de crear/modificar

            return nil
        },
    }
}
```

**Convención de ID:** `DB` + `YYYYMMDD` + `_` + número secuencial del día. Ej: `DB20260409_1`, `DB20260409_2`. Antes de elegir uno, revisa los ya usados:

```bash
grep -rho 'GlobalDBMap\["[^"]*"\]' apps/api/migrations/code-migrations/2026/*.go | sort | tail -5
```

---

## Cómo se ejecutan

- `RunAllMigrations` (en cada arranque de la API) llama a `RunMigrations`, que
  hace `AutoMigrate` y luego `database.ExecuteAll`.
- `ExecuteAll` lee la tabla de migraciones ejecutadas y corre, **en una sola
  transacción**, las que falten, ordenadas por fecha e índice del ID. Si una
  falla, no se registra ninguna y la API no arranca.
- El `db` que recibe `Execute` ya es `tenantdb.System(db)`: las consultas no se
  filtran por tenant, así que la migración ve y modifica datos de todos los
  tenants.
- En dev (`air`), guardar el archivo recompila y ejecuta la migración nueva;
  confirma con `docker logs pengi-api | grep <ID>`.
- **Orden entre años:** `database.ParseFileName` interpreta la fecha del ID
  como `DDMMYYYY`, no `YYYYMMDD`. Dentro de 2026 el orden resultante es
  correcto, pero un ID de 2027 se ordenaría antes que los de fines de 2026.
  Si una migración de 2027 depende de otra anterior, arregla primero
  `ParseFileName`.
- Sufijos no numéricos (`DB20260315_CIE10_SEED`) se ordenan como índice 0 de su
  fecha.

---

## Tipos comunes de migrations

### Seed de datos (crear si no existe)

```go
Execute: func(db *gorm.DB) error {
    item := models.MyModel{
        Name: "Valor inicial",
        Code: "CODIGO",
    }
    if err := db.Where(models.MyModel{Code: item.Code}).FirstOrCreate(&item).Error; err != nil {
        return fmt.Errorf("failed to create item: %w", err)
    }
    fmt.Printf("✅ Item '%s' created/found.\n", item.Name)
    return nil
},
```

### Agregar columna (idempotente)

```go
Execute: func(db *gorm.DB) error {
    if !db.Migrator().HasColumn(&models.MyModel{}, "new_column") {
        if err := db.Migrator().AddColumn(&models.MyModel{}, "new_column"); err != nil {
            return fmt.Errorf("failed to add column: %w", err)
        }
        fmt.Println("✅ Added new_column to my_models table")
    }
    return nil
},
```

### Seed de permisos y asignación a rol

```go
Execute: func(db *gorm.DB) error {
    var adminRole user_models.Role
    if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
        return fmt.Errorf("failed to find admin role: %w", err)
    }

    // El catálogo vive en features/permissions/data/permission-data.go
    for _, perm := range permission_data.ResourcePermissions {
        if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
            return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
        }
        fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)

        if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
            return fmt.Errorf("failed to assign permission '%s': %w", perm.ID, err)
        }
        fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.ID)
    }
    return nil
},
```

Cada entrada del catálogo es
`{BaseStringID: database.BaseStringID{ID: "CREATE_RESOURCE"}, Name: "...", Category: "...", Description: "..."}`.
Ejemplo real: `add_kanban_permissions.go`. Los roles no-admin (doctor,
recepcionista, contador) y la asociación a un `Feature` del plan son pasos
aparte: ver [`permissions-system.md`](permissions-system.md).

### Actualización de datos existentes

```go
Execute: func(db *gorm.DB) error {
    if err := db.Model(&models.MyModel{}).
        Where("status IS NULL OR status = ''").
        Update("status", "active").Error; err != nil {
        return fmt.Errorf("failed to update status: %w", err)
    }
    fmt.Println("✅ Updated NULL status records to 'active'")
    return nil
},
```

---

## Agregar el modelo nuevo a AutoMigrate

Si el feature tiene un modelo nuevo, **también** hay que agregarlo en `migrations/migrate.go`:

```go
import [dominio]_models "pengi-med-saas/features/[dominio]/models"

// Dentro de RunMigrations, en la lista de database.MigrateDB():
[dominio]_models.[Nombre]{},
```

---

## Reglas

1. **IDs son ÚNICOS e INMUTABLES** — una vez creados, no cambiar el ID. Un hook de Claude Code (`.claude/hooks/guard-migrations.mjs`) bloquea automáticamente editar un archivo de migration que ya está commiteado en `origin/main`. Para corregir algo ya mergeado, creá un archivo nuevo con un ID nuevo.
2. **Idempotencia obligatoria** — la migration puede ejecutarse múltiples veces sin efectos adversos.
3. **Múltiples migrations en un mismo archivo** — se pueden agregar varios `database.GlobalDBMap[...]` en el mismo `init()` o en archivos separados del mismo package.
4. **Errores con `fmt.Errorf("...: %w", err)`** — wrapping obligatorio para trazabilidad.
5. **Logs con `fmt.Printf("✅ ...")`** — usar emojis para distinguir visualmente en los logs de inicio.
6. **NO usar `logger.Log`** — las migrations usan `fmt` directamente, no zap.
