# Dashboard Redesign — Design Spec
_2026-05-17_

## Context

The dashboard currently has 5 stat cards in a single row (4 clinical + 1 subscription), followed by a weekly chart and upcoming appointments panel. Below that is empty whitespace. With the addition of the payment and subscription system (dLocal integration, `/subscription` page, `subscription-service.ts`), the dashboard needs to surface subscription status and billing activity in a more intentional way — removing the awkward 5th stat card and filling the empty lower section.

## Design

### Layout: 3 rows

**Row 1 — Clinical KPIs (unchanged content, improved alignment)**
4 equal-width stat cards with consistent structure: label + icon in header, large value below.
- Total de Pacientes → `Users` icon, teal bg
- Pacientes Críticos → `AlertTriangle` icon, red bg, left border accent, red tint background
- Citas de Hoy → `Calendar` icon, blue bg
- Completadas Este Mes → `CheckCircle` icon, green bg

The subscription card is **removed from this row**.

**Row 2 — Activity (unchanged)**
- Left (3fr): Citas de la Semana bar chart
- Right (2fr): Próximas Citas panel with empty state + "Agendar cita" button

**Row 3 — Business (NEW, fills whitespace)**
- Left (1fr): **Subscription card** — plan name, expiry date, days remaining, active/expired badge, "Ver detalles" link to `/subscription`. Border-top teal accent. Conditional amber/red border when expiring ≤30 days or ≤7 days (logic already exists in current card).
- Right (2fr): **Facturas Recientes** — last 5 invoices (no status filter), columns: patient name / amount / status badge. "Ver todas" link to `/billing`. Uses existing `GET /billing/invoices` endpoint via the billing service layer. Slice the first 5 from the response (sorted by `created_at` desc).

### Visual rules
- All cards: `background: white`, `border: 1px solid #e2e8f0`, `border-radius: 12px`, consistent `padding: 16px 18px`
- Icons: lucide-react, 16×16, `stroke-width: 2`
- Stat value font: 28px, weight 800
- No hardcoded strings — all labels via `textGet()`

## Files to modify

| File | Change |
|------|--------|
| `apps/web/src/pages/home/home.tsx` | Full layout restructure — add row 3, remove subscription from row 1, add invoice fetch |
| `apps/api/i18n/messages/messages_es.json` | Add keys for new widget labels |
| `apps/api/i18n/messages/messages_en.json` | Same keys in English |

## Data sources

| Widget | Source | Notes |
|--------|--------|-------|
| Clinical stats | Existing `GET /home/stats` | No change |
| Subscription card | `getMySubscription()` from `subscription-service.ts` | Already used in current dashboard |
| Facturas Recientes | `GET /billing/invoices` via existing billing service | Fetch, slice first 5 sorted by created_at desc |

## i18n keys to add

```
home.recent_invoices.title
home.recent_invoices.view_all
home.subscription.view_details
```

Existing keys already cover subscription status labels.

## Verification

1. Run `just dev` — open dashboard at `http://localhost:5173`
2. Row 1: all 4 cards same height, icon + label aligned in header row, value below
3. Row 3 left: subscription card shows plan, days remaining, badge, link
4. Row 3 right: shows last 4–5 invoices with patient name, amount, status badge
5. Empty whitespace below the appointments panel is gone
6. "Ver todas" navigates to `/billing`
7. "Ver detalles" navigates to `/subscription`
8. Subscription expiry warning colors still work (≤30 days amber, ≤7 days red)
9. No hardcoded strings — `just check` passes lint
