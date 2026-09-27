---
name: Web Frontend — Complete Guide
description: Guía completa del frontend React incluyendo arquitectura, patrones API, state management, y cómo implementar nuevas features
---

# Web Frontend — Complete Guide

Guía de la arquitectura y los patrones del frontend `apps/web` (React + TypeScript + TailwindCSS).

> Para crear o extender una feature de punta a punta (backend, permisos, plan,
> tests), la receta es la skill `create-feature` (`.claude/skills/create-feature/`).
> Dominio de referencia para copiar: el catálogo de billing (`catalog-item`).

## 📐 Arquitectura General

### Stack Técnico

- **Framework:** React 19 + TypeScript, Vite
- **Styling:** TailwindCSS v4; componentes visuales en `@pengi/ui` (`packages/ui`)
- **State:** Zustand
- **HTTP:** Axios + `createHttpService` de `@pengi/shared` (`packages/shared`)
- **Forms:** React Hook Form + Zod, vía los componentes `Form*` de `@pengi/ui`
- **Tablas:** `@tanstack/react-table` vía `DataTable`
- **Routing:** React Router (`createBrowserRouter`, páginas con `lazy()`)
- **i18n:** `useText()` de `@pengi/shared`
- **Tests:** Vitest + Testing Library; e2e con Playwright

### Estructura de Directorios

El código se organiza por tipo y, dentro, por dominio:

```
apps/web/src/
├── api/
│   ├── index.ts                 # Instancias axios: api, apiWithTenant, noAuthApi
│   └── <domain>-service.ts      # Servicio + tipos del dominio
├── components/
│   ├── custom/                  # check-permission, page-header, table/data-table, ...
│   ├── features/<domain>/       # Diálogos y widgets de un dominio
│   ├── forms/                   # Solo web: FormCalendar, FormTagInput, FormIcd11Select
│   └── ui/                      # Pocos primitivos locales (el resto está en @pengi/ui)
├── config/nav-config.ts         # Ítems del sidebar
├── hooks/                       # use-permission, use-auth, user-responsive, ...
├── lib/constants.ts             # PERMISSIONS y constantes
├── pages/<domain>/<kebab>.tsx   # Páginas (la carpeta clínica se llama `clincal/`)
├── routes/routes.tsx            # Rutas + CheckPermission
├── sections/
│   ├── columns/<domain>/        # Definiciones de columnas de tablas
│   ├── forms/<domain>/          # Formularios
│   ├── template/                # DashboardLayout
│   └── ...                      # dashboard, settings, kanban, views
├── store/<domain>-store.ts      # Stores Zustand
├── types/                       # Tipos transversales (user, permission, api...)
└── __tests__/                   # Tests (también se colocan junto al código)
```

Código que usan **las dos** apps (web y backoffice) va en `packages/ui`
(visual) o `packages/shared` (no visual); ver `packages/shared/README.md`.

### Axios Instances (`src/api/index.ts`)

| Instancia | Headers | Uso |
|---|---|---|
| `noAuthApi` | ninguno (cookies) | rutas públicas |
| `api` | Bearer + `Accept-Language` | auth, rutas sin tenant |
| `apiWithTenant` | Bearer + `X-Tenant-Slug` + `Accept-Language` | todas las rutas de tenant |

---

## 🔄 Service Layer Pattern

Los componentes nunca llaman a axios: siempre pasan por un servicio.

