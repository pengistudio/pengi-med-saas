# Órdenes de exámenes — Diseño

**Fecha:** 2026-10-02
**Feature:** Órdenes de exámenes con catálogo por tenant, resultados adjuntos y revisión médica
**Alcance:** `apps/api` (dominio `clinical`), `apps/web`. Sin cambios en backoffice salvo asociar permisos al plan.

Vocabulario en `CONTEXT.md` § Clínica (Orden de examen, Examen, Catálogo de exámenes, Perfil, Resultado, Revisión, Estado de la orden, Adjunto).

Se apoya en **Adjuntos del paciente** (`2026-10-02-adjuntos-del-paciente-design.md`, `PatientAttachment`): un resultado es un adjunto.

---

## Contexto

Un competidor ofrece "órdenes de exámenes" en todos sus planes. La decisión es ofrecer más: orden firmada electrónicamente, catálogo precargado con perfiles, resultados adjuntos a la historia, bandeja de revisión y visor integrado.

## Decisiones (grilling 2026-10-02)

| # | Decisión |
|---|---|
| Gating | Plan + rol (`RequirePermission`). Permisos con `Category` clinical → incluidos en el `Feature` clínico de **todos** los planes. |
| Origen | La orden pertenece siempre a un paciente; la consulta (`MedicalRecord`) es opcional. Una consulta puede tener varias órdenes. El FK vive en la orden (no copiar el 1:1 de `Prescription`). |
| Tipos | Categoría por examen: `laboratory`, `imaging`, `other`. |
| Catálogo | Por tenant, precargado (~150 exámenes + ~10 perfiles, nombres usados en Ecuador). Editable, desactivable, con "restaurar catálogo inicial". Exámenes escritos a mano permitidos (no se agregan al catálogo). |
| Datos de la orden | Por examen: indicaciones (default del catálogo). Por orden: diagnóstico presuntivo CIE-10 (mismo buscador de la consulta), prioridad (`routine`/`urgent`), observaciones, laboratorio de destino (texto libre). |
| Resultados | Un resultado es un **Adjunto** (`PatientAttachment`) del paciente de la orden, vinculado (m2m) a uno o varios exámenes de esa orden. Se sube por el mismo camino que cualquier adjunto: un archivo por petición, PDF/JPEG/PNG detectados por contenido, ≤15 MB, cuota del plan, aviso de duplicado por SHA-256, cifrado en reposo y auditoría. Sin WEBP (los adjuntos no lo aceptan). Modelo preparado para valores estructurados en el futuro. |
| Estados | `issued` → `partial_results` → `complete_results`, o `voided`. Calculado por exámenes con resultado; el médico puede cerrar la orden como completa a mano. Anular exige motivo; la orden anulada sigue visible. |
| Edición | Sin resultados: editable; si estaba firmada, la edición borra la firma (igual que recetas, `clearedSignature`). Con algún resultado: solo agregar exámenes (borra firma) o anular. |
| Firma | Opcional, PAdES + QR con `Signer`, permiso `SIGN_MEDICAL_DOCUMENT`. |
| PDF | Nuevo `pdfrender.Document` `exam_order`, A4 vertical, exámenes agrupados por categoría con casillas. Configurable en `/document-templates`; requiere bloque de firma como los demás documentos clínicos. |
| Envío | La orden por email (PDF adjunto, `mailer.SendMedicalDocumentEmail`) y WhatsApp (mismo mecanismo que recetas). Los resultados **no** se envían al paciente en v1. |
| Revisión | Al subir resultados se notifica (in-app, `features/notifications`) al médico que emitió la orden. Cualquier usuario con `REVIEW_EXAM_RESULTS` marca exámenes como revisados; se guarda quién y cuándo. |
| Numeración | Correlativo por tenant, sin huecos ni duplicados bajo concurrencia, impreso como `ORD-000123`. |
| Borrado de resultados | Es el borrado del adjunto: soft delete con motivo obligatorio, quién y cuándo, auditado y restaurable desde los adjuntos eliminados del paciente. Bloqueado (409) si algún examen que cubre está revisado, tanto desde la orden como desde los adjuntos del paciente. Borrar o restaurar recalcula el estado de la orden. |
| Visor | Panel lateral en la página de la orden que usa las rutas `view`/`download` de adjuntos (permiso `READ_PATIENT_ATTACHMENT`). La orden lista los metadatos de sus resultados con `READ_EXAM_ORDER`; quien no tenga `READ_PATIENT_ATTACHMENT` ve que hay resultado pero no puede abrirlo (403), igual que la recepcionista con los adjuntos. |
| Facturación | Sin enlace a `catalog-item` en v1 (se retoma en la fase de caja). |

