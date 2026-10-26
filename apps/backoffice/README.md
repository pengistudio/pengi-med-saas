# Backoffice

Admin panel for Pengi Med (tenants, plans, users). React + Vite + TypeScript,
sharing `@pengi/ui` and `@pengi/shared` with `apps/web`.

Run it with the rest of the stack (`just dev`, port 3001). JS tooling runs inside
the dev containers, never on the host — see the root `CLAUDE.md`.

```bash
pnpm run dev        # Vite dev server
pnpm run build      # tsc + vite build
pnpm run typecheck
pnpm run test:run   # Vitest
```

Formatting and lint are Biome, from the repo root: `just check`.
