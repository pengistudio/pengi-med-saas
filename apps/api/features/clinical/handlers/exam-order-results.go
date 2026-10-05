package clinical_handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_data "pengi-med-saas/features/clinical/data"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	notifications_service "pengi-med-saas/features/notifications/services"
)

// An exam result is a patient attachment (Adjunto) linked to one or more exams
// of one order. These helpers keep the attachment routes and the exam order
// routes consistent: neither can delete an attachment whose exams were
// reviewed, and both recompute the order status when one is deleted or
// restored.

// attachmentReviewed reports whether any exam the attachment covers was
// already reviewed (db must be tenant-bound).
func attachmentReviewed(db *gorm.DB, attachmentID uint) (bool, error) {
	var reviewed int64
	err := db.Model(&clinical_models.ExamOrderItem{}).
		Joins("JOIN exam_order_item_attachments l ON l.exam_order_item_id = exam_order_items.id").
		Where("l.patient_attachment_id = ? AND exam_order_items.reviewed_at IS NOT NULL", attachmentID).
		Count(&reviewed).Error
	return reviewed > 0, err
}

// refreshOrdersOfAttachment recomputes the status of the orders whose exams
// the attachment covers (none for a plain attachment).
func refreshOrdersOfAttachment(tx *gorm.DB, attachmentID uint) error {
	var orderIDs []uint
	if err := tx.Model(&clinical_models.ExamOrderItem{}).
		Joins("JOIN exam_order_item_attachments l ON l.exam_order_item_id = exam_order_items.id").
		Where("l.patient_attachment_id = ?", attachmentID).
		Distinct().Pluck("exam_order_items.exam_order_id", &orderIDs).Error; err != nil {
		return err
	}
	for _, id := range orderIDs {
		if err := refreshExamOrderStatus(tx, id); err != nil {
			return err
		}
	}
	return nil
}

func reviewedResultResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "clinical.exam_result.error.reviewed", core_errors.ErrClinicalExamResultReviewed)
}

// softDeleteAttachment hides an attachment with who and why in one audited
// UPDATE (the encrypted file stays, so it can be restored) and recomputes the
// status of the orders it is a result of.
func softDeleteAttachment(tx *gorm.DB, attachment *clinical_models.PatientAttachment, reason string, userID uint) error {
	updates := map[string]any{"deleted_at": time.Now(), "delete_reason": reason}
	if userID != 0 {
		updates["deleted_by_id"] = userID
	}
	if err := tx.Model(attachment).Updates(updates).Error; err != nil {
		return err
	}
	return refreshOrdersOfAttachment(tx, attachment.ID)
}

// attachmentDeleteReason validates the required deletion reason.
func attachmentDeleteReason(c *gin.Context) (string, *envelope.Response) {
	var req deleteAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.delete.reason.required", core_errors.ErrClinicalInvalidRequest)
		return "", &resp
	}
	reason := strings.TrimSpace(req.Reason)
	if utf8.RuneCountInString(reason) > attachmentDeleteReasonMax {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.delete.reason.too_long", core_errors.ErrClinicalInvalidRequest)
		return "", &resp
	}
	return reason, nil
}

// parseIDList reads IDs given as repeated values and/or comma-separated.
func parseIDList(values []string) ([]uint, bool) {
	var ids []uint
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseUint(part, 10, 32)
			if err != nil || id == 0 {
				return nil, false
			}
			ids = append(ids, uint(id))
		}
	}
	return ids, len(ids) > 0
}

// orderItemsByID returns the order's items with the given IDs, or ok=false if
// any ID is not an item of the order.
func orderItemsByID(order *clinical_models.ExamOrder, ids []uint) ([]clinical_models.ExamOrderItem, bool) {
	byID := make(map[uint]clinical_models.ExamOrderItem, len(order.Items))
	for _, item := range order.Items {
		byID[item.ID] = item
	}
	seen := map[uint]bool{}
	var out []clinical_models.ExamOrderItem
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			return nil, false
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, item)
		}
	}
	return out, true
}

