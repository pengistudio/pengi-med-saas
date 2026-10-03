package clinical_handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_services "pengi-med-saas/features/companies/services"
	auth_middleware "pengi-med-saas/features/users/middleware"
	user_models "pengi-med-saas/features/users/models"
)

// MaxAttachmentSize is the largest file a patient attachment may have.
const MaxAttachmentSize = 15 << 20

// attachmentFormOverhead is what the multipart request may carry besides the
// file itself (boundaries, part headers, the other fields).
const attachmentFormOverhead = 64 << 10

// attachmentMimeTypes are the only types accepted, as detected from the
// content (never the extension or the client's Content-Type).
var attachmentMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

const (
	attachmentDescriptionMax  = 255
	attachmentFileNameMax     = 255
	attachmentDeleteReasonMax = 500
)

// PatientAttachmentHandler serves a patient's attachments (Adjuntos). files is
// the encrypting store; nil when ATTACHMENT_ENCRYPTION_KEY is not configured,
// in which case every route answers 503 and the rest of the API still works.
type PatientAttachmentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	files  tenantfiles.Store
}

func NewPatientAttachmentHandler(db *gorm.DB, logger *zap.Logger, files tenantfiles.Store) *PatientAttachmentHandler {
	return &PatientAttachmentHandler{db: db, logger: logger, files: files}
}

// Mount registers the attachment routes on the patients group (":id" is the
// patient). guard returns the middleware that requires a permission; in
// production it is subscription_middleware.RequirePermission.
func (h *PatientAttachmentHandler) Mount(patients *gin.RouterGroup, guard func(permissionID string) gin.HandlerFunc) {
	patients.POST("/:id/attachments", guard("UPLOAD_PATIENT_ATTACHMENT"), envelope.Handle(h.UploadAttachment))
	patients.PUT("/:id/attachments/:attachment_id", guard("UPLOAD_PATIENT_ATTACHMENT"), envelope.Handle(h.UpdateAttachment))
	patients.GET("/:id/attachments", guard("READ_PATIENT_ATTACHMENT"), envelope.Handle(h.ListAttachments))
	patients.GET("/:id/attachments/:attachment_id/download", guard("READ_PATIENT_ATTACHMENT"), h.DownloadAttachment)
	patients.GET("/:id/attachments/:attachment_id/view", guard("READ_PATIENT_ATTACHMENT"), h.ViewAttachment)
	patients.DELETE("/:id/attachments/:attachment_id", guard("DELETE_PATIENT_ATTACHMENT"), envelope.Handle(h.DeleteAttachment))
	patients.GET("/:id/attachments-deleted", guard("DELETE_PATIENT_ATTACHMENT"), envelope.Handle(h.ListDeletedAttachments))
	patients.POST("/:id/attachments/:attachment_id/restore", guard("DELETE_PATIENT_ATTACHMENT"), envelope.Handle(h.RestoreAttachment))
}

func unavailableAttachments() envelope.Response {
	return envelope.ErrorResponse(http.StatusServiceUnavailable, "clinical.attachment.unavailable", core_errors.ErrClinicalAttachmentUnavailable)
}

// findPatient loads the request's patient (":id") within the tenant.
func (h *PatientAttachmentHandler) findPatient(c *gin.Context) (*clinical_models.Patient, *envelope.Response) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
		return nil, &resp
	}
	var patient clinical_models.Patient
	if err := tenantdb.For(c, h.db).First(&patient, id).Error; err != nil {
		resp := envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
		return nil, &resp
	}
	return &patient, nil
}

// AttachmentUsage is the tenant's attachment storage against its plan quota.
// Warning is set from 80% of the quota on (always when the quota is 0).
type AttachmentUsage struct {
	UsedBytes  int64 `json:"used_bytes"`
	QuotaBytes int64 `json:"quota_bytes"`
	Warning    bool  `json:"warning"`
}

// ListedAttachment is an attachment in the list, with the date of the
// consultation it is linked to (if any) so the list can show it.
type ListedAttachment struct {
	clinical_models.PatientAttachment
	MedicalRecordDate *time.Time `json:"medical_record_date,omitempty"`
}

// AttachmentList is the list response: the patient's attachments plus the
// tenant's storage usage.
type AttachmentList struct {
	Items []ListedAttachment `json:"items"`
	Usage AttachmentUsage    `json:"usage"`
}

