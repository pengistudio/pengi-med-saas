package clinical_handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/mailer"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/tenantfiles"
	clinical_data "pengi-med-saas/features/clinical/data"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	clinical_services "pengi-med-saas/features/clinical/services"
	clinical_templates "pengi-med-saas/features/clinical/templates"
	company_models "pengi-med-saas/features/companies/models"
	signature_models "pengi-med-saas/features/signatures/models"
	signature_services "pengi-med-saas/features/signatures/services"
	user_models "pengi-med-saas/features/users/models"
)

// ExamOrderHandler serves exam orders: CRUD, void/close, PDF (download, email,
// sign) and results (upload, delete, review). files holds the signed PDFs;
// attachments is the encrypting store results (Adjuntos) live in, nil when
// ATTACHMENT_ENCRYPTION_KEY is not configured (result routes answer 503).
type ExamOrderHandler struct {
	db          *gorm.DB
	logger      *zap.Logger
	mailer      *mailer.Mailer
	renderer    *pdfrender.Renderer
	signer      *signature_services.Signer
	files       tenantfiles.Store
	attachments tenantfiles.Store
}

func NewExamOrderHandler(db *gorm.DB, logger *zap.Logger, mailer *mailer.Mailer, renderer *pdfrender.Renderer, signer *signature_services.Signer, files, attachments tenantfiles.Store) *ExamOrderHandler {
	return &ExamOrderHandler{db: db, logger: logger, mailer: mailer, renderer: renderer, signer: signer, files: files, attachments: attachments}
}

// currentUserID is the authenticated user (AuthMiddleware sets "user_id").
func currentUserID(c *gin.Context) uint {
	switch v := c.Value("user_id").(type) {
	case int64:
		return uint(v)
	case uint:
		return v
	case int:
		return uint(v)
	case float64:
		return uint(v)
	}
	return 0
}

func examOrderNotFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "clinical.exam_order.error.not_found", core_errors.ErrClinicalExamOrderNotFound)
}

func examOrderVoided() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "clinical.exam_order.error.voided", core_errors.ErrClinicalExamOrderVoided)
}

func examOrderInvalidItems() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "clinical.exam_order.error.invalid_items", core_errors.ErrClinicalExamOrderInvalidItems)
}

func (h *ExamOrderHandler) examOrderSaveError(err error) envelope.Response {
	h.logger.Error("failed to save exam order", zap.Error(err))
	return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_order.save.error", core_errors.ErrClinicalExamOrderError)
}

// responseError carries an error response out of a transaction.
type responseError struct{ resp envelope.Response }

func (e responseError) Error() string { return e.resp.Message }

// lockExamOrder locks the order row until tx ends (SELECT ... FOR UPDATE; a
// no-op on SQLite) and reloads the order with its items and results. Edits and
// result uploads both take it, so they serialize: an edit can't slip past the
// "only add exams" lock while a result is being uploaded. tx must be
// tenant-bound.
func lockExamOrder(tx *gorm.DB, id uint) (*clinical_models.ExamOrder, error) {
	var row clinical_models.ExamOrder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&row, id).Error; err != nil {
		return nil, err
	}
	return loadExamOrder(tx, id)
}

// ── Loading and status ──────────────────────────────────────────────────────

func preloadExamOrder(db *gorm.DB) *gorm.DB {
	return db.Preload("Patient").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).
		Preload("Items.Attachments", func(db *gorm.DB) *gorm.DB { return db.Order("id") })
}

