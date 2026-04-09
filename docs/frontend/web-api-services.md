# Web Frontend — API Services

Cómo crear y usar servicios de API en el frontend React.

## 🏗️ Estructura de Servicios

Los servicios son la **única forma** de comunicarse con el backend desde componentes. Nunca uses axios directamente en componentes.

```typescript
// ❌ NUNCA hagas esto en un componente:
const res = await apiWithTenant.get("/patients");

// ✅ SIEMPRE crea un servicio:
import { getPatients } from "@/api/clinical-service";
const res = await getPatients();
```

## HttpService Class

Base para todos los servicios:

```typescript
// apps/web/src/api/fetch.ts

export class HttpService {
    private axios: AxiosInstance;

    constructor(axios: AxiosInstance) {
        this.axios = axios;
    }

    async get<T>(url: string, options?: RequestOptions): Promise<ApiResponse<T>>
    async post<T>(url: string, data?: any, options?: RequestOptions): Promise<ApiResponse<T>>
    async put<T>(url: string, data?: any, options?: RequestOptions): Promise<ApiResponse<T>>
    async delete<T>(url: string, options?: RequestOptions): Promise<ApiResponse<T>>
}

interface RequestOptions {
    notifySuccess?: boolean;  // Mostrar toast de éxito
    notifyError?: boolean;    // Mostrar toast de error
}

interface ApiResponse<T> {
    success: boolean;
    data?: T;
    message?: string;
    code?: string;
}
```

## Axios Instances

```typescript
// apps/web/src/api/index.ts

// 1. Para rutas de auth (sin tenant header)
export const api = axios.create({
    baseURL: import.meta.env.VITE_API_URL,
    headers: {
        "Content-Type": "application/json",
    },
});

// 2. Para rutas tenant-scoped (la mayoría)
export const apiWithTenant = axios.create({
    baseURL: import.meta.env.VITE_API_URL,
    headers: {
        "Content-Type": "application/json",
        "X-Tenant-Slug": /* dinámico desde sesión */,
    },
});

// 3. Para rutas públicas (sin auth)
export const noAuthApi = axios.create({
    baseURL: import.meta.env.VITE_API_URL,
    headers: {
        "Content-Type": "application/json",
    },
});
```

### Cuándo usar cada una

| Instance | Cuándo | Ejemplo |
|----------|--------|---------|
| `api` | Routes de auth / login | POST /auth/login |
| `apiWithTenant` | Rutas con tenant_id | GET /patients, POST /billing/invoices |
| `noAuthApi` | Rutas públicas | GET /public/health |

## Crear un Servicio

### Paso 1: Identificar el dominio

```typescript
// apps/web/src/api/team-service.ts
import { createHttpService } from "./fetch";
import { apiWithTenant } from "./index";

const teamService = createHttpService(apiWithTenant);
```

### Paso 2: Crear funciones por operación

```typescript
// GET
export const getTeams = async () =>
    teamService.get<Team[]>("/team");

// GET by ID
export const getTeamByID = async (id: string) =>
    teamService.get<Team>(`/team/${id}`);

// POST
export const createTeam = async (payload: CreateTeamPayload) =>
    teamService.post<Team>("/team", payload, {
        notifySuccess: true,
        notifyError: true,
    });

// PUT
export const updateTeam = async (id: string, payload: UpdateTeamPayload) =>
    teamService.put<Team>(`/team/${id}`, payload, {
        notifySuccess: true,
        notifyError: true,
    });

// DELETE
export const deleteTeam = async (id: string) =>
    teamService.delete<void>(`/team/${id}`, {
        notifySuccess: true,
        notifyError: true,
    });
```

### Paso 3: Usar en componentes

```typescript
import { createTeam, getTeams } from "@/api/team-service";

function TeamPage() {
    const [teams, setTeams] = React.useState<Team[]>([]);

    React.useEffect(() => {
        getTeams().then((res) => {
            if (res.success) setTeams(res.data || []);
        });
    }, []);

    async function handleCreate(payload: CreateTeamPayload) {
        const res = await createTeam(payload);
        if (res.success) {
            // Service ya mostró toast de éxito
            // Refresca lista
            const updated = await getTeams();
            if (updated.success) setTeams(updated.data || []);
        }
    }

    return (
        <div>
            {teams.map((team) => (
                <TeamCard key={team.id} team={team} />
            ))}
        </div>
    );
}
```

