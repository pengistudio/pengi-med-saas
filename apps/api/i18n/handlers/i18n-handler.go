package i18n_handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	message_cache "pengi-med-saas/i18n/cache"
	message_models "pengi-med-saas/i18n/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MessageHandler struct {
	db *gorm.DB
}

func NewMessageHandler(db *gorm.DB) *MessageHandler {
	return &MessageHandler{db: db}
}

func (h *MessageHandler) GetAllMessages(c *gin.Context) envelope.Response {
	lang := c.Query("lang")
	if lang == "" {
		lang = "es" // Default language
	}

	messages := []message_models.Message{}
	if err := h.db.Where("lang = ?", lang).Find(&messages).Error; err != nil {
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error obtaining messages", core_errors.ErrMessagesNotFound)
	}

	return envelope.SuccessResponse(messages, "Messages obtained successfully")
}

func (h *MessageHandler) GetMessageVersion(c *gin.Context) envelope.Response {
	version := "v0.1.0"
	return envelope.SuccessResponse(version, "Version obtained successfully")
}

func (h *MessageHandler) ReloadMessages(c *gin.Context) envelope.Response {
	logger := zap.L()

	workDir, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to get working directory", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to get working directory", core_errors.ErrInternal)
	}

	for _, lang := range []string{"es", "en"} {
		file := filepath.Join(workDir, "i18n", "messages", fmt.Sprintf("messages_%s.json", lang))
		if err := message_models.LoadMessagesFromFile(h.db, file, lang); err != nil {
			logger.Error("Failed to seed messages", zap.String("lang", lang), zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to seed messages", core_errors.ErrInternal)
		}
	}

	if err := message_cache.Reload(h.db); err != nil {
		logger.Error("Failed to reload message cache", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to reload message cache", core_errors.ErrInternal)
	}

	logger.Info("i18n messages reloaded successfully")
	return envelope.SuccessResponse(nil, "i18n.reload.success")
}
