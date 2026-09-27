# Documentos: PDF, archivos del tenant, email

Rama del paso 0. Para cualquier archivo por tenant (P12, logo, XML firmado,
RIDE, plantillas subidas) usa `core/tenantfiles`; para PDFs, `core/pdfrender`.
Ambos resuelven las rutas `storage/tenants/...` y el cliente de Gotenberg.

## PDF — `core/pdfrender`

```go
pdf, err := h.renderer.Render(tenantdb.TenantID(c), "<name>.html", data, utils.A4Portrait) // o utils.A5Landscape
```

`Render` usa la plantilla que el tenant subió con ese nombre si existe; si no,
la default embebida. Para agregar defaults de un dominio nuevo:

1. `features/<domain>/templates/<name>.html` (sintaxis `html/template`).
2. `features/<domain>/templates/embed.go`:
   ```go
   package <domain>_templates

   import "embed"

   //go:embed *.html
   var FS embed.FS
   ```
3. Suma `<domain>_templates.FS` a `documentRenderer()` en `routes/documents.go`.
4. Inyecta `documentRenderer()` en el constructor del handler desde el archivo
   de rutas.

El nombre de plantilla es único entre todos los FS (gana el primero que lo
tenga). El embed reemplaza cualquier `COPY` en Dockerfiles: una plantilla fuera
del embed es la causa del incidente de producción b74f2f2.

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