```typescript
// src/api/item-service.ts
import {
  type BaseModel,
  createHttpService,
  type ServiceResponse,
} from "@pengi/shared";
import { apiWithTenant } from ".";

const itemService = createHttpService(apiWithTenant);

export interface Item extends BaseModel {
  tenant_id: number;
  name: string;
  description?: string;
}

export type CreateItemPayload = {
  name: string;
  description?: string;
};
export type UpdateItemPayload = Partial<CreateItemPayload>;

export const getItems = async (): Promise<ServiceResponse<Item[]>> =>
  itemService.get<Item[]>("/items", { notifyError: true });

export const createItem = async (
  payload: CreateItemPayload,
): Promise<ServiceResponse<Item>> =>
  itemService.post<Item>("/items", payload, {
    notifySuccess: true,
    notifyError: true,
  });

export const updateItem = async (
  id: number,
  payload: UpdateItemPayload,
): Promise<ServiceResponse<Item>> =>
  itemService.put<Item>(`/items/${id}`, payload, {
    notifySuccess: true,
    notifyError: true,
  });

export const deleteItem = async (id: number): Promise<ServiceResponse<null>> =>
  itemService.delete<null>(`/items/${id}`, {
    notifySuccess: true,
    notifyError: true,
  });
```

- GET → `{ notifyError: true }`; escrituras → `{ notifySuccess: true, notifyError: true }`.
- `notifySuccess`/`notifyError` aceptan también un string (key i18n) que
  reemplaza el mensaje del backend.

### Response Pattern

```typescript
// ServiceResponse<T> = { success, code, message, data, filename? }
// En error, data = { error_code, error_message }
const res = await createItem(payload);
if (res.success) {
  navigate("/items"); // el toast ya lo mostró el servicio
}
```

Toasts de validación puramente de UI (no resultado de una llamada) sí van en
el componente.

---

## 🔐 Types Pattern

Los tipos van junto a su servicio en `src/api/<domain>-service.ts`
(`types/` queda para tipos transversales). Los que reflejan un modelo con
`gorm.Model` extienden `BaseModel` de `@pengi/shared`:

```typescript
// BaseModel = { ID: number; CreatedAt: string; UpdatedAt: string; DeletedAt?: string | null }
export interface Item extends BaseModel {
  tenant_id: number;
  name: string;
}
```

Los nombres de los campos son los `json` tags del modelo Go: cópialos del
modelo (`snake_case` en la mayoría; los campos de `gorm.Model` en PascalCase).
Usa `item.ID`, no `item.id`.

---

## 🎣 State Management with Zustand

Crea un store solo si varios componentes comparten el estado. Para estado de
un componente, `useState`.

```typescript
// src/store/item-store.ts
import { create } from "zustand";
import type { Item } from "@/api/item-service";

interface ItemStore {
  selectedItem?: Item;
  setSelectedItem: (item: Item | undefined) => void;
}

export const useItemStore = create<ItemStore>((set) => ({
  selectedItem: undefined,
  setSelectedItem: (item) => set({ selectedItem: item }),
}));
```

- Por defecto sin persistencia (`store/kanban-store.ts`).
- Persiste solo lo que debe sobrevivir a un refresh:
  `persist(..., { name: "<x>-storage", storage: createJSONStorage(() => sessionStorage) })`.
  El logout limpia `localStorage` y `sessionStorage`.
- Si cambias la forma de datos persistidos, versiona el `name`
  (`notification-storage-v4`) para no leer cachés viejos.
- Selección múltiple de filas en tablas: `useRowStore` (`store/row-store.ts`).

---

## 📝 i18n Pattern

```typescript
import { useText } from "@pengi/shared";
import { Text } from "@pengi/ui";

const { textGet } = useText();
<h1>{textGet("item.title")}</h1>
<Text uuid="item.create.button" /> // equivalente en JSX
```

- Hook `useText()` y función `textGet(key)` (no `t` ni `useTranslation`).
- Una key inexistente se renderiza como `*key*`.
- Las keys viven en el backend: `apps/api/i18n/messages/messages_es.json` y
  `messages_en.json`, array plano:

```json
[
  { "key": "item.title", "value": "Ítems" },
  { "key": "item.create.button", "value": "Crear ítem" },
  { "key": "item.form.error.required", "value": "Campo obligatorio" }
]
```

- **Caché:** los mensajes se guardan en `localStorage["messages"]` y solo se
  vuelven a pedir si cambia `__APP_VERSION__` (arranque de Vite) o el idioma.
  Tras agregar keys: `localStorage.removeItem("messages")` y recarga.

