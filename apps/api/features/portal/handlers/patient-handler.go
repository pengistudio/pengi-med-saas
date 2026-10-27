package portal_handlers

import (
	"fmt"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	agenda_models "pengi-med-saas/features/agenda/models"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_services "pengi-med-saas/features/portal/services"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// timeNow is the clock of the portal handlers (tests may replace it).
var timeNow = time.Now

// servedMimeTypes are the result file types served as such; anything else
// goes out as an opaque download (same rule as staff attachment downloads).
var servedMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

// Audit entity types of portal downloads.
const (
	auditAttachment  = "patient_attachments"
	auditPrescriptn  = "prescriptions"
	auditCertificate = "medical_certificates"
	auditReport      = "medical_reports"
	auditInvoice     = "invoices"
)

// PatientHandler serves the signed-in patient's own data. Every query is
// bound to the session's tenant (tenantdb.ForTenant) and filtered by the
// session's patient; no patient id is ever read from the URL or body.
type PatientHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	// files: signed PDFs, RIDEs and signed XMLs. attachments: the encrypted
	// patient attachment store (nil when ATTACHMENT_ENCRYPTION_KEY is missing).
	files       tenantfiles.Store
	attachments tenantfiles.Store
	sriDocs     sri_document.Documents
}

func NewPatientHandler(db *gorm.DB, logger *zap.Logger, files, attachments tenantfiles.Store, sriDocs sri_document.Documents) *PatientHandler {
	return &PatientHandler{db: db, logger: logger, files: files, attachments: attachments, sriDocs: sriDocs}
}

// session returns the tenant-bound handle and the ids of the session.
func (h *PatientHandler) session(c *gin.Context) (*gorm.DB, uint, uint) {
	tenantID := portal_middleware.Clinic(c).Tenant.ID
	return tenantdb.ForTenant(h.db, tenantID), tenantID, portal_middleware.Patient(c).ID
}

func notFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "portal.error.not_found", core_errors.ErrPortalNotFound)
}

func fileUnavailable() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "portal.error.file_unavailable", core_errors.ErrPortalFileUnavailable)
}

func paramID(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	return uint(id), err == nil && id > 0
}

func clinicDate(t time.Time) string {
	return t.In(whatsapp_services.ClinicLocation()).Format("2006-01-02")
}

func doctorName(d *doctor_models.Doctor, fallback string) string {
	if d != nil && d.FullName != "" {
		return d.FullName
	}
	return fallback
}

func specialty(d *doctor_models.Doctor) (code, other string) {
	if d == nil {
		return "", ""
	}
	return d.Specialty, d.SpecialtyOther
}

// serveFile writes data as a download named fileName.
func serveFile(c *gin.Context, contentType, fileName string, data []byte) {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": fileName})
	if disposition == "" {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, contentType, data)
}

// Me is GET /portal/me.
func (h *PatientHandler) Me(c *gin.Context) envelope.Response {
	p := portal_middleware.Patient(c)
	return envelope.SuccessResponse(portal_dto.MeResponse{
		Patient: portal_dto.PortalPatient{FirstName: p.FirstName, LastName: p.LastName, Document: p.Document, Email: p.Email, Phone: p.Phone},
		Clinic:  ClinicInfo(portal_middleware.Clinic(c)),
	}, "portal.me.get.success")
}

// ListAppointments is GET /portal/appointments, newest first.
func (h *PatientHandler) ListAppointments(c *gin.Context) envelope.Response {
	db, _, patientID := h.session(c)
	settings := portal_middleware.Clinic(c).Settings
	now := timeNow()
	var appts []clinical_models.Appointment
	if err := db.Preload("Doctor").Where("patient_id = ?", patientID).
		Order("date DESC").Order("start_time DESC").Order("id DESC").Find(&appts).Error; err != nil {
		h.logger.Error("portal: failed to list appointments", zap.Error(err))
		return internalError()
	}
	typeIDs := []uint{}
	for _, a := range appts {
		if a.AppointmentTypeID != nil {
			typeIDs = append(typeIDs, *a.AppointmentTypeID)
		}
	}
	typeNames := map[uint]string{}
	if len(typeIDs) > 0 {
		var types []agenda_models.AppointmentType
		if err := db.Unscoped().Select("id", "name").Where("id IN ?", typeIDs).Find(&types).Error; err != nil {
			h.logger.Error("portal: failed to load appointment types", zap.Error(err))
			return internalError()
		}
		for _, t := range types {
			typeNames[t.ID] = t.Name
		}
	}
	items := make([]portal_dto.Appointment, 0, len(appts))
	for _, a := range appts {
		item := portal_dto.Appointment{
			ID: a.ID, Date: clinicDate(a.Date), StartTime: a.StartTime, EndTime: a.EndTime, Status: a.Status,
			DoctorName: doctorName(a.Doctor, ""), Location: a.Location,
			CanCancel: portal_services.CanCancel(a, settings, now),
		}
		item.Specialty, item.SpecialtyOther = specialty(a.Doctor)
		if a.AppointmentTypeID != nil {
			item.AppointmentType = typeNames[*a.AppointmentTypeID]
		}
		items = append(items, item)
	}
	return envelope.SuccessResponse(items, "portal.appointments.list.success")
}

