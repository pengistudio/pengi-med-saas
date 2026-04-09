# Web Frontend — Architecture Reference

Guía de arquitectura y estructura del frontend SaaS en `apps/web`.

## Stack Técnico

- **Runtime**: Node.js + npm/pnpm
- **Framework**: React 19 + TypeScript
- **Build Tool**: Vite
- **Styling**: TailwindCSS v4 + CSS Modules
- **Components**: shadcn/ui (Radix UI + Tailwind)
- **State Management**: Zustand
- **Routing**: React Router
- **HTTP Client**: Axios
- **i18n**: Custom hook `useText()` (traducciones desde backend)

## Estructura de Carpetas

```
apps/web/src/
├── App.tsx                 # Componente raíz
├── main.tsx               # Entrypoint
├── index.css              # Estilos globales
├── api/                   # Servicios de API
│   ├── index.ts          # Axios instances (api, apiWithTenant, noAuthApi)
│   ├── fetch.ts          # HttpService class
│   ├── auth-service.ts
│   ├── clinical-service.ts
│   ├── billing-service.ts
│   ├── team-service.ts
│   └── [domain]-service.ts
├── assets/                # Imágenes, fonts, etc
├── components/            # Componentes reutilizables
│   ├── ui/               # shadcn/ui components
│   ├── custom/           # Componentes custom
│   ├── forms/
│   │   ├── form.tsx
│   │   ├── form-input.tsx
│   │   └── [form-components].tsx
│   └── access-control/
│       └── check-permission.tsx
├── config/               # Configuración
│   └── nav-config.ts    # Items navegación (con features)
├── contexts/             # React contexts (si es necesario)
├── hooks/                # Custom hooks
│   ├── use-auth.tsx
│   ├── use-text.tsx     # i18n
│   ├── use-permission.tsx
│   └── [domain]-hooks.tsx
├── lib/                  # Utilidades
│   ├── constants.ts     # PERMISSIONS, etc
│   └── utils.ts         # cn(), helpers
├── pages/                # Componentes de páginas
│   ├── [domain]/
│   │   ├── page.tsx
│   │   └── components/
│   ├── auth/
│   ├── clinical/
│   ├── billing/
│   └── settings/
├── routes/               # Configuración de routing
│   └── routes.tsx       # Router setup + CheckPermission
├── sections/             # Secciones/layouts reutilizables
│   ├── template/        # Layouts principales
│   │   └── dashboard-template.tsx
│   └── [domain]/
├── store/                # Zustand stores
│   ├── session-store.ts
│   ├── billing-store.ts
│   ├── clinical-store.ts
│   └── sidebar-store.ts
├── types/                # TypeScript types
│   ├── user-type.ts
│   ├── [domain]-type.ts
│   └── api-response.ts
└── globals.d.ts         # Declaraciones globales
```

## Patrones Clave

### API Service Layer

```typescript
// apps/web/src/api/fetch.ts
export class HttpService {
    constructor(axios: AxiosInstance) { ... }
    
    get<T>(url, options?)
    post<T>(url, data, options?)
    put<T>(url, data, options?)
    delete<T>(url, options?)
}

// apps/web/src/api/index.ts
export const api = axios.create(...)        // Auth routes
export const apiWithTenant = axios.create(...) // Tenant-scoped
export const noAuthApi = axios.create(...)  // Public routes

// usage en servicios:
const billingService = createHttpService(apiWithTenant);
export const createInvoice = (payload) =>
    billingService.post("/billing/invoices", payload, {
        notifySuccess: true,   // Service maneja toast
        notifyError: true,
    });
```

### Axios Instances

| Instance | Uso | Headers |
|----------|-----|---------|
| `api` | Login, auth | `Authorization: Bearer token` |
| `apiWithTenant` | Rutas tenant-scoped | `Authorization` + `X-Tenant-Slug` |
| `noAuthApi` | Public endpoints | Sin headers auth |

### Response Pattern

Todos los servicios retornan:
```typescript
interface ApiResponse<T> {
    success: boolean;
    data?: T;
    message?: string;
    code?: string;
}
```

**Uso:**
```typescript
const res = await createInvoice(payload);
if (res.success) {
    navigate("/billing");
    // Service ya mostró toast, no lo hagas acá
}
```

### i18n Pattern

```typescript
import { useText } from "@/hooks/use-text";

export function MyComponent() {
    const { textGet } = useText();
    
    return (
        <h1>{textGet("dashboard.title")}</h1>
        <p>{textGet("billing.invoice.title")}</p>
    );
}
```

