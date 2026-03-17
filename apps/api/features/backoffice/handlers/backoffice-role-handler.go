package backoffice_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	permission_models "pengi-med-saas/features/permissions/models"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BackofficeRoleHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeRoleHandler(db *gorm.DB, logger *zap.Logger) *BackofficeRoleHandler {
	return &BackofficeRoleHandler{db: db, logger: logger}
}

type createRoleRequest struct {
	Role          string   `json:"role" binding:"required"`
	PermissionIDs []string `json:"permission_ids"`
}

type updateRoleRequest struct {
	Role          *string  `json:"role"`
	PermissionIDs []string `json:"permission_ids"`
}

func (h *BackofficeRoleHandler) GetRoles(c *gin.Context) envelope.Response {
	var roles []user_models.Role
	if err := h.db.Preload("Permissions").Find(&roles).Error; err != nil {
		h.logger.Error("Failed to fetch roles", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching roles", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(roles, "backoffice.roles.list.success")
}

func (h *BackofficeRoleHandler) GetRoleByID(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var role user_models.Role
	if err := h.db.Preload("Permissions").First(&role, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Role not found", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(role, "backoffice.roles.get.success")
}

func (h *BackofficeRoleHandler) CreateRole(c *gin.Context) envelope.Response {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrBackofficeInvalidRequest)
	}

	role := user_models.Role{
		Role:        req.Role,
		Permissions: h.findPermissionsByIDs(req.PermissionIDs),
	}
	if err := h.db.Create(&role).Error; err != nil {
		h.logger.Error("Failed to create role", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating role", core_errors.ErrInternal)
	}
	h.db.Preload("Permissions").First(&role, role.ID)
	return envelope.SuccessResponse(role, "backoffice.roles.create.success")
}

func (h *BackofficeRoleHandler) UpdateRole(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrBackofficeInvalidRequest)
	}

	var role user_models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Role not found", core_errors.ErrInternal)
	}

	if req.Role != nil {
		if err := h.db.Model(&role).Update("role", *req.Role).Error; err != nil {
			h.logger.Error("Failed to update role name", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating role", core_errors.ErrInternal)
		}
	}

	if req.PermissionIDs != nil {
		if err := h.db.Model(&role).Association("Permissions").Replace(h.findPermissionsByIDs(req.PermissionIDs)); err != nil {
			h.logger.Error("Failed to update role permissions", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error updating permissions", core_errors.ErrInternal)
		}
	}

	h.db.Preload("Permissions").First(&role, role.ID)
	return envelope.SuccessResponse(role, "backoffice.roles.update.success")
}

func (h *BackofficeRoleHandler) DeleteRole(c *gin.Context) envelope.Response {
	id := c.Param("id")
	var role user_models.Role
	if err := h.db.First(&role, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Role not found", core_errors.ErrInternal)
	}
	h.db.Model(&role).Association("Permissions").Clear()
	if err := h.db.Delete(&role).Error; err != nil {
		h.logger.Error("Failed to delete role", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error deleting role", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(nil, "backoffice.roles.delete.success")
}

func (h *BackofficeRoleHandler) findPermissionsByIDs(ids []string) []permission_models.Permission {
	var perms []permission_models.Permission
	if len(ids) > 0 {
		h.db.Where("id IN ?", ids).Find(&perms)
	}
	return perms
}
