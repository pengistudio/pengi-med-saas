# El cobro es independiente del comprobante electrónico y nunca se borra

Una factura tenía un solo campo `payment_method` y ninguna noción de cuánto se había cobrado. La caja agrega cobros (`Payment`) N:1 con la factura, con su propio estado derivado (sin cobrar, parcial, cobrada) que no depende del estado SRI. Registrar, anular o reembolsar un cobro nunca modifica el comprobante ni su clave de acceso (ver ADR 0001): el SRI solo conoce la forma de pago declarada en el XML, que se mapea desde el método con `MethodSRICodes`.

Se puede cobrar contra una factura en borrador, porque el Consumidor Final también recibe factura y el cobro suele ocurrir antes de la emisión.

Los cobros y movimientos no se borran. Dentro de la sesión abierta se anulan; una vez cerrada la sesión es inmutable y la corrección es un reembolso: un cobro negativo en la sesión vigente, vinculado al original. Así el cierre ciego de cada sesión sigue siendo reproducible.

## Considered Options

- **Guardar el estado de cobro en la factura y editarlo**: descartado porque obliga a tocar un comprobante que el SRI ya recibió, y pierde el historial de abonos.
- **Borrar o editar cobros erróneos**: descartado porque rompe la inmutabilidad de la sesión cerrada y la auditoría.
- **Emitir nota de crédito automática en cada reembolso**: descartado; un reembolso de caja no implica una devolución tributaria. Si hace falta, la emite el usuario.

## Consequences

Las facturas anteriores a la caja no tienen cobros y se muestran con "—", sin backfill. Reportes y cuentas por cobrar quedan fuera de v1 y deberán leer de `Payment`, nunca de `Invoice.payment_method`.
