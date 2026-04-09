# Backoffice — Domains & Operations

Guía detallada de los dominios administrativos principales.

## 🏢 Companies (Empresas)

Gestión de clientes/empresas que usan la plataforma.

### Estructura

```typescript
interface Company {
    id: number;
    legal_name: string;      // Razón Social
    trade_name: string;      // Nombre Comercial
    plan_code: string;       // Plan actual (ENT, PRO, BASIC)
    tenant_id: number;       // Pertenece a este tenant
    tenant: Tenant;
    subscriptions?: Subscription[];
}

interface Tenant {
    id: number;
    name: string;
    slug: string;
    enabled_features: string; // JSON: { clinical: true, billing: true, team: false }
    created_at: string;
}
```

### Operaciones Comunes

**Listar empresas:**
```typescript
const res = await getCompanies();
if (res.success) {
    // res.data = Company[]
    // Mostrar en tabla con nombre, plan, tenant
}
```

**Ver detalles:**
```typescript
const res = await getCompanyByID(companyId);
if (res.success) {
    // Mostrar tenant info
    // Listar suscripciones activas
    // Opción para cambiar plan
}
```

**Filtrados:**
```typescript
// Buscar por nombre, tenant, plan
const filtered = companies.filter(
    c => c.trade_name.includes(searchTerm)
      && (!tenantFilter || c.tenant_id === tenantFilter)
      && (!planFilter || c.plan_code === planFilter)
);
```

### Key Features

- Una empresa = 1 tenant (1 tenant = múltiples empresas)
- El plan determina qué features están habilitados
- Las subscripciones históricas están en el modelo

---

## 📋 Plans (Planes)

Definición de planes de suscripción y sus features.

### Estructura

```typescript
interface Plan {
    id: number;
    name: string;           // "Enterprise", "Pro", "Basic"
    code: string;           // Código único: ENT, PRO, BASIC
    price: number;          // Precio en USD
    Features: Feature[];    // Features asignados a este plan
    Properties: {
        enabled_features?: {    // Auto-calculado según features
            clinical: boolean;
            billing: boolean;
            team: boolean;
        };
        max_users: number;      // -1 = ilimitado
        max_patients: number;
        max_offices: number;
    };
}

interface Feature {
    id: number;
    code: string;           // "bitacora_clinica", "facturacion", "equipo"
    name: string;
    code: string;           // CLINICAL, BILLING, TEAM
    Permissions: Permission[];
}
```

### Operaciones Comunes

**Listar planes:**
```typescript
const res = await getPlans();
// Mostrar tabla: nombre, código, precio, features count, enabled_features
```

**Crear plan:**
```typescript
const res = await createPlan({
    name: "Professional",
    code: "PROF",
    price: 199.99,
    feature_codes: ["bitacora_clinica", "facturacion"],  // Select features
    properties: {
        max_users: 50,
        max_patients: 5000,
        max_offices: 10,
    }
});

// Backend calcula automáticamente enabled_features:
// Si seleccionas features con permisos CLINICAL + BILLING
// → enabled_features: { clinical: true, billing: true, team: false }
```

**Editar plan:**
```typescript
const res = await updatePlan(planId, {
    name: "New Name",
    feature_codes: [...nuevos features...],
    properties: { max_users: 100 }
});

// Backend recalcula enabled_features automáticamente
```

**Eliminar:**
```typescript
const res = await deletePlan(planId);
// Validar que no hay empresas usando este plan
```

### Features Habilitados (Automático)

**No es manual.** El backend calcula automáticamente:

```typescript
// En el formulario, seleccionas features:
selectedFeatures = ["clinical", "billing"]  // Códigos de features

// El backend en updatePlan():
// 1. Obtiene features seleccionados
// 2. Extrae sus permisos (qué categorías tienen)
// 3. Calcula: clinical: true, billing: true, team: false
// 4. Guarda en plan.properties.enabled_features
```

Luego cuando se asigna el plan a una empresa:

```typescript
// En subscription handler:
// Copia plan.properties.enabled_features → tenant.enabled_features
// Y el frontend filtra automáticamente nav items
```

---

## 🔗 Subscriptions (Suscripciones)

Asignar planes a empresas.

### Estructura

```typescript
interface Subscription {
    id: number;
    company_id: number;
    plan_code: string;          // Código del plan actual
    status: "active" | "cancelled" | "paused";
    expires_at: string;         // Fecha de vencimiento
    created_at: string;
    updated_at: string;
    Plan?: Plan;
    Company?: Company;
}
```

### Operaciones Comunes

**Listar suscripciones:**
```typescript
const res = await getSubscriptions();
// Mostrar: empresa, plan, status, fecha expiración
```

**Crear (asignar plan a empresa):**
```typescript
const res = await createSubscription({
    company_id: 123,
    plan_code: "ENTERPRISE",
    status: "active",
    expires_at: "2027-04-09T00:00:00Z",
});

// Backend automáticamente:
// 1. Aplica plan.properties.enabled_features al tenant
// 2. Filtra nav items en frontend
```

