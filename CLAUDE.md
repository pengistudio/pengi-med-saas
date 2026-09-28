# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Full stack (Docker)
```bash
just dev                          # Entire stack via Docker Compose (recommended)
```

### Backend (`apps/api` — Go)
```bash
cd apps/api
go run cmd/main.go                # Run API (port 8080)
go build ./...                    # Build
go vet ./...                      # Static analysis
```

### Frontend (`apps/web` — React)
```bash
cd apps/web
pnpm run dev                      # Dev server (port 5173)
pnpm run build                    # Production build (tsc + vite)
pnpm run lint                     # ESLint
pnpm run typecheck                # tsc (the pre-commit hook runs this)
pnpm run test:run                 # Vitest
```

### Tests
```bash
just tests-api                     # go test ./... inside the api container (stack must be up)
just tests-web                     # Vitest for apps/web
just tests-e2e                     # Playwright (apps/web/e2e), stack must be up
```

CI (`.github/workflows/ci-biome.yml`) also runs `go vet`/`go test`/`go build` and typecheck + tests for `apps/backoffice` and `packages/shared`.

### Code quality (root — applies to all TS/JS)
```bash
just check                         # Check formatting + lint (primary command)
just lint                          # Auto-format all TS/JS
```

### Setup & utilities
```bash
just setup                         # Configure git hooks (run once after cloning)
```

### Infrastructure dependencies (for local backend dev without Docker)
```bash
docker compose -f docker-compose.dev.yaml up -d db rabbitmq gotenberg sri-xml-signer
```

---

## Documentation & Skills

**Complete implementation guides** are in `docs/skills/`:
- [`api-backend-complete-guide.md`](docs/skills/api-backend-complete-guide.md) — Backend architecture + patterns + how-to
- [`web-frontend-complete-guide.md`](docs/skills/web-frontend-complete-guide.md) — Frontend architecture + patterns + how-to
- [`form-creation-standard.md`](docs/skills/form-creation-standard.md) — Standard for creating forms (Zod + Form components)

When creating or extending a feature end to end, use the `create-feature` skill
(`.claude/skills/create-feature/`): ordered steps with completion criteria,
covering permissions, plan gating, migrations, i18n, frontend and tests. The
guides above are the architecture reference it builds on.

---

## Architecture

### Monorepo structure
```
apps/
  api/              # Go backend
  web/              # React SaaS frontend
  backoffice/       # React admin panel
  landing/          # Astro landing page
  sri-xml-signer/   # Node.js SRI XML signing microservice
biome.json          # Formatter + linter for all TS/JS apps
Justfile            # Dev shortcuts
```

---

## Backend (`apps/api`)

**Module name:** `pengi-med-saas`
**Stack:** Gin + GORM + Zap + RabbitMQ + PostgreSQL

### Request/Response — the envelope pattern

Every handler **must** return `envelope.Response`, never write directly to `gin.Context`. Routes are wrapped with `envelope.Handle()`, which translates the i18n key in `Message` and serializes to JSON. Code that can't return a `Response` — middleware and handlers that stream files — writes errors with `envelope.Abort(c, resp)` (also stops the chain) or `envelope.Write(c, resp)`, never `c.JSON`/`c.AbortWithStatusJSON`, so they're translated too. Messages stay plain strings; the catalog test is the guard (`docs/adr/0004-mensajes-de-respuesta-sin-tipar.md`).

```go
// Handler definition
func (h *InvoiceHandler) GetAllInvoices(c *gin.Context) envelope.Response {
    db := tenantdb.For(c, h.db)
    var invoices []billing_models.Invoice
    if err := db.Find(&invoices).Error; err != nil {
        h.logger.Error("failed to fetch invoices", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "...", core_errors.ErrInternal)
    }
    return envelope.SuccessResponse(invoices, "billing.invoices.fetch.success")
}

// Route registration
billingGroup.GET("/invoices", rp(db, "READ_BILLING"), envelope.Handle(invoiceHandler.GetAllInvoices))
```

The second argument to `SuccessResponse`/`ErrorResponse` is always an **i18n key** (never a hardcoded string). Keys live in `apps/api/i18n/messages/messages_es.json` and `messages_en.json`.

