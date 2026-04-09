# Backoffice — Architecture Reference

Guía de arquitectura del panel administrativo (`apps/backoffice`).

## 🎯 Propósito

Backoffice es el **panel de administración** para gestionar:
- Empresas (Companies) y sus suscripciones
- Planes y catálogos de productos
- Roles y permisos globales
- Usuarios y sus environments
- Features y configuración

## Stack Técnico

**Idéntico al frontend principal**, pero con propósito diferente:

- **Framework**: React 19 + TypeScript
- **Build Tool**: Vite
- **Styling**: TailwindCSS v4 + CSS Modules
- **Components**: shadcn/ui
- **State Management**: Zustand
- **Routing**: React Router
- **HTTP Client**: Axios

## Estructura de Carpetas

```
apps/backoffice/src/
├── App.tsx
├── main.tsx
├── index.css
├── api/                    # Servicios administrativos
│   ├── index.ts
│   ├── fetch.ts           # HttpService (idéntico a web)
│   ├── company-service.ts
│   ├── plan-service.ts
│   ├── subscription-service.ts
│   ├── role-service.ts
│   ├── feature-service.ts
│   ├── user-service.ts
│   └── permission-service.ts
├── assets/
├── components/            # Componentes reutilizables
│   ├── ui/
│   ├── custom/
│   └── forms/
├── config/
│   └── nav-config.ts     # Navegación del backoffice
├── contexts/
├── hooks/
├── lib/
│   ├── constants.ts      # Constantes administrativas
│   └── utils.ts
├── pages/                 # Páginas administrativas
│   ├── companies/
│   │   ├── list.tsx
│   │   ├── detail.tsx
│   │   └── components/
│   ├── plans/
│   │   ├── plan-list.tsx
│   │   ├── create-plan.tsx
│   │   ├── edit-plan.tsx
│   │   └── plan-limits-editor.tsx
│   ├── subscriptions/
│   ├── roles/
│   ├── features/
│   ├── users/
│   ├── home/
│   └── login/
├── routes/
│   └── routes.tsx
├── sections/
│   ├── template/
│   │   └── dashboard-template.tsx
│   └── [domain]/
├── store/                # State management
│   └── sidebar-store.ts
└── types/
    ├── company-type.ts
    ├── plan-type.ts
    └── [domain]-type.ts
```

## Diferencias con Frontend Principal

| Aspecto | Frontend (apps/web) | Backoffice (apps/backoffice) |
|--------|---|---|
| Propósito | SaaS público para usuarios | Panel admin interno |
| Acceso | Usuarios normales | Admins solo |
| Features | Multi-tenant por empresa | Gestión global de tenants |
| Datos | Datos de una empresa | Todas las empresas |
| Permisos | Roles por empresa | Admin global |

## Dominios Administrativos

### 1. Companies
Gestión de empresas (clientes):

```
GET    /backoffice/companies           # Listar todas
GET    /backoffice/companies/:id       # Detalles
POST   /backoffice/companies           # Crear nueva
PUT    /backoffice/companies/:id       # Actualizar
DELETE /backoffice/companies/:id       # Eliminar
```

### 2. Plans
Gestión de planes de suscripción:

```
GET    /backoffice/plans               # Listar planes
GET    /backoffice/plans/:id           # Detalles del plan
POST   /backoffice/plans               # Crear plan
PUT    /backoffice/plans/:id           # Editar plan
DELETE /backoffice/plans/:id           # Eliminar plan
```

**Especial**: Plans incluyen features y límites (max_users, max_patients, etc.)

### 3. Subscriptions
Asignar planes a empresas:

```
GET    /backoffice/subscriptions                  # Listar
POST   /backoffice/subscriptions                  # Crear (asignar plan)
PUT    /backoffice/subscriptions/:id              # Cambiar plan/estado
DELETE /backoffice/subscriptions/:id              # Cancelar
```

### 4. Roles & Permissions
Gestión de roles globales (no por empresa):

```
GET    /backoffice/roles               # Listar roles
POST   /backoffice/roles               # Crear rol
PUT    /backoffice/roles/:id           # Editar rol
```

### 5. Features
Gestión de features disponibles:

```
GET    /backoffice/features            # Listar features
POST   /backoffice/features            # Crear feature
PUT    /backoffice/features/:id        # Editar feature
```

### 6. Users
Gestión de usuarios del sistema:

```
GET    /backoffice/users               # Listar
GET    /backoffice/users/:id           # Detalles
POST   /backoffice/users               # Crear
PUT    /backoffice/users/:id           # Editar
DELETE /backoffice/users/:id           # Eliminar
```