## Modelo de datos

Todos con `gorm.Model` + `TenantID uint`. Nombres orientativos; el engineer ajusta a convenciones del dominio.

- **`ExamCatalogItem`**: `Name`, `Category`, `Subgroup` (Hematología, Ecografía…; editable, ordena el selector y agrupa el PDF), `DefaultIndications`, `Active`, `IsDefault` (vino del seed). Contenido del seed: [`2026-10-02-exam-catalog-seed.md`](2026-10-02-exam-catalog-seed.md).
- **`ExamProfile`**: `Name`, `Active`, `IsDefault`; many2many con `ExamCatalogItem`.
- **`ExamOrder`**: `Number uint` (único por tenant), `PatientID`, `MedicalRecordID *uint`, `OrderedByID` (usuario), diagnósticos CIE-10 (mismo formato que la consulta), `Priority`, `Notes`, `DestinationLab`, `Status`, `ClosedManually bool`, `VoidReason`, `VoidedAt`, `VoidedByID`, `DocumentSignature` embebido, `Items []ExamOrderItem`.
- **`ExamOrderItem`**: `ExamOrderID`, `CatalogItemID *uint`, `Name`, `Category` y `Subgroup` copiados (la orden no cambia si se edita el catálogo, ni al editar la orden), `Indications`, `ReviewedByID *uint`, `ReviewedAt *time.Time`, `Attachments` (many2many con `PatientAttachment`, tabla `exam_order_item_attachments`; un adjunto borrado deja de contar).
- **Resultado = `PatientAttachment`** (de la rama de adjuntos): `PatientID` = paciente de la orden; `MedicalRecordID` = consulta de la orden si tiene; `Category` según los exámenes que cubre: `lab_result` si cubre alguno de laboratorio, si no `imaging` si cubre alguno de imagen, si no `external_report` (ECG, espirometría, endoscopía: informes de estudios hechos fuera); el cliente puede enviar otra categoría válida. `TakenAt` = día de subida salvo que se envíe.
- **Contador del correlativo**: tabla por tenant con incremento atómico (`UPDATE ... SET value = value + 1 RETURNING value` dentro de la transacción de creación). Revisar si billing ya tiene un patrón de secuencial reutilizable antes de crear uno nuevo.

`Status` se recalcula en el servidor tras cada cambio de resultados o exámenes; nunca lo envía el cliente (salvo la acción explícita "cerrar como completa").

## API (`/clinical`, tenant)

| Método y ruta | Permiso |
|---|---|
| `GET /exam-catalog`, `GET /exam-profiles` | `READ_EXAM_ORDER` |
| `POST/PUT/DELETE /exam-catalog[/:id]`, `POST /exam-catalog/restore-defaults`, `POST/PUT/DELETE /exam-profiles[/:id]` | `MANAGE_EXAM_CATALOG` |
| `GET /exam-orders` (filtros: `status`, `pending_review`, `ordered_by`, `patient_id`, `record_id`, paginado) | `READ_EXAM_ORDER` |
| `GET /exam-orders/:id`, `GET /exam-orders/:id/download` | `READ_EXAM_ORDER` |
| `POST /exam-orders`, `PUT /exam-orders/:id`, `POST /exam-orders/:id/void`, `POST /exam-orders/:id/close`, `POST /exam-orders/:id/email` | `CREATE_EXAM_ORDER` |
| `POST /exam-orders/:id/sign` | `CREATE_EXAM_ORDER` + `SIGN_MEDICAL_DOCUMENT` |
| `POST /exam-orders/:id/results` (multipart: `file` + `item_ids`, opcionales `category`, `taken_at`, `description`) | `UPLOAD_EXAM_RESULTS` |
| `DELETE /exam-orders/:id/results/:attachmentId` (JSON `{reason}`) | `UPLOAD_EXAM_RESULTS` + `DELETE_PATIENT_ATTACHMENT` |
| `POST /exam-orders/:id/review` (`item_ids`) | `REVIEW_EXAM_RESULTS` |
| Ver/descargar un resultado: `GET /patients/:id/attachments/:attachment_id/view` / `download`; restaurar: `POST /patients/:id/attachments/:attachment_id/restore` | permisos de adjuntos |

