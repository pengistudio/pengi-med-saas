# Adjuntos del paciente

Investigación de base: `docs/research/2026-10-02-funciones-clinicas.md` (sección 2).

## Problem Statement

El médico recibe a diario resultados de laboratorio, informes de imagen, estudios de otros especialistas y papeles que el paciente trae a la consulta. Hoy Pengi no tiene dónde guardarlos: quedan en el correo, en el celular del médico o en una carpeta física, fuera de la historia clínica. En plena consulta el médico no puede ver el resultado anterior, no puede ordenar los estudios en el tiempo y no hay registro de quién los vio. Además, las funciones que vienen después (órdenes de laboratorio con su resultado, consentimiento informado firmado y escaneado) necesitan un lugar donde dejar el archivo.

## Solution

Cada paciente tiene una pestaña "Archivos" en su ficha donde el equipo sube PDFs y fotos (incluida la cámara del celular), los clasifica en categorías fijas, les pone la fecha del examen y los ve sin salir de la app. Desde la consulta en curso también se pueden subir y quedan vinculados a esa consulta. Los archivos se guardan cifrados, aislados por clínica, con un límite de espacio según el plan, y cada vez que alguien los ve o descarga queda registrado. Nada se borra de verdad: se oculta con un motivo y se puede restaurar.

## User Stories

1. As a médico, I want to upload a PDF lab result to a patient's record, so that it lives with the rest of the historia clínica.
2. As a médico, I want to upload several files at once, so that a multi-page result photographed page by page takes one action.
3. As a médico on my phone, I want to take a photo with the camera directly from the upload control, so that I can capture the paper the patient brought without transferring files.
4. As a médico with an iPhone, I want HEIC photos to be accepted, so that I don't have to change my camera settings.
5. As a médico, I want to choose a category (resultado de laboratorio, imagen, informe externo, foto clínica, consentimiento, otro) for each file, so that I can find it later.
6. As a médico, I want to set the date the exam was taken, so that a March result uploaded in June appears in March.
7. As a médico, I want the exam date to default to the upload date when I leave it empty, so that uploading stays fast.
8. As a médico, I want to add a short optional description, so that I can note what the file is ("Hemograma Lab. X").
9. As a médico, I want to see a patient's files ordered by exam date, newest first, so that I read their evolution in order.
10. As a médico, I want to filter the file list by category, so that I can see only lab results.
11. As a médico, I want to open a PDF or image inside the app, so that I review results during the consultation without downloading.
12. As a médico, I want to download the original file, so that I can share or print it.
13. As a médico, I want to upload files from within the consultation I'm recording, so that they're linked to that visit automatically.
14. As a médico, I want to see in a consultation which files are linked to it, so that I know what the patient brought that day.
15. As a médico, I want to link or unlink an existing file to a consultation of the same patient, so that I can fix where it belongs.
16. As a médico, I want to edit a file's category, exam date and description after uploading, so that I can correct mistakes.
17. As a médico, I want to be warned when I upload a file identical to one the patient already has, so that I avoid duplicates without being blocked.
18. As a médico, I want to be told clearly when a file is too large or of an unsupported type, so that I know what to do.
19. As a recepcionista, I want to upload files the patient brings without being able to view clinical files, so that I help without accessing data I shouldn't see.
20. As a usuario with delete permission, I want to delete a file by giving a reason, so that mistakes are corrected while keeping the record traceable.
21. As a usuario with delete permission, I want to see the list of deleted files with who, when and why, so that I can audit removals.
22. As a usuario with delete permission, I want to restore a deleted file, so that an accidental deletion is reversible.
23. As a usuario, I want deleted files hidden from the normal list and the consultation, so that the record stays clean.
24. As an administrador de la clínica, I want to see how much storage we use against our plan's quota, so that I can plan an upgrade.
25. As an administrador de la clínica, I want to be warned at 80% of the quota, so that uploads don't stop by surprise.
26. As a usuario, I want a clear message to upgrade the plan when the quota is full, so that I know why the upload failed.
27. As an administrador de la clínica, I want every view and download of a file recorded, so that I can answer who accessed a patient's data.
28. As an administrador de la clínica, I want every metadata change and deletion recorded, so that the record's history is complete.
29. As a dueño de clínica, I want my files invisible to any other clinic, so that patient data stays confidential.
30. As a dueño de clínica, I want files encrypted at rest, so that a leaked disk or backup doesn't expose patient data.
31. As an operador de backoffice, I want to set the storage quota of each plan, so that pricing controls storage cost.
32. As a médico, I want the exported historia clínica to list the patient's files (name, category, date), so that the export shows what supporting documents exist.
33. As a médico, I want the file tab to show an empty state explaining what can be uploaded, so that I understand the feature the first time.
34. As a médico, I want upload progress per file, so that I know a large photo is still uploading.

## Implementation Decisions

**Domain**
- New term **Adjunto** (patient attachment) in `CONTEXT.md`: a file that belongs to a patient, optionally linked to one consulta (medical record) of the same patient.
- New tenant model `PatientAttachment`: tenant, patient, optional medical record, category, exam date (`taken_at`, defaults to upload date), description, original file name, MIME type, size, SHA-256 of the original content, stored name, uploaded-by user, plus `gorm.Model` soft delete with deleted-by user and deletion reason. Auditable (`IsAuditable()`), so metadata changes go through the existing audit.
- Categories are a fixed set: `lab_result`, `imaging`, `external_report`, `clinical_photo`, `consent`, `other`. No tenant-defined categories.
- The file itself is never replaced; only metadata is editable.

