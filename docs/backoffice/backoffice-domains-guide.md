# Backoffice — Domains & Operations

Qué administra cada sección del backoffice y cómo se conectan. Modelos en
`apps/api/features/companies/models/` salvo que se indique otro lugar.

## 📊 Diagrama de relaciones

```
Company ──1:1── Tenant                  (la clínica; sus datos llevan TenantID)
   │
   ├── Environments ── User + Role      (miembros de la empresa y su rol)
   │
   └── Subscriptions ── Plan (por PlanCode)
                          ├── Features ── Permissions (con Category)
                          ├── Pricings (precio por meses)
                          └── Properties (límites: max_users, max_patients, max_offices)

Role ── Permissions                     (roles globales, compartidos por todas las empresas)
BackofficeUser                          (admins de Pengi; login propio)
Announcement                            (avisos a todos, a una empresa o a un usuario)
```

Qué ve un usuario de la app = permisos de su **rol** ∩ permisos de los
**features del plan** de su empresa. Detalle en
[`../backend/permissions-system.md`](../backend/permissions-system.md).

## 🏢 Companies (Empresas)

`Company {LegalName, TradeName, PlanCode, TenantID, OwnerUserID}`; cada empresa
tiene un `Tenant`.

- CRUD: `/backoffice/companies` (`pages/companies/*`).
- Links de alta: token de registro de empresa nueva
  (`POST /register-token`) y de signup para sumar usuarios a una existente
  (`GET /:id/signup-token`).
- Usuarios de la empresa (`pages/companies/company-users.tsx`): listar, cambiar
  rol, generar link de reseteo de contraseña, quitar.

## 📋 Plans (Planes)

`Plan {Name, Code (único), Tier, CanRenew, Price, Features, Pricings, Properties}`.

- CRUD: `/backoffice/plans` (`pages/plans/*`). El request lleva
  `feature_codes`, `pricings` (`{months, price}`) y `properties`.
- **Límites** (`plan-limits-editor.tsx`): `max_users`, `max_patients`,
  `max_offices` en `Properties`; la app los consulta con
  `GetPlanLimitForCompany` / `ExceedsPlanLimit`.
- **Precios por período** (`plan-pricings-editor.tsx`): `PlanPricing` por
  cantidad de meses.
- `Tier` ordena los planes para upgrades/downgrades; `CanRenew: false` impide
  renovar (plan `TRIAL`, único sembrado en código).

### Features habilitados

Al guardar, el backend guarda en `Properties["enabled_features"]` un resumen
calculado desde los features elegidos (para mostrarlo). **La app no lo lee**:
en cada request recalcula `enabled_features` desde las categorías de los
permisos de los features del plan activo
(`company_services.CalculateEnabledFeaturesFromFeatures`). Cambiar los features
de un plan afecta a sus empresas de inmediato, sin tocar suscripciones ni
tenants.

## 🧩 Features

`Feature {Code (único), Name, Permissions}` — un paquete de permisos que se
vende dentro de un plan.

- CRUD: `/backoffice/features` (`pages/features/*`), con `PermissionPicker`, que
  agrupa los permisos por `Category` (una categoría nueva no requiere código).
- No se siembran en código: cada ambiente (dev, prod) los tiene en su BD.
- Cuando una feature de la app agrega permisos a un dominio con Feature
  existente, una code-migration los asocia (patrón
  `add_medical_document_feature_permissions.go`); si no, hay que agregarlos
  aquí a mano en cada ambiente.

## 🔗 Subscriptions (Suscripciones)

`Subscription {CompanyID, PlanCode, Status, ExpiresAt, NextPlanCode, PlanChangeAt}`.

- CRUD: `/backoffice/subscriptions` (`pages/subscriptions/*`); por empresa:
  `GET /backoffice/subscriptions/company/:id`.
- La app exige una suscripción `active` que no haya vencido hace más de 3 días
  (`SubscriptionMiddleware`); sin ella, los endpoints con plan responden 403.
- `NextPlanCode` + `PlanChangeAt` programan un cambio de plan.
- Pagos: `SubscriptionPayment` (dLocal); `/backoffice/payments` lista y genera
  pagos.

## 👥 Roles

`Role {Role, Permissions}` (`apps/api/features/users/models/user-model.go`).

- CRUD: `/backoffice/roles` (`pages/roles/*`) con `PermissionPicker`.
- Los roles son **globales**: editar uno cambia lo que puede hacer ese rol en
  todas las empresas.
- Roles canónicos: `admin` (todos los permisos), `doctor`, `recepcionista`,
  `contador` (según `RolePermissionMatrix`, `features/users/data/role-data.go`).

## 🔒 Permissions

Catálogo de solo lectura (`GET /backoffice/permissions`), definido en código en
`features/permissions/data/permission-data.go` y sembrado por migraciones. El
backoffice lo usa para los selectores de roles y features.

## 👤 Users (Admins del backoffice)

`BackofficeUser {Name, UserName, Password}` (`features/backoffice/models/`).

- CRUD: `/backoffice/users` (`pages/users/*`).
- Login: `/backoffice/auth/login` (rate limited), refresh por cookie, logout.
- Son distintos de los usuarios de la app; estos se gestionan desde
  *Companies → usuarios*.

## 📣 Announcements (Anuncios)

`notifications_models.Announcement {scope, company_id, user_id, title, body, level, action_url, scheduled_at, status}`.

- `/backoffice/announcements` (`pages/announcements/*`): crear y cancelar.
- `scope`: `global`, `company` o `user`; `level`: `info`, `success`,
  `warning`, `critical`.
- Con `scheduled_at` futuro los envía el `AnnouncementScheduler` (arrancado en
  `cmd/main.go`); sin él, se envían al crearlos como notificaciones de la app.

## 📈 Dashboard

`GET /backoffice/dashboard` (`pages/home/home.tsx`): métricas globales.
