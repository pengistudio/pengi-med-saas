# Investigación: 5 funciones clínicas candidatas (2026-10-02)

Alcance: órdenes de laboratorio/imagen, adjuntos del paciente, consentimiento informado firmable, plantillas SOAP + frases rápidas, medicamentos favoritos + vademécum CNMB + alerta de alergias. Excluido: referencia/contrarreferencia (form. 053). Restricción: sin integraciones pagadas.

Convención de citas: `[Fn]` = fuente de la tabla final; `ruta:línea` = código del repo. "Secundaria" = la afirmación no se verificó en documento oficial. Páginas de [F1] = página del PDF; de [F2] y [F5] = número impreso en el documento.

---

## 0. Marco normativo común

| Tema | Hallazgo | Fuente |
|---|---|---|
| Formularios HCU vigentes | El **AM 00115-2021** "Reglamento para el Manejo de la Historia Clínica Única" (RO n.º 378, 26-ene-2021) actualizó 51 formularios. Su Anexo 1 lista, entre otros: **010A/2021 Laboratorio clínico – solicitud**, **010B/2021 Laboratorio clínico – informe**, **012A/2021 Imagenología – solicitud**, **012B/2021 Imagenología – informe**, **013A/B Anatomía patológica**, **024/2021 Consentimiento informado**, **117/2021 Certificado médico**. | [F1] p. 1 del anexo; [F2] p. 3 |
| HCU electrónica permitida | "La HCU … puede ser en formato físico, digital o electrónico" (Art. 4). Todo profesional debe identificarse "con firma autógrafa o electrónica, si se trata de un sistema informático" (Art. 4). La HCU electrónica se compone de registro de personal, administrativo, integración semántica, registro clínico, seguridad y vigilancia epidemiológica (Art. 8). | [F2] pp. 9-10 |
| "Solicitud de exámenes" | Definida como "documento oficial de la HCU que registra los pedidos de determinaciones o estudios a los servicios de apoyo (laboratorio, imagenología, histopatología)". | [F2] p. 9 |
| Acceso | "Al formato digital o electrónico de la HCU tendrá acceso únicamente el personal autorizado" (Art. 11). | [F2] p. 11 |
| Conservación de la HCU | **No hay plazo de conservación vigente localizado.** El AM 115 solo define "archivo pasivo" (>5 años sin atención) y ordena (Disp. Trans. 1.ª, 120 días) elaborar "la normativa para el archivo, depuración, conservación y eliminación"; no encontré esa normativa publicada. | [F2] pp. 5, 13 |
| Datos de salud (LOPDP) | Son "categoría especial" (Art. 25 c). Los profesionales pueden tratarlos para "diagnóstico médico, prestación de asistencia o tratamiento" sin consentimiento adicional (Art. 30-31). Deber de confidencialidad y "medidas técnicas organizativas apropiadas" contra pérdida o acceso ilícito (Art. 30). Conservación "no mayor al necesario" con plazos de supresión definidos por el responsable (Art. 10 i). Notificar brechas en 5 días (Art. 43). Evaluación de impacto obligatoria en "tratamiento a gran escala de categorías especiales" (Art. 42 b). Pengi actúa como **encargado** (Art. 4) del consultorio (responsable). | [F3] |
| Firma electrónica | Igual validez que la manuscrita (Art. 14 LCE). Requisitos: individual, que "permita verificar inequívocamente la autoría e identidad del signatario, mediante dispositivos técnicos de comprobación establecidos por esta Ley", método confiable e inalterable, bajo control exclusivo del firmante (Art. 15). | [F4] (texto publicado por ARCOTEL 2015; no verifiqué reformas posteriores) |
| Ley Orgánica de Salud | Art. 7 h: derecho a "ejercer la autonomía de su voluntad a través del consentimiento por escrito … salvo en los casos de urgencia, emergencia o riesgo para la vida". | Citado textualmente en [F5] p. 32 (no consulté el texto de la LOS directamente) |

---

## 1. Órdenes de laboratorio e imagen (+ registro de resultados)

