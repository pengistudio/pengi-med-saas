package permission_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	permission_models "pengi-med-saas/features/permissions/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PermissionHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPermissionHandler(db *gorm.DB, logger *zap.Logger) *PermissionHandler {
	return &PermissionHandler{
		db:     db,
		logger: logger,
	}
}

func (h *PermissionHandler) GetAllPermissions(c *gin.Context) envelope.Response {
	var permissions []permission_models.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
		h.logger.Error("Failed to fetch permissions", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrPermissionGetError)
	}
	return envelope.SuccessResponse(permissions, "permission.list.success")
}

type PermissionCategoryResponse struct {
	Category    string                         `json:"category"`
	Permissions []permission_models.Permission `json:"permissions"`
}

func (h *PermissionHandler) GetAllPermissionWithCategory(c *gin.Context) envelope.Response {
	var permissions []permission_models.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
		h.logger.Error("Failed to fetch permissions for categories", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, err.Error(), core_errors.ErrPermissionGetError)
	}

	categoryMap := make(map[string][]permission_models.Permission)
	for _, p := range permissions {
		categoryMap[p.Category] = append(categoryMap[p.Category], p)
	}

	var response []PermissionCategoryResponse
	for cat, perms := range categoryMap {
		response = append(response, PermissionCategoryResponse{
			Category:    cat,
			Permissions: perms,
		})
	}

	return envelope.SuccessResponse(response, "permission.categories.success")
}
