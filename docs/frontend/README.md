# Frontend Development Guide

Guías de desarrollo para el frontend SaaS (`apps/web` y `apps/backoffice`).

## 📚 Documentación

### Architecture & Patterns
- **[web-architecture-reference.md](web-architecture-reference.md)** — Stack, estructura de carpetas, patrones clave
  - React 19 + Vite + TypeScript + Zustand
  - Componentes con shadcn/ui + Tailwind
  - i18n, routing, state management
  - Features habilitados & navegación dináminca

### API Integration
- **[web-api-services.md](web-api-services.md)** — Cómo crear y usar servicios de API
  - HttpService class + axios instances
  - Patrones de request/response
  - Toast notifications (manejadas por servicio)
  - Error handling
  - Ejemplos prácticos por dominio

### State Management
- **[web-state-management.md](web-state-management.md)** — Zustand para estado compartido
  - Crear stores por dominio
  - Persistencia (localStorage/sessionStorage)
  - Casos de uso comunes
  - Testing y debugging
  - Cuándo usar store vs useState

### Implementation Guides
- **[web-new-feature.md](web-new-feature.md)** — Paso a paso para agregar nuevas features
  - Crear tipos TypeScript
  - Servicio de API
  - Store (si necesario)
  - Componentes y páginas
  - Routing y navegación
  - i18n keys
  - Checklist completo

## 🚀 Quick Start

1. Lee [web-architecture-reference.md](web-architecture-reference.md) para entender la estructura
2. Para una nueva feature, sigue [web-new-feature.md](web-new-feature.md)
3. Para comunicación con API, consulta [web-api-services.md](web-api-services.md)
4. Para state compartido, usa [web-state-management.md](web-state-management.md)

## 📂 Related
- Frontend source: `apps/web/`
- Backoffice source: `apps/backoffice/`
- Backend guide: [../backend/README.md](../backend/README.md)
- Main docs: [../README.md](../README.md)

## 🔑 Key Concepts

### API Requests
Nunca uses axios directamente en componentes — usa servicios.

```typescript
// ❌ NUNCA:
const res = await apiWithTenant.get("/patients");

// ✅ SIEMPRE:
const res = await getPatients();
```

### i18n
Todas las strings visibles van a través de `useText()`:

```typescript
const { textGet } = useText();
<h1>{textGet("dashboard.title")}</h1>
```

### State Management
Usa Zustand para estado compartido entre componentes:

```typescript
const { selectedItem, setSelectedItem } = useStore();
```

### Permissions
Protege rutas y componentes con `CheckPermission`:

```typescript
<CheckPermission permissions={[PERMISSIONS.TEAM.PERMISSION_READ_TEAM]}>
    <TeamContent />
</CheckPermission>
```

### Features Habilitados
El frontend filtra automáticamente nav items según:
- Permisos del usuario
- Features habilitados en el plan

## 📋 Stack Versions

- React 19
- Vite 5
- TailwindCSS 4
- shadcn/ui (latest)
- Zustand 4
- Axios (latest)
- React Router 6
- TypeScript 5

## 🛠️ Common Patterns

| Situación | Dónde copiarlo |
|-----------|---|
| Crear un servicio de API | `api/clinical-service.ts` |
| Crear un componente form | `pages/clinical/components/patient-form.tsx` |
| Crear un store | `store/billing-store.ts` |
| Crear una página | `pages/team/page.tsx` |
| Usar permisos | `pages/billing/page.tsx` |

## 📞 Debugging

- **DevTools**: Redux DevTools para inspeccionar Zustand stores
- **Network**: Inspecciona requests/responses en Network tab
- **Console**: Busca errores de API o TypeScript
- **React DevTools**: Inspecciona componentes y props

## 📖 More Info

- [TailwindCSS Docs](https://tailwindcss.com)
- [shadcn/ui Components](https://ui.shadcn.com)
- [Zustand Docs](https://github.com/pmndrs/zustand)
- [React Router](https://reactrouter.com)
- [Vite Docs](https://vitejs.dev)
