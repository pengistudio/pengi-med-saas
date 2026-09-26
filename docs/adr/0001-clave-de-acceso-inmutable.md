# La clave de acceso de un comprobante es inmutable y no se reenvía mientras el SRI lo procesa

Cada comprobante electrónico recibe su clave de acceso una sola vez, en el primer intento de procesamiento, y todos los reintentos la reutilizan. Antes de la recepción el XML se puede regenerar y volver a firmar (P12 renovado, datos del tenant corregidos). Una vez recibido, mientras el SRI lo procesa o se espera su autorización, no se vuelve a firmar ni a enviar: solo se consulta la autorización. Si el SRI lo declara No autorizado (NAT), se corrige la causa y se reenvía con la misma clave y el mismo secuencial, y solo cuando un usuario lo reintenta.

Esto sigue la ficha técnica del SRI (esquema offline, v2.26):

- §5.10 y la nota 1 de la tabla de errores: un comprobante rechazado se reenvía con la misma clave de acceso y secuencial, sin generar nuevos.
- §5.12: un comprobante NAT debe corregirse y enviarse nuevamente.
- Nota 2 (error 70, "clave de acceso en procesamiento"): no se reenvía ni se genera otra clave mientras no haya una respuesta, por un máximo de 24 horas.
- §8.18: controlar que no haya duplicidad de secuencias ni de claves de acceso, y evitar reenvíos innecesarios.

La clave incluye un código numérico aleatorio. Si se regenerara en cada intento, un reenvío registraría el mismo secuencial con una segunda clave. Las respuestas "clave de acceso registrada" (ID 43) y "secuencial registrado" (ID 45) se tratan como recepción exitosa. Así los reintentos automáticos son idempotentes.

## Considered Options

- **Regenerar la clave en cada intento**: era el comportamiento anterior. Descartado porque duplica claves para un mismo secuencial (§8.18).
- **Congelar también el XML firmado desde la primera firma**: descartado porque un comprobante que falló por un P12 vencido o por datos del emisor incorrectos nunca podría corregirse, y §5.10 exige corregir y reenviar.
- **Tratar el NAT como terminal y emitir un comprobante nuevo**: descartado porque contradice §5.10 y §5.12.

## Consequences

La ficha no dice qué responde la recepción cuando se reenvía una clave que quedó NAT. Si respondiera 43 o 45, el reenvío se trataría como "ya recibido" y solo se volvería a consultar la autorización. Falta comprobarlo en el ambiente de pruebas.