### Handler struct pattern

```go
type InvoiceHandler struct {
    db     *gorm.DB
    logger *zap.Logger
}

func NewInvoiceHandler(db *gorm.DB, logger *zap.Logger) *InvoiceHandler {
    return &InvoiceHandler{db: db, logger: logger}
}
```

Handlers are instantiated in `apps/api/routes/` and injected with `db` + `logger.Log`.

### Multi-tenancy — CRITICAL

Tenant isolation lives in the data layer (`core/tenantdb`, see
`docs/adr/0002-aislamiento-por-tenant-en-la-capa-de-datos.md`). A GORM plugin
filters every statement by `tenant_id` and stamps `TenantID` on create — but
only on a handle bound to a tenant:

```go
db := tenantdb.For(c, h.db)          // in a handler: bound to the request's tenant
db.Find(&records)                    // filtered to that tenant
db.First(&record, c.Param("id"))     // another tenant's row → not found
```

| Context | Handle |
|---|---|
| HTTP handler | `tenantdb.For(c, h.db)` |
| Background work on one tenant | `tenantdb.ForTenant(db, tenantID)` |
| Work across all tenants (workers, migrations, backoffice) | `tenantdb.System(db)` |

`TenantMiddleware(db)` resolves the tenant from the `X-Tenant-Slug` header;
`tenantdb.TenantID(c)` returns it. `TENANTDB_MODE` (`permissive` / `warn`
default / `strict`) controls what happens to a query on a tenant table without
a bound handle — in `strict` it fails. `tenant_middleware.TenantScope` is
legacy; don't use it in new code.

Every tenant model has `TenantID uint`. When creating records, set it:

```go
item := &models.Item{TenantID: tenantdb.TenantID(c), ...}
```

### Permissions

Routes are gated per endpoint with `subscription_middleware.RequirePermission(db, "READ_X")`
(role **and** subscription plan must include the permission) or
`RequireRolePermission` (role only). Permission IDs live in
`features/permissions/data/permission-data.go` and are mirrored byte-for-byte in
`apps/web/src/lib/constants.ts` → `PERMISSIONS`. See `docs/backend/permissions-system.md`.

### Error codes

Centralized in `apps/api/core/errors/codes.go`. Format: `E-{DOMAIN}-{NNN}`. Always use existing codes or add new ones there — never pass raw strings as error codes. `envelope.Handle` translates the code too, so every code needs an i18n key (`{"key": "E-X-001", ...}`) in both message files.

### Feature structure

```
features/[domain]/
  handlers/     # One file per resource (e.g. invoice-handler.go) + *_test.go
  models/       # GORM models
  dto/          # Request/response DTOs
  services/     # Optional: logic shared across handlers
  workers/      # Optional: schedulers / consumers, started in cmd/main.go
  templates/    # Optional: default PDF templates + embed.go
  middleware/   # Feature-specific middleware (tenants, users)
```

### i18n Messages

All user-facing strings are stored in `apps/api/i18n/messages/` (flat array of `{"key", "value"}`):
- `messages_es.json` — Spanish
- `messages_en.json` — English

When creating endpoints or features:
1. Use i18n keys in `SuccessResponse`/`ErrorResponse`
2. Add new keys to **both** JSON files
3. Keys follow pattern: `{domain}.{resource}.{action}` (e.g., `billing.invoice.create.success`)

### Migrations

`apps/api/migrations/migrate.go` runs on startup:
1. GORM `AutoMigrate` for all models
2. Code migrations in `migrations/code-migrations/` (keyed by date)

To add a migration: create a file in `migrations/code-migrations/{year}/`, register it in `GlobalDBMap`.

**Never edit a code-migration file that is already committed to `origin/main`** — IDs are immutable, editing one that already ran elsewhere is a silent no-op there (drift between environments). Always create a new file with a new ID instead. This is enforced automatically by `.claude/hooks/guard-migrations.mjs` (blocks Edit/Write on any migration file already in `origin/main`); see `docs/backend/api-code-migration.md` rule #1.

### Tenant files and PDFs

