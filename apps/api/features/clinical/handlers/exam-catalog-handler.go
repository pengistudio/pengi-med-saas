package clinical_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_dto "pengi-med-saas/features/clinical/dto"
	clinical_models "pengi-med-saas/features/clinical/models"
	clinical_services "pengi-med-saas/features/clinical/services"
)

// ExamCatalogHandler manages the tenant's exam catalog and profiles.
type ExamCatalogHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewExamCatalogHandler(db *gorm.DB, logger *zap.Logger) *ExamCatalogHandler {
	return &ExamCatalogHandler{db: db, logger: logger}
}

// examCatalogOrder lists laboratory, imaging, other; seeded exams in seed
// order, the tenant's own after them.
const examCatalogOrder = "CASE category WHEN 'laboratory' THEN 0 WHEN 'imaging' THEN 1 ELSE 2 END, sort_order, name"

func pathID(c *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	return uint(id), err == nil && id > 0
}

// GetExamCatalog lists the catalog. Query: active=true|false, category, q (name).
func (h *ExamCatalogHandler) GetExamCatalog(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	query := db.Model(&clinical_models.ExamCatalogItem{})
	if v := c.Query("active"); v != "" {
		query = query.Where("active = ?", v == "true")
	}
	if v := c.Query("category"); v != "" {
		query = query.Where("category = ?", v)
	}
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(v)+"%")
	}
	var items []clinical_models.ExamCatalogItem
	if err := query.Order(examCatalogOrder).Find(&items).Error; err != nil {
		h.logger.Error("failed to list exam catalog", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.fetch.error", core_errors.ErrClinicalExamCatalogError)
	}
	return envelope.SuccessResponse(items, "clinical.exam_catalog.fetch.success")
}

func (h *ExamCatalogHandler) CreateExamCatalogItem(c *gin.Context) envelope.Response {
	var req clinical_dto.CreateExamCatalogItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	item := clinical_models.ExamCatalogItem{
		TenantID:           tenantdb.TenantID(c),
		Name:               strings.TrimSpace(req.Name),
		Category:           req.Category,
		Subgroup:           strings.TrimSpace(req.Subgroup),
		DefaultIndications: strings.TrimSpace(req.DefaultIndications),
		Active:             true,
		SortOrder:          clinical_services.CustomExamSortOrder,
	}
	if err := tenantdb.For(c, h.db).Create(&item).Error; err != nil {
		h.logger.Error("failed to create exam catalog item", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError)
	}
	return envelope.New(http.StatusCreated, "clinical.exam_catalog.create.success", item)
}

func (h *ExamCatalogHandler) UpdateExamCatalogItem(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req clinical_dto.UpdateExamCatalogItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var item clinical_models.ExamCatalogItem
	if err := db.First(&item, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "clinical.exam_catalog.error.not_found", core_errors.ErrClinicalExamCatalogNotFound)
	}
	updates := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
		}
		updates["name"] = name
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Subgroup != nil {
		updates["subgroup"] = strings.TrimSpace(*req.Subgroup)
	}
	if req.DefaultIndications != nil {
		updates["default_indications"] = strings.TrimSpace(*req.DefaultIndications)
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if len(updates) > 0 {
		if err := db.Model(&item).Updates(updates).Error; err != nil {
			h.logger.Error("failed to update exam catalog item", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError)
		}
	}
	if err := db.First(&item, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "clinical.exam_catalog.error.not_found", core_errors.ErrClinicalExamCatalogNotFound)
	}
	return envelope.SuccessResponse(item, "clinical.exam_catalog.update.success")
}

// DeleteExamCatalogItem soft-deletes an exam. Orders keep their copy of it;
// profiles stop listing it.
func (h *ExamCatalogHandler) DeleteExamCatalogItem(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var item clinical_models.ExamCatalogItem
	if err := db.First(&item, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "clinical.exam_catalog.error.not_found", core_errors.ErrClinicalExamCatalogNotFound)
	}
	if err := db.Delete(&item).Error; err != nil {
		h.logger.Error("failed to delete exam catalog item", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError)
	}
	return envelope.SuccessResponse(nil, "clinical.exam_catalog.delete.success")
}

