---
name: Web Frontend — Complete Guide
description: Guía completa del frontend React incluyendo arquitectura, patrones API, state management, y cómo implementar nuevas features
---

# Web Frontend — Complete Guide

Guía completa de la arquitectura y desarrollo del frontend `apps/web` (React + TypeScript + TailwindCSS + shadcn/ui).

## 📐 Arquitectura General

### Stack Técnico

- **Framework:** React 19 + TypeScript
- **Build:** Vite
- **Styling:** TailwindCSS v4 + shadcn/ui
- **State:** Zustand (no Redux)
- **HTTP:** Axios + custom HttpService
- **Forms:** React Hook Form + Zod
- **i18n:** Custom hook `useText()`

### Estructura de Directorios

```
apps/web/src/
├── api/                         # Service layer
│   ├── index.ts                # Axios instances
│   ├── fetch.ts                # HttpService class
│   └── [domain]-service.ts     # Domain-specific services
├── components/
│   ├── ui/                     # shadcn/ui components (button, dialog, etc.)
│   ├── forms/                  # Form components (FormInput, FormSelect, etc.)
│   ├── access-control/         # Permission guards
│   └── [shared components]/
├── hooks/
│   ├── use-text.tsx            # i18n hook (textGet)
│   └── [custom hooks]/
├── lib/
│   ├── utils.ts                # cn() + utilities
│   ├── constants.ts            # PERMISSIONS, ROUTES
│   └── [helpers]/
├── types/
│   └── [domain]-type.ts        # Domain interfaces
├── store/
│   ├── session-store.ts        # Auth + current environment
│   └── [domain]-store.ts       # Domain state (Zustand)
├── pages/
│   └── [domain]/
│       ├── page.tsx            # Page component
│       └── components/         # Page-specific components
├── routes/
│   └── routes.tsx              # Route definitions + CheckPermission
├── config/
│   └── nav-config.ts           # Navigation items
├── sections/
│   └── [template]/
│       └── [component]/        # Shared sections
└── App.tsx                     # Root component
```

### Axios Instances

```typescript
// apps/web/src/api/index.ts

// Use: No auth header (public routes)
export const noAuthApi = axios.create({...})

// Use: Auth header only (login, auth routes)
export const api = axios.create({...})

// Use: Auth header + X-Tenant-Slug header (ALL tenant routes)
export const apiWithTenant = axios.create({...})
```

**Rule:** Elige la instancia correcta según el endpoint.

---

## 🔄 Service Layer Pattern

**NUNCA** llamar API directo desde componentes. Siempre usar service layer.

### HttpService Class

```typescript
import { createHttpService, type ServiceResponse } from "./fetch";
import { apiWithTenant } from "./index";

const itemService = createHttpService(apiWithTenant);

// GET /items
export const getItems = async (): Promise<ServiceResponse<Item[]>> =>
  itemService.get<Item[]>("/items", {
    notifyError: true,
  });

// POST /items
export const createItem = async (payload: CreateItemPayload): Promise<ServiceResponse<Item>> =>
  itemService.post<Item>("/items", payload, {
    notifySuccess: true,   // ← Service muestra toast automático
    notifyError: true,
  });

// PUT /items/:id
export const updateItem = async (id: number, payload: UpdateItemPayload): Promise<ServiceResponse<Item>> =>
  itemService.put<Item>(`/items/${id}`, payload, {
    notifySuccess: true,
    notifyError: true,
  });

// DELETE /items/:id
export const deleteItem = async (id: number): Promise<ServiceResponse<null>> =>
  itemService.delete<null>(`/items/${id}`, {
    notifySuccess: true,
    notifyError: true,
  });
```

### Response Pattern

```typescript
// Service retorna siempre ServiceResponse<T>
interface ServiceResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

// En componente, chequea success
const res = await createItem(payload);
if (res.success) {
  navigate("/items");
  // No hagas toast aquí, service ya lo hizo
} else {
  // Manejar error UI si es necesario
}
```