Never build `storage/tenants/...` paths or call Gotenberg directly. Tenant files (P12, logo, signed XML, RIDE, custom templates) go through `core/tenantfiles` (`Store`: `Write`/`Read`/`Remove`/`Exists` by tenant + relative name). PDFs go through `core/pdfrender` (`Render(tenantID, template, data, paper)`): it uses the tenant's uploaded template if present, else the default embedded in the binary (`features/*/templates/embed.go`) — add new default templates there, not to the Dockerfile.

Electronic signature of medical documents (report, certificate, prescription): each user uploads their own P12 (`features/signatures`, password encrypted with `core/secretbox`, key `SIGNATURE_ENCRYPTION_KEY`). Signing goes through `signature_services.Signer` + `core/pdfsign` (PAdES + FirmaEC-style QR stamp passed to templates as `.Signature`); the signed PDF is stored and served as-is, never re-rendered.

### Async SRI processing (comprobantes electrónicos)

`features/billing/sri-document/` owns the whole lifecycle of facturas, notas de crédito and notas de débito: `Enqueue` (called by the `*/sri/process` handlers), `Process` (RabbitMQ consumers, one queue per document kind) and `Sweep` (re-queues stuck documents and pending authorizations). Status: `pending → processing → signed → validated (recibido) → authorized`, plus `rejected` (NO AUTORIZADO: corrected and resent with the same key on user retry) and `failed`/`connection_error` (retryable). A document's access key never changes and it is never resent while the SRI is processing it — see `docs/adr/0001-clave-de-acceso-inmutable.md`. Adding a new document type = a new `Kind` in `kinds.go`.

---

## Frontend (`apps/web`)

**Stack:** React 19 + Vite + TypeScript + TailwindCSS v4 + shadcn/ui + Zustand

### Shared packages

`packages/ui` (`@pengi/ui`) holds visual components, including the sidebar nav. `packages/shared` (`@pengi/shared`) holds the non-visual code both apps use: `createHttpService` (the envelope client) and the i18n messages (`useText`, `useMessages`, language context). Each app calls `initShared({ client: noAuthApi })` in `main.tsx`. Only code **both** apps use goes there — see its README.

### API service layer

Never use axios directly in components. Always go through a service file:

```typescript
// Pick the right axios instance:
//   api            → auth routes (no tenant header)
//   apiWithTenant  → tenant-scoped routes
//   noAuthApi      → public routes (no auth)

const billingService = createHttpService(apiWithTenant);

export const createInvoice = async (payload: CreateInvoicePayload) =>
    billingService.post<Invoice>("/billing/invoices", payload, {
        notifySuccess: true,   // service shows toast on success
        notifyError: true,     // service shows toast on error
    });
```

### Toast pattern

Toasts are owned by the **service layer** via `notifySuccess`/`notifyError` flags. Components only handle navigation and state updates after checking `res.success`:

```typescript
const res = await createInvoice(payload);
if (res.success) {
    navigate("/billing");
}
// No manual successToast/errorToast here — service handles it
```

Exception: pure UI validation toasts (e.g. "no patient selected") that are not the result of an API call stay in the component.

### Types from DB

Types live next to their service in `src/api/<domain>-service.ts`. Interfaces that mirror a backend model with `gorm.Model` extend `BaseModel` (from `@pengi/shared`); field names are the Go model's `json` tags (mostly `snake_case`):

```typescript
export interface Invoice extends BaseModel {   // BaseModel has ID, CreatedAt, UpdatedAt, DeletedAt
    tenant_id: number;
    patient_id: number;
    sequential: string;
    status: string;
    total: number;
    items: InvoiceItem[];
}
```

### i18n

```typescript
const { textGet, formatDate, formatMoney } = useText();   // from @pengi/shared (never use `t` or `useTranslation`)
const label = textGet("billing.invoice.title");
// Missing keys render as *billing.invoice.title* — no fallback needed
textGet("billing.sri.status.expires_on", { date, days });   // fills {date} and {days}
textGet("dashboard.tasks.total", { count });                // reads dashboard.tasks.total.one / .other
formatDate(invoice.CreatedAt, "long");                      // also formatDateTime, formatTime, formatRelative
formatMoney(invoice.total);                                 // USD, formatted for the language
```

