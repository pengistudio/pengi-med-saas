---
name: Skills Index
description: Índice de guías de implementación consolidadas para el proyecto Pengi Med SaaS
---

# 🛠️ Skills — Guías de Implementación

Colección **consolidada** de guías completas para implementar features en Pengi Med SaaS. Cada skill combina arquitectura + patrones + guía paso a paso.

## 📚 Skills Disponibles

### 🎯 Guías Completas (Arquitectura + How-To)

#### [API Backend — Complete Guide](api-backend-complete-guide.md)
Guía completa del backend Go/Gin/GORM incluyendo arquitectura, patrones core, y cómo implementar features.

**Cubre:**
- Stack técnico y estructura de directorios
- Modelo multi-tenancy
- Flujo de request (middlewares, handlers, responses)
- Patrones: handlers, DTOs, modelos, error codes, responses
- Paso a paso: crear modelo → handler → rutas → migraciones → i18n
- Logging con Zap
- Reglas multi-tenancy

**Cuándo usar:** Implementar cualquier feature backend, entender arquitectura API.

---

#### [Web Frontend — Complete Guide](web-frontend-complete-guide.md)
Guía completa del frontend React incluyendo arquitectura, service layer, state management, y cómo implementar features.

**Cubre:**
- Stack técnico y estructura de directorios
- Axios instances (cuál usar cuándo)
- Service layer pattern (NUNCA API directo)
- Types pattern (extends BaseModel, payloads)
- Zustand state management (cuándo crear, store pattern)
- i18n pattern (useText hook)
- Paso a paso: crear tipos → servicio → store → página → componentes → rutas → i18n
- Toast ownership (service maneja todo)

**Cuándo usar:** Implementar cualquier feature frontend, entender arquitectura web.

---

### 🎨 Guías Especializadas

#### [Form Creation Standard](form-creation-standard.md)
Estándar para crear formularios reactivos con Zod, Form components y Zustand.

**Cubre:**
- Schema con Zod + infer tipos
- Form component wrapper
- FormInput, FormSelect, FormTextArea, etc.
- Validación y binding tags
- i18n integration
- Ejemplos: simple, editar, validación condicional

**Cuándo usar:** Crear formularios de creación/edición en cualquier módulo.

---

## 🎯 Flujo Típico: Feature Completo (Backend + Frontend)

### Backend (Go)

1. **Modelo** → `features/[domain]/models/[nombre].go`
   - Extender `gorm.Model`
   - Incluir `TenantID`
   - Tags JSON en snake_case

2. **DTOs** → `features/[domain]/dto/[nombre]-dto.go`
   - Create: campos required normal
   - Update: **TODOS** campos como punteros

3. **Handler** → `features/[domain]/handlers/[nombre]-handler.go`
   - Struct con `db` + `logger`
   - Métodos retornan `envelope.Response`
   - Todos los logs con `zap`

4. **Error Codes** → `core/errors/codes.go`
   - Agregar códigos `E-[DOM]-NNN`

5. **Rutas** → `routes/[domain]_routes.go`
   - Usar `envelope.Handle()`
   - Registrar en `routes/index.go`

6. **Migración** → `migrations/migrate.go`
   - Agregar modelo en `AutoMigrate`

7. **i18n** → `i18n/messages/messages_es.json` + `messages_en.json`
   - Keys: `[domain].[recurso].action`

### Frontend (React)

1. **Tipos** → `src/types/[domain]-type.ts`
   - Extender `BaseModel`
   - Payloads separados

2. **Servicio** → `src/api/[domain]-service.ts`
   - Usar `createHttpService(apiWithTenant)`
   - Tipificar responses
   - Toasts via `notifySuccess`/`notifyError`

3. **Store** (si necesario) → `src/store/[domain]-store.ts`
   - Zustand con `create()`
   - Solo para estado compartido

4. **Página** → `src/pages/[domain]/page.tsx`
   - useEffect para cargar datos
   - Usar store para estado compartido

5. **Componentes** → `src/pages/[domain]/components/`
   - Card, Dialog, etc.
   - Props tipadas

6. **Rutas** → `src/routes/routes.tsx`
   - Envolver con `<CheckPermission>`
   - Registrar ruta

7. **Navegación** → `src/config/nav-config.ts`
   - Agregar item con icon + label

