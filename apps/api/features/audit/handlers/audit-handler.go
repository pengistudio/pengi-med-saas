package audit_handlers

import (
	"net/http"
	"pengi-med-saas/core/tenantdb"
	"sort"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuditLogHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAuditLogHandler(db *gorm.DB, logger *zap.Logger) *AuditLogHandler {
	return &AuditLogHandler{db: db, logger: logger}
}

// AuditLogItem is an audit log row with the names the viewer shows.
type AuditLogItem struct {
	audit.AuditLog
	UserName    string `json:"user_name"`
	PatientName string `json:"patient_name,omitempty"`
}

// AuditUser is someone who appears in the tenant's audit trail.
type AuditUser struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// GetAuditLogs returns the tenant-scoped compliance audit trail, filterable by
// user_id, patient_id, entity_type, entity_id, action, and a from/to date
// range, with the user's and patient's names.
func (h *AuditLogHandler) GetAuditLogs(c *gin.Context) envelope.Response {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	baseQuery := tenantdb.For(c, h.db).Model(&audit.AuditLog{})

	if userID := c.Query("user_id"); userID != "" {
		id, err := strconv.ParseUint(userID, 10, 32)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("user_id = ?", uint(id))
	}
	if patientID := c.Query("patient_id"); patientID != "" {
		id, err := strconv.ParseUint(patientID, 10, 32)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("patient_id = ?", uint(id))
	}
	if entityType := c.Query("entity_type"); entityType != "" {
		baseQuery = baseQuery.Where("entity_type = ?", entityType)
	}
	if entityID := c.Query("entity_id"); entityID != "" {
		id, err := strconv.ParseUint(entityID, 10, 32)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("entity_id = ?", uint(id))
	}
	if action := c.Query("action"); action != "" {
		baseQuery = baseQuery.Where("action = ?", action)
	}
	if from := c.Query("from"); from != "" {
		fromDate, err := time.Parse("2006-01-02", from)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("created_at >= ?", fromDate)
	}
	if to := c.Query("to"); to != "" {
		toDate, err := time.Parse("2006-01-02", to)
		if err != nil {
			return envelope.ErrorResponse(http.StatusBadRequest, "audit.log.error.invalid_request", core_errors.ErrAuditInvalidRequest)
		}
		baseQuery = baseQuery.Where("created_at <= ?", toDate.Add(24*time.Hour))
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count audit logs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "audit.log.error.fetch_failed", core_errors.ErrAuditFetchError)
	}

	var logs []audit.AuditLog
	if err := baseQuery.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		h.logger.Error("Failed to fetch audit logs", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "audit.log.error.fetch_failed", core_errors.ErrAuditFetchError)
	}

	userIDs := make([]uint, 0, len(logs))
	patientIDs := make([]uint, 0, len(logs))
	for _, l := range logs {
		userIDs = append(userIDs, l.UserID)
		if id := patientOf(l); id != 0 {
			patientIDs = append(patientIDs, id)
		}
	}
	userNames := h.userNames(c, userIDs)
	patientNames := h.patientNames(c, patientIDs)
	items := make([]AuditLogItem, len(logs))
	for i, l := range logs {
		items[i] = AuditLogItem{AuditLog: l, UserName: userNames[l.UserID], PatientName: patientNames[patientOf(l)]}
	}

	return envelope.PagedSuccessResponse(items, int(total), page, limit, "audit.log.list.success")
}

// patientOf is the patient an audit row is about, or 0. A row on the patient
// itself is about that patient even when patient_id wasn't recorded (rows
// written before it was resolved).
func patientOf(l audit.AuditLog) uint {
	if l.PatientID != nil {
		return *l.PatientID
	}
	if l.EntityType == "patients" {
		return l.EntityID
	}
	return 0
}

// GetAuditUsers lists who appears in the tenant's audit trail (the viewer's
// user filter), including people no longer on the team.
func (h *AuditLogHandler) GetAuditUsers(c *gin.Context) envelope.Response {
	var ids []uint
	if err := tenantdb.For(c, h.db).Model(&audit.AuditLog{}).Distinct("user_id").Pluck("user_id", &ids).Error; err != nil {
		h.logger.Error("Failed to list audit users", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "audit.log.error.fetch_failed", core_errors.ErrAuditFetchError)
	}
	names := h.userNames(c, ids)
	users := make([]AuditUser, 0, len(ids))
	for _, id := range ids {
		users = append(users, AuditUser{ID: id, Name: names[id]})
	}
	sort.Slice(users, func(i, j int) bool { return strings.ToLower(users[i].Name) < strings.ToLower(users[j].Name) })
	return envelope.SuccessResponse(users, "audit.users.list.success")
}

// userNames resolves user IDs to the name they have on the tenant's team,
// falling back to their user name (people who left the team). Best effort: a
// lookup failure leaves the names empty.
func (h *AuditLogHandler) userNames(c *gin.Context, ids []uint) map[uint]string {
	names := map[uint]string{}
	if len(ids) == 0 {
		return names
	}
	db := tenantdb.For(c, h.db)
	var users []user_models.User
	if err := db.Select("id", "user_name").Where("id IN ?", ids).Find(&users).Error; err != nil {
		h.logger.Warn("Failed to resolve audit user names", zap.Error(err))
		return names
	}
	for _, u := range users {
		names[u.ID] = u.UserName
	}
	var company company_models.Company
	if err := db.Select("id").First(&company).Error; err != nil {
		return names
	}
	var envs []user_models.Environment
	db.Select("user_id", "name").Where("company_id = ? AND user_id IN ? AND name <> ''", company.ID, ids).Find(&envs)
	for _, e := range envs {
		names[e.UserID] = e.Name
	}
	return names
}

// patientNames resolves the tenant's patient IDs to "First Last".
func (h *AuditLogHandler) patientNames(c *gin.Context, ids []uint) map[uint]string {
	names := map[uint]string{}
	if len(ids) == 0 {
		return names
	}
	var patients []clinical_models.Patient
	if err := tenantdb.For(c, h.db).Unscoped().Select("id", "first_name", "last_name").Where("id IN ?", ids).Find(&patients).Error; err != nil {
		h.logger.Warn("Failed to resolve audit patient names", zap.Error(err))
		return names
	}
	for _, p := range patients {
		names[p.ID] = strings.TrimSpace(p.FirstName + " " + p.LastName)
	}
	return names
}
