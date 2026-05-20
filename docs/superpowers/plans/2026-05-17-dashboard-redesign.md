# Dashboard Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restructure the home dashboard into 3 rows — 4 clinical stat cards, chart/appointments, and a new bottom row with subscription card + recent invoices widget — eliminating the empty whitespace.

**Architecture:** All changes are contained in `apps/web/src/pages/home/home.tsx`. The subscription card moves from the first stat grid to a dedicated bottom row. A new `RecentInvoicesCard` component is added inline, fetching from the existing `getAllInvoices` billing service. Three new i18n keys are added to both JSON files.

**Tech Stack:** React 19, lucide-react, shadcn/ui (Card, Badge, Button), `getAllInvoices` from `billing-service.ts`, `Invoice`/`Patient` types from existing services, `useText` hook.

---

## File Map

| File | Change |
|------|--------|
| `apps/web/src/pages/home/home.tsx` | Restructure layout: fix stat grid to 4 cols, add invoice fetch + state, add `RecentInvoicesCard` component, add row 3 |
| `apps/api/i18n/messages/messages_es.json` | Add 3 new i18n keys |
| `apps/api/i18n/messages/messages_en.json` | Add 3 new i18n keys |

---

## Task 1: Add i18n keys

**Files:**
- Modify: `apps/api/i18n/messages/messages_es.json`
- Modify: `apps/api/i18n/messages/messages_en.json`

- [ ] **Step 1: Add keys to messages_es.json**

Find the block ending with `"dashboard.subscription.pay_now"` (around line 1459) and add immediately after its closing `},`:

```json
  {
    "key": "dashboard.subscription.view_details",
    "value": "Ver detalles"
  },
  {
    "key": "dashboard.recent_invoices.title",
    "value": "Facturas Recientes"
  },
  {
    "key": "dashboard.recent_invoices.view_all",
    "value": "Ver todas"
  },
```

- [ ] **Step 2: Add keys to messages_en.json**

Find the same block in `messages_en.json` (search for `dashboard.subscription.pay_now`) and add after its closing `},`:

```json
  {
    "key": "dashboard.subscription.view_details",
    "value": "View details"
  },
  {
    "key": "dashboard.recent_invoices.title",
    "value": "Recent Invoices"
  },
  {
    "key": "dashboard.recent_invoices.view_all",
    "value": "View all"
  },
```

- [ ] **Step 3: Commit**

```bash
git add apps/api/i18n/messages/messages_es.json apps/api/i18n/messages/messages_en.json
git commit -m "feat(i18n): add dashboard recent invoices and subscription detail keys"
```

---

## Task 2: Restructure home.tsx — stat grid + invoice fetch

**Files:**
- Modify: `apps/web/src/pages/home/home.tsx`

The current file imports `initiatePayment` from `subscription-service` and `getAllInvoices` is NOT yet imported. The stat grid is dynamic (4 or 5 cols depending on `stats.subscription`). Fix it to always 4 cols and add invoice state.

- [ ] **Step 1: Add `getAllInvoices` import**

Current import block at top of file:
```typescript
import {
	type DashboardStats,
	getDashboardStats,
	type SubscriptionInfo,
} from "@/api/clinical-service";
import { initiatePayment } from "@/api/subscription-service";
```

Replace with:
```typescript
import {
	type DashboardStats,
	getDashboardStats,
	type SubscriptionInfo,
} from "@/api/clinical-service";
import { getAllInvoices, type Invoice } from "@/api/billing-service";
import { initiatePayment } from "@/api/subscription-service";
```

- [ ] **Step 2: Add invoice state and fetch to the `Home` component**

Current state in `Home`:
```typescript
const [stats, setStats] = React.useState<DashboardStats | null>(null);
```

Replace with:
```typescript
const [stats, setStats] = React.useState<DashboardStats | null>(null);
const [recentInvoices, setRecentInvoices] = React.useState<Invoice[]>([]);
```

Current useEffect:
```typescript
React.useEffect(() => {
    getDashboardStats().then((res) => {
        if (res.success && res.data) {
            setStats(res.data as DashboardStats);
        }
    });
}, []);
```

Replace with:
```typescript
React.useEffect(() => {
    getDashboardStats().then((res) => {
        if (res.success && res.data) {
            setStats(res.data as DashboardStats);
        }
    });
    getAllInvoices({ limit: 5 }).then((res) => {
        if (res.success && res.data) {
            setRecentInvoices(res.data.items.slice(0, 5));
        }
    });
}, []);
```