**Reglas:**
- Service maneja **TODOS** los toasts (notifySuccess/notifyError)
- Componente solo chequea `res.success` para navegación/estado
- Exception: toasts de validación UI pura (no API) pueden estar en componente

---

## 🔐 Types Pattern

Todos los tipos que reflejan modelos del API deben extender `BaseModel`:

```typescript
// apps/web/src/types/item-type.ts

export interface Item extends BaseModel {
  tenant_id: number;
  name: string;
  description?: string;
  status: "active" | "inactive";
}

export interface CreateItemPayload {
  name: string;
  description?: string;
  status?: "active" | "inactive";
}

export interface UpdateItemPayload {
  name?: string;
  description?: string;
  status?: "active" | "inactive";
}
```

**Reglas:**
- Extender `BaseModel` (ID, CreatedAt, UpdatedAt, DeletedAt)
- Usar `snake_case` para propiedades (como vienen del API)
- Separar payloads (`CreateItemPayload`, `UpdateItemPayload`) de respuestas

---

## 🎣 State Management with Zustand

**SOLO Zustand.** No Redux, Recoil, etc.

### Cuándo crear store

- Estado compartido entre múltiples componentes
- Filtros/búsqueda a nivel página
- Estado de modal/dialog persistente
- Selección de item

### NO crear store

- Estado local de un componente (`useState` es suficiente)
- Props que pasan directamente

### Store Pattern

```typescript
// apps/web/src/store/item-store.ts

import { create } from "zustand";
import type { Item } from "@/types/item-type";

interface ItemStore {
  items: Item[];
  selectedItem?: Item;
  searchTerm: string;
  sortBy: "name" | "createdAt";
  
  setItems: (items: Item[]) => void;
  setSelectedItem: (item: Item | undefined) => void;
  setSearchTerm: (term: string) => void;
  setSortBy: (sort: "name" | "createdAt") => void;
  addItem: (item: Item) => void;
  updateItem: (item: Item) => void;
  removeItem: (id: number) => void;
  reset: () => void;
}

export const useItemStore = create<ItemStore>((set) => ({
  items: [],
  selectedItem: undefined,
  searchTerm: "",
  sortBy: "name",

  setItems: (items) => set({ items }),
  setSelectedItem: (item) => set({ selectedItem: item }),
  setSearchTerm: (term) => set({ searchTerm: term }),
  setSortBy: (sort) => set({ sortBy: sort }),
  addItem: (item) => set((state) => ({ items: [...state.items, item] })),
  updateItem: (item) =>
    set((state) => ({
      items: state.items.map((i) => (i.id === item.id ? item : i)),
    })),
  removeItem: (id) =>
    set((state) => ({
      items: state.items.filter((i) => i.id !== id),
    })),
  reset: () =>
    set({
      items: [],
      selectedItem: undefined,
      searchTerm: "",
      sortBy: "name",
    }),
}));
```

---

## 📝 i18n Pattern

**NUNCA hardcodear strings.** Usar `useText()` hook siempre.

```typescript
import { useText } from "@/hooks/use-text";

export default function ItemPage() {
  const { textGet } = useText();

  return (
    <>
      <h1>{textGet("item.title")}</h1>
      <p>{textGet("item.description")}</p>
      <Button>{textGet("item.create")}</Button>
    </>
  );
}
```

**Reglas:**
- Hook es `useText()`, NO `t` o `useTranslation()`
- Función es `textGet(key)`, NO `t(key)`
- Missing keys renderean como `*key*` automático
- Keys deben estar en `apps/api/i18n/messages/messages_es.json` y `messages_en.json`

### i18n Keys Pattern