## Patrones Clave

### API Layer

Idéntico a web, pero **sin tenant header** (admin global):

```typescript
// apps/backoffice/src/api/index.ts
export const api = axios.create({
    baseURL: import.meta.env.VITE_API_URL,
    headers: {
        "Authorization": `Bearer ${token}`,  // ← Solo auth
        // NO X-Tenant-Slug (admin ve todos)
    },
});

// En servicios:
const companyService = createHttpService(api);
```

### State Management

Stores por dominio, similar a web:

```typescript
// apps/backoffice/src/store/company-store.ts
type CompanyStore = {
    selectedCompany?: Company;
    filters: CompanyFilters;
    setSelectedCompany: (company: Company) => void;
};

export const useCompanyStore = create<CompanyStore>(...)
```

### Forms

Formularios para CRUD operations:

```typescript
// apps/backoffice/src/pages/plans/create-plan.tsx
const formSchema = z.object({
    name: z.string().min(2),
    code: z.string().min(2),
    price: z.coerce.number().min(0),
    feature_codes: z.array(z.string()),
    properties: z.record(z.unknown()).optional(),
});
```

## Componentes Comunes

### Tables (para listar)

```typescript
<Table>
    <TableHeader>
        <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Actions</TableHead>
        </TableRow>
    </TableHeader>
    <TableBody>
        {companies.map((company) => (
            <TableRow key={company.id}>
                <TableCell>{company.name}</TableCell>
                <TableCell>
                    <EditButton /> <DeleteButton />
                </TableCell>
            </TableRow>
        ))}
    </TableBody>
</Table>
```

### Forms (para crear/editar)

```typescript
<Form schema={formSchema} onSubmit={onSubmit}>
    {(field) => (
        <>
            <FormInput field={field} name="name" label="Plan Name" />
            <FormInput field={field} name="price" label="Price" type="number" />
        </>
    )}
</Form>
```

### Dialogs (para confirmar)

```typescript
<AlertDialog open={isOpen} onOpenChange={onClose}>
    <AlertDialogContent>
        <AlertDialogTitle>Delete {company.name}?</AlertDialogTitle>
        <AlertDialogDescription>This cannot be undone.</AlertDialogDescription>
        <AlertDialogAction onClick={handleDelete}>
            Delete
        </AlertDialogAction>
    </AlertDialogContent>
</AlertDialog>
```

## Authentication

Backoffice requiere credenciales de admin:

```typescript
// Login es igual a web
const res = await loginService.login(email, password);

// Pero después accede a endpoints /backoffice/*
// Si no es admin, recibe 403 Forbidden del backend
```

## Convenciones

| Tipo | Convención | Ejemplo |
|------|-----------|---------|
| Archivo | kebab-case | `create-plan.tsx` |
| Página | PascalCase | `CreatePlan` |
| Service | camelCase | `companyService` |
| Store | camelCase con `use` | `useCompanyStore` |
| i18n key | dot.notation | `backoffice.plan.create.title` |
| Ruta | kebab-case | `/plans`, `/create-plan` |

## i18n

Todas las strings administrativas están en i18n:

```json
{
    "backoffice": {
        "plan": {
            "title": "Plans",
            "create": "Create Plan",
            "created": "Plan created successfully",
            "edit": "Edit Plan",
            "delete": "Delete Plan",
            "list": "Plan List",
            "col": {
                "name": "Plan Name",
                "code": "Code",
                "price": "Price",
                "features": "Features",
                "enabled_features": "Enabled Features"
            }
        },
        "company": { ... }
    }
}
```

## Testing

Patrón similar a web:

```typescript
// Mock services
jest.mock("@/api/plan-service");

// Render y test
it("creates a plan", async () => {
    jest.mocked(createPlan).mockResolvedValue({
        success: true,
        data: { id: 1, name: "Pro" },
    });
    
    // render y assertions...
});
```

## Key Files

| Archivo | Responsabilidad |
|---------|-----------------|
| `api/company-service.ts` | CRUD de empresas |
| `api/plan-service.ts` | CRUD de planes |
| `api/subscription-service.ts` | Asignar planes a empresas |
| `pages/plans/edit-plan.tsx` | Editor de planes con features |
| `pages/companies/detail.tsx` | Detalles de empresa |
| `store/company-store.ts` | Estado compartido de empresas |

## Next Steps

- [backoffice-domains.md](backoffice-domains.md) — Detalles de cada dominio
- [backoffice-admin-operations.md](backoffice-admin-operations.md) — Operaciones comunes
