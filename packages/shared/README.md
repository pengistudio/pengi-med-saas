# @pengi/shared

Non-visual code shared by `apps/web` and `apps/backoffice`:

- `http`: `createHttpService(axios)`, the client that unwraps the API's
  `{ code, message, data }` envelope into a `ServiceResponse` and shows the
  success/error toasts.
- `i18n`: UI messages (loaded from the API, cached in localStorage and
  refreshed on every new build via `useMessages(__APP_VERSION__)`), `useText`,
  the language context and zod's locale.

Call `initShared({ client: noAuthApi })` once in each app's `main.tsx`.

## What goes here

Only code that **both** apps use **and** that is not a visual component.
Visual components (including the sidebar nav) go to `@pengi/ui`. Code only
one app uses stays in that app, even if the other might need it "someday".
