---
name: Skills Index
description: Índice de guías de implementación consolidadas para el proyecto Pengi Med SaaS
---

# 🛠️ Skills — Guías de Implementación

Guías de arquitectura y patrones para implementar features en Pengi Med SaaS.

**Para crear o extender una feature de punta a punta** (backend, permisos,
plan, frontend, tests) la receta es la skill de Claude Code
[`create-feature`](../../.claude/skills/create-feature/SKILL.md): pasos en
orden, cada uno con su criterio de cierre, y referencias por caso (PDFs,
backoffice, flags de navegación). Las guías de esta carpeta son la referencia
de arquitectura que esa skill complementa.

## 📚 Guías

### [API Backend — Complete Guide](api-backend-complete-guide.md)
Backend Go/Gin/GORM: estructura, multi-tenancy con `tenantdb`, flujo de
request, handlers, DTOs, modelos, envelope, error codes, middleware de
permisos, descargas de binarios/PDFs, trabajo en segundo plano, tests.

**Cuándo usar:** implementar o entender cualquier parte de `apps/api`.

### [Web Frontend — Complete Guide](web-frontend-complete-guide.md)
Frontend React: estructura de `apps/web/src`, instancias axios, service layer
(`@pengi/shared`), tipos, Zustand, i18n, listados con `DataTable`, rutas
`lazy()` con `CheckPermission`, navegación, tests.

**Cuándo usar:** implementar o entender cualquier parte de `apps/web`.

### [Form Creation Standard](form-creation-standard.md)
Formularios con Zod + `Form` y componentes `Form*` de `@pengi/ui`: estructura,
componentes disponibles, reglas de i18n y layout, página vs. diálogo.

**Cuándo usar:** cualquier formulario de creación/edición en web o backoffice.

## 📎 Referencias especializadas

- [`docs/backend/api-code-migration.md`](../backend/api-code-migration.md) — code-migrations (IDs inmutables)
- [`docs/backend/permissions-system.md`](../backend/permissions-system.md) — permisos, roles, planes
- [`docs/adr/`](../adr/) — decisiones de arquitectura (clave de acceso SRI, aislamiento por tenant)
- [`docs/backoffice/`](../backoffice/) — panel de administración
- [`packages/shared/README.md`](../../packages/shared/README.md) — qué va en `@pengi/shared`

## 🔑 Patrones Clave

| Tema | Regla |
|---|---|
| Multi-tenancy | `TenantID` en los modelos; consultas vía `tenantdb.For(c, h.db)` |
| Respuestas | handlers devuelven `envelope.Response` con keys i18n |
| Permisos | `RequirePermission` en rutas; `CheckPermission` + `PERMISSIONS` en web, strings idénticos |
| API calls | siempre vía servicio (`createHttpService`), nunca axios en componentes |
| Toasts | los muestra el servicio (`notifySuccess`/`notifyError`) |
| i18n | `useText()` + `textGet(key)`; keys en `messages_es.json` y `messages_en.json` |
| Estado | Zustand solo para estado compartido; `useState` para local |
| PDFs/archivos | `core/pdfrender` y `core/tenantfiles`, nunca Gotenberg ni rutas `storage/` a mano |

## ⚡ Quick Commands

```bash
just dev                              # Stack completo (Docker)
just check                            # Biome (web, backoffice, ui)
just tests-api                        # go test dentro del contenedor api
just tests-web                        # vitest de apps/web
cd apps/api && go build ./... && go vet ./...
cd apps/web && pnpm run typecheck

# Solo infraestructura para correr el backend fuera de Docker
docker compose -f docker-compose.dev.yaml up -d db rabbitmq gotenberg sri-xml-signer
```
