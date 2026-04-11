---
name: Form Creation Standard
description: Estándar para crear formularios reactivos con Zod, Form components y Zustand en la aplicación Pengi Med SaaS
---

# Form Creation Standard

Estándar obligatorio para crear formularios en la aplicación Pengi Med SaaS. Todos los formularios deben seguir esta estructura para garantizar consistencia y mantenibilidad.

## Estructura Base

### 1. Define el Schema con Zod

```typescript
import { z } from "zod";

const formSchema = z.object({
  // Campos requeridos
  field1: z.string().min(1, "Field is required"),
  
  // Campos opcionales
  field2: z.string().optional(),
  
  // Enumerables
  status: z.enum(["active", "inactive"]),
  
  // Fechas
  birthDate: z.string().optional(),
});

type FormData = z.infer<typeof formSchema>;
```

### 2. Define el Handler Submit

```typescript
const handleSubmit = async (data: FormData) => {
  try {
    const res = await apiService.create(data);
    if (res.success) {
      // Service maneja los toasts automáticamente
      navigate("/list");
    }
  } catch (error) {
    console.error("Error submitting form:", error);
  }
};
```

### 3. Estructura del Componente

```typescript
import React from "react";
import { useNavigate } from "react-router";
import { z } from "zod";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { FormSelect } from "@/components/forms/form-select";
import { FormTextArea } from "@/components/forms/form-textarea";
import { Button } from "@/components/ui/button";
import { useText } from "@/hooks/use-text";

const formSchema = z.object({
  name: z.string().min(1, "Name is required"),
  email: z.string().email("Invalid email"),
  message: z.string().optional(),
  status: z.enum(["active", "inactive"]),
});

export default function CreateForm() {
  const navigate = useNavigate();
  const { textGet } = useText();
  const [loading, setLoading] = React.useState(false);

  const handleSubmit = async (data: z.infer<typeof formSchema>) => {
    try {
      setLoading(true);
      const res = await createItem(data);
      if (res.success) {
        navigate("/items");
      }
    } catch (error) {
      console.error("Error:", error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form schema={formSchema} onSubmit={handleSubmit}>
      {(field) => (
        <div className="max-w-4xl mx-auto p-6">
          <div className="space-y-6">
            {/* Header */}
            <div className="space-y-2">
              <h1 className="text-2xl font-bold">{textGet("form.create.title")}</h1>
              <p className="text-sm text-muted-foreground">{textGet("form.create.description")}</p>
            </div>

            {/* Form Fields */}
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormInput
                  field={field}
                  name="name"
                  label={textGet("form.name")}
                  placeholder={textGet("form.name.placeholder")}
                  required
                  autoFocus
                />

                <FormInput
                  field={field}
                  name="email"
                  label={textGet("form.email")}
                  type="email"
                  placeholder={textGet("form.email.placeholder")}
                />
              </div>

              <FormTextArea
                field={field}
                name="message"
                label={textGet("form.message")}
                placeholder={textGet("form.message.placeholder")}
              />

              <FormSelect
                field={field}
                name="status"
                label={textGet("form.status")}
                options={[
                  { label: "Active", value: "active" },
                  { label: "Inactive", value: "inactive" },
                ]}
              />
            </div>

            {/* Actions */}
            <div className="flex gap-3 pt-4 border-t">
              <Button
                type="button"
                variant="outline"
                onClick={() => navigate(-1)}
              >
                {textGet("common.cancel")}
              </Button>
              <Button type="submit" disabled={loading}>
                {loading ? textGet("common.saving") : textGet("common.save")}
              </Button>
            </div>
          </div>
        </div>
      )}
    </Form>
  );
}
```

## Reglas Obligatorias

### ✅ Validación
- Usar **Zod** para validar todos los campos
- Mensajes de error en **i18n keys** o strings claros
- Tipos opcionales con `.optional()`
- Enumerables con `.enum([...])`

### ✅ Componentes
- Usar `Form` de `@/components/forms/form`
- Usar componentes especializados:
  - `FormInput` — texto, email, número, fecha
  - `FormSelect` — dropdown (con `options` array)
  - `FormTextArea` — texto multiline
  - `FormCheckbox` — checkbox
  - `FormRadioGroup` — radio buttons
  - `FormTagInput` — tags
- **NUNCA** usar HTML directo (`<input>`, `<select>`)

### ✅ Layout
- Estructura clara: header + fields + actions
- Grid responsive: `grid-cols-1 md:grid-cols-2 gap-4` mínimo
- Max ancho: `max-w-4xl mx-auto` o `max-w-2xl` si es simple
- Espaciado: `space-y-4` entre secciones, `space-y-6` para el form completo