// RestoreExamCatalogDefaults re-activates (or re-creates) the seeded exams and
// profiles without touching the tenant's own.
func (h *ExamCatalogHandler) RestoreExamCatalogDefaults(c *gin.Context) envelope.Response {
	var res clinical_dto.RestoreExamCatalogResponse
	err := tenantdb.For(c, h.db).Transaction(func(tx *gorm.DB) error {
		var err error
		res.RestoredItems, res.RestoredProfiles, err = clinical_services.RestoreExamCatalogDefaults(tx, tenantdb.TenantID(c))
		return err
	})
	if err != nil {
		h.logger.Error("failed to restore exam catalog", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError)
	}
	return envelope.SuccessResponse(res, "clinical.exam_catalog.restore.success")
}

// ── Profiles ────────────────────────────────────────────────────────────────

func preloadProfileItems(db *gorm.DB) *gorm.DB {
	return db.Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order(examCatalogOrder) })
}

// GetExamProfiles lists profiles with their exams. Query: active=true|false.
func (h *ExamCatalogHandler) GetExamProfiles(c *gin.Context) envelope.Response {
	query := preloadProfileItems(tenantdb.For(c, h.db))
	if v := c.Query("active"); v != "" {
		query = query.Where("active = ?", v == "true")
	}
	var profiles []clinical_models.ExamProfile
	if err := query.Order("name").Find(&profiles).Error; err != nil {
		h.logger.Error("failed to list exam profiles", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.fetch.error", core_errors.ErrClinicalExamCatalogError)
	}
	return envelope.SuccessResponse(profiles, "clinical.exam_profile.fetch.success")
}

func (h *ExamCatalogHandler) CreateExamProfile(c *gin.Context) envelope.Response {
	var req clinical_dto.CreateExamProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	profile := clinical_models.ExamProfile{TenantID: tenantdb.TenantID(c), Name: strings.TrimSpace(req.Name), Active: true}
	if resp, ok := h.saveProfile(db, &profile, &req.ItemIDs, true); !ok {
		return resp
	}
	return envelope.New(http.StatusCreated, "clinical.exam_profile.create.success", profile)
}

func (h *ExamCatalogHandler) UpdateExamProfile(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	var req clinical_dto.UpdateExamProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var profile clinical_models.ExamProfile
	if err := db.First(&profile, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "clinical.exam_profile.error.not_found", core_errors.ErrClinicalExamProfileNotFound)
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
		}
		profile.Name = strings.TrimSpace(*req.Name)
	}
	if req.Active != nil {
		profile.Active = *req.Active
	}
	if resp, ok := h.saveProfile(db, &profile, req.ItemIDs, false); !ok {
		return resp
	}
	return envelope.SuccessResponse(profile, "clinical.exam_profile.update.success")
}

// saveProfile creates or updates profile and, when itemIDs is given, replaces
// its exams; it reloads profile with its exams.
func (h *ExamCatalogHandler) saveProfile(db *gorm.DB, profile *clinical_models.ExamProfile, itemIDs *[]uint, create bool) (envelope.Response, bool) {
	if itemIDs != nil {
		if _, err := clinical_services.CatalogItemsByID(db, *itemIDs); err != nil {
			if errors.Is(err, clinical_services.ErrUnknownCatalogItems) {
				return envelope.ErrorResponse(http.StatusBadRequest, "clinical.exam_catalog.error.not_found", core_errors.ErrClinicalExamCatalogNotFound), false
			}
			h.logger.Error("failed to load exam catalog items", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError), false
		}
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if create {
			if err := tx.Omit("Items").Create(profile).Error; err != nil {
				return err
			}
		} else if err := tx.Model(profile).Select("name", "active").Updates(map[string]any{"name": profile.Name, "active": profile.Active}).Error; err != nil {
			return err
		}
		if itemIDs != nil {
			return clinical_services.SetProfileItems(tx, profile.ID, *itemIDs)
		}
		return nil
	})
	if err == nil {
		err = preloadProfileItems(db).First(profile, profile.ID).Error
	}
	if err != nil {
		h.logger.Error("failed to save exam profile", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError), false
	}
	return envelope.Response{}, true
}

func (h *ExamCatalogHandler) DeleteExamProfile(c *gin.Context) envelope.Response {
	id, ok := pathID(c, "id")
	if !ok {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
	}
	db := tenantdb.For(c, h.db)
	var profile clinical_models.ExamProfile
	if err := db.First(&profile, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "clinical.exam_profile.error.not_found", core_errors.ErrClinicalExamProfileNotFound)
	}
	if err := db.Delete(&profile).Error; err != nil {
		h.logger.Error("failed to delete exam profile", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "clinical.exam_catalog.save.error", core_errors.ErrClinicalExamCatalogError)
	}
	return envelope.SuccessResponse(nil, "clinical.exam_profile.delete.success")
}
