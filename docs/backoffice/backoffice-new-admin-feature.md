# Backoffice — Implementing Admin Features

Guía paso a paso para agregar nuevas operaciones administrativas.

## 🎯 Ejemplo: Agregar "Feature Management"

Agregaremos una página para crear/editar/listar features administrativos.

## Paso 1: Crear Tipos (TypeScript)

```typescript
// apps/backoffice/src/types/feature-type.ts

export interface Feature extends BaseModel {
    code: string;           // Identificador único
    name: string;
    Permissions?: Permission[];
}

export interface Permission {
    id: string;
    name: string;
    category: string;
}

export interface CreateFeaturePayload {
    code: string;
    name: string;
    permission_ids: string[];  // Permisos a incluir
}

export interface UpdateFeaturePayload {
    name?: string;
    permission_ids?: string[];
}
```

## Paso 2: Crear Servicio

```typescript
// apps/backoffice/src/api/feature-service.ts

import { createHttpService } from "./fetch";
import { api } from "./index";
import type { Feature, CreateFeaturePayload, UpdateFeaturePayload } from "@/types/feature-type";

const featureService = createHttpService(api);

// ── Features ────────────────────────────────────

export const getFeatures = async () =>
    featureService.get<Feature[]>("/backoffice/features");

export const getFeatureByID = async (id: number) =>
    featureService.get<Feature>(`/backoffice/features/${id}`);

export const createFeature = async (payload: CreateFeaturePayload) =>
    featureService.post<Feature>("/backoffice/features", payload, {
        notifySuccess: true,
        notifyError: true,
    });

export const updateFeature = async (id: number, payload: UpdateFeaturePayload) =>
    featureService.put<Feature>(`/backoffice/features/${id}`, payload, {
        notifySuccess: true,
        notifyError: true,
    });

export const deleteFeature = async (id: number) =>
    featureService.delete<void>(`/backoffice/features/${id}`, {
        notifySuccess: true,
        notifyError: true,
    });
```

## Paso 3: Crear Store (si es necesario)

```typescript
// apps/backoffice/src/store/feature-store.ts

import { create } from "zustand";
import type { Feature } from "@/types/feature-type";

type FeatureStore = {
    selectedFeature?: Feature;
    searchTerm: string;
    
    setSelectedFeature: (feature: Feature) => void;
    setSearchTerm: (term: string) => void;
    clear: () => void;
};

export const useFeatureStore = create<FeatureStore>((set) => ({
    selectedFeature: undefined,
    searchTerm: "",
    
    setSelectedFeature: (feature) => set({ selectedFeature: feature }),
    setSearchTerm: (term) => set({ searchTerm: term }),
    clear: () => set({ selectedFeature: undefined, searchTerm: "" }),
}));
```

## Paso 4: Crear Página Principal