// ListResults is GET /portal/results: the patient's exam orders (not voided)
// with only their reviewed exams; orders without one are left out.
func (h *PatientHandler) ListResults(c *gin.Context) envelope.Response {
	db, _, patientID := h.session(c)
	var orders []clinical_models.ExamOrder
	err := db.Preload("Doctor").
		Preload("Items", "reviewed_at IS NOT NULL").
		Preload("Items.Attachments", "patient_attachments.patient_id = ?", patientID).
		Where("patient_id = ? AND status <> ?", patientID, clinical_models.ExamOrderStatusVoided).
		Order("created_at DESC").Order("id DESC").Find(&orders).Error
	if err != nil {
		h.logger.Error("portal: failed to list results", zap.Error(err))
		return internalError()
	}
	out := make([]portal_dto.ResultOrder, 0, len(orders))
	for _, o := range orders {
		items := make([]portal_dto.ResultItem, 0, len(o.Items))
		for _, it := range o.Items {
			if it.ReviewedAt == nil {
				continue
			}
			files := make([]portal_dto.ResultFile, 0, len(it.Attachments))
			for _, a := range it.Attachments {
				files = append(files, portal_dto.ResultFile{ID: a.ID, FileName: a.FileName, ContentType: a.MimeType})
			}
			items = append(items, portal_dto.ResultItem{ID: it.ID, Name: it.Name, ReviewedAt: *it.ReviewedAt, Files: files})
		}
		if len(items) == 0 {
			continue
		}
		out = append(out, portal_dto.ResultOrder{
			ID: o.ID, Number: clinical_models.FormatExamOrderNumber(o.Number), Date: clinicDate(o.CreatedAt),
			DoctorName: doctorName(o.Doctor, o.SignerName), Items: items,
		})
	}
	return envelope.SuccessResponse(out, "portal.results.list.success")
}

// DownloadResultFile is GET /portal/results/files/:id: an attachment, only
// when it covers a reviewed exam of a (not voided) order of this patient.
func (h *PatientHandler) DownloadResultFile(c *gin.Context) {
	db, tenantID, patientID := h.session(c)
	id, ok := paramID(c, "id")
	if !ok {
		envelope.Write(c, notFound())
		return
	}
	var attachment clinical_models.PatientAttachment
	err := db.Where("id = ? AND patient_id = ?", id, patientID).
		Where(`EXISTS (SELECT 1 FROM exam_order_item_attachments x
			JOIN exam_order_items i ON i.id = x.exam_order_item_id
			JOIN exam_orders o ON o.id = i.exam_order_id
			WHERE x.patient_attachment_id = patient_attachments.id
			AND i.tenant_id = ? AND i.deleted_at IS NULL AND i.reviewed_at IS NOT NULL
			AND o.tenant_id = ? AND o.deleted_at IS NULL AND o.patient_id = ? AND o.status <> ?)`,
			tenantID, tenantID, patientID, clinical_models.ExamOrderStatusVoided).
		Limit(1).Find(&attachment).Error
	if err != nil {
		h.logger.Error("portal: failed to load result file", zap.Error(err))
		envelope.Write(c, internalError())
		return
	}
	if attachment.ID == 0 {
		envelope.Write(c, notFound())
		return
	}
	if h.attachments == nil {
		envelope.Write(c, fileUnavailable())
		return
	}
	data, err := h.attachments.Read(tenantID, attachment.StoredName)
	if err != nil {
		h.logger.Error("portal: failed to read result file", zap.Uint("id", attachment.ID), zap.Error(err))
		envelope.Write(c, fileUnavailable())
		return
	}
	audit.RecordPortalAccess(h.db, tenantID, patientID, auditAttachment, attachment.ID)
	contentType := attachment.MimeType
	if !servedMimeTypes[contentType] {
		contentType = "application/octet-stream"
	}
	serveFile(c, contentType, attachment.FileName, data)
}

// signedDocument is one signed document with what the list and the download need.
type signedDocument struct {
	item       portal_dto.Document
	signedFile string
	audit      string
}

