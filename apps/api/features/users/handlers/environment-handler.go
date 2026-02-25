package user_handlers

import (
	"net/http"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	company_models "pengi-med-saas/features/companies/models"
	user_dto "pengi-med-saas/features/users/dto"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type EnvironmentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewEnvironmentHandler(db *gorm.DB, logger *zap.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{
		db:     db,
		logger: logger,
	}
}

func (h *EnvironmentHandler) GetEnvironmentsFromUser(c *gin.Context) envelope.Response {
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("User ID not found in context")
		return envelope.ErrorResponse(http.StatusUnauthorized, "Unauthorized", core_errors.ErrAuthInvalidRequest)
	}

	var user user_models.User
	if err := h.db.Preload("Environments.Role.Permissions").Preload("Environments").First(&user, userID).Error; err != nil {
		h.logger.Error("Failed to fetch user and environments", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining environments", core_errors.ErrUserNotFound)
	}

	// Create custom DTO struct

	var responseDTO []user_dto.EnvironmentWithCompany

	for _, env := range user.Environments {
		var company company_models.Company
		if err := h.db.Preload("Tenant").First(&company, env.CompanyID).Error; err == nil {
			responseDTO = append(responseDTO, user_dto.EnvironmentWithCompany{
				Environment: env,
				Company:     company,
			})
		} else {
			responseDTO = append(responseDTO, user_dto.EnvironmentWithCompany{
				Environment: env,
			})
		}
	}

	return envelope.SuccessResponse(responseDTO, "environment.list.success")
}