// storageUsage sums the size of every attachment of the tenant — deleted ones
// too, since their files stay stored until purged — against the quota of its
// current plan.
func (h *PatientAttachmentHandler) storageUsage(c *gin.Context) (AttachmentUsage, error) {
	var used int64
	if err := tenantdb.For(c, h.db).Unscoped().Model(&clinical_models.PatientAttachment{}).
		Select("COALESCE(SUM(size), 0)").Scan(&used).Error; err != nil {
		return AttachmentUsage{}, err
	}
	quota, err := company_services.StorageQuotaBytes(h.db, tenantdb.TenantID(c))
	if err != nil {
		return AttachmentUsage{}, err
	}
	return AttachmentUsage{UsedBytes: used, QuotaBytes: quota, Warning: used*5 >= quota*4}, nil
}

// UploadAttachment stores one file (multipart field "file") with its category,
// optional exam date ("taken_at", the upload time when empty) and optional
// description.
func (h *PatientAttachmentHandler) UploadAttachment(c *gin.Context) envelope.Response {
	if h.files == nil {
		return unavailableAttachments()
	}
	// Cap the body before anything reads it: a larger request fails while
	// parsing instead of being read whole.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxAttachmentSize+attachmentFormOverhead)

	patient, errResp := h.findPatient(c)
	if errResp != nil {
		return *errResp
	}

	tooLarge := envelope.ErrorResponse(http.StatusRequestEntityTooLarge, "clinical.attachment.too_large", core_errors.ErrClinicalAttachmentTooLarge)
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return tooLarge
		}
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.file.required", core_errors.ErrClinicalInvalidRequest)
	}
	defer c.Request.MultipartForm.RemoveAll()

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.file.required", core_errors.ErrClinicalInvalidRequest)
	}
	defer file.Close()
	if header.Size > MaxAttachmentSize {
		return tooLarge
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxAttachmentSize+1))
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.file.required", core_errors.ErrClinicalInvalidRequest)
	}
	if len(data) > MaxAttachmentSize {
		return tooLarge
	}
	if len(data) == 0 {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.file.required", core_errors.ErrClinicalInvalidRequest)
	}

	mimeType := http.DetectContentType(data)
	if !attachmentMimeTypes[mimeType] {
		return envelope.ErrorResponse(http.StatusUnsupportedMediaType, "clinical.attachment.type.unsupported", core_errors.ErrClinicalAttachmentType)
	}

	category := strings.TrimSpace(c.PostForm("category"))
	if !clinical_models.IsAttachmentCategory(category) {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.category.invalid", core_errors.ErrClinicalAttachmentCategory)
	}

	// The exam day: today (server time zone, America/Guayaquil) unless given.
	takenAt := dateOnly(time.Now())
	if raw := strings.TrimSpace(c.PostForm("taken_at")); raw != "" {
		parsed, errResp := parseAttachmentTakenAt(raw)
		if errResp != nil {
			return *errResp
		}
		takenAt = parsed
	}

	description := strings.TrimSpace(c.PostForm("description"))
	if errResp := checkAttachmentDescription(description); errResp != nil {
		return *errResp
	}

	// Optional consultation to link it to: one of this patient's, in the tenant.
	var medicalRecordID *uint
	if raw := strings.TrimSpace(c.PostForm("medical_record_id")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || id == 0 {
			return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
		}
		if errResp := h.checkMedicalRecord(c, patient.ID, uint(id)); errResp != nil {
			return *errResp
		}
		recordID := uint(id)
		medicalRecordID = &recordID
	}

	// Check-then-insert: two uploads racing at the edge of the quota can both
	// pass and overshoot it by up to one file each. Accepted, no locking.
	usage, err := h.storageUsage(c)
	if err != nil {
		h.logger.Error("failed to compute attachment storage usage", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	if usage.UsedBytes+int64(len(data)) > usage.QuotaBytes {
		return envelope.ErrorResponse(http.StatusForbidden, "plan.limit.storage", core_errors.ErrPlanStorageQuota)
	}

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	duplicate := h.findDuplicate(c, patient.ID, hash)
	attachment := clinical_models.PatientAttachment{
		TenantID:        tenantdb.TenantID(c),
		PatientID:       patient.ID,
		MedicalRecordID: medicalRecordID,
		Category:        category,
		TakenAt:         takenAt,
		Description:     description,
		FileName:        attachmentFileName(header.Filename),
		MimeType:        mimeType,
		Size:            int64(len(data)),
		SHA256:          hash,
		StoredName:      "attachments/" + uuid.NewString(),
	}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		attachment.UploadedByID = uint(uid)
	}

	if err := h.files.Write(attachment.TenantID, attachment.StoredName, data); err != nil {
		h.logger.Error("failed to store patient attachment", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.attachment.upload.error", core_errors.ErrClinicalAttachmentSaveError)
	}
	if err := tenantdb.For(c, h.db).Create(&attachment).Error; err != nil {
		h.logger.Error("failed to create patient attachment", zap.Error(err))
		_ = h.files.Remove(attachment.TenantID, attachment.StoredName)
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.attachment.upload.error", core_errors.ErrClinicalAttachmentSaveError)
	}

	return envelope.SuccessResponse(uploadedAttachment{PatientAttachment: attachment, DuplicateOf: duplicate}, "clinical.attachment.upload.success")
}

// AttachmentDuplicate points at an earlier, non-deleted attachment of the same
// patient with identical content (same SHA-256).
type AttachmentDuplicate struct {
	ID        uint      `json:"id"`
	FileName  string    `json:"file_name"`
	CreatedAt time.Time `json:"created_at"`
}

// uploadedAttachment is the upload response: the stored attachment plus, when
// its content already exists for the patient, duplicate_of. The duplicate is
// stored anyway; the field is a non-blocking warning for the client.
type uploadedAttachment struct {
	clinical_models.PatientAttachment
	DuplicateOf *AttachmentDuplicate `json:"duplicate_of,omitempty"`
}

// findDuplicate looks for the oldest live attachment of the patient (within the
// tenant) with the given hash. A deleted original does not count.
func (h *PatientAttachmentHandler) findDuplicate(c *gin.Context, patientID uint, sha string) *AttachmentDuplicate {
	var original clinical_models.PatientAttachment
	err := tenantdb.For(c, h.db).Where("patient_id = ? AND sha256 = ?", patientID, sha).Order("id ASC").First(&original).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Warn("failed to check duplicate attachment", zap.Error(err))
		}
		return nil
	}
	return &AttachmentDuplicate{ID: original.ID, FileName: original.FileName, CreatedAt: original.CreatedAt}
}