### A. Normativa
- Formatos oficiales: **010A/2021** (solicitud lab), **010B/2021** (informe lab), **012A/2021** (solicitud imagen), **012B/2021** (informe imagen) [F1].
- Estructura del 010A (leída del formulario escaneado, [F1] pp. 41-45): A. datos de establecimiento y paciente; B. servicio y prioridad (emergencia/consulta externa/hospitalización; **urgente / rutina / control**), diagnóstico CIE-10, "tratamiento terapéutico"; C. listado de exámenes para marcar con X, agrupados en: Hematología, Coagulación y hemostasia, Química sanguínea, Inmunología/infecciosas, Orina, Coprológico, Marcadores tumorales, Inmunosupresores, Niveles de drogas terapéuticas, Serología, Microbiología, Marcadores cardíacos/vasculares, Hormonas, Citoquímico de líquidos, Gases y electrolitos (+ Medicina transfusional y Biología molecular en el anverso); D. datos del profesional (nombre, código, firma, sello, fecha/hora de toma de muestra).
- 012A ([F1] pp. 52-55): mismos bloques A/B; C. estudio: **Rx convencional, Portátil, Tomografía, Resonancia, Ecografía, Mamografía, Procedimiento, Sedación, Otros** + descripción; motivo de la solicitud; resumen clínico; diagnóstico CIE-10; profesional.
- No es obligatorio para consultorios privados usar el PDF idéntico, pero el Reglamento es "de cumplimiento obligatorio por todos los profesionales … del Sistema Nacional de Salud" (Art. 2, [F2] p. 4) → conviene que la orden impresa contenga los mismos bloques.

### B. Datos libres
| Fuente | Contenido | Licencia |
|---|---|---|
| Listado del form. 010A | ~150 exámenes agrupados (catálogo semilla "oficial" ecuatoriano) | Documento público MSP; transcripción manual |
| **LOINC 2.83** (2026-08-19) | Códigos de observaciones/órdenes; "Universal Lab Order Codes Value Set"; variantes **es-AR, es-MX, es-ES** (no hay es-EC) | "Free for use in both commercial and non-commercial systems"; incluir aviso de copyright en el producto; no modificar ni crear derivados sin permiso; avisar a Regenstrief antes de traducir; descarga requiere cuenta gratuita [F6][F7] |
| Tarifario de Prestaciones del SNS (MSP) | Códigos de prestaciones (lab, imagen) para pagos RPIS/RPC | Publicado en gob.ec [F8]. *Sin verificar*: creo que se basa en códigos tipo CPT (licencia AMA) → no reutilizar sin revisión legal |

Recomendación: catálogo propio corto basado en el 010A/012A, con columna opcional `loinc_code` (es-MX/es-ES como etiqueta de referencia) para interoperar a futuro. No hace falta LOINC para el MVP.

### C. Encaje en el código
- **No existen** órdenes, exámenes ni resultados en el modelo clínico (búsqueda de `laborator|exam|order` sin resultados en `apps/api/features/clinical`).
- Reutilizable:
  - Documento imprimible nuevo `lab_order` / `imaging_order` = HTML + struct + `pdfrender.Document` en `features/clinical/templates/documents.go:14-45`, registrado en `apps/api/routes/documents.go:15-20`; los tenants podrán personalizarlo vía `/document-templates` automáticamente.
  - Firma del médico: `signature_services.Signer.Sign` (`features/signatures/services/signer.go:94`) + `DocumentSignature` embebible (`features/clinical/models/document-signature.go:8-16`), igual que `Prescription` (`prescription.go:11`).
  - Diagnóstico CIE-10: `DiagnosisItem` (`record.go:10-13`) y buscador `/clinical/icd10/search` (`routes/clinical_routes.go:35`).
  - Catálogo global no-tenant con seed CSV embebido: patrón `Cie10Code` (`models/cie10-code.go:5-8`, `migrations/code-migrations/2026/seed_cie10_codes.go:15`).
- Falta: modelos `ExamCatalogItem` (global) + `ExamOrder`/`ExamOrderItem` (tenant, ligados a `MedicalRecord`) + `ExamResult` (valor/unidad/rango/observación o adjunto). Resultados escaneados dependen de **Adjuntos (§2)**.
- Ojo: `Prescription`, `SOAPRecord` y `VitalSigns` **no tienen `TenantID`** (`prescription.go:5-12`, `record.go:40-46`, `vital-signs.go:5-14`); se aíslan vía el `MedicalRecord`. Los modelos nuevos deben llevar `TenantID` para que `tenantdb` los filtre (CLAUDE.md, ADR 0002).

### D. Patrones UX (EHR establecidos)
- OpenEMR: orden desde el encuentro → buscar estudio → imprimir y entregar al paciente; al volver, el médico **transcribe** resultados en "Procedures"; alternativa: subir el escaneo a Documents, "pero queda más lejos de la nota clínica" [F9][F10].
- GNU Health: catálogo de "lab test types" con analitos y rangos normales; la solicitud solo exige ≥1 test; marca de urgente [F11]. Imagen: solicitud + resultado con imágenes no-DICOM [F12].

