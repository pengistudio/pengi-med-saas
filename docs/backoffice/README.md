# Backoffice — Admin Panel Development

Guías del panel de administración de plataforma (`apps/backoffice` + los
endpoints `/backoffice/*` de `apps/api`). Lo usan solo los admins de Pengi:
ve todas las empresas y no tiene tenants, permisos por rol ni planes.

## 📚 Documentación

- **[backoffice-architecture.md](backoffice-architecture.md)** — stack,
  estructura de carpetas, sesión, patrón `resource`, backend.
- **[backoffice-domains-guide.md](backoffice-domains-guide.md)** — qué administra
  cada sección (empresas, planes, features, suscripciones, roles, usuarios,
  anuncios) y cómo se relacionan.
- **[backoffice-new-admin-feature.md](backoffice-new-admin-feature.md)** — paso a
  paso para agregar una sección nueva, de la API a la pantalla.

## 🎯 Common Tasks

| Tarea | Dónde ir |
|---|---|
| Entender la arquitectura | [backoffice-architecture.md](backoffice-architecture.md) |
| Qué hace un plan / feature / suscripción | [backoffice-domains-guide.md](backoffice-domains-guide.md) |
| Agregar una sección admin | [backoffice-new-admin-feature.md](backoffice-new-admin-feature.md) |
| Cómo un plan decide qué ve un tenant | [`../backend/permissions-system.md`](../backend/permissions-system.md) |

## 🛠️ Dónde copiar

| Patrón | Archivo |
|--------|---|
| Servicio de un recurso CRUD | `src/api/feature-service.ts` |
| Listado | `src/pages/features/feature-list.tsx` (`ResourceList`) |
| Crear / editar | `src/pages/features/create-feature.tsx`, `edit-feature.tsx` (`useResourceItem` + `ResourceEditPage`) |
| Formulario con editores anidados | `src/pages/plans/create-plan.tsx` + `plan-limits-editor.tsx`, `plan-pricings-editor.tsx` |
| Selector de permisos | `src/components/features/permission-picker.tsx` |
| Handler + rutas backend | `apps/api/features/backoffice/handlers/backoffice-feature-handler.go`, `apps/api/routes/backoffice_routes.go` |
| Sección con scheduler (punta a punta) | commit f393d80 (anuncios) |

## 📞 FAQ

**¿Por qué el backoffice no envía `X-Tenant-Slug`?**
Porque administra todas las empresas. Su cliente HTTP (`api`) solo lleva el
Bearer del admin, y el backend consulta con `tenantdb.System(db)`.

**¿Cómo se habilitan las secciones de la app para un tenant?**
No se copian ni se guardan: la app calcula `enabled_features` en cada request a
partir de las categorías de los permisos de los features del plan activo. Ver
[`backoffice-domains-guide.md`](backoffice-domains-guide.md#-plans-planes).

**¿Cómo se escriben tests?**
Vitest + Testing Library, junto al código (`*.test.tsx`). Para páginas de
recursos, `memoryResource` reemplaza al HTTP (ver `lib/resource/resource-list.test.tsx`).