**Storage**
- Files go through `tenantfiles.Store`; no direct paths. The stored name is generated (not the user's file name).
- Encryption is a new `Store` implementation that wraps another `Store` and encrypts/decrypts every `Write`/`Read` with `core/secretbox`, keyed by a new env var `ATTACHMENT_ENCRYPTION_KEY` (separate from `SIGNATURE_ENCRYPTION_KEY`). Attachments use the encrypting store; existing tenant files (logo, P12, templates) are unchanged. Key rotation is not supported in this version; a copy of the key is kept outside the server.
- The local disk store stays the backend for now; moving to S3-compatible storage later is a new `Store` implementation, no handler changes.
- Max 15 MB per file, enforced with `http.MaxBytesReader` before reading.

**Accepted files**
- PDF, JPEG, PNG. Type is decided by content (magic bytes), never by extension or client MIME; anything else is rejected.
- HEIC is converted to JPEG in the browser before upload; the server never receives HEIC. The original HEIC is not kept.

**Quota**
- New plan field `storage_quota_mb`, editable in the backoffice. Initial values: 2 GB / 10 GB / 50 GB for the three clinical plans.
- Usage = sum of attachment sizes for the tenant, **including soft-deleted ones**. Other tenant files don't count.
- Upload over quota is rejected with a dedicated error code; usage ≥ 80% is reported to the frontend so it shows a warning.
- Available on every plan that has the clinical module.

**Permissions**
- Three new permissions, mirrored in the frontend `PERMISSIONS`: read attachments, upload attachments, delete attachments. Gated with `RequirePermission` (role + plan).
- Upload without read: the uploader sees the success response but cannot list or open files.
- Delete permission also grants listing deleted attachments and restoring them.

**API (tenant-scoped, under the patient)**
- Upload (multipart, one or more files + shared category, exam date, description, optional medical record): returns created attachments and, per file, a duplicate warning when the same SHA-256 already exists for that patient (not blocking).
- List (filter by category, optional medical record; ordered by exam date desc; excludes deleted) + quota usage.
- Download/view: streams the decrypted file with the stored MIME type; `Content-Disposition` inline for the viewer, attachment for download; never serves HTML. Records an access with `audit.RecordAccess`.
- Update metadata (category, exam date, description, medical record — must belong to the same patient).
- Delete (reason required) → soft delete.
- List deleted / restore.
- All responses through `envelope`; all messages and error codes as i18n keys in both catalogs; new error codes in `core/errors/codes.go`.

**Frontend**
- "Archivos" tab in the patient record next to medical documents; drag & drop, multi-select, camera capture (`capture` input on mobile), per-file progress, category filter, empty state.
- Viewer: images directly; PDFs in a sandboxed viewer (pdf.js or sandboxed iframe), no script execution from the file.
- Upload and linked-files list inside the consultation form, auto-linking to that consultation.
- Quota indicator and 80% warning; deleted-files view with restore for users with delete permission.
- Service file with `notifySuccess`/`notifyError`; all strings via `textGet`.

**Exported historia clínica**
- Lists the patient's non-deleted attachments (name, category, exam date). Files are not included.

**Infrastructure dependency (separate ticket, blocks production)**
- Daily backup of the storage volume and an encrypted `pg_dump` to external object storage, kept 30 days, with at least one tested restore. Attachments must not reach production before it is in place.

## Testing Decisions

- Good tests exercise external behavior through the HTTP routes: request in, envelope response and stored state out. No tests of private helpers.
- **One seam: the attachment routes.** Tests build the Gin router with the test DB (`testutils.SetupTestDB`) and `tenantfiles.Memory()` wrapped by the encrypting store, and cover: upload/list/view/download/update/delete/restore; tenant isolation (another clinic's attachment is not found on every route); permission gating for read/upload/delete separately; 15 MB limit; magic-byte rejection (e.g. HTML or an executable renamed `.pdf`); quota rejection and 80% flag, counting soft-deleted files; duplicate warning; mandatory delete reason; medical record from another patient rejected; access audit entry on view/download; stored bytes differ from the original while download returns the original.
- Prior art: `features/clinical/handlers/tenant_isolation_test.go` and `patient_handler_test.go` (router + test DB), `core/tenantfiles/tenantfiles_test.go`.
- i18n catalog test covers the new keys and error codes automatically.
- Frontend: HEIC→JPEG conversion helper with Vitest; one Playwright flow (upload a PDF from the patient record, see it in the list, open the viewer, delete with reason, restore).

## Out of Scope

- DICOM and any medical-imaging viewer.
- Word, Excel and other office formats.
- Merging several images into one PDF.
- Linking attachments to lab/imaging orders (added when that feature exists).
- Key rotation for `ATTACHMENT_ENCRYPTION_KEY`.
- Including the files themselves (ZIP) in the exported historia clínica.
- Physical (hard) deletion and retention-based purging.
- Patient-facing access (no patient portal).
- Moving storage to S3.

## Further Notes

- Legal questions open, not blocking the design: retention period to promise clinics (no current MSP rule found) and the data-processor (encargado) contract plus impact assessment under the LOPDP. See the research note.
- Storage sizing reference: ~10 PDFs/day at ~1 MB ≈ 3.6 GB/year per clinic.
- This feature is the base for lab/imaging results and scanned signed consents.
