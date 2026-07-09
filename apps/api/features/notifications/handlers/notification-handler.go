package notifications_handlers

import (
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	notifications_dto "pengi-med-saas/features/notifications/dto"
	notifications_models "pengi-med-saas/features/notifications/models"
	notifications_service "pengi-med-saas/features/notifications/services"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"
	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type NotificationHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewNotificationHandler(db *gorm.DB, logger *zap.Logger) *NotificationHandler {
	return &NotificationHandler{db: db, logger: logger}
}

func (h *NotificationHandler) currentUserID(c *gin.Context) (uint, bool) {
	userID, _, exists := auth_middleware.GetUserFromContext(c)
	if !exists {
		return 0, false
	}
	return uint(userID), true
}

func (h *NotificationHandler) ListNotifications(c *gin.Context) envelope.Response {
	userID, ok := h.currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrNotificationFetchError)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	baseQuery := h.db.Scopes(tenant_middleware.TenantScope(c)).
		Model(&notifications_models.Notification{}).
		Where("user_id = ?", userID)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("failed to count notifications", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "notification.error.fetch_failed", core_errors.ErrNotificationFetchError)
	}

	var unreadCount int64
	unreadQuery := h.db.Scopes(tenant_middleware.TenantScope(c)).
		Model(&notifications_models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID)
	if err := unreadQuery.Count(&unreadCount).Error; err != nil {
		h.logger.Error("failed to count unread notifications", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "notification.error.fetch_failed", core_errors.ErrNotificationFetchError)
	}

	var notifications []notifications_models.Notification
	if err := baseQuery.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error; err != nil {
		h.logger.Error("failed to fetch notifications", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "notification.error.fetch_failed", core_errors.ErrNotificationFetchError)
	}

	response := notifications_dto.ListNotificationsResponse{
		Items:       notifications_dto.ToNotificationDTOs(notifications),
		UnreadCount: int(unreadCount),
		Total:       int(total),
		Page:        page,
		Limit:       limit,
	}

	return envelope.SuccessResponse(response, "notification.list.success")
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) envelope.Response {
	userID, ok := h.currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrNotificationUpdateError)
	}

	notificationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrNotificationUpdateError)
	}

	tenantID := c.GetUint("tenant_id")
	if err := notifications_service.MarkAsRead(h.db, tenantID, userID, uint(notificationID)); err != nil {
		h.logger.Error("failed to mark notification as read", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "notification.error.update_failed", core_errors.ErrNotificationUpdateError)
	}

	return envelope.SuccessResponse(nil, "notification.mark_read.success")
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) envelope.Response {
	userID, ok := h.currentUserID(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "error.unauthorized", core_errors.ErrNotificationUpdateError)
	}

	tenantID := c.GetUint("tenant_id")
	if err := notifications_service.MarkAllAsRead(h.db, tenantID, userID); err != nil {
		h.logger.Error("failed to mark all notifications as read", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "notification.error.update_failed", core_errors.ErrNotificationUpdateError)
	}

	return envelope.SuccessResponse(nil, "notification.mark_all_read.success")
}