**Reglas:**
- Solo usa `textGet()` para strings de usuario
- Si la key no existe, muestra `*key.name*` (no error)
- Nunca hardcodees strings en español/inglés

### Routing & Permissions

```typescript
// apps/web/src/routes/routes.tsx
import { CheckPermission } from "@/components/access-control/check-permission";

function BillingPage() {
    return (
        <CheckPermission permissions={[PERMISSIONS.BILLING.PERMISSION_READ_BILLING]}>
            <BillingContent />
        </CheckPermission>
    );
}

// O con permisos múltiples:
<CheckPermission permissions={[PERM1, PERM2]} requireAll={true}>
    Contenido
</CheckPermission>
```

### State Management (Zustand)

```typescript
// apps/web/src/store/billing-store.ts
import { create } from "zustand";

type BillingStore = {
    selectedInvoice?: Invoice;
    setSelectedInvoice: (invoice: Invoice) => void;
    clear: () => void;
};

export const useBillingStore = create<BillingStore>((set) => ({
    selectedInvoice: undefined,
    setSelectedInvoice: (invoice) => set({ selectedInvoice: invoice }),
    clear: () => set({ selectedInvoice: undefined }),
}));
```

**Uso:**
```typescript
const { selectedInvoice, setSelectedInvoice } = useBillingStore();
```

## Componentes de UI

Todos los componentes UI vienen de `shadcn/ui` (Radix + Tailwind):

```typescript
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";

// Siempre usa cn() para clases dinámicas:
import { cn } from "@/lib/utils";

<Button className={cn("px-4", isActive && "bg-primary")}>
    Click me
</Button>
```

## Form Patterns

```typescript
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import z from "zod";

const schema = z.object({
    name: z.string().min(2),
    email: z.string().email(),
});

function MyForm() {
    async function onSubmit(values: z.infer<typeof schema>) {
        // Hacer request
    }

    return (
        <Form schema={schema} onSubmit={onSubmit} defaultValues={{...}}>
            {(field) => (
                <>
                    <FormInput field={field} name="name" label="Name" />
                    <FormInput field={field} name="email" label="Email" />
                </>
            )}
        </Form>
    );
}
```

## Features Habilitados & Navigation

```typescript
// apps/web/src/config/nav-config.ts
const enabledFeatures = { clinical: true, billing: true, team: false };

const navItems = createNavItems(textGet, enabledFeatures)
    .filter(
        (item) =>
            (!item.permission || checkPermission([item.permission])) &&
            (!item.feature || enabledFeatures[item.feature] !== false)
    );
```

Los items de navegación se filtran automáticamente según:
1. **Permiso**: Usuario tiene el permiso requerido
2. **Feature**: El feature está habilitado en el plan

## Convenciones

| Tipo | Convención | Ejemplo |
|------|-----------|---------|
| Archivo | kebab-case | `user-profile.tsx` |
| Componente | PascalCase | `UserProfile` |
| Hook | camelCase con `use` | `useUserData` |
| Store | camelCase con `use` | `useBillingStore` |
| Servicio | camelCase | `billingService.ts` |
| Variable | camelCase | `selectedUser` |
| Constante | UPPERCASE | `DEFAULT_TIMEOUT` |
| Ruta | kebab-case | `/clinical/waiting-room` |
| i18n key | dot.notation | `dashboard.title`, `billing.invoice.create` |

## Imports

Usa path aliases (`@/`) del `tsconfig.json`:

```typescript
// ❌ No hagas esto:
import { Button } from "../../../components/ui/button";

// ✅ Haz esto:
import { Button } from "@/components/ui/button";
```

## CSS

TailwindCSS v4:

```typescript
// Usa cn() para clases condicionales:
import { cn } from "@/lib/utils";

<div className={cn(
    "px-4 py-2 rounded",
    isActive && "bg-primary text-white",
    isDisabled && "opacity-50 cursor-not-allowed"
)}>
    Content
</div>
```

No uses CSS modules en componentes nuevos (Tailwind ya es suficiente).

## TypeScript Types

Todos los tipos del backend que reflejamos en el frontend extienden `BaseModel`:

```typescript
export interface Invoice extends BaseModel {  // ID, CreatedAt, UpdatedAt, DeletedAt
    tenant_id: number;
    patient_id: number;
    sequential: string;
    status: string;
    total: number;
    items: InvoiceItem[];
}
```

**Usa snake_case** para propiedades (como vienen del API).
