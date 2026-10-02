# Documentos: PDF, archivos del tenant, email

Rama del paso 0. Para cualquier archivo por tenant (P12, logo, XML firmado,
RIDE, plantillas subidas) usa `core/tenantfiles`; para PDFs, `core/pdfrender`.
Ambos resuelven las rutas `storage/tenants/...` y el cliente de Gotenberg.

## PDF — `core/pdfrender`

Cada documento imprimible es un `pdfrender.Document` declarado junto a su
plantilla default (`features/<domain>/templates/documents.go`): `ID` (slug en
URLs), `File` (nombre de la plantilla y del archivo del tenant: nunca lo
renombres), `Defaults` (el FS embebido), `Paper`, `Feature` (`clinical` |
`billing`), `Sample` (datos de ejemplo realistas, del tipo de datos de la
plantilla), `Variants` (otras formas de los datos, p. ej. sin firmar) y
`Required` (valores que la salida debe mostrar). El tipo de datos vive en el
mismo paquete (`data.go`); el handler solo lo construye:

```go
pdf, err := h.renderer.Render(tenantdb.TenantID(c), clinical_templates.Report, data)
```

`Render` usa la plantilla que el tenant subió (Ajustes → Plantillas de
documentos, `/document-templates`) si existe; si no, la default. Para un
documento nuevo:

1. `features/<domain>/templates/<name>.html` (sintaxis `html/template`) y, si
   el dominio no lo tiene, `embed.go` con `//go:embed *.html` + `var FS embed.FS`.
2. El tipo de datos en `data.go` y el `pdfrender.Document` con su `Sample` en
   `documents.go`.
3. Súmalo a `printableDocuments` en `routes/documents.go` (aparece en Ajustes)
   y a la tabla de `core/pdfrender/catalog_test.go`, que valida y renderiza
   cada default con su `Sample`.
4. Inyecta `documentRenderer()` en el constructor del handler desde el archivo
   de rutas, y agrega las keys `document_templates.doc.<id>`.

El embed reemplaza cualquier `COPY` en Dockerfiles: una plantilla fuera del
embed es la causa del incidente de producción b74f2f2.

Ejemplos: `generatePrescriptionPDF` en
`features/clinical/handlers/download-record-handler.go`,
`features/clinical/handlers/medical-document-handler.go`.

## Archivos del tenant — `core/tenantfiles`

`Store` con `Write/Read/Remove/Exists(tenantID uint, name string)`. En rutas usa
la instancia compartida `tenantFiles` (`routes/documents.go`); en tests,
`tenantfiles.Memory()`.

## Handler que devuelve un binario

Escribe con `c.Data(http.StatusOK, "application/pdf", pdf)` (y `c.JSON(status,
envelope.ErrorResponse(...))` en error) y regístralo **sin** `envelope.Handle`:

```go
group.GET("/<x>/:id/download", rp(db, "DOWNLOAD_<X>"), h.Download<X>)
```

Ejemplos: `DownloadPrescription`, `DownloadMedicalReport`, `DownloadInvoiceRide`.

## Email con adjunto — `core/mailer`

`mailer.NewMailer()` inyectado en el handler;
`SendMedicalDocumentEmail(to, subject, title, filename, pdfBytes)`. Un tipo de
email nuevo sigue el patrón de `sendWithAttachments` (Resend). Un
`resend API error: status 4xx` en dev suele ser el API key en modo sandbox, que
solo entrega a la dirección verificada del dueño de la cuenta.