// resultCategory is the attachment category of a result covering items: lab
// result if it covers a laboratory exam, else imaging if it covers an imaging
// one, else external report (ECG, spirometry, endoscopy… are reports of
// studies done elsewhere).
func resultCategory(items []clinical_models.ExamOrderItem) string {
	hasImaging := false
	for _, item := range items {
		switch item.Category {
		case clinical_data.ExamCategoryLaboratory:
			return clinical_models.AttachmentCategoryLabResult
		case clinical_data.ExamCategoryImaging:
			hasImaging = true
		}
	}
	if hasImaging {
		return clinical_models.AttachmentCategoryImaging
	}
	return clinical_models.AttachmentCategoryExternalReport
}

// ExamResultUpload is the result upload response: the order (status
// recomputed), the attachment created and, when the patient already has a
// file with the same content, duplicate_of (a non-blocking warning).
type ExamResultUpload struct {
	Order       *clinical_models.ExamOrder        `json:"order"`
	Attachment  clinical_models.PatientAttachment `json:"attachment"`
	DuplicateOf *AttachmentDuplicate              `json:"duplicate_of,omitempty"`
}

// UploadExamResult stores one result file through the attachment upload path
// (multipart "file", same limit, types, quota, encryption and duplicate check
// as any Adjunto) covering the order's exams in "item_ids" (repeated or
// comma-separated). Optional: "category" (default from the exams), "taken_at"
// (default today), "description". The attachment belongs to the order's
// patient and is linked to its consultation, if any. It recomputes the status
// and notifies the ordering doctor.
func (h *ExamOrderHandler) UploadExamResult(c *gin.Context) envelope.Response {
	if h.attachments == nil {
		return unavailableAttachments()
	}
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	order, err := loadExamOrder(db, id)
	if err != nil {
		return examOrderNotFound()
	}
	if order.VoidedAt != nil {
		return examOrderVoided()
	}

	upload, errResp := readAttachmentUpload(c)
	if errResp != nil {
		return *errResp
	}
	itemIDs, ok := parseIDList(c.Request.MultipartForm.Value["item_ids"])
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	items, ok := orderItemsByID(order, itemIDs)
	if !ok {
		return examOrderInvalidItems()
	}

	attachment := newPatientAttachment(c, order.PatientID, upload)
	attachment.MedicalRecordID = order.MedicalRecordID
	attachment.Category = resultCategory(items)
	if category := strings.TrimSpace(c.PostForm("category")); category != "" {
		if !clinical_models.IsAttachmentCategory(category) {
			return envelope.ErrorResponse(http.StatusBadRequest, "clinical.attachment.category.invalid", core_errors.ErrClinicalAttachmentCategory)
		}
		attachment.Category = category
	}
	if errResp := parseAttachmentMeta(c, &attachment); errResp != nil {
		return *errResp
	}
	if errResp := checkAttachmentQuota(c, h.db, h.logger, attachment.Size); errResp != nil {
		return *errResp
	}
	duplicate := findAttachmentDuplicate(c, h.db, h.logger, order.PatientID, attachment.SHA256)

	err = storeAttachment(h.attachments, &attachment, upload.Data, func() error {
		return db.Transaction(func(tx *gorm.DB) error {
			// Same row lock as UpdateExamOrder: re-check the order (not voided,
			// exams still in it) once concurrent edits are done.
			current, err := lockExamOrder(tx, order.ID)
			if err != nil {
				return err
			}
			if current.VoidedAt != nil {
				return responseError{examOrderVoided()}
			}
			if _, ok := orderItemsByID(current, itemIDs); !ok {
				return responseError{examOrderInvalidItems()}
			}
			if err := tx.Create(&attachment).Error; err != nil {
				return err
			}
			links := make([]map[string]any, 0, len(items))
			for _, item := range items {
				links = append(links, map[string]any{"exam_order_item_id": item.ID, "patient_attachment_id": attachment.ID})
			}
			if err := tx.Table("exam_order_item_attachments").Create(&links).Error; err != nil {
				return err
			}
			return refreshExamOrderStatus(tx, order.ID)
		})
	})
	var respErr responseError
	if errors.As(err, &respErr) {
		return respErr.resp
	}
	if err != nil {
		h.logger.Error("failed to save exam result", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.attachment.upload.error", core_errors.ErrClinicalAttachmentSaveError)
	}

	h.notifyResultsUploaded(order, tenantdb.TenantID(c), currentUserID(c))
	resp := h.respondExamOrder(c, order.ID, http.StatusCreated, "clinical.exam_result.upload.success")
	if updated, ok := resp.Data.(*clinical_models.ExamOrder); ok {
		resp.Data = ExamResultUpload{Order: updated, Attachment: attachment, DuplicateOf: duplicate}
	}
	return resp
}

// notifyResultsUploaded tells the ordering doctor (unless they uploaded the
// results themselves) that the order has results to review.
func (h *ExamOrderHandler) notifyResultsUploaded(order *clinical_models.ExamOrder, tenantID, uploaderID uint) {
	if order.OrderedByID == 0 || order.OrderedByID == uploaderID {
		return
	}
	patientName := ""
	if order.Patient != nil {
		patientName = strings.TrimSpace(order.Patient.FirstName + " " + order.Patient.LastName)
	}
	err := notifications_service.CreateIfNotExists(h.db, h.logger, notifications_service.CreateNotificationInput{
		TenantID:     tenantID,
		UserID:       order.OrderedByID,
		Type:         "clinical.exam_order.results",
		ResourceType: "exam_order",
		ResourceID:   order.ID,
		MessageKey:   "notification.clinical.exam_order.results_uploaded",
		Params: map[string]string{
			"patient_name": patientName,
			"order_code":   clinical_models.FormatExamOrderNumber(order.Number),
		},
		ActionURL: fmt.Sprintf("/clinical/exam-orders/%d", order.ID),
	})
	if err != nil {
		h.logger.Error("failed to notify exam results", zap.Uint("exam_order_id", order.ID), zap.Error(err))
	}
}

// DeleteExamResult soft-deletes a result of the order (JSON {reason}, like any
// attachment deletion; audited), unless an exam it covers was reviewed. It
// can be restored from the patient's deleted attachments.
func (h *ExamOrderHandler) DeleteExamResult(c *gin.Context) envelope.Response {
	if h.attachments == nil {
		return unavailableAttachments()
	}
	id, ok := pathID(c, "id")
	attachmentID, okAttachment := pathID(c, "attachmentId")
	if !ok || !okAttachment {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	reason, errResp := attachmentDeleteReason(c)
	if errResp != nil {
		return *errResp
	}
	db := tenantdb.For(c, h.db)
	var order clinical_models.ExamOrder
	if err := db.First(&order, id).Error; err != nil {
		return examOrderNotFound()
	}
	if order.VoidedAt != nil {
		return examOrderVoided()
	}
	// The attachment must be a result of this order.
	var attachment clinical_models.PatientAttachment
	err := db.Where(`id IN (SELECT l.patient_attachment_id FROM exam_order_item_attachments l
		JOIN exam_order_items i ON i.id = l.exam_order_item_id WHERE i.exam_order_id = ?)`, order.ID).
		First(&attachment, attachmentID).Error
	if err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalAttachmentNotFound)
	}
	reviewed, err := attachmentReviewed(db, attachment.ID)
	if err != nil {
		return h.examOrderSaveError(err)
	}
	if reviewed {
		return reviewedResultResponse()
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return softDeleteAttachment(tx, &attachment, reason, currentUserID(c))
	}); err != nil {
		return h.examOrderSaveError(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusOK, "clinical.exam_result.delete.success")
}

// ReviewExamResults marks exams with results as reviewed by the current user.
func (h *ExamOrderHandler) ReviewExamResults(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req clinical_dto.ReviewExamResultsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	order, err := loadExamOrder(db, id)
	if err != nil {
		return examOrderNotFound()
	}
	items, ok := orderItemsByID(order, req.ItemIDs)
	if !ok {
		return examOrderInvalidItems()
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		if !item.HasResult() {
			return envelope.ErrorResponse(http.StatusConflict, "clinical.exam_result.error.missing", core_errors.ErrClinicalExamResultMissing)
		}
		ids = append(ids, item.ID)
	}
	if err := db.Model(&clinical_models.ExamOrderItem{}).
		Where("id IN ? AND exam_order_id = ? AND reviewed_at IS NULL", ids, order.ID).
		Updates(map[string]any{"reviewed_by_id": currentUserID(c), "reviewed_at": time.Now()}).Error; err != nil {
		return h.examOrderSaveError(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusOK, "clinical.exam_result.review.success")
}
