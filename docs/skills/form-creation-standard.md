---
name: Form Creation Standard
description: Estándar para crear formularios reactivos con Zod y los componentes Form de @pengi/ui en la aplicación Pengi Med SaaS
---

# Form Creation Standard

Estándar para los formularios de `apps/web` y `apps/backoffice`: schema Zod +
`<Form>` + componentes `Form*` de `@pengi/ui`.

Ejemplo de referencia completo: `apps/web/src/sections/forms/billing/catalog-item-form.tsx`
(usado por `pages/billing/create-catalog-item.tsx` y `edit-catalog-item.tsx`).

## Dónde vive

- **Web:** el formulario en `src/sections/forms/<domain>/<x>-form.tsx`, y las
  páginas `src/pages/<domain>/create-<x>.tsx` / `edit-<x>.tsx` que lo usan.
- **Backoffice:** CRUD estándar con `src/lib/resource`; formularios a mano solo
  si el recurso no encaja.

El formulario recibe `onSubmit`, `initialData` y `loading` por props; la
página arma el payload, llama al servicio y navega. Así el mismo formulario
sirve para crear y editar.

## Estructura Base

### 1. Schema con Zod

Los mensajes de error son **keys i18n**: los componentes los traducen al
mostrarlos.

```typescript
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(1, "item.form.error.required"),
  email: z.string().email("item.form.error.email").optional(),
  unit_price: z.coerce.number().min(0, "item.form.error.positive"),
  status: z.enum(["active", "inactive"]),
});

export type FormValues = z.infer<typeof formSchema>;
```

### 2. Componente de formulario

```typescript
// src/sections/forms/item/item-form.tsx
import { useText } from "@pengi/shared";
import {
  Button,
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  Form,
  FormInput,
  FormSelect,
  FormTextArea,
} from "@pengi/ui";
import { Loader2 } from "lucide-react";
import { z } from "zod";
import type { Item } from "@/api/item-service";

const formSchema = z.object({
  name: z.string().min(1, "item.form.error.required"),
  description: z.string().optional(),
  status: z.enum(["active", "inactive"]),
});

export type FormValues = z.infer<typeof formSchema>;

interface ItemFormProps {
  initialData?: Item;
  loading?: boolean;
  onSubmit: (values: FormValues) => void;
}

export default function ItemForm({ initialData, loading, onSubmit }: ItemFormProps) {
  const { textGet } = useText();
  const isEditing = Boolean(initialData);

  const defaultValues: FormValues = {
    name: initialData?.name ?? "",
    description: initialData?.description ?? "",
    status: initialData?.status ?? "active",
  };

  return (
    <Form schema={formSchema} onSubmit={onSubmit} defaultValues={defaultValues}>
      {(field) => (
        <Card className="max-w-4xl mx-auto">
          <CardHeader>
            <CardTitle>
              {textGet(isEditing ? "item.form.edit.title" : "item.form.create.title")}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4">
              <FormInput
                field={field}
                name="name"
                label={textGet("item.form.name")}
                placeholder={textGet("item.form.name.placeholder")}
              />
              <FormSelect
                field={field}
                name="status"
                label={textGet("item.form.status")}
                options={[
                  { label: textGet("item.status.active"), value: "active" },
                  { label: textGet("item.status.inactive"), value: "inactive" },
                ]}
              />
            </div>
            <FormTextArea
              field={field}
              name="description"
              label={textGet("item.form.description")}
              isOptional
            />
          </CardContent>
          <CardFooter className="flex justify-end">
            <Button type="submit" disabled={loading}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {textGet("form.save")}
            </Button>
          </CardFooter>
        </Card>
      )}
    </Form>
  );
}
```

`Form` pasa los métodos de React Hook Form (`field`) como render prop; cada
campo los recibe en `field={field}`. Para leer otros campos dentro del form
(cálculos, campos condicionales) usa `useFormContext()` en un subcomponente.
Si el submit tiene errores de validación, `Form` muestra un toast resumen.

### 3. Página que lo usa

