# Web Frontend — State Management (Zustand)

Cómo usar Zustand para state management en el frontend React.

## 🎯 Principios

- **Zustand only** — No uses Redux, Context (excepto para casos especiales)
- **Store per domain** — Un store por feature/dominio
- **Persistence** — Usa localStorage/sessionStorage cuando sea necesario
- **Simplicity** — Solo guarda lo que necesitas

## Estructura

```typescript
// apps/web/src/store/[domain]-store.ts

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

type DomainStore = {
    // State
    selectedItem?: Item;
    filters: FilterOptions;
    loading: boolean;
    
    // Actions
    setSelectedItem: (item: Item) => void;
    setFilters: (filters: FilterOptions) => void;
    setLoading: (loading: boolean) => void;
    clear: () => void;
};

export const useDomainStore = create<DomainStore>((set) => ({
    selectedItem: undefined,
    filters: {},
    loading: false,
    
    setSelectedItem: (item) => set({ selectedItem: item }),
    setFilters: (filters) => set({ filters }),
    setLoading: (loading) => set({ loading }),
    clear: () => set({ selectedItem: undefined, filters: {}, loading: false }),
}));
```

## Uso en Componentes

```typescript
import { useDomainStore } from "@/store/domain-store";

function MyComponent() {
    // Selecciona solo lo que necesitas
    const { selectedItem, setSelectedItem, clear } = useDomainStore();
    
    return (
        <div>
            {selectedItem && <p>{selectedItem.name}</p>}
            <button onClick={() => setSelectedItem(item)}>Select</button>
            <button onClick={clear}>Clear</button>
        </div>
    );
}
```

## Persistencia

### SessionStorage (por sesión del usuario)

Para datos que duran mientras el usuario está logueado:

```typescript
// apps/web/src/store/session-store.ts

export const useSessionStore = create<SessionState>(
    persist(
        (set) => ({
            environment: undefined,
            subscriptionExpired: false,
            
            setEnvironment: (env) => set({ environment: env }),
            clean: () => set({ environment: undefined, subscriptionExpired: false }),
        }),
        {
            name: "session",
            storage: createJSONStorage(() => sessionStorage),
        }
    )
);
```

### LocalStorage (persistente en tiempo)

Para datos que se mantienen entre sesiones:

```typescript
// apps/web/src/store/billing-store.ts

export const useBillingStore = create<BillingStore>(
    persist(
        (set) => ({
            selectedInvoice: undefined,
            filters: {},
            
            setSelectedInvoice: (invoice) => set({ selectedInvoice: invoice }),
            setFilters: (filters) => set({ filters }),
        }),
        {
            name: "billing",
            storage: createJSONStorage(() => localStorage),
        }
    )
);
```

**Regla:**
- `sessionStorage` = Datos temporales por sesión
- `localStorage` = Preferencias, filtros que persisten

## Casos de Uso Comunes

### 1. Selección de Item

```typescript
type SelectionStore = {
    selectedPatientId?: number;
    setSelectedPatientId: (id: number) => void;
};

export const useSelectionStore = create<SelectionStore>((set) => ({
    selectedPatientId: undefined,
    setSelectedPatientId: (id) => set({ selectedPatientId: id }),
}));
```

### 2. Filtros & Búsqueda

```typescript
type ClinicalFilters = {
    searchTerm: string;
    status?: string;
    dateFrom?: string;
    dateTo?: string;
};

type ClinicalStore = {
    filters: ClinicalFilters;
    setFilters: (filters: Partial<ClinicalFilters>) => void;
    clearFilters: () => void;
};

export const useClinicalStore = create<ClinicalStore>(
    persist(
        (set) => ({
            filters: { searchTerm: "" },
            setFilters: (newFilters) =>
                set((state) => ({
                    filters: { ...state.filters, ...newFilters },
                })),
            clearFilters: () => set({ filters: { searchTerm: "" } }),
        }),
        { name: "clinical", storage: createJSONStorage(() => localStorage) }
    )
);

// Uso:
function SearchPatients() {
    const { filters, setFilters } = useClinicalStore();
    
    return (
        <Input
            value={filters.searchTerm}
            onChange={(e) => setFilters({ searchTerm: e.target.value })}
            placeholder="Search..."
        />
    );
}
```

### 3. Modal/Dialog State