### E. Alcance
- Backend: catálogo seed (010A/012A), `exam_orders`, `exam_order_items`, `exam_results`; endpoints CRUD por paciente/consulta; PDF A5/A4 firmable; permisos `CREATE_EXAM_ORDER`, `RECORD_EXAM_RESULT` (Category `CLINICAL`).
- Frontend: sección "Exámenes" en el tab "complementario" del formulario de consulta (`medical-record-create-form.tsx:142,930`) con selección por grupos + "paquetes favoritos" (p. ej. "perfil lipídico"); vista de órdenes pendientes y carga de resultados en la ficha del paciente; gráfico de tendencia por analito (fase 2).
- Riesgos: resultados numéricos sin unidades/rangos normalizados; doble captura (PDF + valores).
- Preguntas: ¿basta adjuntar el PDF del laboratorio o se quieren valores estructurados (tendencias)? ¿Paquetes por tenant o por médico? ¿Incluir anatomía patológica (013A)?

---

## 2. Archivos adjuntos del paciente

### A. Normativa
- Las HCU deben ser confidenciales, cronológicas y de acceso solo autorizado (AM 115 Arts. 5, 11) [F2].
- Datos de salud = categoría especial; obligación de seguridad, confidencialidad, minimización y plazos de supresión (LOPDP Arts. 10 i, 25, 30, 31, 43) [F3]. Pengi como encargado necesita contrato de encargo con cada consultorio (Art. 4 define encargado; las obligaciones contractuales concretas no las verifiqué).
- **Retención**: sin plazo normativo vigente encontrado (ver §0). Recomendación de producto: conservar mientras exista la HCU (no permitir borrado físico por defecto; soft delete + auditoría) y documentar la política en el contrato. Pregunta legal abierta.

### B. Datos libres
N/A (no requiere catálogos). Formatos: PDF, JPG/PNG/HEIC; DICOM fuera de alcance (GNU Health tampoco lo soporta por defecto [F12]).

### C. Encaje en el código
- **No existen adjuntos de paciente.** Los únicos uploads multipart son: P12 del usuario (`features/signatures/handlers/signature-handler.go:97`), P12 SRI y logo del tenant (`features/tenants/handlers/tenant-handler.go:56,146-177`) y plantillas (`features/document-templates/handlers/document-template-handler.go:186-196`, con `http.MaxBytesReader`). "attachment" solo aparece en emails (`core/mailer/mailer.go:36-144`) y en `Content-Disposition`.
- Reutilizable: `tenantfiles.Store` (`core/tenantfiles/tenantfiles.go:27-33`) aísla por tenant y valida nombres (`:36-46`). Limitaciones: API `[]byte` en memoria (`:58-68`), sin listado, sin cifrado en reposo (escribe 0644 en disco, `:67`), un único volumen local (`docker-compose.dev.yaml` `./storage:/app/storage`; `routes/documents.go:10` "api_storage volume"). Para archivos ≤10-20 MB es suficiente.
- Auditoría de lectura: `audit.RecordAccess` (`core/audit/access.go:13-17`) ya existe para registrar visualizaciones/descargas de datos clínicos sensibles; los modelos con `IsAuditable()` auditan cambios.
- Falta: modelo `PatientAttachment` {TenantID, PatientID, MedicalRecordID?, Category (lab_result, imaging, external_report, photo, consent, other), FileName, MimeType, Size, SHA-256, StoredName, UploadedByID, TakenAt?} + endpoints upload/list/download/delete; validación de MIME por *magic bytes*; cuota por plan.

### D. Patrones UX
- OpenMRS Attachments: lugar central de adjuntos del paciente, también los subidos desde formularios (como "complex obs"); vinculables a la visita [F13].
- OpenEMR: árbol de "Documents" por categorías, con opción de asociar ("tag") el documento a un encuentro [F10][F14].

### E. Alcance
- Backend: modelo + 4 endpoints + migración; permisos `READ_PATIENT_ATTACHMENT`, `UPLOAD_PATIENT_ATTACHMENT`, `DELETE_PATIENT_ATTACHMENT`; límite de tamaño por request; quota por tenant/plan.
- Frontend: tab "Archivos" en la ficha del paciente (junto a `pages/clincal/patient/medical-documents-list.tsx`), drag & drop, captura con cámara del móvil, visor PDF/imagen, filtro por categoría, enlace desde la consulta.
- Riesgos: crecimiento de disco del droplet y respaldo del volumen (hoy no hay backup de archivos documentado aquí); fotos con datos sensibles; cifrado en reposo; malware en PDFs (servir siempre con `Content-Disposition: attachment` o visor sandbox).
- Preguntas: ¿límite de almacenamiento por plan? ¿Migrar a object storage (S3-compatible, coste) más adelante? ¿Quién puede borrar y si el borrado es lógico? ¿Plazo de retención a declarar al cliente?

