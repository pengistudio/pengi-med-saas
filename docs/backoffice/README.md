# Backoffice — Admin Panel Development

Guías de desarrollo para el panel administrativo (`apps/backoffice`).

## 📚 Documentación

### Architecture & Setup
- **[backoffice-architecture.md](backoffice-architecture.md)** — Stack, estructura, diferencias con frontend
  - Mismo stack que web (React, Vite, TypeScript, Zustand)
  - Estructura de carpetas
  - Dominios administrativos principales
  - Patrones clave (API, state, forms)

### Domains & Operations
- **[backoffice-domains-guide.md](backoffice-domains-guide.md)** — Detalles de cada dominio administrativo
  - Companies (empresas/clientes)
  - Plans (planes y features)
  - Subscriptions (asignar planes)
  - Roles & Permissions (control de acceso)
  - Features (módulos)
  - Users (usuarios del sistema)
  - Diagrama de relaciones

### Implementation Guide
- **[backoffice-new-admin-feature.md](backoffice-new-admin-feature.md)** — Paso a paso para agregar nuevas operaciones
  - Crear tipos TypeScript
  - Servicio de API
  - Store (si es necesario)
  - Página de lista (tabla)
  - Página de crear/editar (form)
  - Rutas y navegación
  - i18n keys
  - Checklist completo

## 🚀 Quick Start

1. Lee [backoffice-architecture.md](backoffice-architecture.md) para entender la estructura
2. Consulta [backoffice-domains-guide.md](backoffice-domains-guide.md) para entender cada dominio
3. Sigue [backoffice-new-admin-feature.md](backoffice-new-admin-feature.md) para agregar nuevas operaciones

## 🎯 Common Tasks

| Tarea | Dónde ir |
|---|---|
| Entender la arquitectura | [backoffice-architecture.md](backoffice-architecture.md) |
| Trabajar con plans | [backoffice-domains-guide.md](backoffice-domains-guide.md#-plans-planes) |
| Asignar plan a empresa | [backoffice-domains-guide.md](backoffice-domains-guide.md#-subscriptions-suscripciones) |
| Agregar nueva sección admin | [backoffice-new-admin-feature.md](backoffice-new-admin-feature.md) |
| Entender features habilitados | [backoffice-domains-guide.md](backoffice-domains-guide.md#-diagrama-de-relaciones) |
| Crear form con validación | [backoffice-new-admin-feature.md](backoffice-new-admin-feature.md#paso-5-crear-página-de-crareditar) |

## 📂 Key Files

| Archivo | Responsabilidad |
|---------|-----------------|
| `api/company-service.ts` | CRUD de empresas |
| `api/plan-service.ts` | CRUD de planes + cálculo automático de features |
| `api/subscription-service.ts` | Asignar planes a empresas |
| `pages/plans/create-plan.tsx` | Crear plan con selector de features |
| `pages/plans/edit-plan.tsx` | Editar plan con features automáticos |
| `pages/companies/list.tsx` | Listar empresas |
| `pages/subscriptions/list.tsx` | Gestionar suscripciones |

## 🔑 Key Concepts

### Automatic Features Calculation

**No es manual.** El backend calcula automáticamente:

1. Admin selecciona features para un plan
2. Backend obtiene permisos de esos features
3. Backend calcula enabled_features según categorías:
   - Si hay permisos CLINICAL → clinical: true
   - Si hay permisos BILLING → billing: true
   - Si hay permisos TEAM → team: true
4. Guarda en plan.properties.enabled_features

### Subscription Flow

```
Admin crea/edita Plan
    ↓ (features → enabled_features automático)
Admin asigna Plan a Empresa (crea Subscription)
    ↓ (backend copia enabled_features al tenant)
Usuario inicia sesión en app
    ↓ (frontend lee tenant.enabled_features)
    ↓ (filtra nav items automáticamente)
```

### API Patterns

Como backoffice es admin, no usa tenant header:

```typescript
// ❌ NO X-Tenant-Slug (backoffice ve todo)
const api = axios.create({
    headers: { "Authorization": `Bearer ${token}` }
});

// Servicios:
const companyService = createHttpService(api);
```

## 🛠️ Common Patterns

| Patrón | Dónde copiarlo |
|--------|---|
| CRUD service | `api/plan-service.ts` |
| List table | `pages/plans/plan-list.tsx` |
| Create form | `pages/plans/create-plan.tsx` |
| Edit form | `pages/plans/edit-plan.tsx` |
| Zustand store | `store/company-store.ts` |
| Sidebar layout | `sections/template/dashboard-template.tsx` |

## 📋 Implementation Checklist

Cuando agregues un nuevo dominio administrativo:

```
1. Crear tipos (TypeScript)
2. Crear servicio (API)
3. Crear store (si compartis estado)
4. Crear páginas (list, create, edit)
5. Registrar rutas
6. Agregar navegación
7. Agregar i18n keys
8. Escribir tests
```

## 🔗 Related Documentation

- [Frontend (apps/web)](../frontend/README.md) — Frontend principal para usuarios
- [Backend (apps/api)](../backend/README.md) — Backend API en Go
- [Permissions System](../backend/permissions-system.md) — Sistema de permisos

## 📞 FAQ

**¿Por qué backoffice no tiene X-Tenant-Slug?**
Porque admin ve TODAS las empresas, no una sola. No hay tenant scope.

**¿Cómo se habilitan los features?**
Automáticamente. Admin selecciona features → backend calcula enabled_features → copia al tenant → frontend filtra.

**¿Qué diferencia hay con el frontend principal?**
Frontend es multi-tenant (ve datos de UNA empresa). Backoffice es admin global (ve TODAS las empresas).

**¿Cómo se escriben tests?**
Como en web. Mock servicios, render componentes, test interacciones.