// signedDocuments returns the patient's signed documents of the given type
// (all types when docType is ""), narrowed to id when non-zero.
func (h *PatientHandler) signedDocuments(db *gorm.DB, tenantID, patientID uint, docType string, id uint) ([]signedDocument, error) {
	var out []signedDocument
	signed := "signed_at IS NOT NULL AND signed_file IS NOT NULL AND signed_file <> ''"
	byID := func(q *gorm.DB, column string) *gorm.DB {
		if id != 0 {
			return q.Where(column+" = ?", id)
		}
		return q
	}

	if docType == "" || docType == portal_dto.DocumentPrescription {
		// Prescriptions have no tenant_id: they belong to the medical record.
		var rows []clinical_models.Prescription
		q := db.Preload("Doctor").
			Joins("JOIN medical_records ON medical_records.id = prescriptions.medical_record_id").
			Where("medical_records.tenant_id = ? AND medical_records.patient_id = ? AND medical_records.deleted_at IS NULL", tenantID, patientID).
			Where("prescriptions.signed_at IS NOT NULL AND prescriptions.signed_file IS NOT NULL AND prescriptions.signed_file <> ''")
		if err := byID(q, "prescriptions.id").Find(&rows).Error; err != nil {
			return nil, err
		}
		recordIDs := make([]uint, 0, len(rows))
		for _, r := range rows {
			recordIDs = append(recordIDs, r.MedicalRecordID)
		}
		dates := map[uint]time.Time{}
		if len(recordIDs) > 0 {
			var records []clinical_models.MedicalRecord
			if err := db.Select("id", "date").Where("id IN ? AND patient_id = ?", recordIDs, patientID).Find(&records).Error; err != nil {
				return nil, err
			}
			for _, r := range records {
				dates[r.ID] = r.Date
			}
		}
		for _, r := range rows {
			date, ok := dates[r.MedicalRecordID]
			if !ok {
				date = *r.SignedAt
			}
			out = append(out, signedDocument{
				item:       portal_dto.Document{Type: portal_dto.DocumentPrescription, ID: r.ID, Date: clinicDate(date), DoctorName: doctorName(r.Doctor, r.SignerName), SignedAt: *r.SignedAt},
				signedFile: r.SignedFile, audit: auditPrescriptn,
			})
		}
	}

	if docType == "" || docType == portal_dto.DocumentCertificate {
		var rows []clinical_models.MedicalCertificate
		q := db.Preload("Doctor").Where("patient_id = ?", patientID).Where(signed)
		if err := byID(q, "id").Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, signedDocument{
				item:       portal_dto.Document{Type: portal_dto.DocumentCertificate, ID: r.ID, Date: clinicDate(r.CreatedAt), DoctorName: doctorName(r.Doctor, r.SignerName), SignedAt: *r.SignedAt},
				signedFile: r.SignedFile, audit: auditCertificate,
			})
		}
	}

	if docType == "" || docType == portal_dto.DocumentReport {
		var rows []clinical_models.MedicalReport
		q := db.Preload("Doctor").Where("patient_id = ?", patientID).Where(signed)
		if err := byID(q, "id").Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, signedDocument{
				item:       portal_dto.Document{Type: portal_dto.DocumentReport, ID: r.ID, Date: clinicDate(r.CreatedAt), DoctorName: doctorName(r.Doctor, r.SignerName), SignedAt: *r.SignedAt},
				signedFile: r.SignedFile, audit: auditReport,
			})
		}
	}
	return out, nil
}

// ListDocuments is GET /portal/documents: signed prescriptions, certificates
// and reports, most recently signed first.
func (h *PatientHandler) ListDocuments(c *gin.Context) envelope.Response {
	db, tenantID, patientID := h.session(c)
	docs, err := h.signedDocuments(db, tenantID, patientID, "", 0)
	if err != nil {
		h.logger.Error("portal: failed to list documents", zap.Error(err))
		return internalError()
	}
	sort.SliceStable(docs, func(i, j int) bool { return docs[i].item.SignedAt.After(docs[j].item.SignedAt) })
	items := make([]portal_dto.Document, 0, len(docs))
	for _, d := range docs {
		items = append(items, d.item)
	}
	return envelope.SuccessResponse(items, "portal.documents.list.success")
}

