# Backend Development Guide

Guías de desarrollo para el backend API (`apps/api`).

## 📚 Documentación

### 🎯 Complete Guide (Recommended)
**👉 [API Backend — Complete Guide](../skills/api-backend-complete-guide.md)**

Guía consolidada que cubre:
- Stack técnico, estructura de directorios
- Multi-tenancy model & TenantScope
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

- **[permissions-system.md](permissions-system.md)** — Sistema RBAC granular
  - Permisos por rol
  - Requerimientos de permiso
  - Integración con subscriptions

## 🚀 Quick Start

1. Lee [API Backend — Complete Guide](../skills/api-backend-complete-guide.md) — cubre todo
2. Para migraciones específicas: [api-code-migration.md](api-code-migration.md)
3. Para permisos: [permissions-system.md](permissions-system.md)

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
