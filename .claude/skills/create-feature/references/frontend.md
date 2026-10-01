# Frontend (apps/web)

React 19 + Vite + Tailwind v4 + Zustand. Dominio canónico: catálogo de billing
(`catalog-item`). El código se organiza por tipo, luego por dominio:

| Qué | Dónde | Ejemplo |
|---|---|---|
| Servicio + tipos | `src/api/<domain>-service.ts` | `api/billing-service.ts` |
| Páginas | `src/pages/<domain>/<kebab>.tsx` | `pages/billing/catalog-item-list.tsx`, `create-catalog-item.tsx`, `edit-catalog-item.tsx` |
| Formularios | `src/sections/forms/<domain>/<x>-form.tsx` | `sections/forms/billing/catalog-item-form.tsx` |
| Columnas de tabla | `src/sections/columns/<domain>/<x>-columns.tsx` | `sections/columns/billing/catalog-item-columns.tsx` |
| Diálogos y widgets | `src/components/features/<domain>/` | `components/features/patient/edit-prescription-dialog.tsx` |
| Store | `src/store/<domain>-store.ts` | `store/kanban-store.ts` |
| Tests | `src/__tests__/<x>.test.tsx` o `<x>.test.ts` al lado | `__tests__/kanban-settings.test.tsx` |

(La carpeta clínica se llama `pages/clincal/`, con typo; respétalo.)

Código que usan **las dos** apps va en `packages/ui` (`@pengi/ui`, visual) o
`packages/shared` (`@pengi/shared`, no visual) — ver `packages/shared/README.md`.

## Servicio y tipos

```ts
import { type BaseModel, createHttpService, type ServiceResponse } from "@pengi/shared";
import { apiWithTenant } from "@/api"; // api = sin tenant, noAuthApi = público

const <domain>Service = createHttpService(apiWithTenant);

export interface <X> extends BaseModel { // ID, CreatedAt, UpdatedAt, DeletedAt
  tenant_id: number;
  name: string; // los campos llevan el nombre del json tag Go
}

export const get<X>s = async (): Promise<ServiceResponse<<X>[]>> =>
  <domain>Service.get<<X>[]>("/<domain>/<xs>", { notifyError: true });

export const create<X> = async (payload: Create<X>Payload) =>
  <domain>Service.post<<X>>("/<domain>/<xs>", payload, { notifySuccess: true, notifyError: true });
```

- Respuesta: `{ success, code, message, data }`; el componente decide solo
  navegación/estado según `res.success`. El toast lo muestra el servicio.
- Tipos: junto al servicio. Los nombres de campos siguen los `json` tags del
  modelo Go (`gorm.Model` sin tags → `ID`, `CreatedAt`…); cópialos del modelo.

## Store

Crea uno solo si varios componentes comparten el estado; si no, `useState`.
Por defecto `create()` sin persistencia (`store/kanban-store.ts`). Persiste
(`persist` + `createJSONStorage(() => sessionStorage)`) solo lo que debe
sobrevivir a un refresh; el logout limpia ambos storages. Si cambias la forma
de datos persistidos, versiona el `name` (`notification-storage-v4`).

## Páginas y listados

- La página NO se envuelve en `<DashboardLayout>`: `routes.tsx` lo monta una vez
  (con `<Outlet>`) para todas las rutas autenticadas. Empieza con
  `<PageHeader>` (`components/custom/page-header`).
- Listados: `<DataTable>` (`components/custom/table/data-table`) + un solo juego
  de columnas (`get<X>Columns`) con `meta.phone` para la lista del teléfono +
  `useRowStore` para selección múltiple.
- Acciones condicionadas por permiso dentro de la página:
  `const { checkPermission } = usePermission()`.

## Formularios

Estándar completo: `docs/skills/form-creation-standard.md`. En resumen:

- Zod schema + `<Form schema={schema} onSubmit={...}>{(methods) => ...}</Form>`;
  cada campo recibe `field={methods}`.
- De `@pengi/ui`: `Form`, `FormInput`, `FormPasswordInput`, `FormSelect`,
  `FormTextArea`, `FormRadioGroup`. Solo en web (`src/components/forms/`):
  `FormCalendar`, `FormTagInput`, `FormIcd11Select`. Para booleanos, el
  primitivo `Checkbox` de `@pengi/ui`.
- Mensajes de validación de Zod también son keys i18n.

## Rutas y nav

`src/routes/routes.tsx`:

```tsx
const <X>ListPage = lazy(() => import("@/pages/<domain>/<x>-list"));

const <domain>Routes: RouteObject = {
  path: "/<domain>",
  element: <CheckPermission permissions={[PERMISSIONS.<GROUP>.PERMISSION_READ_<X>]} />,
  children: [
    { index: true, element: <<X>ListPage /> },
    {
      path: "create",
      element: (
        <CheckPermission permissions={[PERMISSIONS.<GROUP>.PERMISSION_CREATE_<X>]}>
          <Create<X>Page />
        </CheckPermission>
      ),
    },
  ],
};

const routes: RouteObject[] = [clinicalRoutes, billingRoutes, <domain>Routes];
```

`CheckPermission` es default export de `@/components/custom/check-permission`,
exige **todos** los permisos listados y redirige a `/` si falta alguno. Una
página suelta (sin subrutas) va inline como `/team` o `/tasks`.

`src/config/nav-config.ts` → `createNavItems(textGet)`:

```ts
{
  label: textGet("<domain>.nav.title"),
  icon: Package, // componente de lucide-react, no JSX
  href: "/<domain>",
  permission: PERMISSIONS.<GROUP>.PERMISSION_READ_<X>,
  feature: "<key>", // solo si aplica — ver permisos-y-plan.md § Flag de navegación
}
```

Solo los ítems de primer nivel se filtran por `permission`/`feature`; los
`accordionItems` se muestran a quien vea el padre.

## i18n

`const { textGet } = useText()` de `@pengi/shared`. Las keys viven en el
backend (`apps/api/i18n/messages/`); una key inexistente se renderiza como
`*key*`. El navegador cachea los mensajes en `localStorage["messages"]` con el
ETag del catálogo y los revalida en cada carga: tras agregar keys basta con
recargar. Un test de `@pengi/shared` falla si una key literal no existe en el JSON.
