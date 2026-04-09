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

**Convención de ID:** `DB` + `YYYYMMDD` + `_` + número secuencial del día. Ej: `DB20260409_1`, `DB20260409_2`.

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

    permissions := []permission_models.Permission{
        {BaseStringID: "CREATE_RESOURCE", Name: "Crear recurso"},
        {BaseStringID: "READ_RESOURCE", Name: "Leer recurso"},
        {BaseStringID: "UPDATE_RESOURCE", Name: "Actualizar recurso"},
        {BaseStringID: "DELETE_RESOURCE", Name: "Eliminar recurso"},
    }

    for _, perm := range permissions {
        if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
            return fmt.Errorf("failed to create permission '%s': %w", perm.BaseStringID, err)
        }
        fmt.Printf("✅ Permission '%s' created/found.\n", perm.BaseStringID)

        if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
            return fmt.Errorf("failed to assign permission '%s': %w", perm.BaseStringID, err)
        }
        fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.BaseStringID)
    }
    return nil
},
```

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

// Dentro de RunMigrations, en database.MigrateDB():
[dominio]_models.[Nombre]{},
```

---

## Reglas

1. **IDs son ÚNICOS e INMUTABLES** — una vez creados, no cambiar el ID.
2. **Idempotencia obligatoria** — la migration puede ejecutarse múltiples veces sin efectos adversos.
3. **Múltiples migrations en un mismo archivo** — se pueden agregar varios `database.GlobalDBMap[...]` en el mismo `init()` o en archivos separados del mismo package.
4. **Errores con `fmt.Errorf("...: %w", err)`** — wrapping obligatorio para trazabilidad.
5. **Logs con `fmt.Printf("✅ ...")`** — usar emojis para distinguir visualmente en los logs de inicio.
6. **NO usar `logger.Log`** — las migrations usan `fmt` directamente, no zap.