**Cambiar plan (upgrade/downgrade):**
```typescript
const res = await updateSubscription(subscriptionId, {
    plan_code: "PRO",
    expires_at: "2027-04-09T00:00:00Z",
});

// Backend recalcula enabled_features del tenant
```

**Cancelar:**
```typescript
const res = await deleteSubscription(subscriptionId);
// Establece status = "cancelled"
```

### Flujo Completo

1. Admin crea plan con features específicos
   → Backend calcula `enabled_features` automáticamente

2. Admin asigna plan a empresa (crea subscription)
   → Backend copia `enabled_features` al tenant

3. Usuario inicia sesión en app
   → Frontend lee `tenant.enabled_features`
   → Filtra nav items automáticamente

---

## 👥 Roles (Roles Globales)

Roles y permisos para usuarios del sistema.

### Estructura

```typescript
interface Role {
    id: number;
    role: string;              // "admin", "user", "moderator"
    Permissions: Permission[];
}

interface Permission {
    id: string;               // "READ_PATIENT", "CREATE_BILLING", etc
    name: string;
    category: string;         // "CLINICAL", "BILLING", "TEAM"
    description: string;
}
```

### Operaciones

**Listar roles:**
```typescript
const res = await getRoles();
// Mostrar: nombre, número de permisos
```

**Ver detalles:**
```typescript
const res = await getRoleByID(roleId);
// Mostrar permisos asignados
// Opción para agregar/quitar permisos
```

**Crear rol:**
```typescript
const res = await createRole({
    role: "manager",
});
// Inicialmente sin permisos, luego agregar
```

---

## 🎨 Features (Features Disponibles)

Definición de features que se pueden asignar a planes.

### Estructura

```typescript
interface Feature {
    id: number;
    code: string;           // Identificador único
    name: string;           // Nombre visible
    Permissions: Permission[];  // Permisos que incluye
}
```

**Importante:** Features y Permissions son diferentes
- **Feature** = Módulo administrativo (bitácora clínica, facturación, equipo)
- **Permission** = Acción específica (READ_PATIENT, CREATE_BILLING)

### Operaciones

**Listar features:**
```typescript
const res = await getFeatures();
// Mostrar: nombre, categoría de permisos, usado en X planes
```

**Crear feature:**
```typescript
const res = await createFeature({
    code: "team_management",
    name: "Team Management",
    category: "TEAM",
});
```

---

## 👤 Users (Usuarios del Sistema)

Gestión de usuarios que acceden al backoffice.

### Estructura

```typescript
interface User {
    id: number;
    user_name: string;
    email: string;
    // password: no se devuelve en API
    Environments: Environment[];  // Qué empresas/roles tiene
}

interface Environment {
    id: number;
    user_id: number;
    company_id: number;
    role_id: number;
    role: Role;
    company: Company;
}
```

### Operaciones

**Listar usuarios:**
```typescript
const res = await getUsers();
// Mostrar: username, email, número de environments
```

**Ver detalles:**
```typescript
const res = await getUserByID(userId);
// Mostrar environments (empresa + rol)
// Opción para agregar environment
```

**Crear usuario:**
```typescript
const res = await createUser({
    user_name: "john.doe",
    email: "john@company.com",
    password: "secure123",
});
// Se crea sin environments, luego agregar
```

**Agregar a empresa:**
```typescript
const res = await addUserToCompany({
    user_id: userId,
    company_id: companyId,
    role_id: roleId,
});
// Crea un Environment
```

---

## 🔒 Permissions (Permisos Globales)

Permisos centralizados para toda la plataforma.

### Estructura

```typescript
interface Permission {
    id: string;           // "READ_PATIENT"
    name: string;
    category: string;     // "CLINICAL", "BILLING", "TEAM"
    description: string;
}
```

### Operaciones

**Listar todos:**
```typescript
const res = await getPermissions();
// Agrupar por categoría para mostrar
```

### Relación con Features

```
Permissions (granular)
    ↓
Features (agrupan permisos)
    ↓
Plans (asignan features)
    ↓
Subscriptions (dan plan a empresa)
    ↓
Tenant.enabled_features (habilita features en frontend)
```

---

## 📊 Diagrama de Relaciones

```
Plan
├── feature_codes[] ─→ Feature
│   └── permissions[] ─→ Permission (categoría)
├── properties.enabled_features (AUTO-CALCULADO)
│   └── Basado en categorías de permisos en features
└── price, limits, etc

    ↓ Create Subscription

Subscription
├── plan_code → Plan
└── company_id → Company
    └── tenant_id → Tenant
        └── enabled_features (COPIA de Plan.properties.enabled_features)
            ↓
            Frontend filtra nav items automáticamente
```

---

## Checklist: Agregar Nueva Funcionalidad

```
1. Crear permisos en backend
   □ Agregado en permission-data.go (CLINICAL, BILLING, TEAM, etc)
   □ Migración que asigna al admin

2. Crear feature (opcional, si es un módulo)
   □ Crear Feature en backoffice
   □ Asignar permisos

3. Crear plan o actualizar
   □ Plan incluye feature
   □ Backend auto-calcula enabled_features
   □ Subscription copia a tenant

4. Frontend filtra
   □ Nav items se ocultan según enabled_features
   □ No necesita cambios (automático)
```