// DownloadDocument is GET /portal/documents/:type/:id/file: the stored signed
// PDF as-is (never re-rendered).
func (h *PatientHandler) DownloadDocument(c *gin.Context) {
	db, tenantID, patientID := h.session(c)
	docType := c.Param("type")
	id, ok := paramID(c, "id")
	if !ok || (docType != portal_dto.DocumentPrescription && docType != portal_dto.DocumentCertificate && docType != portal_dto.DocumentReport) {
		envelope.Write(c, notFound())
		return
	}
	docs, err := h.signedDocuments(db, tenantID, patientID, docType, id)
	if err != nil {
		h.logger.Error("portal: failed to load document", zap.Error(err))
		envelope.Write(c, internalError())
		return
	}
	if len(docs) != 1 {
		envelope.Write(c, notFound())
		return
	}
	doc := docs[0]
	pdf, err := h.files.Read(tenantID, doc.signedFile)
	if err != nil {
		h.logger.Error("portal: failed to read signed document", zap.String("type", docType), zap.Uint("id", id), zap.Error(err))
		envelope.Write(c, fileUnavailable())
		return
	}
	audit.RecordPortalAccess(h.db, tenantID, patientID, doc.audit, id)
	serveFile(c, "application/pdf", fmt.Sprintf("%s-%d.pdf", docType, id), pdf)
}

// authorizedInvoices is the patient's SRI-authorized invoices query.
func authorizedInvoices(db *gorm.DB, patientID uint) *gorm.DB {
	return db.Where("patient_id = ? AND status = ? AND access_key IS NOT NULL AND access_key <> ''", patientID, billing_models.InvoiceStatusAuthorized)
}

func invoiceNumber(inv billing_models.Invoice) string {
	return fmt.Sprintf("%s-%s-%s", inv.EstablishmentCode, inv.EmissionPointCode, inv.Sequential)
}

// ListInvoices is GET /portal/invoices: authorized invoices issued to the patient.
func (h *PatientHandler) ListInvoices(c *gin.Context) envelope.Response {
	db, _, patientID := h.session(c)
	var invoices []billing_models.Invoice
	if err := authorizedInvoices(db, patientID).Order("issue_date DESC").Order("id DESC").Find(&invoices).Error; err != nil {
		h.logger.Error("portal: failed to list invoices", zap.Error(err))
		return internalError()
	}
	items := make([]portal_dto.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, portal_dto.Invoice{
			ID: inv.ID, Number: invoiceNumber(inv), Date: clinicDate(inv.IssueDate), Total: inv.Total, AccessKey: *inv.AccessKey,
		})
	}
	return envelope.SuccessResponse(items, "portal.invoices.list.success")
}

func (h *PatientHandler) invoice(c *gin.Context, preload bool) (billing_models.Invoice, bool) {
	db, _, patientID := h.session(c)
	var invoice billing_models.Invoice
	id, ok := paramID(c, "id")
	if !ok {
		envelope.Write(c, notFound())
		return invoice, false
	}
	q := authorizedInvoices(db, patientID).Where("id = ?", id)
	if preload {
		q = q.Preload("Patient").Preload("Items")
	}
	if err := q.Limit(1).Find(&invoice).Error; err != nil {
		h.logger.Error("portal: failed to load invoice", zap.Error(err))
		envelope.Write(c, internalError())
		return invoice, false
	}
	if invoice.ID == 0 {
		envelope.Write(c, notFound())
		return invoice, false
	}
	return invoice, true
}

// DownloadInvoiceRide is GET /portal/invoices/:id/ride: the stored RIDE PDF
// (rendered and cached when missing, like the staff download).
func (h *PatientHandler) DownloadInvoiceRide(c *gin.Context) {
	invoice, ok := h.invoice(c, true)
	if !ok {
		return
	}
	clinic := portal_middleware.Clinic(c)
	pdf, err := sri_document.InvoiceRide(h.sriDocs, invoice, clinic.Tenant)
	if err != nil {
		h.logger.Error("portal: failed to get RIDE", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
		envelope.Write(c, fileUnavailable())
		return
	}
	audit.RecordPortalAccess(h.db, clinic.Tenant.ID, portal_middleware.Patient(c).ID, auditInvoice, invoice.ID)
	serveFile(c, "application/pdf", "factura_"+invoiceNumber(invoice)+".pdf", pdf)
}

// DownloadInvoiceXML is GET /portal/invoices/:id/xml: the signed XML the SRI
// authorized.
func (h *PatientHandler) DownloadInvoiceXML(c *gin.Context) {
	invoice, ok := h.invoice(c, false)
	if !ok {
		return
	}
	clinic := portal_middleware.Clinic(c)
	xml, err := h.files.Read(clinic.Tenant.ID, sri_document.InvoiceXMLName(*invoice.AccessKey))
	if err != nil {
		h.logger.Error("portal: failed to read invoice XML", zap.Uint("invoice_id", invoice.ID), zap.Error(err))
		envelope.Write(c, fileUnavailable())
		return
	}
	audit.RecordPortalAccess(h.db, clinic.Tenant.ID, portal_middleware.Patient(c).ID, auditInvoice, invoice.ID)
	serveFile(c, "application/xml", "factura_"+invoiceNumber(invoice)+".xml", xml)
}
