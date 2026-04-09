# Web Frontend — Implementing New Features

Guía paso a paso para agregar una nueva página o feature en el frontend.

## 🎯 Ejemplo: Agregar "Team Management"

Seguiremos este flujo para agregar una nueva feature.

## Paso 1: Crear Tipos (TypeScript)

```typescript
// apps/web/src/types/team-type.ts

export interface Team extends BaseModel {
    name: string;
    slug: string;
    description?: string;
    member_count: number;
}

export interface TeamMember extends BaseModel {
    team_id: number;
    user_id: number;
    role: string; // "admin" | "member"
}

export interface CreateTeamPayload {
    name: string;
    description?: string;
}

export interface UpdateTeamPayload {
    name?: string;
    description?: string;
}
```

**Reglas:**
- Extende `BaseModel` (ID, CreatedAt, UpdatedAt, DeletedAt)
- Usa `snake_case` para propiedades (como vienen del API)
- Separa tipos de request (`CreateTeamPayload`) de response (`Team`)

## Paso 2: Crear Servicio de API

```typescript
// apps/web/src/api/team-service.ts

import { createHttpService } from "./fetch";
import { apiWithTenant } from "./index";
import type { Team, TeamMember, CreateTeamPayload, UpdateTeamPayload } from "@/types/team-type";

const teamService = createHttpService(apiWithTenant);

// ── Teams ────────────────────────────────────────

export const getTeams = async () =>
    teamService.get<Team[]>("/team");

export const getTeamByID = async (id: number) =>
    teamService.get<Team>(`/team/${id}`);

export const createTeam = async (payload: CreateTeamPayload) =>
    teamService.post<Team>("/team", payload, {
        notifySuccess: true,
        notifyError: true,
    });

export const updateTeam = async (id: number, payload: UpdateTeamPayload) =>
    teamService.put<Team>(`/team/${id}`, payload, {
        notifySuccess: true,
        notifyError: true,
    });

export const deleteTeam = async (id: number) =>
    teamService.delete<void>(`/team/${id}`, {
        notifySuccess: true,
        notifyError: true,
    });

// ── Team Members ────────────────────────────────

export const getTeamMembers = async (teamId: number) =>
    teamService.get<TeamMember[]>(`/team/${teamId}/members`);

export const addTeamMember = async (teamId: number, userId: number, role: string) =>
    teamService.post<TeamMember>(`/team/${teamId}/members`, { user_id: userId, role }, {
        notifySuccess: true,
        notifyError: true,
    });

export const removeTeamMember = async (teamId: number, memberId: number) =>
    teamService.delete<void>(`/team/${teamId}/members/${memberId}`, {
        notifySuccess: true,
        notifyError: true,
    });
```

**Reglas:**
- Usa el instance correcto (`api`, `apiWithTenant`, `noAuthApi`)
- Tipifica todos los responses
- Usa `notifySuccess`/`notifyError` en mutaciones
- Una función por operación CRUD

## Paso 3: Crear Store (si es necesario)

```typescript
// apps/web/src/store/team-store.ts

import { create } from "zustand";
import type { Team } from "@/types/team-type";

type TeamStore = {
    selectedTeam?: Team;
    searchTerm: string;
    
    setSelectedTeam: (team: Team) => void;
    setSearchTerm: (term: string) => void;
    clear: () => void;
};

export const useTeamStore = create<TeamStore>((set) => ({
    selectedTeam: undefined,
    searchTerm: "",
    
    setSelectedTeam: (team) => set({ selectedTeam: team }),
    setSearchTerm: (term) => set({ searchTerm: term }),
    clear: () => set({ selectedTeam: undefined, searchTerm: "" }),
}));
```

**Cuándo crear un store:**
- Compartir selección entre múltiples componentes
- Guardar filtros/búsqueda
- Estado de modal/dialog

## Paso 4: Crear Componentes

### Layout principal

```typescript
// apps/web/src/pages/team/page.tsx

import React from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { DashboardLayout } from "@/sections/template/dashboard-template";
import { useText } from "@/hooks/use-text";
import { getTeams, createTeam } from "@/api/team-service";
import { useTeamStore } from "@/store/team-store";
import type { Team } from "@/types/team-type";

function TeamPage() {
    const { textGet } = useText();
    const [teams, setTeams] = React.useState<Team[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [showCreateDialog, setShowCreateDialog] = React.useState(false);
    
    const { selectedTeam, setSelectedTeam } = useTeamStore();

    // Cargar datos
    React.useEffect(() => {
        loadTeams();
    }, []);

    async function loadTeams() {
        setLoading(true);
        const res = await getTeams();
        if (res.success) {
            setTeams(res.data || []);
        }
        setLoading(false);
    }

    async function handleCreate(payload: CreateTeamPayload) {
        const res = await createTeam(payload);
        if (res.success) {
            // Service ya mostró toast
            setShowCreateDialog(false);
            await loadTeams();
        }
    }

    if (loading) {
        return <DashboardLayout><Spinner /></DashboardLayout>;
    }

    return (
        <DashboardLayout>
            <div className="space-y-4">
                <div className="flex justify-between items-center">
                    <h1>{textGet("team.title")}</h1>
                    <Button onClick={() => setShowCreateDialog(true)}>
                        {textGet("team.create")}
                    </Button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {teams.map((team) => (
                        <TeamCard 
                            key={team.id} 
                            team={team}
                            selected={selectedTeam?.id === team.id}
                            onSelect={setSelectedTeam}
                        />
                    ))}
                </div>

                {selectedTeam && (
                    <TeamDetails team={selectedTeam} onClose={() => setSelectedTeam(undefined)} />
                )}

                {showCreateDialog && (
                    <CreateTeamDialog 
                        onCreate={handleCreate}
                        onClose={() => setShowCreateDialog(false)}
                    />
                )}
            </div>
        </DashboardLayout>
    );
}

export default TeamPage;
```