// loadExamOrder loads an order of the tenant with its patient, items and results.
func loadExamOrder(db *gorm.DB, id uint) (*clinical_models.ExamOrder, error) {
	var order clinical_models.ExamOrder
	if err := preloadExamOrder(db).First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// examOrderStatus derives the status from the items' results: issued (none
// with result), partial_results, complete_results (all, or closed by hand),
// voided.
func examOrderStatus(order *clinical_models.ExamOrder) string {
	if order.VoidedAt != nil {
		return clinical_models.ExamOrderStatusVoided
	}
	if order.ClosedManually {
		return clinical_models.ExamOrderStatusCompleteResults
	}
	withResult := 0
	for _, item := range order.Items {
		if item.HasResult() {
			withResult++
		}
	}
	switch {
	case withResult == 0:
		return clinical_models.ExamOrderStatusIssued
	case withResult == len(order.Items):
		return clinical_models.ExamOrderStatusCompleteResults
	default:
		return clinical_models.ExamOrderStatusPartialResults
	}
}

// refreshExamOrderStatus reloads the order inside tx and stores its derived status.
func refreshExamOrderStatus(tx *gorm.DB, orderID uint) error {
	order, err := loadExamOrder(tx, orderID)
	if err != nil {
		return err
	}
	if status := examOrderStatus(order); status != order.Status {
		return tx.Model(&clinical_models.ExamOrder{Model: gorm.Model{ID: orderID}}).Update("status", status).Error
	}
	return nil
}

func hasAnyResult(order *clinical_models.ExamOrder) bool {
	for _, item := range order.Items {
		if item.HasResult() {
			return true
		}
	}
	return false
}

// decorateExamOrders fills the computed fields (code, pending review, the
// ordering and reviewing users' names).
func decorateExamOrders(db *gorm.DB, orders []*clinical_models.ExamOrder) {
	ids := make([]uint, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.OrderedByID)
		for _, item := range o.Items {
			if item.ReviewedByID != nil {
				ids = append(ids, *item.ReviewedByID)
			}
		}
	}
	names := map[uint]string{}
	if len(ids) > 0 {
		var users []user_models.User
		db.Select("id", "user_name").Where("id IN ?", ids).Find(&users)
		for _, u := range users {
			names[u.ID] = u.UserName
		}
	}
	for _, o := range orders {
		o.Code = clinical_models.FormatExamOrderNumber(o.Number)
		o.OrderedByName = names[o.OrderedByID]
		o.PendingReview = false
		for i := range o.Items {
			if id := o.Items[i].ReviewedByID; id != nil {
				o.Items[i].ReviewedByName = names[*id]
			}
		}
		for _, item := range o.Items {
			if item.HasResult() && item.ReviewedAt == nil {
				o.PendingReview = true
				break
			}
		}
		if o.Patient != nil {
			o.Patient.MedicalRecords = nil
		}
	}
}

// respondExamOrder reloads the order and returns it.
func (h *ExamOrderHandler) respondExamOrder(c *gin.Context, id uint, code int, message string) envelope.Response {
	db := tenantdb.For(c, h.db)
	order, err := loadExamOrder(db, id)
	if err != nil {
		return examOrderNotFound()
	}
	decorateExamOrders(db, []*clinical_models.ExamOrder{order})
	return envelope.New(code, message, order)
}

// ── Items ───────────────────────────────────────────────────────────────────

// resolvedExamItem is what an ExamOrderItemInput stands for once the catalog
// is applied. Indications nil = not given.
type resolvedExamItem struct {
	CatalogItemID *uint
	Name          string
	Category      string
	Subgroup      string
	Indications   *string
	Default       string // the catalog's default indications
}

// resolveExamItems resolves inputs against the tenant's catalog; ok=false if
// an exam is unknown or has no name.
func resolveExamItems(db *gorm.DB, inputs []clinical_dto.ExamOrderItemInput) ([]resolvedExamItem, bool, error) {
	var catalogIDs []uint
	for _, in := range inputs {
		if in.CatalogItemID != nil {
			catalogIDs = append(catalogIDs, *in.CatalogItemID)
		}
	}
	// Unscoped: an order keeps referencing exams the tenant later deleted.
	catalogItems, err := clinical_services.CatalogItemsByID(db.Unscoped(), catalogIDs)
	if err != nil {
		if errors.Is(err, clinical_services.ErrUnknownCatalogItems) {
			return nil, false, nil
		}
		return nil, false, err
	}
	catalog := make(map[uint]clinical_models.ExamCatalogItem, len(catalogItems))
	for _, item := range catalogItems {
		catalog[item.ID] = item
	}

	out := make([]resolvedExamItem, 0, len(inputs))
	for _, in := range inputs {
		var r resolvedExamItem
		if in.Indications != nil {
			ind := strings.TrimSpace(*in.Indications)
			r.Indications = &ind
		}
		if in.CatalogItemID != nil {
			item := catalog[*in.CatalogItemID]
			id := item.ID
			r.CatalogItemID = &id
			r.Name, r.Category, r.Subgroup, r.Default = item.Name, item.Category, item.Subgroup, item.DefaultIndications
		} else {
			r.Name = strings.TrimSpace(in.Name)
			r.Category = in.Category
			if r.Category == "" {
				r.Category = clinical_data.ExamCategoryOther
			}
			r.Subgroup = strings.TrimSpace(in.Subgroup)
		}
		if r.Name == "" {
			return nil, false, nil
		}
		out = append(out, r)
	}
	return out, true, nil
}