---

## 🛠️ Cómo Implementar un Feature Nuevo

### Paso 1: Servicio y tipos

`src/api/<domain>-service.ts`, como en el patrón de arriba.

### Paso 2: Store (solo si hace falta)

`src/store/<domain>-store.ts`.

### Paso 3: Columnas de la tabla

`src/sections/columns/<domain>/<x>-columns.tsx`, con versión desktop y mobile
(patrón `sections/columns/billing/catalog-item-columns.tsx`):

```typescript
import { DataTableColumnHeader, Text } from "@pengi/ui";
import type { ColumnDef } from "@tanstack/react-table";
import type { Item } from "@/api/item-service";

interface ItemColumnProps {
  onEdit: (id: number) => void;
  onDelete: (id: number) => void;
}

export const getItemColumns = ({ onEdit, onDelete }: ItemColumnProps): ColumnDef<Item>[] => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={<Text uuid="item.column.name" />} />
    ),
  },
  // columna "select" con Checkbox para selección múltiple, columna de acciones...
];

export const getItemColumnsMobile = (props: ItemColumnProps): ColumnDef<Item>[] => [/* ... */];
```

### Paso 4: Página de listado

`src/pages/<domain>/<x>-list.tsx` (patrón `pages/billing/catalog-item-list.tsx`):

```typescript
import { useText } from "@pengi/shared";
import { Button, Text } from "@pengi/ui";
import { Plus } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router";
import { deleteItem, getItems, type Item } from "@/api/item-service";
import { PageHeader } from "@/components/custom/page-header";
import { DataTable } from "@/components/custom/table/data-table";
import usePermission from "@/hooks/use-permission";
import { useResponsive } from "@/hooks/user-responsive";
import { PERMISSIONS } from "@/lib/constants";
import { getItemColumns, getItemColumnsMobile } from "@/sections/columns/item/item-columns";
import { DashboardLayout } from "@/sections/template/dashboard-template";

const ItemList = () => {
  const { textGet } = useText();
  const { checkPermission } = usePermission();
  const { isMobile } = useResponsive();
  const navigate = useNavigate();
  const [items, setItems] = React.useState<Item[]>([]);
  const [loading, setLoading] = React.useState(true);

  const load = React.useCallback(async () => {
    setLoading(true);
    const res = await getItems();
    if (res.success && res.data) setItems(res.data);
    setLoading(false);
  }, []);

  React.useEffect(() => {
    load();
  }, [load]);

  const handlers = {
    onEdit: (id: number) => navigate(`/items/edit/${id}`),
    onDelete: async (id: number) => {
      const res = await deleteItem(id);
      if (res.success) load();
    },
  };

  return (
    <DashboardLayout>
      <main className="grid items-start gap-4">
        <PageHeader
          title={textGet("item.title")}
          description={textGet("item.page.description")}
          actions={
            checkPermission([PERMISSIONS.ITEM.PERMISSION_CREATE_ITEM]) && (
              <Button onClick={() => navigate("/items/create")}>
                <Plus className="mr-2 h-4 w-4" />
                <Text uuid="item.create.button" />
              </Button>
            )
          }
        />
        <DataTable
          columns={isMobile ? getItemColumnsMobile(handlers) : getItemColumns(handlers)}
          data={items}
          loading={loading}
        />
      </main>
    </DashboardLayout>
  );
};

export default ItemList;
```

`DataTable` también acepta paginación (`page`, `pageCount`, `onPageChange`),
búsqueda (`searchValue`, `onSearchChange`, `searchPlaceholder`) y
`bulkActions`.

### Paso 5: Formulario y páginas de crear/editar

Formulario en `src/sections/forms/<domain>/<x>-form.tsx`, siguiendo
[`form-creation-standard.md`](form-creation-standard.md). Las páginas
`create-<x>.tsx` / `edit-<x>.tsx` solo arman el payload, llaman al servicio y
navegan si `res.success` (patrón `pages/billing/create-catalog-item.tsx`).