```typescript
// apps/backoffice/src/pages/features/list.tsx

import React from "react";
import { useNavigate } from "react-router";
import { Button } from "@/components/ui/button";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useText } from "@/hooks/use-text";
import { getFeatures, deleteFeature } from "@/api/feature-service";
import { useFeatureStore } from "@/store/feature-store";
import type { Feature } from "@/types/feature-type";

function FeatureListPage() {
    const { textGet } = useText();
    const navigate = useNavigate();
    const [features, setFeatures] = React.useState<Feature[]>([]);
    const [loading, setLoading] = React.useState(true);
    
    const { searchTerm, setSearchTerm } = useFeatureStore();

    React.useEffect(() => {
        loadFeatures();
    }, []);

    async function loadFeatures() {
        setLoading(true);
        const res = await getFeatures();
        if (res.success) {
            setFeatures(res.data || []);
        }
        setLoading(false);
    }

    async function handleDelete(featureId: number) {
        if (!confirm(textGet("backoffice.features.confirm_delete"))) return;
        
        const res = await deleteFeature(featureId);
        if (res.success) {
            await loadFeatures();
        }
    }

    const filtered = features.filter(f => 
        f.code.toLowerCase().includes(searchTerm.toLowerCase()) ||
        f.name.toLowerCase().includes(searchTerm.toLowerCase())
    );

    if (loading) {
        return <DashboardLayout><div>Loading...</div></DashboardLayout>;
    }

    return (
        <DashboardLayout>
            <div className="space-y-4">
                <div className="flex justify-between items-center">
                    <h1>{textGet("backoffice.features.title")}</h1>
                    <Button onClick={() => navigate("/features/create")}>
                        {textGet("backoffice.features.create")}
                    </Button>
                </div>

                <input
                    type="text"
                    placeholder={textGet("common.search")}
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="px-4 py-2 border rounded"
                />

                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{textGet("backoffice.features.col.code")}</TableHead>
                            <TableHead>{textGet("backoffice.features.col.name")}</TableHead>
                            <TableHead>{textGet("backoffice.features.col.permissions")}</TableHead>
                            <TableHead>{textGet("common.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {filtered.map((feature) => (
                            <TableRow key={feature.id}>
                                <TableCell className="font-mono">{feature.code}</TableCell>
                                <TableCell>{feature.name}</TableCell>
                                <TableCell>{feature.Permissions?.length || 0}</TableCell>
                                <TableCell className="space-x-2">
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={() => navigate(`/features/${feature.id}/edit`)}
                                    >
                                        {textGet("common.edit")}
                                    </Button>
                                    <Button
                                        variant="destructive"
                                        size="sm"
                                        onClick={() => handleDelete(feature.id)}
                                    >
                                        {textGet("common.delete")}
                                    </Button>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </div>
        </DashboardLayout>
    );
}

export default FeatureListPage;
```

## Paso 5: Crear Página de Crear/Editar

```typescript
// apps/backoffice/src/pages/features/create.tsx

import React from "react";
import { useNavigate } from "react-router";
import z from "zod";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Form } from "@/components/forms/form";
import { FormInput } from "@/components/forms/form-input";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useText } from "@/hooks/use-text";
import { createFeature, getPermissions } from "@/api/feature-service";
import type { Permission } from "@/types/feature-type";

const formSchema = z.object({
    code: z.string().min(2),
    name: z.string().min(2),
});

function CreateFeaturePage() {
    const { textGet } = useText();
    const navigate = useNavigate();
    const [loading, setLoading] = React.useState(false);
    const [permissions, setPermissions] = React.useState<Permission[]>([]);
    const [selectedPermissions, setSelectedPermissions] = React.useState<string[]>([]);

    React.useEffect(() => {
        loadPermissions();
    }, []);

    async function loadPermissions() {
        const res = await getPermissions();
        if (res.success) {
            setPermissions(res.data || []);
        }
    }

    async function onSubmit(values: z.infer<typeof formSchema>) {
        setLoading(true);
        const res = await createFeature({
            ...values,
            permission_ids: selectedPermissions,
        });
        setLoading(false);
        
        if (res.success) {
            navigate("/features");
        }
    }

    return (
        <DashboardLayout>
            <div className="max-w-2xl mx-auto">
                <Form
                    schema={formSchema}
                    onSubmit={onSubmit}
                    defaultValues={{ code: "", name: "" }}
                >
                    {(field) => (
                        <Card>
                            <CardHeader>
                                <CardTitle>{textGet("backoffice.features.create")}</CardTitle>
                            </CardHeader>
                            <CardContent className="space-y-4">
                                <FormInput
                                    field={field}
                                    name="code"
                                    label={textGet("backoffice.features.col.code")}
                                    placeholder="feature_code"
                                />
                                <FormInput
                                    field={field}
                                    name="name"
                                    label={textGet("backoffice.features.col.name")}
                                    placeholder="Feature Name"
                                />

                                <div className="border-t pt-4">
                                    <Label>{textGet("backoffice.features.col.permissions")}</Label>
                                    <div className="space-y-2 mt-2 max-h-48 overflow-y-auto">
                                        {permissions.map((perm) => (
                                            <div key={perm.id} className="flex items-center gap-2">
                                                <Checkbox
                                                    checked={selectedPermissions.includes(perm.id)}
                                                    onCheckedChange={(checked) => {
                                                        if (checked) {
                                                            setSelectedPermissions([
                                                                ...selectedPermissions,
                                                                perm.id,
                                                            ]);
                                                        } else {
                                                            setSelectedPermissions(
                                                                selectedPermissions.filter(
                                                                    (id) => id !== perm.id
                                                                )
                                                            );
                                                        }
                                                    }}
                                                />
                                                <div>
                                                    <Label className="cursor-pointer">
                                                        {perm.name}
                                                    </Label>
                                                    <p className="text-xs text-muted-foreground">
                                                        {perm.category}
                                                    </p>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                </div>

                                <div className="flex gap-2">
                                    <Button
                                        type="button"
                                        variant="outline"
                                        onClick={() => navigate("/features")}
                                    >
                                        {textGet("common.cancel")}
                                    </Button>
                                    <Button type="submit" disabled={loading}>
                                        {textGet("common.create")}
                                    </Button>
                                </div>
                            </CardContent>
                        </Card>
                    )}
                </Form>
            </div>
        </DashboardLayout>
    );
}

export default CreateFeaturePage;
```