8. **i18n** → Mismas keys que backend
   - `[domain].title`, `[domain].create`, etc.

---

## 🏗️ Arquitectura de Alto Nivel

### Backend

```
Request
    ↓
[Auth Middleware] → user_id
    ↓
[Tenant Middleware] → tenant_id
    ↓
[Permission Middleware] → granular RBAC (opcional)
    ↓
envelope.Handle(handler.Method)
    ├─ Handler procesa lógica
    ├─ SIEMPRE usa TenantScope(c)
    └─ Retorna envelope.Response
    ↓
[i18n Translation] → traduce message key
    ↓
JSON Response
```

### Frontend

```
User Action
    ↓
Component (page.tsx)
    ↓
Service ([domain]-service.ts)
    ├─ Elige axios instance correcto
    ├─ Tipifica response
    └─ Maneja toasts
    ↓
API Backend
    ↓
Response → Service → Component
    ├─ Actualiza estado (store/useState)
    └─ Navega si es necesario
```

---

## 🔑 Patrones Clave

### Multi-Tenancy
- **Modelo:** `TenantID uint` en TODOS los modelos
- **Query:** **SIEMPRE** `TenantScope(c)` en cada query
- **Create:** Inyectar `tenantID` desde context

### Error Handling
- **Backend:** `core_errors.ErrXxx` + i18n keys
- **Frontend:** Service maneja toasts automáticamente
- **Componente:** Solo chequea `res.success` para navegación

### i18n
- **Backend:** JSON files (es + en)
- **Frontend:** `useText()` hook + `textGet("key")`
- **Rule:** NUNCA hardcodear strings

### API Calls
- **Nunca:** Direct axios en componentes
- **Siempre:** Service layer
- **Toasts:** Responsabilidad del service

### State
- **Zustand only** (no Redux)
- **Store:** Solo para compartido
- **useState:** Para estado local

---

## ⚡ Quick Commands

```bash
# Code quality
just check                          # Biome lint + format check
npx @biomejs/biome format --write . # Auto-format

# Development
just dev                            # Full stack (Docker)
cd apps/api && go run cmd/main.go  # Backend solo
cd apps/web && pnpm run dev        # Frontend solo

# Database
docker compose -f docker-compose.dev.yaml up pengi-db-dev
```

---

## 📖 Cómo Usar These Skills

### Como Agente
1. Lee la skill relevante (API, Web, Forms, etc.)
2. Localiza la sección "Paso a paso"
3. Sigue en orden
4. Usa los ejemplos como plantilla
5. Referencia el checklist

### Como Developer
1. Identifica el tipo: backend, frontend, o ambos
2. Abre la guía completa correspondiente (API o Web)
3. Saltea a la sección "Paso a paso"
4. Copy-paste ejemplos
5. Adapta nombres
6. Corre `just check`

---

## 🗂️ Archivos Removidos (Consolidados)

Las siguientes referencias individuales han sido **consolidadas** en las guías completas:

- `docs/backend/api-architecture-reference.md` → `api-backend-complete-guide.md`
- `docs/backend/api-errors-routes-responses.md` → `api-backend-complete-guide.md`
- `docs/backend/api-new-feature.md` → `api-backend-complete-guide.md`
- `docs/frontend/web-architecture-reference.md` → `web-frontend-complete-guide.md`
- `docs/frontend/web-api-services.md` → `web-frontend-complete-guide.md`
- `docs/frontend/web-state-management.md` → `web-frontend-complete-guide.md`
- `docs/frontend/web-new-feature.md` → `web-frontend-complete-guide.md`

**Documentación aún disponible (referencias especializadas):**
- `docs/backend/api-code-migration.md` — Migraciones de código específicas
- `docs/backend/permissions-system.md` — Sistema de permisos granular
- `docs/backoffice/*` — Guías backoffice
- `docs/frontend/README.md` — Resumen frontend

---

## ✨ Novedades en Consolidación

✅ **Un solo lugar** para encontrar información  
✅ **Arquitectura + How-To** combinados  
✅ **Ejemplos funcionales** listos para copy-paste  
✅ **Checklists** al final de cada skill  
✅ **Frontmatter YAML** para uso como skills formales  
✅ **Navegación clara** entre secciones