func parseAttachmentTakenAt(raw string) (time.Time, *envelope.Response) {
	parsed, err := parseFlexibleDate(raw)
	if err != nil {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.taken_at.invalid", core_errors.ErrClinicalInvalidRequest)
		return time.Time{}, &resp
	}
	return dateOnly(parsed), nil
}

func checkAttachmentDescription(description string) *envelope.Response {
	if utf8.RuneCountInString(description) > attachmentDescriptionMax {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.description.too_long", core_errors.ErrClinicalInvalidRequest)
		return &resp
	}
	return nil
}

// checkMedicalRecord makes sure a consultation an attachment is linked to
// exists in the tenant (another clinic's is simply not found) and belongs to
// the attachment's patient.
func (h *PatientAttachmentHandler) checkMedicalRecord(c *gin.Context, patientID, recordID uint) *envelope.Response {
	var record clinical_models.MedicalRecord
	if err := tenantdb.For(c, h.db).Select("id", "patient_id").First(&record, recordID).Error; err != nil {
		resp := envelope.ErrorResponse(http.StatusNotFound, "clinical.attachment.medical_record.not_found", core_errors.ErrClinicalRecordNotFound)
		return &resp
	}
	if record.PatientID != patientID {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.medical_record.other_patient", core_errors.ErrClinicalAttachmentRecordPatient)
		return &resp
	}
	return nil
}

// optionalRecordID is a JSON field that tells "absent" (Set false) from
// "null" (Set true, ID nil) and from a number.
type optionalRecordID struct {
	Set bool
	ID  *uint
}

func (o *optionalRecordID) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.ID = nil
		return nil
	}
	var id uint
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}
	o.ID = &id
	return nil
}