---

## 3. Consentimiento informado firmable

### A. Normativa (fuente primaria: AM 5316, RO Edición Especial 510, 22-feb-2016 [F5])
- Formulario oficial **024** (DNEAIS-HCU-form.024; reemplazó al 024 del AM 138/2008) [F5] pp. 32-33; el AM 115 lo mantiene como **SNS-MSP/HCU-form.024/2021** con el mismo layout (bloques C Declaración, D Negativa, E Revocatoria) [F1] pp. 108-113.
- **Escrito obligatorio** en: cirugías de riesgo mayor; radiología bajo anestesia o con medio de contraste; radio/quimioterapia; endoscopías; **biopsias**; reproducción asistida; **prueba de VIH**; todo procedimiento de riesgo mayor; donante vivo (notariado); transfusiones (LOS Art. 77) [F5] pp. 41-42.
- Excepciones: emergencia (fundamentar en HC), tratamientos exigidos por ley, corrección intraoperatoria, **intervenciones de riesgo mínimo** [F5] p. 46.
- Contenido mínimo: título por procedimiento; establecimiento y servicio; fecha y hora; cédula/HCU; nombre; tipo de atención; diagnóstico CIE-10; descripción (qué es, cómo se hace, duración, **gráfico**); beneficios; riesgos frecuentes / poco frecuentes / específicos del paciente; alternativas; post-tratamiento; consecuencias de no hacerlo; texto de declaratoria literal; firmas [F5] pp. 48-50.
- Firmas: paciente (o representante legal: parentesco, cédula, teléfono; **huella digital si es analfabeto**), profesional (firma, sello, código), anestesiólogo si aplica. **Negativa**: firma del paciente o, si se niega, profesional + **testigo externo**. **Revocatoria**: firma del paciente y del profesional que la recibe [F5] pp. 42-50.
- Menores: firma padre/madre/tutor; informar verbalmente a >12 años [F5] pp. 43-44.
- Proceso: entregar el documento en la cita previa en cirugías planificadas; no puede exigirse como requisito de admisión; contenidos por procedimiento validados por el establecimiento y revisados cada 2 años [F5] pp. 40, 44-46.

### Validez de la firma del paciente (riesgo principal)
- La firma electrónica tiene validez de manuscrita solo si cumple el Art. 15 LCE (identidad verificable con dispositivos técnicos de la Ley, control exclusivo del firmante) [F4]. Una **firma dibujada en pantalla** no es claramente una firma electrónica conforme al Art. 15 → **su validez probatoria es dudosa; requiere opinión legal**. (Inferencia propia, no hay pronunciamiento oficial localizado.)
- Opciones por orden de seguridad jurídica:
  1. **Imprimir → firma manuscrita/huella → escanear y adjuntar** (depende de §2). Cumple el AM 5316 sin discusión.
  2. Paciente con su propio certificado (FirmaEC/P12): poco realista en consulta.
  3. Firma manuscrita capturada en tablet + PDF sellado con PAdES por el médico (`core/pdfsign`) + hash/fecha/IP: buena evidencia de integridad, validez como firma del paciente **no confirmada**.

### B. Datos libres
- Modelo de contenidos y textos literales del AM 5316 (licencia CC BY-NC 4.0 indicada en el documento de socialización, [F5] página de créditos — uso comercial de *ese PDF* no cubierto; los textos legales del acuerdo son normativa pública; confirmar).
- No hay catálogo oficial de plantillas por procedimiento: cada establecimiento las redacta ([F5] p. 44).

### C. Encaje en el código
- **No existe** consentimiento informado en el repo.
- Reutilizable: `pdfrender.Document` + plantilla por tenant (`core/pdfrender/pdfrender.go:37-59`); `signatureRequired` exige QR y nombre del firmante en las plantillas clínicas (`features/clinical/templates/documents.go:51-54`); `Signer.Sign` firma con el **P12 del usuario médico** (`signer.go:50,94`), no hay firma de paciente; el PDF firmado es inmutable (`document-signature.go:5-7`).
- Falta: `ConsentTemplate` (por tenant: procedimiento, descripción, riesgos, alternativas, imagen) + `InformedConsent` {PatientID, MedicalRecordID?, template snapshot, status: draft/signed/refused/revoked, firmante (paciente/representante + parentesco/cédula), testigo, adjunto escaneado, signed_at} y eventos de revocación.

