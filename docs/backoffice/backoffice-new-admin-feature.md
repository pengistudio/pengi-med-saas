# Backoffice — Implementing Admin Features

Paso a paso para agregar una sección nueva al backoffice, de la API a la
pantalla. El backoffice no tiene tenants, permisos por rol ni planes: se salta
todo ese cableado (ver [`backoffice-architecture.md`](backoffice-architecture.md)).

Ejemplo: una sección **Coupons** (cupones de descuento). Referencia real para
copiar: la sección **Features** (`backoffice-feature-handler.go` +
`src/api/feature-service.ts` + `src/pages/features/*`).

## Paso 1: Modelo (si es nuevo)

En el dominio al que pertenece el dato (no en `features/backoffice/`), p. ej.
`apps/api/features/companies/models/coupon-model.go`:

```go
type Coupon struct {
    gorm.Model
    Code     string  `gorm:"not null;unique" json:"code"`
    Discount float64 `gorm:"not null" json:"discount"`
    Active   bool    `gorm:"not null;default:true" json:"active"`
}
```

Agrégalo a `RunMigrations` en `apps/api/migrations/migrate.go`. Si es un dato
de un tenant, lleva `TenantID` como cualquier modelo de la app.

## Paso 2: Handler

`apps/api/features/backoffice/handlers/backoffice-coupon-handler.go`:

```go
type BackofficeCouponHandler struct {
    db     *gorm.DB
    logger *zap.Logger
}

// System: el backoffice trabaja sobre todos los tenants (ADR 0002).
func NewBackofficeCouponHandler(db *gorm.DB, logger *zap.Logger) *BackofficeCouponHandler {
    return &BackofficeCouponHandler{db: tenantdb.System(db), logger: logger}
}

type CreateCouponRequest struct {
    Code     string  `json:"code" binding:"required"`
    Discount float64 `json:"discount" binding:"required,gt=0"`
}

func (h *BackofficeCouponHandler) GetCoupons(c *gin.Context) envelope.Response {
    var coupons []company_models.Coupon
    if err := h.db.Order("created_at DESC").Find(&coupons).Error; err != nil {
        h.logger.Error("failed to fetch coupons", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "backoffice.coupon.list.error", core_errors.ErrInternal)
    }
    return envelope.SuccessResponse(coupons, "backoffice.coupon.list.success")
}

func (h *BackofficeCouponHandler) CreateCoupon(c *gin.Context) envelope.Response {
    var req CreateCouponRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        return envelope.ErrorResponse(http.StatusBadRequest, "backoffice.coupon.invalid.request", core_errors.ErrBackofficeInvalidRequest)
    }
    coupon := company_models.Coupon{Code: req.Code, Discount: req.Discount, Active: true}
    if err := h.db.Create(&coupon).Error; err != nil {
        h.logger.Error("failed to create coupon", zap.Error(err))
        return envelope.ErrorResponse(http.StatusInternalServerError, "backoffice.coupon.create.error", core_errors.ErrInternal)
    }
    return envelope.New(http.StatusCreated, "backoffice.coupon.create.success", coupon)
}

// GetCouponByID, UpdateCoupon, DeleteCoupon: igual, con c.Param("id").
```

- Los mensajes son **keys i18n** (algunos handlers viejos del backoffice pasan
  texto plano; no los copies en eso).
- DTOs: inline en el handler si son pocos (como features) o en
  `features/backoffice/dto/`.
- Error codes nuevos: `E-BO-NNN` en `core/errors/codes.go`, con su key i18n.
- Tests: `backoffice-coupon-handler_test.go` con `testutils.SetupTestDB`
  (patrón `backoffice-announcement-handler_test.go`).

## Paso 3: Rutas

`apps/api/routes/backoffice_routes.go`, dentro de `RegisterBackofficeRoutes`:

```go
backofficeCouponHandler := backoffice_handlers.NewBackofficeCouponHandler(db, logger.Log)
backofficeCouponRoutes := router.Group("/backoffice/coupons", backofficeAuth)
{
    backofficeCouponRoutes.GET("", envelope.Handle(backofficeCouponHandler.GetCoupons))
    backofficeCouponRoutes.GET("/:id", envelope.Handle(backofficeCouponHandler.GetCouponByID))
    backofficeCouponRoutes.POST("", envelope.Handle(backofficeCouponHandler.CreateCoupon))
    backofficeCouponRoutes.PUT("/:id", envelope.Handle(backofficeCouponHandler.UpdateCoupon))
    backofficeCouponRoutes.DELETE("/:id", envelope.Handle(backofficeCouponHandler.DeleteCoupon))
}
```

Con esas cinco rutas el recurso encaja en `lib/resource` sin código extra.

## Paso 4: Servicio

`apps/backoffice/src/api/coupon-service.ts`:

```typescript
import { resource } from "@/lib/resource/http-resource";

export interface Coupon {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  code: string;
  discount: number;
  active: boolean;
}

export interface CreateCouponRequest extends Record<string, unknown> {
  code: string;
  discount: number;
}

export interface UpdateCouponRequest extends Record<string, unknown> {
  discount?: number;
  active?: boolean;
}

export const coupons = resource<Coupon, CreateCouponRequest, UpdateCouponRequest>("coupons");
```

Endpoints fuera del CRUD (acciones, listados filtrados) son funciones extra con
`createHttpService(api)` en el mismo archivo (patrón
`announcement-service.ts` → `cancelAnnouncement`).

## Paso 5: Listado

`apps/backoffice/src/pages/coupons/coupon-list.tsx`:

```typescript
import { type Coupon, coupons } from "@/api/coupon-service";
import { type ResourceColumn, ResourceList } from "@/lib/resource";

const columns: ResourceColumn<Coupon>[] = [
  { header: "backoffice.coupons.col.code", cell: (c) => c.code, className: "font-mono text-sm" },
  { header: "backoffice.coupons.col.discount", cell: (c) => `${c.discount}%` },
];

const CouponList = () => (
  <ResourceList resource={coupons} columns={columns} itemLabel={(c) => c.code} />
);

export default CouponList;
```

`ResourceList` pone título, botón crear, estados de carga y vacío, y editar /
borrar (con confirmación) por fila. `rowActions` y `headerActions` agregan
acciones propias.

## Paso 6: Crear y editar

`apps/backoffice/src/pages/coupons/create-coupon.tsx` (patrón
`pages/features/create-feature.tsx`):

```typescript
import { useText } from "@pengi/shared";
import { Button, Card, CardContent, CardFooter, CardHeader, CardTitle, Form, FormInput, Spinner } from "@pengi/ui";
import { useNavigate } from "react-router";
import z from "zod";
import { coupons } from "@/api/coupon-service";
import { ResourceEditPage, useResourceItem } from "@/lib/resource";

const formSchema = z.object({
  code: z.string().min(2, "backoffice.coupons.error.code"),
  discount: z.coerce.number().gt(0, "backoffice.coupons.error.discount"),
});

const CreateCoupon = () => {
  const { textGet } = useText();
  const navigate = useNavigate();
  const { saving, save } = useResourceItem(coupons);

  return (
    <ResourceEditPage>
      <div className="max-w-2xl mx-auto">
        <Form schema={formSchema} onSubmit={save} defaultValues={{ code: "", discount: 0 }}>
          {(field) => (
            <Card>
              <CardHeader>
                <CardTitle>{textGet("backoffice.coupons.create.title")}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <FormInput field={field} name="code" label={textGet("backoffice.coupons.col.code")} />
                <FormInput field={field} name="discount" type="number" label={textGet("backoffice.coupons.col.discount")} />
              </CardContent>
              <CardFooter className="flex justify-between">
                <Button type="button" variant="outline" onClick={() => navigate("/coupons")}>
                  {textGet("backoffice.common.cancel")}
                </Button>
                <Button type="submit" disabled={saving}>
                  {saving && <Spinner />}
                  {textGet("backoffice.coupons.create")}
                </Button>
              </CardFooter>
            </Card>
          )}
        </Form>
      </div>
    </ResourceEditPage>
  );
};

export default CreateCoupon;
```

La página de editar es igual, con
`const { item, loading, saving, save } = useResourceItem(coupons, id)`
(`id` de `useParams`), `<ResourceEditPage loading={loading}>` y
`defaultValues` desde `item` (patrón `pages/features/edit-feature.tsx`). `save`
crea o actualiza según haya `id` y vuelve al listado.

## Paso 7: Rutas y navegación

`apps/backoffice/src/routes/routes.tsx` — dentro de los hijos de `RequireSession`:

```typescript
const CouponList = lazy(() => import("@/pages/coupons/coupon-list"));
const CreateCoupon = lazy(() => import("@/pages/coupons/create-coupon"));
const EditCoupon = lazy(() => import("@/pages/coupons/edit-coupon"));

{ path: "/coupons", element: <CouponList /> },
{ path: "/coupons/create", element: <CreateCoupon /> },
{ path: "/coupons/edit/:id", element: <EditCoupon /> },
```

Las rutas siguen `resourceRoutes("coupons")`, que es a donde navegan
`ResourceList` y `useResourceItem`.

`apps/backoffice/src/config/nav-config.ts`:

```typescript
{
  icon: TicketPercent, // componente de lucide-react
  label: textGet("backoffice.nav.coupons"),
  href: "/coupons",
},
```

## Paso 8: i18n

En **ambos** `apps/api/i18n/messages/messages_es.json` y `messages_en.json`:

- Las que usa `ResourceList`/`ResourceEditPage` por convención:
  `backoffice.coupons.title`, `backoffice.coupons.create`,
  `backoffice.coupons.list.title`, `backoffice.coupons.list.description`,
  `backoffice.coupons.empty`.
- Las de tus columnas, formularios y errores de Zod
  (`backoffice.coupons.col.*`, `backoffice.coupons.error.*`).
- Las del backend (`backoffice.coupon.*.success|error`) y los `E-BO-NNN` nuevos.
- `backoffice.nav.coupons`.

Tras agregarlas: `localStorage.removeItem("messages")` y recarga.

## Paso 9: Tests y verificación

- Frontend: tests junto al código; `memoryResource` evita el HTTP
  (patrón `lib/resource/resource-list.test.tsx`).
- `cd apps/backoffice && pnpm run typecheck && pnpm test:run`
- `cd apps/api && go build ./... && go vet ./... && go test ./features/backoffice/...`
- `just check`

## Checklist

- [ ] Modelo en su dominio y en `RunMigrations` (si es nuevo)
- [ ] Handler con `tenantdb.System(db)`, mensajes como keys i18n
- [ ] Error codes `E-BO-NNN` con key i18n
- [ ] Rutas en `backoffice_routes.go` detrás de `backofficeAuth`
- [ ] `src/api/<x>-service.ts` con `resource<T, C, U>("<x>")`
- [ ] Listado con `ResourceList`; crear/editar con `useResourceItem` + `ResourceEditPage`
- [ ] Rutas `lazy()` en `routes.tsx` siguiendo `resourceRoutes`; ítem en `nav-config.ts`
- [ ] i18n de convención + propias, en es y en
- [ ] Tests backend y frontend; typecheck, vet, `just check`
