# Frontend Development Guide

Guías de desarrollo para el frontend SaaS (`apps/web` y `apps/backoffice`).

## 📚 Documentación

### 🎯 Complete Guide (Recommended)
**👉 [Web Frontend — Complete Guide](../skills/web-frontend-complete-guide.md)**

Guía consolidada que cubre:
- Stack: React 19 + TypeScript + Vite + TailwindCSS + Zustand
- Estructura de directorios y patrones clave
- Axios instances (cuál usar cuándo)
- Service layer pattern (NUNCA API directo)
- Types pattern (extends BaseModel, payloads)
- Zustand state management (cuándo crear, patrones)
- i18n pattern (useText hook)
- Paso a paso: implementar un feature nuevo

### Form Creation
- **[Form Creation Standard](../skills/form-creation-standard.md)** — Estándar para formularios
  - Schema con Zod
  - Form components (Input, Select, TextArea)
  - Validación y binding
  - i18n integration
  - Ejemplos por tipo

## 🚀 Quick Start

1. Lee [Web Frontend — Complete Guide](../skills/web-frontend-complete-guide.md) — cubre todo
2. Para crear formularios: [Form Creation Standard](../skills/form-creation-standard.md)

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