### D. Patrones UX
- OpenEMR "Templates for Patient Documents": plantillas de documentos que el paciente revisa/firma en el portal [F14]. (Solo referencia de UX; no implica validez legal en Ecuador.)

### E. Alcance
- Backend: plantillas por tenant con variables (`{{paciente}}`, `{{diagnóstico}}`), PDF 024 (A4, 2 páginas), estados aceptado/negado/revocado, adjuntar escaneo firmado, firma PAdES del médico.
- Frontend: biblioteca de plantillas en Ajustes; desde la consulta "Generar consentimiento" → imprimir → "Subir firmado".
- Riesgos: validez de la firma en tablet; responsabilidad por contenidos médicos (los redacta el médico; no ofrecer textos clínicos "oficiales" de Pengi); versionar la plantilla usada (snapshot).
- Preguntas: ¿qué procedimientos hacen los clientes (biopsias, endoscopías, estéticos)? ¿Queremos captura en tablet aun con validez dudosa? ¿Se exige testigo/huella en la UI?

---

## 4. Plantillas SOAP por especialidad + frases rápidas

### A. Normativa
- El form. **002/2021 Consulta externa – anamnesis/examen físico** y el Art. 6 del AM 115 definen los bloques mínimos: motivo, antecedentes personales/familiares, enfermedad actual, constantes vitales y antropometría, revisión por sistemas, examen físico, diagnóstico CIE-10, plan de tratamiento [F1] p. 1; [F2] p. 10. La HCU debe contener datos "objetivos, científicos y veraces" y llenarse simultáneamente a la atención (Art. 5 f, g) [F2] p. 10 → riesgo de texto copiado no revisado (*copy-paste*), mitigable obligando a editar.
- No hay norma que regule plantillas/frases en sí.

### B. Datos libres
- No hay catálogo oficial de plantillas por especialidad. Fuente práctica: los bloques del form. 002/003 [F1] y las guías de práctica clínica del MSP (no revisadas aquí).

### C. Encaje en el código
- SOAP hoy son 4 textos libres (`record.go:40-46`), capturados con `FormTextArea` en el tab "soap" (`medical-record-create-form.tsx:747-…`), con borrador autoguardado (`useSoapDraft`, `:60,318`; modelo `medical-record-draft.go:10-12`).
- No existen plantillas ni frases. Falta: `NoteTemplate` {TenantID, OwnerUserID? (personal o compartida), Specialty, Name, Subjective/Objective/Assessment/Plan, Diagnoses?} y `QuickPhrase` {TenantID, OwnerUserID?, Shortcut (p. ej. `/normal`), Text, Field?}.
- El SOAP es libre → no requiere migrar datos existentes; es puramente aditivo.

### D. Patrones UX
- OpenEMR Nation Notes: categorías con "componentes" (texto reutilizable) que se insertan en la nota; plantillas compartibles entre usuarios para uniformar la documentación [F15].
- OpenEMR "Simple Note Templates" para notas cortas [F16].

### E. Alcance
- Backend: 2 modelos + CRUD; permiso `MANAGE_NOTE_TEMPLATES` (o reutilizar `CREATE_MEDICAL_RECORD` para las personales).
- Frontend: selector "Aplicar plantilla" en el tab SOAP (rellena solo campos vacíos o pide confirmar sobrescritura); autocompletado de frases con `/atajo` o `Ctrl+Espacio` en `FormTextArea`; gestión en Ajustes; 3-5 plantillas semilla (medicina general, pediatría, ginecología) escritas por un médico, no por Pengi.
- Riesgos: notas clonadas idénticas (calidad/medicolegal); i18n: las plantillas son contenido del usuario, no claves i18n.
- Preguntas: ¿plantillas por médico, por tenant o ambas? ¿Especialidades prioritarias de los clientes actuales? ¿Las plantillas pueden precargar diagnósticos CIE-10 y exámenes?

---

## 5. Medicamentos favoritos + vademécum CNMB + alerta de alergias