### Componentes secundarios

```typescript
// apps/web/src/pages/team/components/team-card.tsx

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import type { Team } from "@/types/team-type";

interface TeamCardProps {
    team: Team;
    selected?: boolean;
    onSelect?: (team: Team) => void;
}

export function TeamCard({ team, selected, onSelect }: TeamCardProps) {
    return (
        <Card 
            className={selected ? "ring-2 ring-primary" : ""}
            onClick={() => onSelect?.(team)}
        >
            <CardHeader>
                <CardTitle>{team.name}</CardTitle>
            </CardHeader>
            <CardContent>
                <p className="text-sm text-muted-foreground">{team.description}</p>
                <p className="text-xs mt-2">{team.member_count} members</p>
            </CardContent>
        </Card>
    );
}
```

## Paso 5: Crear Rutas (Routing)

```typescript
// apps/web/src/routes/routes.tsx

import { CheckPermission } from "@/components/access-control/check-permission";
import { PERMISSIONS } from "@/lib/constants";
import TeamPage from "@/pages/team/page";

const routes = [
    {
        path: "/team",
        element: (
            <CheckPermission permissions={[PERMISSIONS.TEAM.PERMISSION_READ_TEAM]}>
                <TeamPage />
            </CheckPermission>
        ),
    },
    // ... más rutas
];
```

**Reglas:**
- Envuelve con `<CheckPermission>` si requiere permisos
- Una ruta por página principal

## Paso 6: Agregar a Navegación (si es visible)

```typescript
// apps/web/src/config/nav-config.ts

import { Users } from "lucide-react";

export const createNavItems = (textGet, enabledFeatures) => [
    // ...
    {
        icon: Users,
        label: textGet("team.title"),
        href: "/team",
        permission: PERMISSIONS.TEAM.PERMISSION_READ_TEAM,
        feature: "team",  // ← Se filtra automáticamente
    },
    // ...
];
```

**Reglas:**
- `feature` debe coincidir con categoria de permisos
- `permission` es el mínimo requerido para ver el item

## Paso 7: Agregar i18n Keys

```json
// apps/api/i18n/messages/messages_es.json

{
    "team": {
        "title": "Equipo",
        "create": "Crear Equipo",
        "created": "Equipo creado exitosamente",
        "updated": "Equipo actualizado",
        "deleted": "Equipo eliminado",
        "description_placeholder": "Descripción del equipo...",
        "members": "Miembros",
        "add_member": "Agregar Miembro",
        "remove_member": "Remover Miembro"
    }
}

// apps/api/i18n/messages/messages_en.json

{
    "team": {
        "title": "Team",
        "create": "Create Team",
        ...
    }
}
```

## Paso 8: Testing (opcional)

```typescript
// apps/web/src/pages/team/__tests__/page.test.tsx

import { render, screen } from "@testing-library/react";
import TeamPage from "../page";
import * as teamService from "@/api/team-service";

jest.mock("@/api/team-service");
jest.mock("@/hooks/use-text", () => ({
    useText: () => ({ textGet: (key: string) => key }),
}));

it("renders teams", async () => {
    jest.mocked(teamService.getTeams).mockResolvedValue({
        success: true,
        data: [{ id: 1, name: "Team A" }],
    });

    render(<TeamPage />);
    expect(await screen.findByText("Team A")).toBeInTheDocument();
});
```

## Checklist de Implementación

- [ ] Tipos creados (`types/team-type.ts`)
- [ ] Servicio creado (`api/team-service.ts`)
- [ ] Store creado si es necesario (`store/team-store.ts`)
- [ ] Página principal creada (`pages/team/page.tsx`)
- [ ] Componentes secundarios creados
- [ ] Rutas registradas en `routes/routes.tsx`
- [ ] Navegación actualizada si es visible
- [ ] i18n keys agregadas en backend (JSON)
- [ ] Permisos asignados (backend + frontend)
- [ ] Tests escritos

## Estructura de Carpetas Final

```
pages/team/
├── page.tsx                 # Página principal
├── components/
│   ├── team-card.tsx
│   ├── team-details.tsx
│   └── create-team-dialog.tsx
└── __tests__/
    └── page.test.tsx

types/
└── team-type.ts

api/
└── team-service.ts

store/
└── team-store.ts
```
