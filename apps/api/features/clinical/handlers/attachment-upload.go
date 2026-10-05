package clinical_handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_services "pengi-med-saas/features/companies/services"
	auth_middleware "pengi-med-saas/features/users/middleware"
)

// The upload path shared by every way an Adjunto enters the system: the
// patient's attachments (UploadAttachment) and exam results
// (UploadExamResult). Both get the same size limit, content-sniffed types,
// plan quota, duplicate warning, encrypted store and audit.

// attachmentUpload is the validated "file" part of a multipart upload.
type attachmentUpload struct {
	Data     []byte
	FileName string
	MimeType string
}

// readAttachmentUpload caps the request body, parses the multipart form and
// reads its "file" part, enforcing MaxAttachmentSize and the accepted types
// (by content). The form's other values stay available via c.PostForm.
func readAttachmentUpload(c *gin.Context) (*attachmentUpload, *envelope.Response) {
	fail := func(resp envelope.Response) (*attachmentUpload, *envelope.Response) { return nil, &resp }
	tooLarge := envelope.ErrorResponse(http.StatusRequestEntityTooLarge, "clinical.attachment.too_large", core_errors.ErrClinicalAttachmentTooLarge)
	required := envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.file.required", core_errors.ErrClinicalInvalidRequest)

	// Cap the body before anything reads it: a larger request fails while
	// parsing instead of being read whole.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxAttachmentSize+attachmentFormOverhead)
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return fail(tooLarge)
		}
		return fail(required)
	}
	defer c.Request.MultipartForm.RemoveAll()

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return fail(required)
	}
	defer file.Close()
	if header.Size > MaxAttachmentSize {
		return fail(tooLarge)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxAttachmentSize+1))
	if err != nil {
		return fail(required)
	}
	if len(data) > MaxAttachmentSize {
		return fail(tooLarge)
	}
	if len(data) == 0 {
		return fail(required)
	}
	mimeType := http.DetectContentType(data)
	if !attachmentMimeTypes[mimeType] {
		return fail(envelope.ErrorResponse(http.StatusUnsupportedMediaType, "clinical.attachment.type.unsupported", core_errors.ErrClinicalAttachmentType))
	}
	return &attachmentUpload{Data: data, FileName: attachmentFileName(header.Filename), MimeType: mimeType}, nil
}

// attachmentStorageUsage sums the size of every attachment of the tenant —
// deleted ones too, since their files stay stored until purged — against the
// quota of its current plan.
func attachmentStorageUsage(c *gin.Context, db *gorm.DB) (AttachmentUsage, error) {
	var used int64
	if err := tenantdb.For(c, db).Unscoped().Model(&clinical_models.PatientAttachment{}).
		Select("COALESCE(SUM(size), 0)").Scan(&used).Error; err != nil {
		return AttachmentUsage{}, err
	}
	quota, err := company_services.StorageQuotaBytes(db, tenantdb.TenantID(c))
	if err != nil {
		return AttachmentUsage{}, err
	}
	return AttachmentUsage{UsedBytes: used, QuotaBytes: quota, Warning: used*5 >= quota*4}, nil
}

// checkAttachmentQuota rejects an upload of size bytes that would exceed the
// plan's quota. Check-then-insert: two uploads racing at the edge of the quota
// can both pass and overshoot it by up to one file each. Accepted, no locking.
func checkAttachmentQuota(c *gin.Context, db *gorm.DB, logger *zap.Logger, size int64) *envelope.Response {
	usage, err := attachmentStorageUsage(c, db)
	if err != nil {
		logger.Error("failed to compute attachment storage usage", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		return &resp
	}
	if usage.UsedBytes+size > usage.QuotaBytes {
		resp := envelope.ErrorResponse(http.StatusForbidden, "plan.limit.storage", core_errors.ErrPlanStorageQuota)
		return &resp
	}
	return nil
}

// newPatientAttachment builds the row for an upload (hash, generated stored
// name, uploader); the caller fills the metadata and saves it.
func newPatientAttachment(c *gin.Context, patientID uint, upload *attachmentUpload) clinical_models.PatientAttachment {
	sum := sha256.Sum256(upload.Data)
	attachment := clinical_models.PatientAttachment{
		TenantID:   tenantdb.TenantID(c),
		PatientID:  patientID,
		FileName:   upload.FileName,
		MimeType:   upload.MimeType,
		Size:       int64(len(upload.Data)),
		SHA256:     hex.EncodeToString(sum[:]),
		StoredName: "attachments/" + uuid.NewString(),
		TakenAt:    dateOnly(time.Now()),
	}
	if uid, _, ok := auth_middleware.GetUserFromContext(c); ok {
		attachment.UploadedByID = uint(uid)
	}
	return attachment
}

// findAttachmentDuplicate looks for the oldest live attachment of the patient
// (within the tenant) with the given hash. A deleted original does not count.
func findAttachmentDuplicate(c *gin.Context, db *gorm.DB, logger *zap.Logger, patientID uint, sha string) *AttachmentDuplicate {
	var original clinical_models.PatientAttachment
	err := tenantdb.For(c, db).Where("patient_id = ? AND sha256 = ?", patientID, sha).Order("id ASC").First(&original).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("failed to check duplicate attachment", zap.Error(err))
		}
		return nil
	}
	return &AttachmentDuplicate{ID: original.ID, FileName: original.FileName, CreatedAt: original.CreatedAt}
}

// storeAttachment writes the (encrypting) file, then runs save — which must
// create the row, and whatever goes with it, in one transaction. If save
// fails the file is removed again.
func storeAttachment(files tenantfiles.Store, attachment *clinical_models.PatientAttachment, data []byte, save func() error) error {
	if err := files.Write(attachment.TenantID, attachment.StoredName, data); err != nil {
		return err
	}
	if err := save(); err != nil {
		_ = files.Remove(attachment.TenantID, attachment.StoredName)
		return err
	}
	return nil
}

// parseAttachmentMeta reads the optional "taken_at" and "description" form
// values shared by every upload.
func parseAttachmentMeta(c *gin.Context, attachment *clinical_models.PatientAttachment) *envelope.Response {
	if raw := strings.TrimSpace(c.PostForm("taken_at")); raw != "" {
		parsed, errResp := parseAttachmentTakenAt(raw)
		if errResp != nil {
			return errResp
		}
		attachment.TakenAt = parsed
	}
	description := strings.TrimSpace(c.PostForm("description"))
	if errResp := checkAttachmentDescription(description); errResp != nil {
		return errResp
	}
	attachment.Description = description
	return nil
}