- [ ] **Step 3: Fix stat grid to always 4 columns**

Current stat grid opening:
```typescript
<div
    className={cn(
        "grid gap-4 sm:grid-cols-2 md:grid-cols-3",
        stats.subscription ? "xl:grid-cols-5" : "xl:grid-cols-4",
    )}
>
```

Replace with:
```typescript
<div className="grid gap-4 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4">
```

- [ ] **Step 4: Remove subscription card from stat grid**

Delete this block from inside the stat grid (after the 4 StatCards):
```typescript
{stats.subscription && (
    <SubscriptionCard
        subscription={stats.subscription}
        textGet={textGet}
    />
)}
```

- [ ] **Step 5: Verify lint passes**

```bash
cd apps/web && pnpm run lint
```

Expected: no errors. If unused import warnings appear for `cn`, keep it — it's used elsewhere in the file.

- [ ] **Step 6: Commit**

```bash
git add apps/web/src/pages/home/home.tsx
git commit -m "feat(dashboard): add invoice fetch, fix stat grid to 4 cols, remove sub from row 1"
```

---

## Task 3: Add RecentInvoicesCard component + bottom row

**Files:**
- Modify: `apps/web/src/pages/home/home.tsx`

Add the `RecentInvoicesCard` component and wire the bottom row into the JSX. Add this component definition between `SubscriptionCard` and the `Home` function.

- [ ] **Step 1: Add invoice status badge helper**

After the closing `}` of `SubscriptionCard` (around line 206), insert:

```typescript
// ─── Invoice Status Badge ────────────────────────────────────────────────────

const INVOICE_STATUS_STYLES: Record<
    string,
    { label: string; className: string }
> = {
    authorized: {
        label: "Autorizada",
        className: "bg-emerald-500/10 text-emerald-600",
    },
    validated: {
        label: "Validada",
        className: "bg-emerald-500/10 text-emerald-600",
    },
    pending: {
        label: "Pendiente",
        className: "bg-amber-500/10 text-amber-600",
    },
    processing: {
        label: "Procesando",
        className: "bg-blue-500/10 text-blue-600",
    },
    signed: {
        label: "Firmada",
        className: "bg-blue-500/10 text-blue-600",
    },
};

function InvoiceStatusBadge({ status }: { status: string }) {
    const style = INVOICE_STATUS_STYLES[status] ?? {
        label: status,
        className: "bg-muted text-muted-foreground",
    };
    return (
        <span
            className={cn(
                "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                style.className,
            )}
        >
            {style.label}
        </span>
    );
}
```

- [ ] **Step 2: Add RecentInvoicesCard component**

Insert after `InvoiceStatusBadge`:

```typescript
// ─── Recent Invoices Card ────────────────────────────────────────────────────

function RecentInvoicesCard({
    invoices,
    textGet,
    onViewAll,
}: {
    invoices: Invoice[];
    textGet: (k: string) => string;
    onViewAll: () => void;
}) {
    return (
        <Card>
            <CardHeader className="flex flex-row items-center justify-between pb-3">
                <CardTitle className="text-sm font-semibold">
                    {textGet("dashboard.recent_invoices.title")}
                </CardTitle>
                <button
                    type="button"
                    onClick={onViewAll}
                    className="text-xs font-medium text-primary hover:underline flex items-center gap-1"
                >
                    {textGet("dashboard.recent_invoices.view_all")}
                    <ArrowRight className="h-3 w-3" />
                </button>
            </CardHeader>
            <CardContent className="pt-0">
                {invoices.length === 0 ? (
                    <p className="text-sm text-muted-foreground py-4 text-center">
                        {textGet("dashboard.recent_invoices.empty")}
                    </p>
                ) : (
                    <div className="divide-y">
                        {invoices.map((invoice) => {
                            const patientName = invoice.patient
                                ? `${invoice.patient.first_name} ${invoice.patient.last_name}`
                                : `#${invoice.sequential}`;
                            return (
                                <div
                                    key={invoice.ID}
                                    className="flex items-center justify-between gap-3 py-2.5"
                                >
                                    <span className="text-sm text-foreground truncate flex-1">
                                        {patientName}
                                    </span>
                                    <span className="text-sm font-semibold tabular-nums">
                                        ${invoice.total.toFixed(2)}
                                    </span>
                                    <InvoiceStatusBadge status={invoice.status} />
                                </div>
                            );
                        })}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