func (r resolvedExamItem) newItem(tenantID, orderID uint) clinical_models.ExamOrderItem {
	indications := r.Default
	if r.Indications != nil {
		indications = *r.Indications
	}
	return clinical_models.ExamOrderItem{
		TenantID:      tenantID,
		ExamOrderID:   orderID,
		CatalogItemID: r.CatalogItemID,
		Name:          r.Name,
		Category:      r.Category,
		Subgroup:      r.Subgroup,
		Indications:   indications,
	}
}

func sameUintPtr(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func diagnosesJSON(items []clinical_models.DiagnosisItem) datatypes.JSON {
	if items == nil {
		items = []clinical_models.DiagnosisItem{}
	}
	raw, _ := json.Marshal(items)
	return raw
}

func sameDiagnoses(stored datatypes.JSON, items []clinical_models.DiagnosisItem) bool {
	var current []clinical_models.DiagnosisItem
	_ = json.Unmarshal(stored, &current)
	return string(diagnosesJSON(current)) == string(diagnosesJSON(items))
}

// validateMedicalRecord checks the consultation belongs to the patient.
func validateMedicalRecord(db *gorm.DB, recordID *uint, patientID uint) bool {
	if recordID == nil {
		return true
	}
	var count int64
	db.Model(&clinical_models.MedicalRecord{}).Where("id = ? AND patient_id = ?", *recordID, patientID).Count(&count)
	return count == 1
}

// nextExamOrderNumber takes the tenant's next order number. It must run in
// the creation transaction: the counter row stays locked until commit, so
// concurrent creations get distinct numbers and a rollback leaves no gap.
func nextExamOrderNumber(tx *gorm.DB, tenantID uint) (uint, error) {
	var number uint
	err := tx.Raw(`INSERT INTO exam_order_counters (tenant_id, value) VALUES (?, 1)
		ON CONFLICT (tenant_id) DO UPDATE SET value = exam_order_counters.value + 1
		RETURNING value`, tenantID).Scan(&number).Error
	if err == nil && number == 0 {
		err = errors.New("exam order counter returned no value")
	}
	return number, err
}

// ── CRUD ────────────────────────────────────────────────────────────────────

// GetExamOrders lists orders, newest first. Query: status (comma-separated),
// pending_review=true, ordered_by (user id or "me"), patient_id, record_id,
// page, limit.
func (h *ExamOrderHandler) GetExamOrders(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	query := db.Model(&clinical_models.ExamOrder{})
	if v := c.Query("status"); v != "" {
		query = query.Where("status IN ?", strings.Split(v, ","))
	}
	if c.Query("pending_review") == "true" {
		query = query.Where(`EXISTS (SELECT 1 FROM exam_order_items i
			JOIN exam_order_item_attachments l ON l.exam_order_item_id = i.id
			JOIN patient_attachments pa ON pa.id = l.patient_attachment_id AND pa.deleted_at IS NULL
			WHERE i.exam_order_id = exam_orders.id AND i.deleted_at IS NULL AND i.reviewed_at IS NULL)`)
	}
	if v := c.Query("ordered_by"); v != "" {
		userID := currentUserID(c)
		if v != "me" {
			id, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
			}
			userID = uint(id)
		}
		query = query.Where("ordered_by_id = ?", userID)
	}
	for param, column := range map[string]string{"patient_id": "patient_id", "record_id": "medical_record_id"} {
		if v := c.Query(param); v != "" {
			id, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
			}
			query = query.Where(column+" = ?", id)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.logger.Error("failed to count exam orders", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_order.fetch.error", core_errors.ErrClinicalExamOrderError)
	}
	var orders []*clinical_models.ExamOrder
	if err := preloadExamOrder(query).Order("id DESC").Limit(limit).Offset((page - 1) * limit).Find(&orders).Error; err != nil {
		h.logger.Error("failed to list exam orders", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_order.fetch.error", core_errors.ErrClinicalExamOrderError)
	}
	decorateExamOrders(db, orders)
	return envelope.PagedSuccessResponse(orders, int(total), page, limit, "clinical.exam_order.fetch.success")
}

func (h *ExamOrderHandler) GetExamOrder(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	order, err := loadExamOrder(db, id)
	if err != nil {
		return examOrderNotFound()
	}
	audit.RecordAccess(h.db, c, "exam_orders", order.ID, &order.PatientID)
	decorateExamOrders(db, []*clinical_models.ExamOrder{order})
	return envelope.SuccessResponse(order, "clinical.exam_order.fetch.success")
}

func (h *ExamOrderHandler) CreateExamOrder(c *gin.Context) envelope.Response {
	var req clinical_dto.CreateExamOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	tenantID := tenantdb.TenantID(c)

	var patient clinical_models.Patient
	if err := db.First(&patient, req.PatientID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalPatientNotFound)
	}
	if !validateMedicalRecord(db, req.MedicalRecordID, patient.ID) {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	resolved, ok, err := resolveExamItems(db, req.Items)
	if err != nil {
		return h.examOrderSaveError(err)
	}
	if !ok {
		return examOrderInvalidItems()
	}

	priority := req.Priority
	if priority == "" {
		priority = clinical_models.ExamPriorityRoutine
	}
	order := clinical_models.ExamOrder{
		TenantID:        tenantID,
		PatientID:       patient.ID,
		MedicalRecordID: req.MedicalRecordID,
		OrderedByID:     currentUserID(c),
		Diagnoses:       diagnosesJSON(req.Diagnoses),
		Priority:        priority,
		Notes:           strings.TrimSpace(req.Notes),
		DestinationLab:  strings.TrimSpace(req.DestinationLab),
		Status:          clinical_models.ExamOrderStatusIssued,
	}
	for _, r := range resolved {
		order.Items = append(order.Items, r.newItem(tenantID, 0))
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		number, err := nextExamOrderNumber(tx, tenantID)
		if err != nil {
			return err
		}
		order.Number = number
		return tx.Create(&order).Error
	})
	if err != nil {
		return h.examOrderSaveError(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusCreated, "clinical.exam_order.create.success")
}

// UpdateExamOrder replaces the order's content. Without results everything is
// editable; once an exam has a result only new exams can be added. Any change
// clears the signature (the signed PDF no longer matches).
func (h *ExamOrderHandler) UpdateExamOrder(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req clinical_dto.UpdateExamOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
	locked := hasAnyResult(order)
	lockedResponse := envelope.ErrorResponse(http.StatusConflict, "clinical.exam_order.error.locked", core_errors.ErrClinicalExamOrderLocked)

	resolved, ok, err := resolveExamItems(db, req.Items)
	if err != nil {
		return h.examOrderSaveError(err)
	}
	if !ok {
		return examOrderInvalidItems()
	}

	existing := make(map[uint]clinical_models.ExamOrderItem, len(order.Items))
	for _, item := range order.Items {
		existing[item.ID] = item
	}
	seen := map[uint]bool{}
	itemUpdates := map[uint]map[string]any{}
	var toCreate []clinical_models.ExamOrderItem
	for i, in := range req.Items {
		r := resolved[i]
		if in.ID == nil {
			toCreate = append(toCreate, r.newItem(order.TenantID, order.ID))
			continue
		}
		current, found := existing[*in.ID]
		if !found || seen[*in.ID] {
			return examOrderInvalidItems()
		}
		seen[*in.ID] = true
		if r.CatalogItemID != nil && sameUintPtr(current.CatalogItemID, r.CatalogItemID) {
			// Same catalog exam: keep the order's copy even if the catalog changed since.
			r.Name, r.Category, r.Subgroup = current.Name, current.Category, current.Subgroup
		}
		indications := current.Indications
		if r.Indications != nil {
			indications = *r.Indications
		}
		if sameUintPtr(current.CatalogItemID, r.CatalogItemID) && current.Name == r.Name && current.Category == r.Category &&
			current.Subgroup == r.Subgroup && current.Indications == indications {
			continue
		}
		if locked {
			return lockedResponse
		}
		itemUpdates[current.ID] = map[string]any{
			"catalog_item_id": r.CatalogItemID, "name": r.Name, "category": r.Category,
			"subgroup": r.Subgroup, "indications": indications,
		}
	}
	var removed []clinical_models.ExamOrderItem
	for _, item := range order.Items {
		if !seen[item.ID] {
			removed = append(removed, item)
		}
	}
	if len(removed) > 0 && locked {
		return lockedResponse
	}

	priority := req.Priority
	if priority == "" {
		priority = clinical_models.ExamPriorityRoutine
	}
	notes, lab := strings.TrimSpace(req.Notes), strings.TrimSpace(req.DestinationLab)
	headerChanged := !sameUintPtr(order.MedicalRecordID, req.MedicalRecordID) || order.Priority != priority ||
		order.Notes != notes || order.DestinationLab != lab || !sameDiagnoses(order.Diagnoses, req.Diagnoses)
	if headerChanged && locked {
		return lockedResponse
	}
	if !sameUintPtr(order.MedicalRecordID, req.MedicalRecordID) && !validateMedicalRecord(db, req.MedicalRecordID, order.PatientID) {
		return envelope.ErrorResponse(http.StatusNotFound, "error.not_found", core_errors.ErrClinicalRecordNotFound)
	}
	if !headerChanged && len(itemUpdates) == 0 && len(removed) == 0 && len(toCreate) == 0 {
		return h.respondExamOrder(c, order.ID, http.StatusOK, "clinical.exam_order.update.success")
	}

	updates := clearedSignature()
	updates["medical_record_id"] = req.MedicalRecordID
	updates["priority"] = priority
	updates["notes"] = notes
	updates["destination_lab"] = lab
	updates["diagnoses"] = diagnosesJSON(req.Diagnoses)
	if len(toCreate) > 0 {
		updates["closed_manually"] = false // new exams reopen an order closed by hand
	}
	onlyAdds := !headerChanged && len(itemUpdates) == 0 && len(removed) == 0
	err = db.Transaction(func(tx *gorm.DB) error {
		// Re-check under the row lock: a result uploaded since the order was
		// read locks it, and then only additions may go through.
		current, err := lockExamOrder(tx, order.ID)
		if err != nil {
			return err
		}
		if current.VoidedAt != nil {
			return responseError{examOrderVoided()}
		}
		if hasAnyResult(current) && !onlyAdds {
			return responseError{lockedResponse}
		}
		if err := tx.Model(&clinical_models.ExamOrder{Model: gorm.Model{ID: order.ID}}).Select(mapKeys(updates)).Updates(updates).Error; err != nil {
			return err
		}
		for itemID, u := range itemUpdates {
			if err := tx.Model(&clinical_models.ExamOrderItem{}).Where("id = ? AND exam_order_id = ?", itemID, order.ID).
				Select(mapKeys(u)).Updates(u).Error; err != nil {
				return err
			}
		}
		for i := range removed {
			if err := tx.Delete(&removed[i]).Error; err != nil {
				return err
			}
		}
		if len(toCreate) > 0 {
			if err := tx.Create(&toCreate).Error; err != nil {
				return err
			}
		}
		return refreshExamOrderStatus(tx, order.ID)
	})
	var respErr responseError
	if errors.As(err, &respErr) {
		return respErr.resp
	}
	if err != nil {
		return h.examOrderSaveError(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusOK, "clinical.exam_order.update.success")
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// VoidExamOrder voids the order with a reason; it stays visible.
func (h *ExamOrderHandler) VoidExamOrder(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req clinical_dto.VoidExamOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var order clinical_models.ExamOrder
	if err := db.First(&order, id).Error; err != nil {
		return examOrderNotFound()
	}
	if order.VoidedAt != nil {
		return examOrderVoided()
	}
	now := time.Now()
	userID := currentUserID(c)
	if err := db.Model(&order).Updates(map[string]any{
		"status":       clinical_models.ExamOrderStatusVoided,
		"void_reason":  strings.TrimSpace(req.Reason),
		"voided_at":    now,
		"voided_by_id": userID,
	}).Error; err != nil {
		return h.examOrderSaveError(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusOK, "clinical.exam_order.void.success")
}

// CloseExamOrder marks the order as complete by hand (e.g. results arrived on paper).
func (h *ExamOrderHandler) CloseExamOrder(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var order clinical_models.ExamOrder
	if err := db.First(&order, id).Error; err != nil {
		return examOrderNotFound()
	}
	if order.VoidedAt != nil {
		return examOrderVoided()
	}
	if err := db.Model(&order).Updates(map[string]any{
		"closed_manually": true,
		"status":          clinical_models.ExamOrderStatusCompleteResults,
	}).Error; err != nil {
		return h.examOrderSaveError(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusOK, "clinical.exam_order.close.success")
}

// ── PDF: download, email, sign ──────────────────────────────────────────────

var examCategoryLabels = map[string]string{
	clinical_data.ExamCategoryLaboratory: "Laboratorio",
	clinical_data.ExamCategoryImaging:    "Imagen",
	clinical_data.ExamCategoryOther:      "Otros",
}

// examOrderGroups groups the items by category (laboratory, imaging, other)
// and subgroup, keeping the order in which they were requested.
func examOrderGroups(items []clinical_models.ExamOrderItem) []clinical_templates.ExamOrderGroup {
	var groups []clinical_templates.ExamOrderGroup
	for _, category := range []string{clinical_data.ExamCategoryLaboratory, clinical_data.ExamCategoryImaging, clinical_data.ExamCategoryOther} {
		group := clinical_templates.ExamOrderGroup{Category: examCategoryLabels[category]}
		subgroupIndex := map[string]int{}
		for _, item := range items {
			itemCategory := item.Category
			if _, known := examCategoryLabels[itemCategory]; !known {
				itemCategory = clinical_data.ExamCategoryOther
			}
			if itemCategory != category {
				continue
			}
			i, ok := subgroupIndex[item.Subgroup]
			if !ok {
				i = len(group.Subgroups)
				subgroupIndex[item.Subgroup] = i
				group.Subgroups = append(group.Subgroups, clinical_templates.ExamOrderSubgroup{Name: item.Subgroup})
			}
			group.Subgroups[i].Exams = append(group.Subgroups[i].Exams, clinical_templates.ExamOrderExam{Name: item.Name, Indications: item.Indications})
		}
		if len(group.Subgroups) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

// examOrderDoctorName is the name printed under the signature line: the
// ordering doctor's certificate holder if they uploaded one, else their user name.
func examOrderDoctorName(db *gorm.DB, order *clinical_models.ExamOrder) string {
	var sig signature_models.UserSignature
	if err := db.Where("user_id = ?", order.OrderedByID).Limit(1).Find(&sig).Error; err == nil && sig.SubjectName != "" {
		return sig.SubjectName
	}
	var user user_models.User
	if err := db.Select("id", "user_name").Limit(1).Find(&user, order.OrderedByID).Error; err == nil && user.UserName != "" {
		return user.UserName
	}
	return "Médico Tratante"
}

func (h *ExamOrderHandler) generateExamOrderPDF(c *gin.Context, order *clinical_models.ExamOrder, stamp *pdfsign.Stamp) ([]byte, error) {
	db := tenantdb.For(c, h.db)
	var company company_models.Company
	db.Limit(1).Find(&company)
	tradeName := "Consultorio Médico"
	if company.TradeName != "" {
		tradeName = company.TradeName
	}

	patient := order.Patient
	patientName := ""
	if patient != nil {
		if patient.FullName != nil && *patient.FullName != "" {
			patientName = *patient.FullName
		} else {
			patientName = strings.TrimSpace(patient.FirstName + " " + patient.LastName)
		}
	} else {
		patient = &clinical_models.Patient{}
	}

	var diagnoses []clinical_models.DiagnosisItem
	_ = json.Unmarshal(order.Diagnoses, &diagnoses)

	data := clinical_templates.ExamOrderData{
		TradeName:       tradeName,
		DoctorName:      examOrderDoctorName(db, order),
		Date:            order.CreatedAt.Format("02/01/2006"),
		Code:            clinical_models.FormatExamOrderNumber(order.Number),
		PatientName:     patientName,
		PatientDocument: patient.Document,
		PatientAge:      patientAge(patient),
		PatientPhone:    patient.Phone,
		Urgent:          order.Priority == clinical_models.ExamPriorityUrgent,
		Priority:        map[bool]string{true: "Urgente", false: "Rutina"}[order.Priority == clinical_models.ExamPriorityUrgent],
		Diagnoses:       diagnoses,
		DestinationLab:  order.DestinationLab,
		Notes:           order.Notes,
		Groups:          examOrderGroups(order.Items),
		Signature:       stamp,
	}
	return h.renderer.Render(tenantdb.TenantID(c), clinical_templates.ExamOrder, data)
}

func examOrderFileName(order *clinical_models.ExamOrder) string {
	return medicalDocumentFileName("orden_examenes_"+clinical_models.FormatExamOrderNumber(order.Number), order.Patient)
}

// DownloadExamOrder streams the order's PDF (the signed one if signed).
func (h *ExamOrderHandler) DownloadExamOrder(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		envelope.Write(c, envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest))
		return
	}
	order, err := loadExamOrder(tenantdb.For(c, h.db), id)
	if err != nil {
		envelope.Write(c, examOrderNotFound())
		return
	}
	audit.RecordAccess(h.db, c, "exam_orders", order.ID, &order.PatientID)

	pdf, err := storedOrRendered(c, h.files, order.DocumentSignature, func() ([]byte, error) {
		return h.generateExamOrderPDF(c, order, nil)
	})
	if err != nil {
		h.logger.Error("failed to generate exam order PDF", zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_order.pdf.error", core_errors.ErrClinicalExamOrderPDFError))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", examOrderFileName(order)))
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func (h *ExamOrderHandler) EmailExamOrder(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req clinical_dto.EmailMedicalDocumentDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	order, err := loadExamOrder(tenantdb.For(c, h.db), id)
	if err != nil {
		return examOrderNotFound()
	}
	if order.VoidedAt != nil {
		return examOrderVoided()
	}
	pdf, err := storedOrRendered(c, h.files, order.DocumentSignature, func() ([]byte, error) {
		return h.generateExamOrderPDF(c, order, nil)
	})
	if err != nil {
		h.logger.Error("failed to generate exam order PDF for email", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_order.pdf.error", core_errors.ErrClinicalExamOrderPDFError)
	}
	title := "Orden de exámenes " + clinical_models.FormatExamOrderNumber(order.Number)
	if err := h.mailer.SendMedicalDocumentEmail(req.Email, title, title, examOrderFileName(order), pdf); err != nil {
		h.logger.Error("failed to email exam order", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrClinicalDocumentEmailError)
	}
	return envelope.SuccessResponse(nil, "clinical.exam_order.email.success")
}

// SignExamOrder signs the order's PDF with the current user's certificate.
func (h *ExamOrderHandler) SignExamOrder(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	order, err := loadExamOrder(tenantdb.For(c, h.db), id)
	if err != nil {
		return examOrderNotFound()
	}
	if order.VoidedAt != nil {
		return examOrderVoided()
	}
	if order.IsSigned() {
		return signature_services.AlreadySignedResponse()
	}
	_, err = signDocument(c, h.db, h.signer, h.files, &clinical_models.ExamOrder{}, "exam_order", order.ID, "Orden de exámenes",
		func(stamp *pdfsign.Stamp) ([]byte, error) { return h.generateExamOrderPDF(c, order, stamp) })
	if err != nil {
		if errors.Is(err, errAlreadySigned) {
			return signature_services.AlreadySignedResponse()
		}
		h.logger.Error("failed to sign exam order", zap.Error(err))
		return signature_services.ErrorResponse(err)
	}
	return h.respondExamOrder(c, order.ID, http.StatusOK, "signature.document.sign.success")
}