```typescript
// src/pages/item/create-item.tsx
export default function CreateItemPage() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: FormValues) => {
    setLoading(true);
    const res = await createItem({ name: values.name, description: values.description });
    if (res.success) {
      navigate("/items"); // el servicio ya mostró el toast
    }
    setLoading(false);
  };

  return (
    <DashboardLayout>
      <main className="grid items-start gap-4">
        <ItemForm onSubmit={handleSubmit} loading={loading} />
      </main>
    </DashboardLayout>
  );
}
```

Los servicios no lanzan excepciones: devuelven `{ success: false }` y muestran
el toast de error. No hace falta `try/catch`.

## Componentes Form Disponibles

Desde `@pengi/ui` (web y backoffice):

| Componente | Uso | Props principales |
|-----------|-----|-------|
| `Form` | Contenedor con Zod + React Hook Form | `schema`, `onSubmit`, `defaultValues`, `onUpdateValuesCallback`, `className` |
| `FormInput` | Texto, email, número, fecha | `field`, `name`, `label`, `isOptional`, `description`, `startAddon`, `endAddon` + atributos de `<input>` (`type`, `placeholder`, `disabled`...) |
| `FormPasswordInput` | Contraseña con mostrar/ocultar | `field`, `name`, `label`, `isOptional`, `description` |
| `FormSelect` | Dropdown | `field`, `name`, `label`, `options: {label, value}[]`, `placeholder`, `disabled`, `emptyMessage` |
| `FormTextArea` | Texto multilínea | `field`, `name`, `label`, `isOptional`, `description` |
| `FormRadioGroup` | Radio buttons | `field`, `name`, `label`, `options: {label, value}[]`, `isRow` |

Solo en `apps/web`, en `@/components/forms/`:

| Componente | Uso |
|-----------|-----|
| `FormCalendar` | Fecha (y hora) con selector visual |
| `FormTagInput` | Tags |
| `FormIcd11Select` | Selector de diagnóstico ICD-11 |

Para booleanos no hay `FormCheckbox`: usa el primitivo `Checkbox`
de `@pengi/ui` controlado vía `useFormContext()`, como el toggle de ICE en
`catalog-item-form.tsx`.

Todo campo de formulario usa estos componentes, no `<input>`/`<select>` HTML.

## Reglas

### Validación
- Zod para todos los campos; `.optional()` para opcionales, `.enum([...])` para enumerables.
- `z.coerce.number()` para campos numéricos que llegan como string del input.
- Mensajes de error como keys i18n.
- Validación cruzada con `.superRefine((data, ctx) => ctx.addIssue({ code: "custom", path: ["campo"], message: "item.form.error.x" }))`.

### i18n
- Labels, placeholders, opciones y mensajes de error vía `textGet()` o keys.
- Keys: `<domain>.form.<campo>`, `<domain>.form.<campo>.placeholder`, `<domain>.form.error.<regla>`.
- Agregar las keys a ambos `apps/api/i18n/messages/messages_es.json` y `messages_en.json`.

### Layout
- `Card` con header / content / footer, `max-w-4xl mx-auto` (o `max-w-2xl` si es corto).
- Grid responsive: `grid md:grid-cols-2 grid-cols-1 gap-2 md:gap-4`.
- Botón submit con estado `loading` (spinner `Loader2`).

### Página o diálogo
- Un registro que el usuario querrá volver a ver, imprimir o reenviar →
  página + listado.
- Edición rápida de 1-2 campos desde una fila → `Dialog` en
  `components/features/<domain>/` (patrón `edit-prescription-dialog.tsx`), con
  los mismos componentes `Form*`. En diálogos angostos, evita grids de 3
  columnas con fechas largas.

## Checklist

- [ ] Schema Zod con mensajes como keys i18n; tipo con `z.infer`
- [ ] Formulario en `sections/forms/<domain>/`, reutilizable para crear/editar
- [ ] `Form` y campos de `@pengi/ui` (o `FormCalendar`/`FormTagInput` locales en web)
- [ ] Todos los textos vía `textGet()`; keys en ambos JSON
- [ ] Grid responsive y `max-w-*` aplicados
- [ ] Submit con estado `loading`; la página navega si `res.success`
