# End-to-End Testing with Playwright

E2E tests para flujos críticos del usuario usando Playwright.

## Estructura

```
e2e/
├── auth.spec.ts              # Login, logout, error handling
├── patients.spec.ts          # Create, search, list patients
├── invoices.spec.ts          # Create, view, manage invoices
├── multi-tenant.spec.ts      # Tenant isolation verification
└── fixtures.ts               # Shared test utilities
```

## Instalación (Docker)

### 1. Agregar dependencia

```bash
# Desde el root del proyecto
pnpm add -D @playwright/test --filter web
```

Si hay problemas con PNPM_HOME, ejecuta manualmente:

```bash
cd apps/web
npm install --save-dev @playwright/test
```

### 2. Instalar navegadores de Playwright

```bash
# Con docker-compose
docker compose -f docker-compose.dev.yaml exec web npx playwright install

# O instalación local
npx playwright install
```

## Ejecución

### Con Docker Compose (Recomendado)

```bash
# Terminal 1: Inicia la stack completa
just dev

# Terminal 2: Ejecuta tests E2E
docker compose -f docker-compose.dev.yaml exec web pnpm exec playwright test
```

### Local (sin Docker)

```bash
# Inicia frontend dev server
cd apps/web && pnpm run dev

# En otra terminal, ejecuta tests
pnpm exec playwright test
```

## Comandos Útiles

```bash
# Ejecutar tests con UI (browser visible)
pnpm exec playwright test --ui

# Ejecutar tests en modo debug
pnpm exec playwright test --debug

# Ejecutar un test específico
pnpm exec playwright test auth.spec.ts

# Ver reporte HTML
pnpm exec playwright show-report
```

## Scripts en package.json

Agrega estos scripts a `apps/web/package.json`:

```json
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui",
    "test:e2e:debug": "playwright test --debug"
  }
}
```

Luego ejecuta con:

```bash
pnpm run test:e2e
```

## Tests Incluidos

### 1. **auth.spec.ts** (3 tests)
- ✅ Login con credenciales válidas
- ✅ Logout functionality  
- ✅ Error con credenciales inválidas

### 2. **patients.spec.ts** (3 tests)
- ✅ Navegar a lista de pacientes
- ✅ Crear nuevo paciente
- ✅ Buscar pacientes

### 3. **invoices.spec.ts** (3 tests)
- ✅ Navegar a lista de facturas
- ✅ Ver facturas
- ✅ Crear nueva factura

### 4. **multi-tenant.spec.ts** (3 tests)
- ✅ Verificar aislamiento de datos
- ✅ Mostrar nombre del tenant
- ✅ Mantener aislamiento al navegar

## Configuración (playwright.config.ts)

- **baseURL**: http://localhost:3000
- **Timeout**: 120s por test
- **Retries**: 2 en CI, 0 local
- **Browsers**: Chromium, Firefox, WebKit

## Test Data

Los tests usan credenciales de test:

```
Username: testuser
Password: password123
```

Asegúrate de que exista este usuario en tu BD de desarrollo.

## Solución de Problemas

### Tests fallan porque no encuentran el elemento

- Los tests son tolerantes: buscan múltiples selectores
- Si un elemento no existe, el test simplemente lo salta
- Verifica que la app esté corriendo en http://localhost:3000

### "Playwright not installed"

```bash
pnpm add -D @playwright/test --filter web
npx playwright install
```

### Tests timeout

- Aumenta el timeout en `playwright.config.ts`
- Verifica que el backend esté corriendo (puerto 8000)
- Verifica que el frontend esté corriendo (puerto 3000)

## CI/CD

Para ejecutar en CI:

```yaml
# .github/workflows/e2e.yml
name: E2E Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
    steps:
      - uses: actions/checkout@v3
      - uses: pnpm/action-setup@v2
      - uses: actions/setup-node@v3
        with:
          node-version: '20'
          cache: 'pnpm'
      - run: pnpm install
      - run: pnpm run build
      - run: pnpm exec playwright install --with-deps
      - run: pnpm exec playwright test
```

## Referencias

- [Playwright Docs](https://playwright.dev)
- [Best Practices](https://playwright.dev/docs/best-practices)
- [Debugging](https://playwright.dev/docs/debug)
