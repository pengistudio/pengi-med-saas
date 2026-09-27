---
name: create-feature
description: Crear o extender una feature/dominio en pengi-med-saas de punta a punta (apps/api, apps/web, apps/backoffice). Úsala al agregar un dominio nuevo, endpoints o pantallas nuevas a un dominio existente, o una sección de administración en el backoffice — "crear feature", "nueva funcionalidad", "add new domain", "agregar X a clinical".
---

# Crear o extender una feature

Receta en orden. Cada paso termina en un **criterio de cierre**: no pases al
siguiente hasta cumplirlo. El detalle de cada paso vive en `references/`; abre
solo el archivo que el paso nombra.

Copia de un **dominio canónico**, no de las guías en `docs/skills/`, que están
desactualizadas en varios puntos (listados en
[`references/docs-desactualizadas.md`](references/docs-desactualizadas.md)).
Los canónicos son **kanban** (backend y rutas, el más limpio) y el catálogo de
**billing** (`catalog-item`, el frontend CRUD más completo).

## 0. Mapa y decisión de gating

Antes de escribir código, deja por escrito:

1. **¿Dominio nuevo o extensión?** Si `apps/api/features/<domain>/` existe, lee
   sus handlers, rutas y migraciones actuales primero.
2. **Ramas que toca**, cada una con su referencia:
   - Genera PDF, sirve archivos del tenant o envía email →
     [`references/documentos.md`](references/documentos.md)
   - Trabajo en segundo plano (scheduler, consumer RabbitMQ) →
     sección *Segundo plano* de [`references/backend.md`](references/backend.md)
   - Pantallas de administración de plataforma (solo admins de Pengi) →
     [`references/backoffice.md`](references/backoffice.md). Esta rama no usa
     permisos ni plan; salta los pasos 2 y 6.
3. **Decisión de gating**: elige exactamente una opción y anótala en el mensaje
   del commit o en el PR:
   - **Plan + rol**: `RequirePermission`. Es el caso normal para funcionalidad clínica o de facturación.
   - **Solo rol**: `RequireRolePermission`, como en team.
   - **Sin permisos**: basta con que el usuario pertenezca al tenant, como en notifications.

   Las reglas de cada opción están en
   [`references/permisos-y-plan.md`](references/permisos-y-plan.md).

**Criterio de cierre:** hay una lista de ramas y una decisión de gating
escrita, y leíste el dominio canónico o el existente.

## 1. Backend: modelo, DTO, handler

Sigue [`references/backend.md`](references/backend.md). Lo que no se negocia:

- Toda consulta de datos de tenant usa `db := tenantdb.For(c, h.db)`, y los
  handlers devuelven `envelope.Response`.
- El `Message` de cada respuesta es una key i18n.

**Criterio de cierre:** `go build ./...` pasa en `apps/api`, y ningún handler
nuevo consulta `h.db` sin pasar por `tenantdb.For`
(`grep -n 'h\.db\.' features/<domain>/handlers/*.go` solo debe devolver las
líneas `tenantdb.For(c, h.db)`).

## 2. Permisos

Solo aplica si la decisión fue **plan + rol** o **solo rol**. Sigue
[`references/permisos-y-plan.md`](references/permisos-y-plan.md) § Permisos:
el catálogo en `permission-data.go`, la migración que los asigna al rol admin,
la decisión por rol canónico (doctor, recepcionista, contador) y el middleware
en las rutas.

**Criterio de cierre:** cada ID de permiso aparece en `permission-data.go`, en
una migración nueva, en su ruta y en `PERMISSIONS` de
`apps/web/src/lib/constants.ts`. En los cuatro lugares el string es idéntico
byte a byte.

## 3. Migraciones, error codes, i18n, registro

Sigue [`references/backend.md`](references/backend.md) § Cableado:

- Los modelos nuevos se agregan a `RunMigrations`.
- Los error codes van en `codes.go`, cada uno con su key i18n.
- Las keys van en **ambos** `messages_es.json` y `messages_en.json`.
- `Register<Domain>Routes` se agrega en `routes/index.go`.
- Cada migración usa un archivo nuevo con una key nueva. Las migraciones ya
  presentes en `origin/main` son inmutables, y el hook `guard-migrations`
  bloquea cualquier edición.