### A. Normativa
- **CNMB Décima Segunda Revisión (2026)**: 512 principios activos; aprobado por CONASA y vigente en 2026 tras su publicación en RO [F17][F18]. Oficializado por **AM 00007-2026, RO Suplemento 363 del 7-sep-2026** — *secundaria* [F19] (el PDF oficial incluye el acuerdo como imagen, no lo verifiqué).
- El CNMB es obligatorio para compras públicas (Ley de genéricos Art. 6, citada en [F18], sección "Base legal"); para consultorios privados lo entiendo como referencia, no como restricción de prescripción (inferencia; no hallé norma que lo limite).
- **Niveles de prescripción** [F18] Tabla 3: G (general/especialista/odontólogo/obstetriz, ambulatorio), E (especialista en la patología), E(p) (especialista con protocolo), H/H(p) (hospitalario/hospital del día), HE/HE(p), (p) requiere protocolo, K (kits de primer nivel).
- Alertas: AM 5316 pide registrar riesgos "específicos del paciente … por ejemplo alergias" [F5] p. 52; el form. 022/2021 (administración de medicamentos) registra alergias [F1] p. 106 (lectura de escaneo de baja resolución).

### B. Datos libres
| Fuente | Formato | Campos | Licencia |
|---|---|---|---|
| **cnmb.gob.ec** (portal oficial CONASA) | Export **CSV/XLS/PDF** sin login: `https://cnmb.gob.ec/app-api/public/medications/export/csv`; API JSON `…/app-api/public/medications` | `Codigo medicamento` (ATC nivel 5), `Codigo ATC`, `Clasificacion ATC`, `DCI`, `Nombre mostrado`, `Concentracion`, `Codigo/Nivel de prescripcion`, `Nivel de atencion`, `Via de administracion`, `Forma farmaceutica`, `Uso terapeutico`, `Riesgo en embarazo` (+ en JSON: sustancias controladas). Descargado hoy: **890 presentaciones, 512 DCI, 528 ATC-5**. | El PDF dice: "autoriza la reproducción … para fines académicos, científicos, docentes, de investigación o gestión pública … sin finalidad comercial. La reproducción … con fines comerciales requerirá autorización previa y expresa del CONASA" [F18] página de créditos. → **Pedir autorización escrita a CONASA** o validar con abogado que la lista (acto normativo) no es objeto de derecho de autor. |
| PDF CNMB 2026 | 142 págs. | Igual + glosario | Igual |
| ARCSA – base de registros sanitarios | Consulta web pública [F20] | Nombres comerciales | No evaluado (scraping no recomendado) |

Nota: el `Codigo medicamento` (ATC-5) **no es único por presentación** (p. ej. Amoxicilina J01CA04 tiene 500 mg y 250 mg/5 mL); la clave natural es ATC-5 + concentración + forma.

### C. Encaje en el código
- `PrescriptionItem.Medication` es texto libre (`prescription.go:17`); el formulario valida `medication: z.string().min(1)` (`medical-record-create-form.tsx:70`) y une items en `content` (`:225`); hay modo texto vs. estructurado (`:1042-1045`).
- Alergias: `Patient.Allergies` string "JSON array" (`patient.go:30`) y `MedicalRecord.Allergies` (`record.go:37`); se copian al paciente solo en primera consulta (`record-handler.go:30-56`); el front parsea JSON o CSV (`apps/web/src/lib/allergies.ts:1-21`) y muestra un **banner ámbar** en la consulta (`medical-record-create-form.tsx:485-500`). **No hay cruce alergia↔medicamento.**
- Catálogo global: mismo patrón que CIE-10 (modelo sin `TenantID` + seed CSV embebido + búsqueda `ILIKE … LIMIT 50`, `icd10-handler.go:37-39`).
- Falta: `CnmbMedication` (global, versionado por revisión), `FavoriteMedication` {TenantID, UserID, CnmbMedicationID? o texto libre, dosis/frecuencia/duración por defecto}, `PrescriptionItem.CnmbMedicationID *uint` (opcional, compatible con lo existente), y alergias estructuradas (lista de DCI/grupos ATC además del texto).

### Alerta de alergias (sin base de interacciones pagada)
- Viable: coincidencia **DCI exacta** y por **grupo ATC** (p. ej. alergia "penicilina" → bloquear J01C*), con tabla curada pequeña de sinónimos → ATC (penicilinas, cefalosporinas J01D, sulfonamidas J01E, AINE M01A, etc.). Alerta *no bloqueante* con confirmación registrada.
- No viable gratis: interacciones fármaco-fármaco y reactividad cruzada fina.
- Requisito previo: alergias como datos estructurados; las actuales son texto libre ("Penicilina: no. Sulfas: sí." aparece incluso en la muestra de `templates/documents.go:112`) → falsos negativos si no se migra/normaliza.

