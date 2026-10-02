# Pengi Med

SaaS multi-tenant para consultorios médicos en Ecuador: gestión clínica (pacientes, citas, historias clínicas) y facturación electrónica ante el SRI.

## Language

### Organización

**Tenant**:
Espacio aislado de datos de un consultorio; ningún dato de un tenant es visible desde otro. Se identifica en cada petición por su slug.
_Avoid_: cuenta, organización

**Empresa**:
Entidad comercial dueña de un tenant (uno a uno): razón social, plan y suscripción. En código: `Company`.
_Avoid_: compañía, cliente

**Miembro**:
Usuario con un rol en una empresa; solo un miembro puede operar sobre el tenant de esa empresa. En código: `Environment`.
_Avoid_: environment, entorno

### Facturación electrónica

**Comprobante electrónico**:
Documento tributario emitido al SRI con clave de acceso propia y ciclo de recepción/autorización; hoy son la factura, la nota de crédito y la nota de débito. En código: `SriDocument`.
_Avoid_: documento SRI, invoice (para referirse a cualquier tipo)

**Clave de acceso**:
Identificador de 49 dígitos que identifica un comprobante electrónico ante el SRI; es también su número de autorización. Un comprobante tiene una sola clave de acceso durante toda su vida.
_Avoid_: número de autorización, access code

**Secuencial**:
Número correlativo del comprobante dentro de su establecimiento, punto de emisión y tipo de comprobante.

**Recepción**:
Etapa en la que el SRI recibe y valida la estructura de un comprobante firmado. Mientras el SRI lo procesa no debe reenviarse, y nunca con otra clave de acceso.
_Avoid_: validación (a secas)

**Devolución**:
Rechazo del SRI en la recepción (estado DEVUELTA); el comprobante no queda registrado en el SRI y se reenvía con la misma clave de acceso y secuencial una vez corregida la causa.
_Avoid_: rechazo (a secas)

**Autorización**:
Etapa en la que el SRI aprueba un comprobante ya recibido; puede quedar pendiente ("en procesamiento") y consultarse de nuevo sin reenviar el comprobante.

**No autorizado**:
Rechazo del SRI en la autorización (estado NAT). El emisor debe corregir la causa y reenviar el comprobante con la misma clave de acceso y secuencial; un comprobante puede tener varias respuestas No autorizado y una sola Autorizado.
_Avoid_: fallido, rechazado (a secas), anulado

**RIDE**:
Representación impresa (PDF) de un comprobante autorizado; tiene validez tributaria y jurídica.

**Consumidor Final**:
Comprador genérico declarado cuando la factura no tiene paciente asociado.

### Plataforma

**Catálogo de mensajes**:
Todos los textos de la API y de las apps (mensajes, códigos de error, etiquetas) por clave i18n e idioma, en el JSON embebido en el binario; cambiar un texto requiere un deploy. En código: `i18n/catalog`.
_Avoid_: tabla de mensajes, traducciones (a secas)

**Idioma de la interfaz**:
El idioma (`es`/`en`) en que el usuario ve textos, fechas y montos; cambia junto con los mensajes cargados, no al elegirlo. Define el locale de formato (`es`→`es-EC`, `en`→`en-US`); la moneda es siempre USD. En código: `useText` de `@pengi/shared`.
_Avoid_: locale (a secas), idioma del navegador

**Documento imprimible**:
Documento que el sistema genera como PDF a partir de una plantilla: receta, informe médico, certificado médico, RIDE. Cada uno tiene un tamaño de papel fijo y depende de un módulo del plan (clínico o facturación). Un documento ya emitido (RIDE autorizado, documento firmado) no se vuelve a generar al cambiar la plantilla. En código: `pdfrender.Document`.
_Avoid_: reporte, PDF (a secas)

**Plantilla**:
El HTML con que se genera un documento imprimible. La **por defecto** viene en el binario; la **del tenant**, si existe, la reemplaza para todo ese tenant. Una plantilla del tenant no puede pedir recursos externos y debe mostrar los datos obligatorios de su documento (clave de acceso en el RIDE, firma en los documentos clínicos).
_Avoid_: template, formato