Handlers con `tenantdb.For(c, h.db)` y `envelope.Response`. La lógica de subida (lectura y validación del archivo, cuota, duplicado, guardado cifrado) vive en `features/clinical/handlers/attachment-upload.go` y la usan `UploadAttachment` y `UploadExamResult`. Error codes nuevos `E-CLIN-040`…`050` (los `018`…`025` son de adjuntos) con keys i18n.

## Permisos y roles

| Permiso | doctor | recepcionista | contador |
|---|---|---|---|
| `READ_EXAM_ORDER` | ✓ | ✓ | — |
| `CREATE_EXAM_ORDER` | ✓ | — | — |
| `UPLOAD_EXAM_RESULTS` | ✓ | ✓ | — |
| `REVIEW_EXAM_RESULTS` | ✓ | — | — |
| `MANAGE_EXAM_CATALOG` | ✓ | — | — |

Admin: todos. Borrar un resultado exige además `DELETE_PATIENT_ATTACHMENT` (un resultado es un adjunto): la recepcionista sube resultados pero no los borra. Migraciones nuevas: permisos → admin + roles canónicos (`role-data.go` `RolePermissionMatrix`), y asociación al `Feature` clínico del plan. Mirror en `apps/web/src/lib/constants.ts`.

## Seed del catálogo

- Datos en Go (`features/clinical/data/`), revisados por el usuario antes de implementar.
- Se cargan al crear un tenant y, con una migración nueva, en los tenants existentes que no tengan catálogo.
- "Restaurar catálogo inicial" reactiva/re-crea los `IsDefault` sin tocar los creados por el tenant.

## Frontend (`apps/web`)

- `src/api/exam-order-service.ts`: tipos + servicio (`apiWithTenant`, toasts por flags).
- **Consulta y ficha del paciente**: sección "Órdenes de exámenes" con listado y botón "Nueva orden".
- **Página de orden** (crear/editar/ver): selector de exámenes del catálogo + perfiles + texto libre, indicaciones editables, CIE-10, prioridad, laboratorio, observaciones; acciones descargar/firmar/email/WhatsApp/anular/cerrar; subida de resultados marcando exámenes cubiertos; visor lateral; marcar revisado.
- **`/clinical/exam-orders`**: pestañas Pendientes de resultado / Por revisar (por defecto las del médico conectado, filtro "todas") / Todas.
- **Configuración**: CRUD de catálogo y perfiles, restaurar catálogo inicial.
- Rutas con `<CheckPermission>`, ítem en `nav-config.ts` con `permission` y `feature: "clinical"`.
- Todo texto vía `textGet`; fechas con `formatDate`.

## Tests

- Handlers: crear/editar/anular, transición de estados al subir/borrar resultados, bloqueo de edición con resultados, bloqueo de borrado de resultado revisado, firma borrada al editar.
- Aislamiento entre tenants en órdenes, catálogo y resultados (por la orden y por los adjuntos del paciente).
- Correlativo: sin duplicados con creaciones concurrentes.
- Validación de archivos (camino de adjuntos): MIME falso y WEBP rechazados, >15 MB rechazado, cuota del plan, aviso de duplicado, guardado cifrado.
- Bloqueo de borrado de un adjunto con examen revisado desde ambas rutas; borrar/restaurar recalcula el estado.
- Web: tests de servicio/columnas según el patrón del dominio.

## Fuera de alcance (v1)

- Resultados con valores numéricos, rangos y gráfica de evolución.
- Enlace con facturación / cobro de exámenes.
- Envío de resultados al paciente; portal del paciente.