### D. Patrones UX
- OpenMRS: mostrar las alergias del paciente en el mismo lugar y momento en que se ordena el fármaco ("Allergy UI in Orders Workflow") [F21]; "Order Sets" para órdenes predefinidas en un clic [F22].

### E. Alcance
- Backend: seed CNMB (CSV embebido o job de importación por revisión), búsqueda `/clinical/medications/search`, favoritos CRUD por usuario, endpoint de verificación de alergias.
- Frontend: autocompletar medicamento (favoritos primero, luego CNMB, luego texto libre), rellena concentración/forma/vía; badge de nivel de prescripción y riesgo en embarazo; alerta modal si hay alergia coincidente; editor de alergias con chips (DCI/grupo).
- Riesgos: **licencia comercial del CNMB**; actualizaciones trimestrales/anuales del cuadro (re-seed con migración nueva, nunca editar la ya publicada); responsabilidad por alertas falsas negativas (comunicar como ayuda, no garantía); medicamentos fuera del CNMB (marcas comerciales) siguen en texto libre.
- Preguntas: ¿pedimos autorización a CONASA? ¿Favoritos por médico o por consultorio? ¿Migramos alergias de texto a estructurado (manual/asistido)? ¿Mostrar marcas comerciales (ARCSA) o solo DCI?

---

## Recomendación priorizada

| Orden | Función | Por qué | Depende de | Tamaño |
|---|---|---|---|---|
| 1 | **Adjuntos del paciente** | Base de todo: resultados escaneados (§1), consentimiento firmado a mano (§3), estudios externos. Infra lista (`tenantfiles`, `audit`). | — | M |
| 2 | **Plantillas SOAP + frases rápidas** | Ahorro de tiempo diario, 100 % aditivo, sin riesgo regulatorio ni licencias. | — | S-M |
| 3 | **Órdenes lab/imagen** (orden impresa + resultado como adjunto; valores estructurados en fase 2) | Formato oficial claro (010A/012A), reutiliza `pdfrender` + firma. | 1 (resultados) | M |
| 4 | **Favoritos + CNMB + alerta de alergias** | Alto valor, pero bloqueado por licencia CONASA y por alergias en texto libre. Favoritos en texto libre pueden salir antes sin CNMB. | Autorización CONASA; alergias estructuradas | M-L |
| 5 | **Consentimiento informado** | Necesario solo para procedimientos de riesgo mayor; validez de firma en tablet incierta. MVP = plantilla + PDF 024 + subir escaneo firmado. | 1, plantillas (patrón de §2/§4) | M |

Dependencias clave: **Adjuntos → Órdenes (resultados) y → Consentimiento (escaneo firmado)**. **Alergias estructuradas → alerta al recetar**. Las plantillas de consentimiento pueden reutilizar el modelo de plantillas de §4.

## Preguntas abiertas para el PO
1. ¿Cuota de almacenamiento por plan y política de retención/borrado de adjuntos que declaramos al cliente (no hay plazo MSP vigente)?
2. ¿Contrato de encargado de tratamiento (LOPDP) y evaluación de impacto (Art. 42 b) ya existen?
3. ¿Solicitar autorización comercial a CONASA para el CNMB, o consultar si es de dominio público como acto normativo?
4. ¿Firma del paciente en tablet aceptable con validez incierta, o solo escaneo de papel?
5. ¿Resultados de laboratorio estructurados (tendencias) o solo PDF adjunto en el MVP?
6. ¿Plantillas/favoritos por médico, por consultorio o ambos? ¿Especialidades prioritarias?

---

## Fuentes