```typescript
type DialogStore = {
    isOpen: boolean;
    data?: ModalData;
    open: (data?: ModalData) => void;
    close: () => void;
};

export const useModalStore = create<DialogStore>((set) => ({
    isOpen: false,
    data: undefined,
    open: (data) => set({ isOpen: true, data }),
    close: () => set({ isOpen: false, data: undefined }),
}));

// Uso:
function Dialog() {
    const { isOpen, data, open, close } = useModalStore();
    
    return (
        <AlertDialog open={isOpen} onOpenChange={close}>
            <AlertDialogContent>
                <AlertDialogTitle>Confirm</AlertDialogTitle>
                <AlertDialogDescription>{data?.message}</AlertDialogDescription>
            </AlertDialogContent>
        </AlertDialog>
    );
}

// Desde otro componente:
function DeleteButton() {
    const { open } = useModalStore();
    
    return (
        <Button onClick={() => open({ message: "Delete this item?" })}>
            Delete
        </Button>
    );
}
```

### 4. Sidebar State

```typescript
type SidebarStore = {
    isOpen: boolean;
    toggle: () => void;
    open: () => void;
    close: () => void;
};

export const useSidebarStore = create<SidebarStore>((set) => ({
    isOpen: true,
    toggle: () => set((state) => ({ isOpen: !state.isOpen })),
    open: () => set({ isOpen: true }),
    close: () => set({ isOpen: false }),
}));
```

## Anti-patrones

### ❌ NO: Guardar datos del servidor

```typescript
// ❌ NO hagas esto:
export const usePatientStore = create((set) => ({
    patients: [],  // Esto debería venir de una fetch, no de store
    setPatients: (patients) => set({ patients }),
}));
```

**¿Por qué?** Zustand no es caché. Si necesitas cachear datos del servidor, usa:
- React Query / TanStack Query
- O maneja la fetch localmente en el componente

### ❌ NO: Store gigante con todo

```typescript
// ❌ NO:
export const useMainStore = create((set) => ({
    user, company, team, patients, invoices, ... // Demasiado
}));

// ✅ SÍ:
export const useSessionStore = create(...)    // User, company
export const useClinicalStore = create(...)   // Patients, appointments
export const useBillingStore = create(...)    // Invoices
```

### ❌ NO: Lógica compleja en actions

```typescript
// ❌ NO:
export const usePatientStore = create((set) => ({
    createPatient: async (payload) => {
        const res = await patientService.create(payload);
        if (res.success) {
            set((state) => ({
                patients: [...state.patients, res.data],
            }));
        }
    },
}));

// ✅ SÍ: Lógica en componente o hook
async function handleCreatePatient(payload) {
    const res = await createPatient(payload);
    if (res.success) {
        // Refresca lista desde API, no guardando en store
        const updated = await getPatients();
        // ...
    }
}
```

## DevTools (Debugging)

```typescript
import { devtools } from "zustand/middleware";

export const useDomainStore = create<DomainStore>(
    devtools(
        (set) => ({
            // ...
        }),
        { name: "DomainStore" }
    )
);
```

Luego abre Redux DevTools en el navegador para inspeccionar cambios de estado.

## Testing

```typescript
// ✅ Test de store:
import { renderHook, act } from "@testing-library/react";
import { useDomainStore } from "@/store/domain-store";

it("updates selected item", () => {
    const { result } = renderHook(() => useDomainStore());
    
    act(() => {
        result.current.setSelectedItem({ id: 1, name: "Test" });
    });
    
    expect(result.current.selectedItem?.id).toBe(1);
});
```

## Prácticas Recomendadas

### ✅ DO

- Un store por dominio/feature
- Nombres descriptivos: `useBillingStore`, `useModalStore`
- Acciones simples (no lógica de negocio)
- Persistencia solo cuando sea necesario
- Tipifica completamente el state

### ❌ DON'T

- No guardes datos del servidor (eso es caché)
- No hagas stores gigantes
- No mezcles lógica de API con store
- No abusos de Zustand para cosas que React state puede hacer

## Cuándo usar Store vs useState

| Situación | Usa |
|-----------|-----|
| Estado local de componente | `useState` |
| Compartir entre hermanos cercanos | `useState` + prop drilling |
| Compartir entre componentes lejanos | `useStore` |
| Persistir entre sesiones | `useStore` + localStorage |
| Manejo de modales/dialogs | `useStore` |
| Caché de API | React Query / TanStack Query |