```

- [ ] **Step 3: Add `ArrowRight` to lucide imports**

Current lucide import:
```typescript
import {
    AlertTriangle,
    Calendar,
    CalendarPlus,
    CheckCircle,
    Clock,
    CreditCard,
    Users,
} from "lucide-react";
```

Replace with:
```typescript
import {
    AlertTriangle,
    ArrowRight,
    Calendar,
    CalendarPlus,
    CheckCircle,
    Clock,
    CreditCard,
    Users,
} from "lucide-react";
```

- [ ] **Step 4: Update SubscriptionCard to add "Ver detalles" link**

In `SubscriptionCard`, after the existing `{isExpiringSoon && (<Button ...>)}` block (before the closing `</CardContent>`), add:

```typescript
<button
    type="button"
    onClick={() => navigate("/subscription")}
    className="text-xs font-medium text-primary hover:underline flex items-center gap-1 mt-1"
>
    {textGet("dashboard.subscription.view_details")}
    <ArrowRight className="h-3 w-3" />
</button>
```

Also add `navigate` to the `SubscriptionCard` props and signature:

Current `SubscriptionCard` props type:
```typescript
function SubscriptionCard({
    subscription,
    textGet,
}: {
    subscription: SubscriptionInfo;
    textGet: (k: string) => string;
}) {
```

Replace with:
```typescript
function SubscriptionCard({
    subscription,
    textGet,
    navigate,
}: {
    subscription: SubscriptionInfo;
    textGet: (k: string) => string;
    navigate: (path: string) => void;
}) {
```

- [ ] **Step 5: Wire bottom row into the Home JSX**

After the closing `</div>` of the charts row (end of `{/* Charts Row */}` block), add:

```typescript
{/* Bottom Row — Subscription + Recent Invoices */}
{stats.subscription && (
    <div className="grid gap-4 xl:grid-cols-3">
        <SubscriptionCard
            subscription={stats.subscription}
            textGet={textGet}
            navigate={navigate}
        />
        <div className="xl:col-span-2">
            <RecentInvoicesCard
                invoices={recentInvoices}
                textGet={textGet}
                onViewAll={() => navigate("/billing/invoices")}
            />
        </div>
    </div>
)}
```

- [ ] **Step 6: Add empty-state i18n key for invoices (ES + EN)**

In `messages_es.json`, add after the `dashboard.recent_invoices.view_all` entry:
```json
  {
    "key": "dashboard.recent_invoices.empty",
    "value": "No hay facturas recientes"
  },
```

In `messages_en.json`:
```json
  {
    "key": "dashboard.recent_invoices.empty",
    "value": "No recent invoices"
  },
```

- [ ] **Step 7: Verify lint**

```bash
cd apps/web && pnpm run lint
```

Expected: no errors.

- [ ] **Step 8: Commit**

```bash
git add apps/web/src/pages/home/home.tsx \
        apps/api/i18n/messages/messages_es.json \
        apps/api/i18n/messages/messages_en.json
git commit -m "feat(dashboard): add RecentInvoicesCard and subscription row below chart"
```

---

## Task 4: Visual QA

- [ ] **Step 1: Start the stack**

```bash
just dev
```

Open `http://localhost:5173` and log in to a tenant with an active subscription.

- [ ] **Step 2: Check row 1**

All 4 stat cards must:
- Have equal height
- Show label (uppercase, small) + icon in a single header row, right-aligned icon
- Show large value below
- Critical Patients card: red left border + red tinted background when value > 0

- [ ] **Step 3: Check row 2 (unchanged)**

Weekly chart and Próximas Citas panel look identical to before.

- [ ] **Step 4: Check row 3**

- Subscription card visible with plan name, expiry, days remaining, active badge
- "Ver detalles" link navigates to `/subscription`
- If subscription expires in ≤30 days: amber border + "Pagar ahora" button appears
- Facturas Recientes shows last 5 invoices: patient name / amount / status badge
- "Ver todas" navigates to `/billing/invoices`
- If no invoices: shows `dashboard.recent_invoices.empty` text

- [ ] **Step 5: Check no subscription case**

Log in as a tenant without subscription. Row 3 must not render. Rows 1 and 2 unchanged.

- [ ] **Step 6: Run lint**

```bash
just check
```

Expected: passes with no errors.
