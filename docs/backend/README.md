# Backend Development Guide

Guías de desarrollo para el backend API (`apps/api`).

## 📚 Documentación

### 🎯 Complete Guide (Recommended)
**👉 [API Backend — Complete Guide](../skills/api-backend-complete-guide.md)**

Guía consolidada que cubre:
- Stack técnico, estructura de directorios
- Multi-tenancy con `tenantdb` (aislamiento en la capa de datos)
- Flujo de request, middlewares
- Patrones core: handlers, DTOs, modelos, error codes
- Paso a paso: implementar un feature nuevo
- Logging con Zap, reglas multi-tenancy

### Specialized References
- **[api-code-migration.md](api-code-migration.md)** — Cómo crear migraciones de código
  - Ubicación de migraciones
  - Estructura de una migración
  - Ejecutar migraciones
  - Debugging

- **[permissions-system.md](permissions-system.md)** — Permisos, roles y planes
  - Rol + plan: `RequirePermission` / `RequireRolePermission`
  - Roles canónicos y `RolePermissionMatrix`
  - Features habilitados calculados desde el plan

## 🚀 Quick Start

1. Para crear o extender una feature de punta a punta: skill `create-feature` (`.claude/skills/create-feature/`)
2. Arquitectura y patrones: [API Backend — Complete Guide](../skills/api-backend-complete-guide.md)
3. Para migraciones específicas: [api-code-migration.md](api-code-migration.md)
4. Para permisos: [permissions-system.md](permissions-system.md)

## 📂 Related
- Backend source: `apps/api/`
- Main docs: [../README.md](../README.md)
- Frontend guide: [../frontend/README.md](../frontend/README.md)

### Decisiones de arquitectura
- [ADR 0001 — Clave de acceso SRI inmutable](../adr/0001-clave-de-acceso-inmutable.md)
- [ADR 0002 — Aislamiento por tenant en la capa de datos](../adr/0002-aislamiento-por-tenant-en-la-capa-de-datos.md)
