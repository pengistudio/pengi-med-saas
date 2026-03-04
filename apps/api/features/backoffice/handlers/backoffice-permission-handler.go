package backoffice_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	permission_models "pengi-med-saas/features/permissions/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficePermissionHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficePermissionHandler(db *gorm.DB, logger *zap.Logger) *BackofficePermissionHandler {
	return &BackofficePermissionHandler{db: db, logger: logger}
}

func (h *BackofficePermissionHandler) GetPermissions(c *gin.Context) envelope.Response {
	var permissions []permission_models.Permission
	if err := h.db.Find(&permissions).Error; err != nil {
		h.logger.Error("Failed to fetch permissions", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining permissions", core_errors.ErrPermissionGetError)
	}
	return envelope.SuccessResponse(permissions, "backoffice.permission.list.success")
}
