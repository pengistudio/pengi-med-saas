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
```

### Code quality (root — applies to all TS/JS)
```bash
npx @biomejs/biome ci .           # Check formatting + lint
npx @biomejs/biome format --write . # Auto-format
```

### Infrastructure dependencies (for local backend dev without Docker)
```bash
docker compose -f docker-compose.dev.yaml up pengi-db-dev pengi-rabbitmq-dev gotenberg sri-xml-signer -d
```

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

Every handler **must** return `envelope.Response`, never write directly to `gin.Context`. Routes are wrapped with `envelope.Handle()`, which translates the i18n key in `Message` and serializes to JSON.

```go
// Handler definition
func (h *InvoiceHandler) GetAllInvoices(c *gin.Context) envelope.Response {
    tenantScope := tenant_middleware.TenantScope(c)
    var invoices []billing_models.Invoice
    if err := h.db.Scopes(tenantScope).Find(&invoices).Error; err != nil {
        h.logger.Error("failed to fetch invoices", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "...", core_errors.ErrInternal)
    }
    return envelope.SuccessResponse(invoices, "billing.invoices.fetch.success")
}

// Route registration
billingGroup.GET("/invoices", envelope.Handle(invoiceHandler.GetAllInvoices))
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

### Multi-tenancy

Every DB query that touches tenant data **must** apply the GORM scope:

```go
tenantScope := tenant_middleware.TenantScope(c)
h.db.Scopes(tenantScope).Find(&records)
```

The scope is populated by `TenantMiddleware(db)`, which reads the `X-Tenant-Slug` header.

### Error codes

Centralized in `apps/api/core/errors/codes.go`. Format: `E-{DOMAIN}-{N}`. Always use existing codes or add new ones there — never pass raw strings as error codes.

### Feature structure

```
features/[domain]/
  handlers/     # One file per resource (e.g. invoice-handler.go)
  models/       # GORM models
  dto/          # Request/response DTOs
  workers/      # Background consumers (billing only)
  middleware/   # Feature-specific middleware (tenants, users)
```

### Migrations

`apps/api/migrations/migrate.go` runs on startup:
1. GORM `AutoMigrate` for all models
2. Code migrations in `migrations/code-migrations/` (keyed by date)
3. Seeds i18n messages from JSON files into DB

To add a migration: create a file in `migrations/code-migrations/{year}/`, register it in `GlobalDBMap`.

### Async invoice processing

`POST /billing/invoices/:id/sri/process` publishes a message to RabbitMQ queue `invoice_tasks`. The worker in `features/billing/workers/invoice-worker.go` consumes it, calls the `sri-xml-signer` microservice, and updates the invoice status through these stages: `pending → processing → signed → validated → authorized`.

---

## Frontend (`apps/web`)

**Stack:** React 19 + Vite + TypeScript + TailwindCSS v4 + shadcn/ui + Zustand

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

All interfaces that mirror backend models extend `BaseModel` and use `snake_case` keys:

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
const { textGet } = useText();   // hook from @/hooks/use-text
const label = textGet("billing.invoice.title");
// Missing keys render as *billing.invoice.title* — no fallback needed
```

Never hardcode user-visible strings. Every label, placeholder, and message must be an i18n key. Add new keys to both `apps/api/i18n/messages/messages_es.json` and `messages_en.json`.

### State management

Zustand only — no Redux. Stores live in `src/store/`. Use `sessionStorage` persistence for page-level data (e.g. `billing-store.ts`) and `localStorage` for session data (e.g. `token-store.ts`).

### Routing & permissions

Routes are defined in `src/routes/routes.tsx` and wrapped with `<CheckPermission permissions={[...]} />`. Permission constants are in `src/lib/constants.ts` under `PERMISSIONS`.

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