## Toast Notifications

**El servicio controla los toasts**, no el componente.

```typescript
// En el servicio:
export const createInvoice = async (payload: CreateInvoicePayload) =>
    invoiceService.post<Invoice>("/billing/invoices", payload, {
        notifySuccess: true,   // ← Service muestra "Invoice created"
        notifyError: true,     // ← Service muestra el error
    });

// En el componente:
const res = await createInvoice(payload);
if (res.success) {
    // No hagas: useToast().toast({ ... })
    // El servicio ya lo hizo
    navigate("/billing");
}
```

**Excepción**: Validación de UI local (no API)

```typescript
// Esto SÍ va en el componente (no es resultado de API):
const { toast } = useToast();

if (!selectedPatient) {
    toast({
        title: "Error",
        description: "Please select a patient",
        variant: "destructive",
    });
}
```

## Error Handling

El servicio maneja errores automáticamente:

```typescript
const res = await createInvoice(payload);

// res.success = false
// res.message = Error message (i18n desde backend)
// res.code = Error code (ej: "ERR_BILLING_001")

if (!res.success) {
    // Service ya mostró toast de error
    console.log("Error:", res.code);
}
```

## Tipos de Datos

### Response Types

Siempre tipifica el response:

```typescript
// ✅ Tipificado
const getInvoices = async () =>
    invoiceService.get<Invoice[]>("/billing/invoices");

// ❌ Sin tipificar (evita)
const getInvoices = async () =>
    invoiceService.get("/billing/invoices");
```

### Payload DTOs

```typescript
// apps/web/src/types/[domain]-type.ts

export interface CreateInvoicePayload {
    patient_id: number;
    date: string;
    items: {
        description: string;
        amount: number;
    }[];
}

export interface UpdateInvoicePayload {
    status?: string;
    notes?: string;
}

// En el servicio:
export const createInvoice = async (payload: CreateInvoicePayload) =>
    invoiceService.post<Invoice>("/billing/invoices", payload, {
        notifySuccess: true,
    });
```

## Patrón Completo: Clinical Service

```typescript
// apps/web/src/api/clinical-service.ts

import { createHttpService } from "./fetch";
import { apiWithTenant } from "./index";
import type { Patient, PatientCreatePayload, PatientUpdatePayload } from "@/types/clinical-type";

const clinicalService = createHttpService(apiWithTenant);

// ── Patients ────────────────────────────────────────────

export const getPatients = async () =>
    clinicalService.get<Patient[]>("/clinical/patients");

export const getPatientByID = async (id: number) =>
    clinicalService.get<Patient>(`/clinical/patients/${id}`);

export const createPatient = async (payload: PatientCreatePayload) =>
    clinicalService.post<Patient>("/clinical/patients", payload, {
        notifySuccess: true,
        notifyError: true,
    });

export const updatePatient = async (id: number, payload: PatientUpdatePayload) =>
    clinicalService.put<Patient>(`/clinical/patients/${id}`, payload, {
        notifySuccess: true,
        notifyError: true,
    });

export const deletePatient = async (id: number) =>
    clinicalService.delete<void>(`/clinical/patients/${id}`, {
        notifySuccess: true,
        notifyError: true,
    });

// ── Appointments ────────────────────────────────────────

export const getAppointments = async () =>
    clinicalService.get<Appointment[]>("/clinical/appointments");

export const createAppointment = async (payload: AppointmentCreatePayload) =>
    clinicalService.post<Appointment>("/clinical/appointments", payload, {
        notifySuccess: true,
        notifyError: true,
    });
```

## Prácticas Recomendadas

### ✅ DO

- Crear un servicio por dominio
- Tipificar todos los responses
- Usar `notifySuccess`/`notifyError` en mutaciones (POST, PUT, DELETE)
- Manejar errores en componentes verificando `res.success`
- Usar constantes para URLs si se repiten

### ❌ DON'T

- Usar axios directamente en componentes
- Hardcodear URLs (`"/billing/invoices"` → crear function)
- Ignorar el `success` flag del response
- Mostrar toast en componente si el servicio ya lo hizo
- Mezclar lógica de API con lógica de componente

## Testing

```typescript
// Para tests, mockea el servicio:
jest.mock("@/api/team-service", () => ({
    getTeams: jest.fn().mockResolvedValue({
        success: true,
        data: [{ id: 1, name: "Team A" }],
    }),
}));
```
