# @pengi/shared

Non-visual code shared by `apps/web` and `apps/backoffice`:

- `http`: `createHttpService(axios)`, the client that unwraps the API's
  `{ code, message, data }` envelope into a `ServiceResponse` and shows the
  success/error toasts.
- `i18n`: UI messages (loaded from the API, cached in localStorage and
  revalidated on each load by content hash via `useMessages()`), `useText`,
  the language context and zod's locale.

Call `initShared({ client: noAuthApi })` once in each app's `main.tsx`.

## What goes here

Only code that **both** apps use **and** that is not a visual component.
Visual components (including the sidebar nav) go to `@pengi/ui`. The one
exception is `SelectLanguage`: it is the control of the language state that
lives here, and `@pengi/ui` cannot import `@pengi/shared` (shared depends on
ui). A ui component that needs text or the current language reads it from
`useUiText()`, which `AppTextBridge` fills. Code only
one app uses stays in that app, even if the other might need it "someday".
