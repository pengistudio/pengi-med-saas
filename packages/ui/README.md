# @pengi/ui

Shared React component library for `apps/web` and `apps/backoffice`. Consumed as
source (no build step) via the pnpm workspace - both apps import it directly as
`@pengi/ui`.

## Text lookup

Components that need translated strings call `useUiText()` (from
`src/context/text-context.tsx`) instead of depending on either app's
`useMessageStore` directly. Each app wraps its root in a small
`AppTextBridge` (`apps/*/src/components/app-text-bridge.tsx`) that reads its
own `useText()` and feeds it into `UiTextProvider`.

## Non-goals

The following were evaluated for extraction and deliberately left duplicated
in both apps because they pull in app-specific state beyond text lookup, or
diverge in actual behavior rather than just implementation detail:

- **`select-language.tsx`, `nav-item.tsx`** - depend on
  `language-context`/`message-store` and `sidebar-store` respectively, both
  app-local zustand stores. Small, low-duplication-cost components; not worth
  a new context bridge each.
- **`form.tsx`** - the two apps summarize validation errors differently (web
  shows a count; backoffice lists every invalid field), a real UX difference,
  not a superset.
- **`form-calendar.tsx`** - web supports locale switching (es/en) driven by
  `message-store`; backoffice doesn't have this at all.
- **`use-auth.tsx`** - `token-store` persists to `localStorage` in web vs
  `sessionStorage` in backoffice, the two apps have separate user types, and
  backoffice's `auth-service` exposes far fewer endpoints than web's
  (no register/verify-email/company-signup/reset-password). Both already use
  the shared `useToast`.
- **`data-table.tsx`** - server-side pagination, loading skeleton,
  `rowClassName`/`emptyState` props, and `useRowStore` sync in web vs a
  simpler version in backoffice. Only the reusable subcomponents
  (`data-table-view-options`, `data-table-column-header`,
  `data-table-pagination`) were extracted.

Don't move these into `@pengi/ui` without re-checking whether the underlying
app-specific dependency has since been unified - otherwise you'll need a new
context bridge (as with `useUiText`) or you'll silently change behavior for
one of the two apps.
