# Verificación

CI (`.github/workflows/ci-biome.yml`) es la barra; el pre-commit
(`.githooks/pre-commit`, activo solo tras `just setup`) cubre solo Biome +
typecheck de web y backoffice.

## Backend

```bash
cd apps/api && go build ./... && go vet ./...
just tests-api            # go test ./... dentro del contenedor api (stack arriba)
```

Tests de handler: `*_test.go` al lado del handler, patrón de
`features/billing/handlers/invoice_handler_test.go`:

- `db := testutils.SetupTestDB(t, &<domain>_models.<X>{}, ...)` — Postgres del
  contenedor si está, si no sqlite en memoria (CI); registra el plugin
  `tenantdb` y migra solo los modelos pasados.
- `c, w := testutils.NewGinContext(tenantID, userID)` e invoca el handler
  directo; aserta sobre el `envelope.Response` devuelto.
- `NewGinContext` pone `"userId"`, mientras auth y `tenantdb.For` leen
  `"user_id"`: si el test depende del usuario (auditoría, creador), haz
  `c.Set("user_id", userID)`.
- Aislamiento: crea datos en dos tenants y verifica que el handler de uno no ve
  ni modifica los del otro — patrón `features/clinical/handlers/tenant_isolation_test.go`.
- `tenantfiles.Memory()` para handlers que leen o escriben archivos.

## Frontend

Por cada paquete que tocaste:

```bash
just check                                   # Biome (web, backoffice, ui)
pnpm --filter web typecheck && pnpm --filter web test:run
pnpm --filter backoffice typecheck && pnpm --filter backoffice test:run
pnpm --filter @pengi/shared typecheck && pnpm --filter @pengi/shared test:run
```

`just check` no incluye `packages/shared`, pero CI sí: corre
`npx @biomejs/biome check packages/shared` si lo tocaste. Tests con Vitest +
Testing Library; e2e con Playwright en `apps/web/e2e/` (`just tests-e2e`,
stack arriba) para flujos críticos.

## Navegador

`/doctor` (skill `react-doctor`, en `apps/web` y `apps/backoffice`) antes de
cerrar el trabajo frontend.