```json
{
  "item": {
    "title": "Ítems",
    "description": "Gestiona tus ítems",
    "create": "Crear Ítem",
    "edit": "Editar Ítem",
    "delete": "Eliminar",
    "created": "Ítem creado exitosamente",
    "updated": "Ítem actualizado",
    "deleted": "Ítem eliminado",
    "empty": "No hay ítems aún",
    "name": "Nombre",
    "name.placeholder": "Ingresa el nombre",
    "status": "Estado"
  }
}
```

---

## 🛠️ Cómo Implementar un Feature Nuevo

### Paso 1: Crear Tipos

**Archivo:** `apps/web/src/types/item-type.ts`

```typescript
export interface Item extends BaseModel {
  tenant_id: number;
  name: string;
  description?: string;
  status: "active" | "inactive";
}

export interface CreateItemPayload {
  name: string;
  description?: string;
}

export interface UpdateItemPayload {
  name?: string;
  description?: string;
  status?: "active" | "inactive";
}
```

### Paso 2: Crear Servicio

**Archivo:** `apps/web/src/api/item-service.ts`

```typescript
import type {
  CreateItemPayload,
  Item,
  UpdateItemPayload,
} from "@/types/item-type";

import { apiWithTenant } from ".";
import {
  createHttpService,
  type ServiceResponse,
} from "./fetch";

const itemService = createHttpService(apiWithTenant);

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

### Paso 3: Crear Store (si necesario)

**Archivo:** `apps/web/src/store/item-store.ts`

```typescript
import { create } from "zustand";
import type { Item } from "@/types/item-type";

interface ItemStore {
  items: Item[];
  selectedItem?: Item;
  
  setItems: (items: Item[]) => void;
  setSelectedItem: (item: Item | undefined) => void;
  addItem: (item: Item) => void;
  removeItem: (id: number) => void;
}

export const useItemStore = create<ItemStore>((set) => ({
  items: [],
  selectedItem: undefined,
  
  setItems: (items) => set({ items }),
  setSelectedItem: (item) => set({ selectedItem: item }),
  addItem: (item) => set((state) => ({ items: [...state.items, item] })),
  removeItem: (id) =>
    set((state) => ({
      items: state.items.filter((i) => i.id !== id),
    })),
}));
```

### Paso 4: Crear Página

**Archivo:** `apps/web/src/pages/item/page.tsx`

```typescript
import { useEffect, useState } from "react";
import { deleteItem, getItems } from "@/api/item-service";
import { Button } from "@/components/ui/button";
import { useText } from "@/hooks/use-text";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useItemStore } from "@/store/item-store";
import type { Item } from "@/types/item-type";
import ItemCard from "./components/item-card";
import CreateItemDialog from "./components/create-item-dialog";

export default function ItemPage() {
  const { textGet } = useText();
  const [loading, setLoading] = useState(true);
  const [showDialog, setShowDialog] = useState(false);
  const { items, setItems, selectedItem, setSelectedItem } = useItemStore();

  useEffect(() => {
    loadItems();
  }, []);

  async function loadItems() {
    setLoading(true);
    const res = await getItems();
    if (res.success) {
      setItems(res.data || []);
    }
    setLoading(false);
  }

  async function handleDelete(id: number) {
    const res = await deleteItem(id);
    if (res.success) {
      await loadItems();
      setSelectedItem(undefined);
    }
  }

  if (loading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center h-full">
          <div className="text-center space-y-2">
            <div className="h-10 w-10 border-2 border-muted rounded-full animate-spin mx-auto" />
            <p>{textGet("common.loading")}</p>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">{textGet("item.title")}</h1>
            <p className="text-sm text-muted-foreground mt-1">
              {textGet("item.description")}
            </p>
          </div>
          <Button onClick={() => setShowDialog(true)}>
            {textGet("item.create")}
          </Button>
        </div>

        {/* Items Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {items.map((item) => (
            <ItemCard
              key={item.id}
              item={item}
              selected={selectedItem?.id === item.id}
              onSelect={setSelectedItem}
              onDelete={handleDelete}
            />
          ))}
        </div>

        {items.length === 0 && (
          <div className="text-center py-12">
            <p className="text-muted-foreground">{textGet("item.empty")}</p>
          </div>
        )}

        {/* Create Dialog */}
        {showDialog && (
          <CreateItemDialog
            onClose={() => setShowDialog(false)}
            onSuccess={() => {
              setShowDialog(false);
              loadItems();
            }}
          />
        )}
      </div>
    </DashboardLayout>
  );
}
```

### Paso 5: Crear Componentes

**Archivo:** `apps/web/src/pages/item/components/item-card.tsx`

```typescript
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Trash2, Edit2 } from "lucide-react";
import { useText } from "@/hooks/use-text";
import type { Item } from "@/types/item-type";

