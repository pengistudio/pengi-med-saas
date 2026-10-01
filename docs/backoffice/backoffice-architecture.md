# Backoffice — Architecture Reference

## 🎯 Propósito

Panel de administración de plataforma: empresas, planes, features,
suscripciones, pagos, roles, usuarios admin y anuncios. A diferencia de
`apps/web`:

| | `apps/web` | `apps/backoffice` |
|---|---|---|
| Usuarios | usuarios de una clínica | admins de Pengi (`BackofficeUser`) |
| Alcance de datos | un tenant (`X-Tenant-Slug`) | todas las empresas |
| Autorización | rol + plan (`RequirePermission`) | admin autenticado (`BackofficeAuthMiddleware`) |
| Rutas API | `/<dominio>/*` | `/backoffice/*` |
| Estado | Zustand por dominio | sin stores de dominio: `lib/resource` |

## Stack

React 19 + TypeScript + Vite, TailwindCSS v4, componentes de `@pengi/ui`,
`createHttpService` y `useText` de `@pengi/shared`, React Router (páginas
`lazy()`), React Hook Form + Zod vía `Form*` de `@pengi/ui`, Vitest.

## Estructura de carpetas

```
apps/backoffice/src/
├── api/
│   ├── index.ts, http-clients.ts   # instancias axios: api (Bearer admin) y noAuthApi
│   └── <x>-service.ts              # tipos + resource<T, C, U>("<x>") o funciones propias
├── components/features/            # widgets compartidos (permission-picker)
├── config/
│   ├── nav-config.ts               # sidebar
│   └── zod.ts
├── lib/
│   ├── resource/                   # CRUD genérico: Resource, ResourceList, ResourceEditPage, hooks
│   ├── session/                    # sesión del admin (token en memoria + refresh cookie)
│   └── subscription/               # lógica de términos de suscripción
├── pages/<seccion>/                # <x>-list.tsx, create-<x>.tsx, edit-<x>.tsx
├── routes/routes.tsx               # rutas lazy bajo <RequireSession>
└── sections/                       # template (DashboardLayout), forms/login
```

No hay `store/`, `types/`, `components/ui` ni `components/forms` locales.

## Sesión

`lib/session`: el access token vive solo en memoria; al cargar la página se
restaura desde la cookie de refresh (`POST /backoffice/auth/refresh`), y un 401
dispara un único refresh compartido tras el cual se reintentan las requests.
Si el refresh falla, la sesión expira y `RequireSession` manda al login.

## Patrón `resource`

Un recurso es una colección en `/backoffice/<name>` con list/get/create/update/remove:

```typescript
// src/api/feature-service.ts
export interface Feature {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  code: string;
  name: string;
  permissions: Permission[];
}
export interface CreateFeatureRequest extends Record<string, unknown> { code: string; name: string; permission_ids?: string[] }
export interface UpdateFeatureRequest extends Record<string, unknown> { name?: string; permission_ids?: string[] }

export const features = resource<Feature, CreateFeatureRequest, UpdateFeatureRequest>("features");
```

El `name` fija las convenciones:

- Rutas: `/<name>`, `/<name>/create`, `/<name>/edit/:id` (`resourceRoutes(name)`).
- i18n: `backoffice.<name>.{title,create,list.title,list.description,empty}`.
- Las escrituras muestran el mensaje del backend como toast.

Piezas de `@/lib/resource`:

| Pieza | Uso |
|---|---|
| `ResourceList` | página de listado completa: título, botón crear, carga, vacío, tabla con `columns`, borrar con confirmación, `rowActions`, `headerActions` |
| `useResourceItem(resource, id?)` | carga el ítem (si hay `id`) y da `save`, que crea o actualiza y vuelve al listado |
| `ResourceEditPage` | marco de crear/editar: estado de carga (el `DashboardLayout` lo monta `routes.tsx`) |
| `useResourceList` | listado sin `ResourceList` |
| `memoryResource` | adaptador en memoria para tests |

Un recurso que no es CRUD (anuncios con cancelación, pagos, usuarios de una
empresa) usa funciones propias con `createHttpService(api)` en su servicio.

## Backend

- Handlers en `apps/api/features/backoffice/handlers/backoffice-<x>-handler.go`
  (DTOs en `features/backoffice/dto/`), construidos con `tenantdb.System(db)`
  porque trabajan sobre todos los tenants.
- Rutas en `apps/api/routes/backoffice_routes.go`:
  `router.Group("/backoffice/<x>", backofficeAuth)` con `envelope.Handle`.
- Login, refresh y logout en `/backoffice/auth/*`, con rate limiter.
- Error codes `E-BO-NNN`; keys i18n `backoffice.*`.

## i18n

Igual que en web: `useText()` / `textGet(key)` de `@pengi/shared`, keys en
`apps/api/i18n/messages/messages_{es,en}.json`, y caché en
`localStorage["messages"]`.

## Testing

Vitest + Testing Library, tests junto al código (`lib/resource/*.test.tsx`,
`components/features/permission-picker.test.tsx`). `pnpm test:run` en
`apps/backoffice` (no lo cubre `just tests-web`; CI sí).

## Key Files

| Archivo | Responsabilidad |
|---|---|
| `src/lib/resource/*` | CRUD genérico |
| `src/lib/session/*` | sesión del admin |
| `src/api/http-clients.ts`, `src/api/index.ts` | clientes HTTP |
| `src/routes/routes.tsx` | rutas |
| `src/config/nav-config.ts` | sidebar |
| `apps/api/routes/backoffice_routes.go` | endpoints |
