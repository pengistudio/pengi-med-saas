package backoffice_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	permission_models "pengi-med-saas/features/permissions/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeFeatureHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeFeatureHandler(db *gorm.DB, logger *zap.Logger) *BackofficeFeatureHandler {
	return &BackofficeFeatureHandler{db: db, logger: logger}
}

// ── DTOs ────────────────────────────────────────────────────────────────────

type CreateFeatureRequest struct {
	Code          string   `json:"code" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	PermissionIDs []string `json:"permission_ids"`
}

type UpdateFeatureRequest struct {
	Name          string   `json:"name"`
	PermissionIDs []string `json:"permission_ids"`
}

// ── Handlers ────────────────────────────────────────────────────────────────

func (h *BackofficeFeatureHandler) GetFeatures(c *gin.Context) envelope.Response {
	var features []company_models.Feature
	if err := h.db.Preload("Permissions").Find(&features).Error; err != nil {
		h.logger.Error("Failed to fetch features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining features", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(features, "backoffice.feature.list.success")
}

func (h *BackofficeFeatureHandler) GetFeatureByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var feature company_models.Feature
	if err := h.db.Preload("Permissions").First(&feature, id).Error; err != nil {
		h.logger.Error("Feature not found", zap.String("id", id), zap.Error(err))
		return envelope.ErrorResponse(http.StatusNotFound, "Feature not found", core_errors.ErrBackofficeFeatureNotFound)
	}
	return envelope.SuccessResponse(feature, "backoffice.feature.found")
}

func (h *BackofficeFeatureHandler) CreateFeature(c *gin.Context) envelope.Response {
	var req CreateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	feature := company_models.Feature{
		Code: req.Code,
		Name: req.Name,
	}

	if err := h.db.Create(&feature).Error; err != nil {
		h.logger.Error("Failed to create feature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating feature", core_errors.ErrInternal)
	}

	// Attach permissions if provided
	if len(req.PermissionIDs) > 0 {
		if err := h.db.Model(&feature).Association("Permissions").Replace(h.findPermissionsByIDs(req.PermissionIDs)); err != nil {
			h.logger.Error("Failed to attach permissions", zap.Error(err))
		}
	}

	h.db.Preload("Permissions").First(&feature, feature.ID)
	h.logger.Info("Feature created", zap.String("code", req.Code))
	return envelope.New(http.StatusCreated, "backoffice.feature.create.success", feature)
}

func (h *BackofficeFeatureHandler) UpdateFeature(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var feature company_models.Feature
	if err := h.db.First(&feature, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Feature not found", core_errors.ErrBackofficeFeatureNotFound)
	}

	var req UpdateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrBackofficeInvalidRequest)
	}

	if req.Name != "" {
		h.db.Model(&feature).Update("name", req.Name)
	}

	if req.PermissionIDs != nil {
		if err := h.db.Model(&feature).Association("Permissions").Replace(h.findPermissionsByIDs(req.PermissionIDs)); err != nil {
			h.logger.Error("Failed to update permissions", zap.Error(err))
		}
	}

	h.db.Preload("Permissions").First(&feature, feature.ID)
	return envelope.SuccessResponse(feature, "backoffice.feature.update.success")
}

func (h *BackofficeFeatureHandler) DeleteFeature(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var feature company_models.Feature
	if err := h.db.First(&feature, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Feature not found", core_errors.ErrBackofficeFeatureNotFound)
	}

	// Clear associations first
	h.db.Model(&feature).Association("Permissions").Clear()

	if err := h.db.Delete(&feature).Error; err != nil {
		h.logger.Error("Failed to delete feature", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting feature", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(nil, "backoffice.feature.delete.success")
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func (h *BackofficeFeatureHandler) findPermissionsByIDs(ids []string) []permission_models.Permission {
	var permissions []permission_models.Permission
	if len(ids) > 0 {
		h.db.Where("id IN ?", ids).Find(&permissions)
	}
	return permissions
}