### ✅ I18n
- Todos los labels desde `textGet()`
- Keys pattern: `form.create.title`, `form.fieldname`, `form.fieldname.placeholder`
- **NUNCA** hardcodear strings
- Agregar keys a ambos `messages_es.json` y `messages_en.json`

### ✅ Estado
- Usar `useState` para `loading`
- Submit debe ser `async`
- Manejar errores con try/catch
- Service maneja toasts automáticamente

### ✅ Tipos
- Extraer tipo con `z.infer<typeof formSchema>`
- Types en archivo separado si se reutiliza
- Interfaces deben extender `BaseModel` para datos del backend

### ✅ Responsive
- Mínimo: `grid-cols-1 md:grid-cols-2 gap-4`
- Content centered: `max-w-4xl mx-auto`
- Padding: `p-6` para contenedores principales

## Componentes Form Disponibles

| Componente | Uso | Props |
|-----------|-----|-------|
| `FormInput` | Campos de texto, email, número, fecha | `field`, `name`, `label`, `placeholder`, `type`, `required`, `autoFocus`, `disabled` |
| `FormSelect` | Dropdowns | `field`, `name`, `label`, `options: {label, value}[]`, `disabled` |
| `FormTextArea` | Texto multiline | `field`, `name`, `label`, `placeholder`, `description`, `disabled` |
| `FormCheckbox` | Checkbox single | `field`, `name`, `label`, `disabled` |
| `FormRadioGroup` | Radio buttons | `field`, `name`, `label`, `options: {label, value}[]`, `disabled` |
| `FormTagInput` | Tags | `field`, `name`, `label`, `placeholder`, `disabled` |

## Ejemplos por Tipo

### Crear Item Simple

```typescript
const createSchema = z.object({
  name: z.string().min(1, "Required"),
  description: z.string().optional(),
});

export default function CreateItemForm() {
  const { textGet } = useText();
  const navigate = useNavigate();

  return (
    <Form schema={createSchema} onSubmit={async (data) => {
      const res = await createItem(data);
      if (res.success) navigate("/items");
    }}>
      {(field) => (
        <div className="max-w-2xl mx-auto space-y-4 p-6">
          <h1>{textGet("item.create.title")}</h1>
          <FormInput field={field} name="name" label="Name" required />
          <FormTextArea field={field} name="description" label="Description" />
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => navigate(-1)}>Cancel</Button>
            <Button type="submit">Create</Button>
          </div>
        </div>
      )}
    </Form>
  );
}
```

### Editar con Estado Inicial

```typescript
interface UpdateFormProps {
  item: Item;
}

export default function UpdateItemForm({ item }: UpdateFormProps) {
  const { textGet } = useText();
  const navigate = useNavigate();

  return (
    <Form
      schema={updateSchema}
      defaultValues={{
        name: item.name,
        email: item.email,
        status: item.status,
      }}
      onSubmit={async (data) => {
        const res = await updateItem(item.id, data);
        if (res.success) navigate("/items");
      }}
    >
      {(field) => (
        // ... fields igual que crear
      )}
    </Form>
  );
}
```

### Con Validación Condicional

```typescript
const schema = z.object({
  type: z.enum(["individual", "company"]),
  name: z.string().min(1),
  taxId: z.string().optional(),
  companyName: z.string().optional(),
}).superRefine((data, ctx) => {
  if (data.type === "company" && !data.companyName) {
    ctx.addIssue({
      code: "custom",
      path: ["companyName"],
      message: "Company name required",
    });
  }
  if (data.type === "individual" && !data.taxId) {
    ctx.addIssue({
      code: "custom",
      path: ["taxId"],
      message: "Tax ID required",
    });
  }
});
```

## Cuándo NO Usar Esta Estructura

- **Formularios muy simples** (2-3 campos): puedes usar componentes más simples directamente
- **Inline edits**: usar inputs inline en lugar de un formulario completo
- **Quick actions**: diálogos con 1-2 campos simples

Para esos casos, simplifica: usa componentes de Form directamente sin envoltorio, o inputs HTML simples si es realmente trivial.

## Checklist

- [ ] Schema creado con Zod
- [ ] Tipos extraídos con `z.infer`
- [ ] Handler submit async con try/catch
- [ ] Form component envuelve los fields
- [ ] Todos los labels usan `textGet()`
- [ ] i18n keys agregadas en backend
- [ ] Grid responsive (cols-1 md:cols-2)
- [ ] Max-width y padding aplicados
- [ ] Actions (Cancel/Save) en footer
- [ ] Loading state en botón submit
- [ ] Sin HTML `<input>` directo