**Criterio de cierre:** existe cada key usada en Go y en TS, junto con cada
error code nuevo, en los dos JSON (verificado con grep, no a ojo), y la API
arranca en docker (`docker logs pengi-api`) con tus migraciones
imprimiendo ✅.

## 4. Frontend: servicio, tipos, store, páginas

Sigue [`references/frontend.md`](references/frontend.md):

- Servicio y tipos en `src/api/<domain>-service.ts`.
- Páginas `lazy()` en `src/pages/<domain>/`.
- Formularios en `src/sections/forms/<domain>/` (Zod + componentes `Form*` de
  `@pengi/ui`).
- Listados con `DataTable` y columnas en `src/sections/columns/<domain>/`.
- Store Zustand solo si el estado es compartido.

Un registro que el usuario querrá volver a ver, imprimir o reenviar vive en
una **página + listado**. Un `Dialog` es para ediciones rápidas de uno o dos
campos.

**Criterio de cierre:** `pnpm run typecheck` pasa en `apps/web`, y cada string
visible pasa por `textGet`.

## 5. Rutas y navegación (web)

Sigue [`references/frontend.md`](references/frontend.md) § Rutas y nav:

- Agrega el grupo al array `routes` de `routes.tsx`, con `<CheckPermission>` en
  el grupo **y** en cada ruta.
- Agrega el ítem de `nav-config.ts` con su `permission`.

**Criterio de cierre:** la URL nueva carga para admin y redirige a `/` para un
rol sin el permiso.

## 6. Plan y flag de navegación

Solo aplica si la decisión fue **plan + rol**. Sigue
[`references/permisos-y-plan.md`](references/permisos-y-plan.md) § Plan:

- Asocia los permisos al `Feature` del plan con una migración propia.
- Si el nav item debe ocultarse según el plan, sigue § Flag de navegación. El
  flag lo decide la `Category` de los permisos.

Este es el paso que más se olvida, y ni el build ni el lint lo detectan. El
síntoma es un toast 403 que dice *"Your plan does not include this feature"*,
aunque el rol tenga el permiso.

**Criterio de cierre:** con un usuario de un tenant con suscripción activa,
el endpoint nuevo responde 200 en el navegador, no 403.

## 7. Tests y verificación

Sigue [`references/verificacion.md`](references/verificacion.md). CI corre
todo esto, pero el pre-commit solo corre Biome y el typecheck de web y
backoffice:

- Backend: `go vet`, `go test` y `go build`.
- Frontend: typecheck y `test:run` en web, backoffice y `@pengi/shared`.

**Criterio de cierre:** al menos un `*_test.go` nuevo cubre el handler
principal, incluido el aislamiento entre tenants si el modelo tiene
`TenantID`. Todo lo que corre CI pasa localmente para los paquetes que tocaste.

## 8. Smoke test en navegador

- Recorre cada pantalla nueva con un usuario admin y con uno sin permiso.
- Si agregaste keys i18n, antes de juzgar corre en la consola del navegador
  `localStorage.removeItem("messages")` y recarga.
- Corre `/doctor` (skill `react-doctor`) sobre el frontend.

**Criterio de cierre:** ninguna pantalla muestra un `*key*` literal, la consola
no tiene errores, no hay 403 de plan, y `/doctor` no reporta issues nuevos.

## Definition of Done

```
- [ ] Ramas y decisión de gating escritas en el commit/PR (paso 0)
- [ ] Handlers con tenantdb.For + envelope.Response; go build OK (1)
- [ ] Permisos: data + migración admin + roles canónicos decididos + rutas + constants.ts idénticos (2)
- [ ] Modelos en RunMigrations; migraciones nuevas con key nueva, sin editar las existentes (3)
- [ ] Error codes con key i18n; keys en es Y en (3)
- [ ] Routes registradas en routes/index.go (3)
- [ ] Servicio/tipos/páginas lazy/forms/listados en su carpeta (4)
- [ ] routes.tsx (grupo en `routes`, CheckPermission) + nav-config.ts (5)
- [ ] Permisos asociados al Feature del plan; flag de nav si aplica (6)
- [ ] Tests nuevos; vet/test/build + typecheck/test:run de los paquetes tocados (7)
- [ ] Smoke test en navegador sin *key*, sin 403 de plan; /doctor limpio (8)
```