// updateAttachmentRequest carries the editable metadata; a field left out is
// left as it was. medical_record_id links the attachment to a consultation of
// the same patient (a number) or unlinks it (null). The file (bytes, name,
// type, hash) is never editable.
type updateAttachmentRequest struct {
	Category        *string          `json:"category"`
	TakenAt         *string          `json:"taken_at"`
	Description     *string          `json:"description"`
	MedicalRecordID optionalRecordID `json:"medical_record_id"`
}

// UpdateAttachment edits an attachment's category, exam date, description and
// consultation link. The change is recorded by the audit callbacks (old and
// new values).
func (h *PatientAttachmentHandler) UpdateAttachment(c *gin.Context) envelope.Response {
	if h.files == nil {
		return unavailableAttachments()
	}
	patient, errResp := h.findPatient(c)
	if errResp != nil {
		return *errResp
	}
	attachmentID, err := strconv.ParseUint(c.Param("attachment_id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req updateAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}

	updates := map[string]interface{}{}
	if req.Category != nil {
		category := strings.TrimSpace(*req.Category)
		if !clinical_models.IsAttachmentCategory(category) {
			return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.category.invalid", core_errors.ErrClinicalAttachmentCategory)
		}
		updates["category"] = category
	}
	if req.TakenAt != nil {
		takenAt, errResp := parseAttachmentTakenAt(strings.TrimSpace(*req.TakenAt))
		if errResp != nil {
			return *errResp
		}
		updates["taken_at"] = takenAt
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if errResp := checkAttachmentDescription(description); errResp != nil {
			return *errResp
		}
		updates["description"] = description
	}

	db := tenantdb.For(c, h.db)
	var attachment clinical_models.PatientAttachment
	if err := db.Where("patient_id = ?", patient.ID).First(&attachment, attachmentID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAttachmentNotFound)
	}
	if req.MedicalRecordID.Set {
		if id := req.MedicalRecordID.ID; id != nil {
			if errResp := h.checkMedicalRecord(c, patient.ID, *id); errResp != nil {
				return *errResp
			}
			updates["medical_record_id"] = *id
		} else {
			updates["medical_record_id"] = nil
		}
	}
	if len(updates) > 0 {
		if err := db.Model(&attachment).Updates(updates).Error; err != nil {
			h.logger.Error("failed to update patient attachment", zap.Uint("id", attachment.ID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.attachment.upload.error", core_errors.ErrClinicalAttachmentSaveError)
		}
		if err := db.First(&attachment, attachment.ID).Error; err != nil {
			h.logger.Error("failed to reload patient attachment", zap.Uint("id", attachment.ID), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		}
	}
	return envelope.SuccessResponse(attachment, "clinical.attachment.update.success")
}

// dateOnly is the calendar day t falls on (in t's own location), as UTC
// midnight: how a date column reads back.
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// attachmentFileName is the uploaded file's base name, shortened to fit.
func attachmentFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, `\`, "/")))
	if name == "." || name == "/" || name == "" {
		return "file"
	}
	if utf8.RuneCountInString(name) > attachmentFileNameMax {
		name = string([]rune(name)[:attachmentFileNameMax])
	}
	return name
}

// ListAttachments returns the patient's attachments, newest exam date first,
// optionally only one category ("?category=") and/or only those linked to one
// consultation ("?medical_record_id="), with the tenant's storage usage.
func (h *PatientAttachmentHandler) ListAttachments(c *gin.Context) envelope.Response {
	if h.files == nil {
		return unavailableAttachments()
	}
	patient, errResp := h.findPatient(c)
	if errResp != nil {
		return *errResp
	}

	query := tenantdb.For(c, h.db).Where("patient_id = ?", patient.ID)
	if category := c.Query("category"); category != "" {
		if !clinical_models.IsAttachmentCategory(category) {
			return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.category.invalid", core_errors.ErrClinicalAttachmentCategory)
		}
		query = query.Where("category = ?", category)
	}
	if raw := c.Query("medical_record_id"); raw != "" {
		recordID, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
		}
		query = query.Where("medical_record_id = ?", recordID)
	}

	attachments := []clinical_models.PatientAttachment{}
	if err := query.Order("taken_at DESC").Order("id DESC").Find(&attachments).Error; err != nil {
		h.logger.Error("failed to list patient attachments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	items, err := h.withRecordDates(c, attachments)
	if err != nil {
		h.logger.Error("failed to load attachment consultations", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	usage, err := h.storageUsage(c)
	if err != nil {
		h.logger.Error("failed to compute attachment storage usage", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(AttachmentList{Items: items, Usage: usage}, "clinical.attachment.list.success")
}

// withRecordDates adds the date of each linked consultation, in one query.
func (h *PatientAttachmentHandler) withRecordDates(c *gin.Context, attachments []clinical_models.PatientAttachment) ([]ListedAttachment, error) {
	var ids []uint
	for _, a := range attachments {
		if a.MedicalRecordID != nil {
			ids = append(ids, *a.MedicalRecordID)
		}
	}
	dates := map[uint]time.Time{}
	if len(ids) > 0 {
		var records []clinical_models.MedicalRecord
		if err := tenantdb.For(c, h.db).Select("id", "date").Where("id IN ?", ids).Find(&records).Error; err != nil {
			return nil, err
		}
		for _, r := range records {
			dates[r.ID] = r.Date
		}
	}
	items := make([]ListedAttachment, 0, len(attachments))
	for _, a := range attachments {
		item := ListedAttachment{PatientAttachment: a}
		if a.MedicalRecordID != nil {
			if date, ok := dates[*a.MedicalRecordID]; ok {
				item.MedicalRecordDate = &date
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// DownloadAttachment streams the original (decrypted) file as a download with
// its detected MIME type, and records the access.
func (h *PatientAttachmentHandler) DownloadAttachment(c *gin.Context) {
	h.serveAttachment(c, false)
}

// attachmentViewCSP locks a viewed file down: no scripts, no network, no
// framing of anything else, even if a browser were to mis-render the content.
const attachmentViewCSP = "sandbox; default-src 'none'; img-src 'self' data: blob:; object-src 'self'"

// ViewAttachment serves the same decrypted file inline (to show it in the
// app), only ever as pdf/jpeg/png, under a restrictive CSP, and records the
// access like a download.
func (h *PatientAttachmentHandler) ViewAttachment(c *gin.Context) {
	h.serveAttachment(c, true)
}

func (h *PatientAttachmentHandler) serveAttachment(c *gin.Context, inline bool) {
	if h.files == nil {
		envelope.Write(c, unavailableAttachments())
		return
	}
	patientID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	attachmentID, err2 := strconv.ParseUint(c.Param("attachment_id"), 10, 32)
	if err != nil || err2 != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest))
		return
	}

	var attachment clinical_models.PatientAttachment
	if err := tenantdb.For(c, h.db).Where("patient_id = ?", patientID).First(&attachment, attachmentID).Error; err != nil {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAttachmentNotFound))
		return
	}

	data, err := h.files.Read(attachment.TenantID, attachment.StoredName)
	if err != nil {
		h.logger.Error("failed to read patient attachment", zap.Uint("id", attachment.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalAttachmentReadError))
		return
	}

	audit.RecordAccess(h.db, c, "patient_attachments", attachment.ID, &attachment.PatientID)

	// Only the accepted types are ever served: anything else (which upload
	// never stores) goes out as an opaque download, never as HTML.
	contentType := attachment.MimeType
	known := attachmentMimeTypes[contentType]
	if !known {
		contentType = "application/octet-stream"
	}
	dispositionType := "attachment"
	if inline && known {
		dispositionType = "inline"
		c.Header("Content-Security-Policy", attachmentViewCSP)
	}
	disposition := mime.FormatMediaType(dispositionType, map[string]string{"filename": attachment.FileName})
	if disposition == "" { // a name FormatMediaType can't encode
		disposition = dispositionType
	}
	c.Header("Content-Disposition", disposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, data)
}

type deleteAttachmentRequest struct {
	Reason string `json:"reason"`
}

// DeleteAttachment soft-deletes an attachment. The reason is required; who and
// why are stored in the same UPDATE that sets deleted_at (audited by the
// callbacks). The encrypted file stays in the store so it can be restored.
func (h *PatientAttachmentHandler) DeleteAttachment(c *gin.Context) envelope.Response {
	if h.files == nil {
		return unavailableAttachments()
	}
	patient, errResp := h.findPatient(c)
	if errResp != nil {
		return *errResp
	}
	attachmentID, err := strconv.ParseUint(c.Param("attachment_id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req deleteAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.delete.reason.required", core_errors.ErrClinicalInvalidRequest)
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.delete.reason.required", core_errors.ErrClinicalInvalidRequest)
	}
	if utf8.RuneCountInString(reason) > attachmentDeleteReasonMax {
		return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.delete.reason.too_long", core_errors.ErrClinicalInvalidRequest)
	}

	db := tenantdb.For(c, h.db)
	var attachment clinical_models.PatientAttachment
	if err := db.Where("patient_id = ?", patient.ID).First(&attachment, attachmentID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAttachmentNotFound)
	}
	updates := map[string]interface{}{"deleted_at": time.Now(), "delete_reason": reason}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		updates["deleted_by_id"] = uint(uid)
	}
	if err := db.Model(&attachment).Updates(updates).Error; err != nil {
		h.logger.Error("failed to delete patient attachment", zap.Uint("id", attachment.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.attachment.delete.error", core_errors.ErrClinicalAttachmentSaveError)
	}
	return envelope.SuccessResponse(nil, "clinical.attachment.delete.success")
}

// deletedAttachment is a deleted attachment with who deleted it.
type deletedAttachment struct {
	clinical_models.PatientAttachment
	DeletedByName string `json:"deleted_by_name,omitempty"`
}

// ListDeletedAttachments returns the patient's deleted attachments, most
// recently deleted first, with who, when (deleted_at) and why.
func (h *PatientAttachmentHandler) ListDeletedAttachments(c *gin.Context) envelope.Response {
	if h.files == nil {
		return unavailableAttachments()
	}
	patient, errResp := h.findPatient(c)
	if errResp != nil {
		return *errResp
	}
	attachments := []clinical_models.PatientAttachment{}
	if err := tenantdb.For(c, h.db).Unscoped().
		Where("patient_id = ? AND deleted_at IS NOT NULL", patient.ID).
		Order("deleted_at DESC").Order("id DESC").Find(&attachments).Error; err != nil {
		h.logger.Error("failed to list deleted patient attachments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	names := map[uint]string{}
	var ids []uint
	for _, a := range attachments {
		if a.DeletedByID != nil {
			ids = append(ids, *a.DeletedByID)
		}
	}
	if len(ids) > 0 {
		var users []user_models.User
		if err := h.db.Select("id", "user_name").Where("id IN ?", ids).Find(&users).Error; err == nil {
			for _, u := range users {
				names[u.ID] = u.UserName
			}
		}
	}
	out := make([]deletedAttachment, 0, len(attachments))
	for _, a := range attachments {
		d := deletedAttachment{PatientAttachment: a}
		if a.DeletedByID != nil {
			d.DeletedByName = names[*a.DeletedByID]
		}
		out = append(out, d)
	}
	return envelope.SuccessResponse(out, "clinical.attachment.deleted.list.success")
}

// RestoreAttachment brings a deleted attachment back: deleted_at, deleted_by_id
// and delete_reason are cleared in one audited UPDATE. One that is not deleted
// (or not in scope) is 404.
func (h *PatientAttachmentHandler) RestoreAttachment(c *gin.Context) envelope.Response {
	if h.files == nil {
		return unavailableAttachments()
	}
	patient, errResp := h.findPatient(c)
	if errResp != nil {
		return *errResp
	}
	attachmentID, err := strconv.ParseUint(c.Param("attachment_id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var attachment clinical_models.PatientAttachment
	if err := tenantdb.For(c, h.db).Unscoped().Where("patient_id = ? AND deleted_at IS NOT NULL", patient.ID).First(&attachment, attachmentID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAttachmentNotFound)
	}
	if err := tenantdb.For(c, h.db).Unscoped().Model(&attachment).Updates(map[string]interface{}{"deleted_at": nil, "deleted_by_id": nil, "delete_reason": ""}).Error; err != nil {
		h.logger.Error("failed to restore patient attachment", zap.Uint("id", attachment.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.attachment.delete.error", core_errors.ErrClinicalAttachmentSaveError)
	}
	return envelope.SuccessResponse(nil, "clinical.attachment.restore.success")
}
