# Backend Development Guide

Guías de desarrollo para el backend API (`apps/api`).

## 📚 Documentación

### Architecture & Patterns
- **[api-architecture-reference.md](api-architecture-reference.md)** — Stack técnico, estructura de directorios, patrones de la arquitectura
  - Stack: Go + Gin + GORM + Zap + RabbitMQ
  - Estructura de carpetas y dónde va cada cosa
  - Multi-tenancy con GORM scopes
  - Error handling centralizado

### Implementation Guides
- **[api-new-feature.md](api-new-feature.md)** — Cómo agregar nuevas funcionalidades al API paso a paso
  - Crear modelos GORM
  - Estructurar handlers
  - Implementar DTOs
  - Registrar rutas
  - Trabajar con multi-tenancy
  - Ejemplos prácticos (Patient, Invoice, etc.)

### Database & Migrations
- **[api-code-migration.md](api-code-migration.md)** — Cómo crear y ejecutar migraciones de código
  - Ubicación de migraciones
  - Estructura de una migración
  - Ejecutar migraciones
  - Debugging

### API Response & Error Handling
- **[api-errors-routes-responses.md](api-errors-routes-responses.md)** — Patrones de responses, errores y rutas
  - Response envelope pattern
  - Error codes centralizados
  - Route registration
  - i18n messages
  - Request validation

## 🚀 Quick Start

1. Lee [api-architecture-reference.md](api-architecture-reference.md) para entender la estructura
2. Cuando crees un feature, sigue [api-new-feature.md](api-new-feature.md)
3. Para cambios en BD, consulta [api-code-migration.md](api-code-migration.md)
4. Para responses/errores, revisa [api-errors-routes-responses.md](api-errors-routes-responses.md)

## 📂 Related
- Backend source: `apps/api/`
- Main docs: [../README.md](../README.md)
- Frontend guide: [../frontend/README.md](../frontend/README.md)

### Permissions & Authorization
- **[permissions-system.md](permissions-system.md)** — Sistema granular de permisos, roles y features habilitados
  - Cómo crear nuevos permisos
  - Validación en backend y frontend
  - Cálculo automático de features habilitados
  - Sincronización plan ↔ tenant ↔ navegación
