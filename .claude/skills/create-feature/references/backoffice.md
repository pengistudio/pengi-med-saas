# Backoffice (plataforma, solo admins de Pengi)

Rama del paso 0. El backoffice no tiene tenants, permisos ni planes: todo su
API va detrás de `backofficeAuth`. Salta los pasos 2 y 6 de la receta.

## API

- Handlers y DTOs en `apps/api/features/backoffice/{handlers,dto}/backoffice-<x>-handler.go`;
  modelos propios del dominio en `features/<domain>/models` si los usa también la app.
- Rutas en `apps/api/routes/backoffice_routes.go`:
  `router.Group("/backoffice/<xs>", backofficeAuth)`.
- Consultas cross-tenant → `tenantdb.System(h.db)`.
- Error codes con prefijo `E-BO-`, keys i18n `backoffice.<x>.*`.

## Frontend (apps/backoffice)

- Servicio en `src/api/<x>-service.ts` con tipos inline y el helper
  `resource<T, Create, Update>("<xs>")` (`api/feature-service.ts`).
- CRUD estándar con `src/lib/resource`: `ResourceList` (listado, ~30 líneas —
  `pages/features/feature-list.tsx`) y `resource-edit-page`. Solo escribe una
  página a mano si el recurso no encaja.
- Páginas `lazy()` en `src/routes/routes.tsx` bajo `RequireSession`; ítem en
  `src/config/nav-config.ts`.
- No hay `store/`, `types/` ni `components/forms` locales: formularios con los
  `Form*` de `@pengi/ui`.
- Tests al lado del código (`*.test.tsx`), `pnpm test:run` en `apps/backoffice`.

`docs/backoffice/*.md` describe carpetas (`store/`, `types/`, `pages/*/list.tsx`)
que ya no existen; el código de `pages/features` y `pages/plans` es la referencia.
Ejemplo reciente de punta a punta: commit f393d80 (anuncios, con scheduler).
