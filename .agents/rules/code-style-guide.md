---
trigger: always_on
---

# Pengi Med SaaS — Code Style Guide

Este documento define los estándares arquitectónicos y de estilo del proyecto, diseñado para optimizar el uso de contexto por parte de LLMs.

---

##  Monorepo y Formateo
- **Estructura:** `apps/api` (Go), `apps/web` (React), `apps/backoffice` (React), `apps/landing` (Astro).
- **Biome:** Usamos `biome.json`. **Tabs**, **dobles comillas**, importaciones automáticas. No usar ESLint/Prettier.
- Ignorar reglas justificadas solo con `// biome-ignore lint/rule-name: reason`.

---

##  Backend (Go / Gin / GORM)

### Arquitectura por Features
El código vive en `features/[dominio]/` (ej. `clinical`, `users`) que contiene:
- `handlers/`: Lógica de controladores.
- `models/`: Modelos de GORM.
- `dto/`: Data Transfer Objects.
Las rutas se agrupan en `routes/` (un archivo por feature) y se registran en `index.go`.

### Convenciones
- **Packages:** `snake_case` con prefijo del dominio (ej. `clinical_handlers`, `core_errors`).
- **Imports:** Usa alias para evitar conflictos (ej. `clinical_dto "pengi-med-saas/features/clinical/dto"`).

### Handlers & Responses
- Usa handlers basados en **structs** con inyección de dependencias (`*gorm.DB`, `*zap.Logger`). Constructor: `NewXHandler`.
- **SIEMPRE** retorna `envelope.Response`. Los métodos no son `gin.HandlerFunc` directamente.
- Envuelve las rutas con `envelope.Handle(handler.Metodo)`.
- Éxito: `envelope.SuccessResponse(data, "i18n.key")`.
- Error: `envelope.ErrorResponse(code, err.Error(), core_errors.ErrCode)`.

### Base de Datos (GORM)
- **Modelos:** Heredan de `gorm.Model`. Los tags JSON van en `snake_case`.
- **Multi-tenant:** Todo modelo tenant-aware debe tener `TenantID uint`. Usa `tenant_middleware.TenantScope(c)` en TODAS las consultas a BD.

### DTOs
- `Create[X]DTO`: Valores `binding:"required"`. Campos opcionales como punteros.
- `Update[X]DTO`: **TODO** como puntero (para actualizaciones parciales).

### Logs & Errores
- **No uses `fmt.Println`**. Usa `logger.Log` estructurado (`zap`): `zap.Error(err)`, `zap.Uint("id", x)`.
- Define errores en `core/errors/codes.go` como variables `AppError`.

---

## Frontend (React / Vite)

### Stack y Estructura
- **Stack:** Vite, TypeScript, TailwindCSS v4, shadcn/ui.
- **Rutas Relativas:** Usa siempre el alias `@/` que apunta a `src/`.
- Archivos en `kebab-case.ts(x)`, componentes React en `PascalCase`.

### API Layer (`src/api/`)
- Crea llamadas a API envolviendo axios con `createHttpService(instanciaAxios)`.
- Elige la instancia correcta según el contexto y feature: `api` (autenticado general), `apiWithTenant` (operaciones de dominio que requieren scope de tenant), o `noAuthApi` (pública).
- Retorno estricto: `Promise<ServiceResponse<T>>`.
- Usa las opciones `{ notifySuccess: true, notifyError: true }` para disparar notificaciones toast (Sonner) automáticamente sin ensuciar los componentes.

### Manejo de Estado (Zustand)
- Archivos `[nombre]-store.ts`.  No usar Redux ni Context API (excepto para inyecciones que cambian en runtime como Auth).
- Usar middleware `persist` con `createJSONStorage` para persistencia (localStorage/sessionStorage).

### Componentes y Rutas
- Rutas centralizadas en `routes/routes.tsx` con React Router v7 (`createBrowserRouter`).
- Protege rutas con wrappers `<CheckAuth>` y `<CheckPermission permissions={[PERMISSIONS.X.Y]} />`.
- Usa la librería `shadcn/ui` en `components/ui/` (no la edites a menos que sea necesario). Usa `components/forms/` para formularios.
- Clases CSS condicionales usan el wrapper `cn()` (`@/lib/utils`).

### Tipos (TypeScript)
- `type` para Payloads/DTOs.
- `interface` extendiendo de `BaseModel` para las respuestas que mapean modelos de base de datos.
- Las propiedades JSON que vienen del back (y van hacia él) **DEBEN estar en `snake_case`** (ej. `first_name`).

### Internalización (i18n)
- **Nunca quemes strings en componentes.** Los textos vienen del back.
- Usa `const { t } = useText();` y claves con dot notation: `t("clinical.patient.create.success")`.
- **SIEMPRE que crees un texto con dot notation**, debes agregarlo a los diccionarios del backend en `apps/api/i18n/messages/`. Debes agregar la clave y su respectiva traducción en ambos archivos: `messages_es.json` y `messages_en.json`.