interface ItemCardProps {
  item: Item;
  selected?: boolean;
  onSelect?: (item: Item) => void;
  onDelete?: (id: number) => void;
}

export default function ItemCard({
  item,
  selected,
  onSelect,
  onDelete,
}: ItemCardProps) {
  const { textGet } = useText();

  return (
    <Card
      className={`cursor-pointer transition-all ${
        selected ? "ring-2 ring-primary" : ""
      }`}
      onClick={() => onSelect?.(item)}
    >
      <CardHeader>
        <CardTitle className="text-base">{item.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {item.description && (
          <p className="text-sm text-muted-foreground line-clamp-2">
            {item.description}
          </p>
        )}
        <div className="flex gap-2">
          <Button
            size="sm"
            variant="ghost"
            onClick={(e) => {
              e.stopPropagation();
            }}
          >
            <Edit2 className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="text-red-600"
            onClick={(e) => {
              e.stopPropagation();
              onDelete?.(item.id);
            }}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
```

### Paso 6: Registrar Rutas

**Archivo:** `apps/web/src/routes/routes.tsx`

```typescript
import { CheckPermission } from "@/components/access-control/check-permission";
import { PERMISSIONS } from "@/lib/constants";
import ItemPage from "@/pages/item/page";

const routes = [
  // ... rutas existentes
  {
    path: "/items",
    element: (
      <CheckPermission permissions={[PERMISSIONS.ITEM.READ]}>
        <ItemPage />
      </CheckPermission>
    ),
  },
];
```

### Paso 7: Agregar a Navegación

**Archivo:** `apps/web/src/config/nav-config.ts`

```typescript
import { Package } from "lucide-react";

export const createNavItems = (textGet: any, enabledFeatures: string[]) => [
  // ... items existentes
  {
    icon: Package,
    label: textGet("item.title"),
    href: "/items",
    permission: PERMISSIONS.ITEM.READ,
    feature: "item",
  },
];
```

### Paso 8: Agregar i18n Keys

**Archivo:** `apps/api/i18n/messages/messages_es.json`

```json
{
  "item": {
    "title": "Ítems",
    "description": "Gestiona tus ítems aquí",
    "create": "Crear Ítem",
    "created": "Ítem creado exitosamente",
    "updated": "Ítem actualizado",
    "deleted": "Ítem eliminado",
    "empty": "No hay ítems aún"
  }
}
```

Y en `messages_en.json` con equivalentes.

---

## ✅ Checklist para Feature Nuevo

- [ ] Tipos creados con `extends BaseModel`
- [ ] Servicio creado con `createHttpService()`
- [ ] Store creado si es necesario
- [ ] Página creada en `pages/[domain]/page.tsx`
- [ ] Componentes creados en `pages/[domain]/components/`
- [ ] Rutas registradas en `routes/routes.tsx`
- [ ] Navegación actualizada en `nav-config.ts`
- [ ] i18n keys agregadas en ambos JSON
- [ ] Sin hardcoded strings (todo textGet)
- [ ] Sin console.logs en código de producción
- [ ] `just check` pasa sin errores
- [ ] Tipos correctos, sin `any`