## Paso 6: Registrar Rutas

```typescript
// apps/backoffice/src/routes/routes.tsx

import FeatureListPage from "@/pages/features/list";
import CreateFeaturePage from "@/pages/features/create";

export const routes = [
    {
        path: "/features",
        element: <FeatureListPage />,
    },
    {
        path: "/features/create",
        element: <CreateFeaturePage />,
    },
    // ...más rutas
];
```

## Paso 7: Agregar a Navegación

```typescript
// apps/backoffice/src/config/nav-config.ts

import { Layers } from "lucide-react";

export const navItems = [
    // ...
    {
        icon: Layers,
        label: textGet("backoffice.features.title"),
        href: "/features",
    },
    // ...
];
```

## Paso 8: Agregar i18n Keys

```json
// apps/api/i18n/messages/messages_es.json

{
    "backoffice": {
        "features": {
            "title": "Features",
            "create": "Crear Feature",
            "created": "Feature creado exitosamente",
            "updated": "Feature actualizado",
            "deleted": "Feature eliminado",
            "confirm_delete": "¿Eliminar este feature?",
            "col": {
                "code": "Código",
                "name": "Nombre",
                "permissions": "Permisos"
            }
        }
    }
}
```

## Checklist Completo

```
Paso a paso:

Types
□ CreateFeaturePayload
□ UpdateFeaturePayload
□ Feature interface

Services
□ getFeatures()
□ createFeature()
□ updateFeature()
□ deleteFeature()

Store
□ useFeatureStore creado
□ Acciones definidas

Pages
□ List/table page
□ Create/edit form
□ Detail page (opcional)

Routing
□ Rutas registradas en routes.tsx
□ Links correctos

Navigation
□ Agregado a nav-config.ts
□ Icon seleccionado

i18n
□ Keys en messages_es.json
□ Keys en messages_en.json

Testing
□ Tests básicos para servicios
□ Tests para componentes
```

## Patrón para Operaciones CRUD

```typescript
// ── LIST ────────────────────────────────────
async function handleList() {
    const res = await getFeatures();
    if (res.success) setItems(res.data || []);
}

// ── CREATE ────────────────────────────────────
async function handleCreate(payload) {
    const res = await createFeature(payload);
    if (res.success) {
        navigate("/features");
        // Service ya mostró toast
    }
}

// ── UPDATE ────────────────────────────────────
async function handleUpdate(id, payload) {
    const res = await updateFeature(id, payload);
    if (res.success) {
        await handleList(); // Refresca
    }
}

// ── DELETE ────────────────────────────────────
async function handleDelete(id) {
    if (!confirm("¿Eliminar?")) return;
    const res = await deleteFeature(id);
    if (res.success) {
        await handleList();
    }
}
```