Un registro que el usuario querrá volver a ver, imprimir o reenviar vive en
una página + listado; un `Dialog` (`components/features/<domain>/`) es para
ediciones rápidas de uno o dos campos.

### Paso 6: Rutas

`src/routes/routes.tsx`:

```typescript
import { lazy } from "react";
import CheckPermission from "@/components/custom/check-permission";

const ItemList = lazy(() => import("@/pages/item/item-list"));
const CreateItemPage = lazy(() => import("@/pages/item/create-item"));

const itemRoutes: RouteObject = {
  path: "/items",
  element: <CheckPermission permissions={[PERMISSIONS.ITEM.PERMISSION_READ_ITEM]} />,
  children: [
    { index: true, element: <ItemList /> },
    {
      path: "create",
      element: (
        <CheckPermission permissions={[PERMISSIONS.ITEM.PERMISSION_CREATE_ITEM]}>
          <CreateItemPage />
        </CheckPermission>
      ),
    },
  ],
};

const routes: RouteObject[] = [clinicalRoutes, billingRoutes, itemRoutes];
```

- `CheckPermission` es default export, exige **todos** los permisos listados y
  redirige a `/` si falta alguno.
- Una página suelta sin subrutas va inline en el router, como `/team` o `/tasks`.

### Paso 7: Permisos y navegación

`src/lib/constants.ts` — los strings son idénticos a los IDs del backend:

```typescript
export const PERMISSIONS = {
  // ...
  ITEM: {
    PERMISSION_READ_ITEM: "READ_ITEM",
    PERMISSION_CREATE_ITEM: "CREATE_ITEM",
  },
};
```

`src/config/nav-config.ts`:

```typescript
import { Package } from "lucide-react";

export const createNavItems = (textGet: (key: string) => string): NavItemType[] => [
  // ... items existentes
  {
    icon: Package,
    label: textGet("item.title"),
    href: "/items",
    permission: PERMISSIONS.ITEM.PERMISSION_READ_ITEM,
    feature: "clinical", // opcional: oculta el ítem si el plan no incluye esa categoría
  },
];
```

- Solo los ítems de primer nivel se filtran por `permission`/`feature`; los
  `accordionItems` se muestran a quien vea el padre.
- `feature` es una key de `EnabledFeatures` (`clinical`, `billing`, `team`,
  `kanban`), calculada en el backend desde el plan. Una categoría nueva requiere
  cambios en backend y frontend: ver la skill `create-feature`,
  `references/permisos-y-plan.md` § Flag de navegación.

### Paso 8: i18n Keys

Agrega todas las keys usadas (textos, placeholders, mensajes de Zod) en ambos
JSON del backend, como en el patrón de i18n.

### Paso 9: Tests

- Vitest: `src/__tests__/<x>.test.tsx` o `<x>.test.ts` junto al código
  (patrón `__tests__/kanban-settings.test.tsx`).
- e2e: `apps/web/e2e/*.spec.ts` (`just tests-e2e`, con el stack arriba).

---

## ✅ Checklist para Feature Nuevo (web)

- [ ] Servicio + tipos en `src/api/<domain>-service.ts` (tipos con `BaseModel` si aplica)
- [ ] Store solo si el estado es compartido
- [ ] Columnas en `sections/columns/<domain>/` (desktop + mobile)
- [ ] Páginas en `pages/<domain>/` con `DashboardLayout` + `PageHeader`
- [ ] Formularios en `sections/forms/<domain>/` según `form-creation-standard.md`
- [ ] Rutas `lazy()` con `CheckPermission`, grupo agregado a `routes`
- [ ] `PERMISSIONS` en `constants.ts` idénticos al backend
- [ ] Ítem en `nav-config.ts` con `permission` (y `feature` si aplica)
- [ ] i18n keys en ambos JSON; nada hardcodeado
- [ ] `pnpm run typecheck`, `pnpm test:run` y `just check` pasan
