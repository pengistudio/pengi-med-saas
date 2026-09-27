package backoffice_handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	backoffice_dto "pengi-med-saas/features/backoffice/dto"
	company_models "pengi-med-saas/features/companies/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	notifications_service "pengi-med-saas/features/notifications/services"
	user_models "pengi-med-saas/features/users/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// announcementHistoryLimit caps the history list; older announcements stay in
// the table but aren't shown.
const announcementHistoryLimit = 200

type BackofficeAnnouncementHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewBackofficeAnnouncementHandler(db *gorm.DB, logger *zap.Logger) *BackofficeAnnouncementHandler {
	return &BackofficeAnnouncementHandler{
		db:     tenantdb.System(db),
		logger: logger,
	}
}

func (h *BackofficeAnnouncementHandler) GetAnnouncements(c *gin.Context) envelope.Response {
	var announcements []notifications_models.Announcement
	if err := h.db.Order("created_at DESC").Limit(announcementHistoryLimit).Find(&announcements).Error; err != nil {
		h.logger.Error("Failed to fetch announcements", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching announcements", core_errors.ErrInternal)
	}

	companyIDs, userIDs := []uint{}, []uint{}
	for _, a := range announcements {
		if a.CompanyID != nil {
			companyIDs = append(companyIDs, *a.CompanyID)
		}
		if a.UserID != nil {
			userIDs = append(userIDs, *a.UserID)
		}
	}

	companyNames := map[uint]string{}
	if len(companyIDs) > 0 {
		var companies []company_models.Company
		if err := h.db.Unscoped().Select("id", "trade_name").Where("id IN ?", companyIDs).Find(&companies).Error; err != nil {
			h.logger.Error("Failed to fetch announcement companies", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching announcements", core_errors.ErrInternal)
		}
		for _, company := range companies {
			companyNames[company.ID] = company.TradeName
		}
	}

	userNames := map[uint]string{}
	if len(userIDs) > 0 {
		var users []user_models.User
		if err := h.db.Unscoped().Select("id", "user_name", "email").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			h.logger.Error("Failed to fetch announcement users", zap.Error(err))
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error fetching announcements", core_errors.ErrInternal)
		}
		for _, u := range users {
			userNames[u.ID] = u.UserName
			if u.UserName == "" {
				userNames[u.ID] = u.Email
			}
		}
	}

	result := make([]backoffice_dto.AnnouncementDTO, 0, len(announcements))
	for _, a := range announcements {
		dto := backoffice_dto.AnnouncementDTO{
			ID:             a.ID,
			CreatedAt:      a.CreatedAt,
			UpdatedAt:      a.UpdatedAt,
			Scope:          a.Scope,
			CompanyID:      a.CompanyID,
			UserID:         a.UserID,
			Title:          a.Title,
			Body:           a.Body,
			Level:          a.Level,
			ActionURL:      a.ActionURL,
			ScheduledAt:    a.ScheduledAt,
			SentAt:         a.SentAt,
			Status:         a.Status,
			RecipientCount: a.RecipientCount,
		}
		if a.CompanyID != nil {
			dto.CompanyName = companyNames[*a.CompanyID]
		}
		if a.UserID != nil {
			dto.UserName = userNames[*a.UserID]
		}
		result = append(result, dto)
	}

	return envelope.SuccessResponse(result, "backoffice.announcement.list.success")
}

// CreateAnnouncement sends an announcement right away, or stores it for the
// AnnouncementScheduler when it has a future scheduled_at.
func (h *BackofficeAnnouncementHandler) CreateAnnouncement(c *gin.Context) envelope.Response {
	var req backoffice_dto.CreateAnnouncementDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid request", core_errors.ErrAnnouncementInvalidRequest)
	}
	if !validActionURL(req.ActionURL) {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid action url", core_errors.ErrAnnouncementInvalidRequest)
	}

	announcement := notifications_models.Announcement{
		Scope:     req.Scope,
		Title:     strings.TrimSpace(req.Title),
		Body:      strings.TrimSpace(req.Body),
		Level:     req.Level,
		ActionURL: strings.TrimSpace(req.ActionURL),
	}
	switch req.Scope {
	case notifications_models.AnnouncementScopeCompany:
		announcement.CompanyID = req.CompanyID
	case notifications_models.AnnouncementScopeUser:
		announcement.CompanyID = req.CompanyID
		announcement.UserID = req.UserID
	}
	if userID, ok := c.Get("user_id"); ok {
		if id, ok := userID.(int64); ok {
			announcement.CreatedByID = uint(id)
		}
	}

	// Reject a target that reaches nobody before storing anything, so a
	// scheduled announcement can't silently fail later for that reason.
	recipients, err := notifications_service.ResolveRecipients(h.db, &announcement)
	if err != nil && !errors.Is(err, notifications_service.ErrNoRecipients) {
		h.logger.Error("Failed to resolve announcement recipients", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error resolving recipients", core_errors.ErrInternal)
	}
	if len(recipients) == 0 {
		return envelope.ErrorResponse(http.StatusBadRequest, "Announcement target has no recipients", core_errors.ErrAnnouncementInvalidTarget)
	}

	// A scheduled time that has already passed means "send now".
	scheduled := req.ScheduledAt != nil && req.ScheduledAt.After(time.Now())
	if scheduled {
		announcement.ScheduledAt = req.ScheduledAt
		announcement.Status = notifications_models.AnnouncementStatusScheduled
	} else {
		announcement.Status = notifications_models.AnnouncementStatusFailed // until Dispatch succeeds
	}

	if err := h.db.Create(&announcement).Error; err != nil {
		h.logger.Error("Failed to create announcement", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error creating announcement", core_errors.ErrInternal)
	}

	if !scheduled {
		if err := notifications_service.Dispatch(h.db, h.logger, &announcement); err != nil {
			return envelope.ErrorResponse(http.StatusInternalServerError, "Error sending announcement", core_errors.ErrAnnouncementDispatchError)
		}
	}

	h.logger.Info("Announcement created", zap.Uint("id", announcement.ID), zap.String("status", announcement.Status))
	return envelope.SuccessResponse(announcement, "backoffice.announcement.create.success")
}

func (h *BackofficeAnnouncementHandler) CancelAnnouncement(c *gin.Context) envelope.Response {
	id := c.Param("id")

	var announcement notifications_models.Announcement
	if err := h.db.First(&announcement, id).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Announcement not found", core_errors.ErrAnnouncementNotFound)
	}

	// Conditional update: the scheduler may be sending it at this very moment.
	result := h.db.Model(&announcement).
		Where("status = ?", notifications_models.AnnouncementStatusScheduled).
		Update("status", notifications_models.AnnouncementStatusCancelled)
	if result.Error != nil {
		h.logger.Error("Failed to cancel announcement", zap.String("id", id), zap.Error(result.Error))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Error cancelling announcement", core_errors.ErrInternal)
	}
	if result.RowsAffected == 0 {
		return envelope.ErrorResponse(http.StatusConflict, "Announcement is not scheduled", core_errors.ErrAnnouncementNotCancellable)
	}

	return envelope.SuccessResponse(announcement, "backoffice.announcement.cancel.success")
}

// validActionURL accepts nothing, an in-app path ("/billing") or an https
// link. "//host" is rejected: browsers treat it as an external link.
func validActionURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return true
	}
	if strings.HasPrefix(url, "https://") {
		return len(url) > len("https://")
	}
	return strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "//")
}