| Id | Fuente | Tipo |
|---|---|---|
| F1 | MSP, AM 00115-2021 Reglamento e instructivo de manejo de la HCU (Anexo 1 + formularios + instructivos, 304 págs.), RO n.º 378 26-ene-2021. http://www.acess.gob.ec/wp-content/uploads/2023/01/AM-00115-2021-Reglamento-e-instructivo-de-manejo-de-la-historia-clinica-unica.pdf | Primaria (ACESS, agencia adscrita al MSP) |
| F2 | MSP, AM 00115-2021 (articulado, 21 págs.). http://www.acess.gob.ec/wp-content/uploads/2023/12/Acuerdo-Ministerial-00115-2021-ENE-10-FORMULARIOS-HISTORIA-CLINICA.pdf | Primaria |
| F3 | Ley Orgánica de Protección de Datos Personales, 5.º Supl. RO 459, 26-may-2021. https://consejodecomunicacion.gob.ec/wp-content/uploads/downloads/2021/07/lotaip/Ley%20Org%C3%A1nica%20de%20Protecci%C3%B3n%20de%20Datos%20Personales.pdf | Primaria (copia institucional) |
| F4 | Ley de Comercio Electrónico, Firmas Electrónicas y Mensajes de Datos (Ley 2002-67). https://www.arcotel.gob.ec/wp-content/uploads/downloads/2015/04/LEY-COMERCIO-ELECTRONICO-FIRMAS-ELECTRONICAS-Y-MENSAJE-DE-DATOS.pdf | Primaria (copia ARCOTEL 2015; reformas posteriores no verificadas) |
| F5 | MSP, AM 5316 Modelo de Gestión de Aplicación del Consentimiento Informado, RO EE 510 22-feb-2016 (documento de socialización 2017). https://www.salud.gob.ec/wp-content/uploads/2022/09/A.M.5316-Consentimiento-Informado_-AM-5316.pdf | Primaria |
| F6 | LOINC License. https://loinc.org/license/ | Primaria |
| F7 | LOINC International / Downloads (v2.83, variantes es-AR/es-MX/es-ES). https://loinc.org/international/ , https://loinc.org/downloads/ | Primaria |
| F8 | Tarifario de Prestaciones para el SNS. https://www.gob.ec/regulaciones/tarifario-prestaciones-sistema-nacional-salud | Primaria (no descargado) |
| F9 | OpenEMR wiki, Procedures Module Configuration for Manual Result Entry. https://open-emr.org/wiki/index.php/Procedures_Module_Configuration_for_Manual_Result_Entry | Primaria (producto) |
| F10 | OpenEMR wiki, Procedure configuration & order process. https://www.open-emr.org/wiki/index.php/Procedure_configuration_%26_order_process | Primaria (producto) |
| F11 | GNU Health docs, Laboratory. https://docs.gnuhealth.org/his/userguide/modules/laboratory.html | Primaria (producto) |
| F12 | GNU Health docs, Imaging. https://docs.gnuhealth.org/his/_sources/userguide/modules/imaging.rst.txt | Primaria (producto) |
| F13 | OpenMRS wiki, Attachments module. https://openmrs.atlassian.net/wiki/spaces/projects/pages/26936249/Build+new+UI+for+Attachments+Module+as+an+OWA | Primaria (producto) |
| F14 | OpenEMR wiki, Templates for Patient Documents. https://www.open-emr.org/wiki/index.php/Templates_for_Patient_Documents | Primaria (producto) |
| F15 | OpenEMR wiki, Nation Notes. https://www.open-emr.org/wiki/index.php/Nation_Notes | Primaria (producto) |
| F16 | OpenEMR wiki, Simple Note Templates. https://www.open-emr.org/wiki/index.php/Simple_Note_Templates | Primaria (producto) |
| F17 | MSP, nota de prensa aprobación CNMB 12.ª revisión (512 principios activos). https://www.salud.gob.ec/gobierno-del-presidente-noboa-marca-un-hito-en-salud-aprueba-un-nuevo-cuadro-nacional-de-medicamentos-basicos-con-512-principios-activos-y-la-politica-nacional-de-recursos-humanos-en-salud/ | Primaria (comunicado) |
| F18 | CONASA, CNMB Décima Segunda Revisión 2026 (PDF) https://www.conasa.gob.ec/wp-content/uploads/2026/09/Cuadro%20Nacional%20de%20Medicamentos%20FINAL%202026.pdf y portal/export https://cnmb.gob.ec/es | Primaria |
| F19 | Consultorsalud / Primicias, AM 00007-2026 RO Supl. 363. https://consultorsalud.com/ecuador-cuadro-nacional-medicamentos-basicos/ , https://www.primicias.ec/sociedad/nuevo-cuadro-medicamentos-basicos-2026-crisis-salud-ecuador-129641/ | **Secundaria** |
| F20 | ARCSA, consulta pública de registros sanitarios. https://aplicaciones.controlsanitario.gob.ec/publico/consultas/index | Primaria (no evaluada) |
| F21 | OpenMRS wiki, Allergy UI in Orders Workflow. https://openmrs.atlassian.net/wiki/spaces/projects/pages/470515713/Allergy+UI+in+Orders+Workflow | Primaria (producto) |
| F22 | OpenMRS wiki, Order Sets in OpenMRS O3. https://openmrs.atlassian.net/wiki/spaces/projects/pages/962756610/Order+Sets+in+OpenMRS+O3 | Primaria (producto) |