`useText` is the only place that interpolates or formats: it follows the interface language (`es`→`es-EC`, `en`→`en-US`, see `CONTEXT.md`). Never write `toLocaleDateString`, `Intl.*Format`, `.replace("{x}", …)` or `$${x.toFixed(2)}` — a test in `@pengi/shared` (`formatting-guard.test.ts`) fails on them. Column definitions (memoized, no hooks) render `<Money>` / `<FormattedDate>` from `@/components/custom/formatted`.

Never hardcode user-visible strings. Every label, placeholder, and message must be an i18n key. Add new keys to **both** `apps/api/i18n/messages/messages_es.json` and `messages_en.json`.

Keys live in the backend JSON, embedded in the API binary (the message catalog, `docs/adr/0003-catalogo-de-mensajes-en-el-binario.md`); there is no messages table. The browser caches them in `localStorage["messages"]` with the catalog's ETag and revalidates on every load, so a reload is enough to see new keys. Tests fail on keys that don't exist: a Go test checks every literal key passed to `envelope.*Response` (plus error-code coverage and es/en parity), and a Vitest test in `@pengi/shared` checks literal `textGet`/`<Text uuid>` keys.

### State management

**Zustand only** — no Redux, Context, or Recoil. Stores live in `src/store/`.

Create a store only if state is **shared across multiple components**. For single-component state, use `useState`.

```typescript
interface ItemStore {
  items: Item[];
  selectedItem?: Item;
  setItems: (items: Item[]) => void;
  setSelectedItem: (item: Item | undefined) => void;
}

export const useItemStore = create<ItemStore>((set) => ({
  items: [],
  selectedItem: undefined,
  setItems: (items) => set({ items }),
  setSelectedItem: (item) => set({ selectedItem: item }),
}));
```

### Routing & permissions

Routes are defined in `src/routes/routes.tsx`: pages are `lazy()` imports, wrapped with `<CheckPermission permissions={[...]} />` (default export of `@/components/custom/check-permission`; requires all listed permissions). Permission constants are in `src/lib/constants.ts` under `PERMISSIONS`.

Navigation items are managed in `src/config/nav-config.ts`. Top-level items can be filtered by:
- `permission` — RBAC permission requirement
- `feature` — key of `EnabledFeatures` (`clinical`, `billing`, `team`, `kanban`). The backend computes it from the permission `Category`s in the tenant's plan (`features/companies/services/enabled-features-service.go`); the item hides only when the flag is `false`. It hides the nav item only — the backend `RequirePermission` is what protects data.

---

## Environment variables

Backend template: `apps/api/.env.example`. Key vars:
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `AUTH_KEY` — JWT signing secret
- `SRI_SIGNER_SERVICE_URL` — Points to `sri-xml-signer` service
- `SRI_ENV` — `test` or `prod` (Ecuador SRI environment)
- `RABBITMQ_USER`, `RABBITMQ_PASSWORD`
- `GIN_MODE` — `development` or `release`

Frontend env: Vite reads `VITE_API_URL` (defaults to `http://localhost:8000/api/v1`).

---

## Important Patterns to Remember

### 1. Query Tenant Data Through `tenantdb` (Backend)
```go
// ❌ WRONG — unbound handle, sees every tenant
h.db.Find(&items)

// ✅ CORRECT
tenantdb.For(c, h.db).Find(&items)
```

### 2. Never Call API Directly (Frontend)
```typescript
// ❌ WRONG
const res = await apiWithTenant.get("/items");

// ✅ CORRECT
const res = await itemService.getItems();  // Service layer handles it
```

### 3. All User-Facing Strings Must Be i18n Keys
```typescript
// ❌ WRONG
<h1>Items</h1>

// ✅ CORRECT
<h1>{textGet("item.title")}</h1>
```

### 4. Toasts Are Service Responsibility (Frontend)
```typescript
// ❌ WRONG
const res = await createItem(data);
if (res.success) showSuccessToast("Item created");

// ✅ CORRECT
const res = await createItem(data);  // Service shows toast automatically
if (res.success) navigate("/items");
```
