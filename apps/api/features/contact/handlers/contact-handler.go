package contact_handlers

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/mailer"
	contact_dto "pengi-med-saas/features/contact/dto"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const rateLimitWindow = time.Hour

func maxRequests() int {
	if v, err := strconv.Atoi(os.Getenv("CONTACT_RATE_LIMIT")); err == nil && v > 0 {
		return v
	}
	return 3
}

type ContactHandler struct {
	logger *zap.Logger
	mailer *mailer.Mailer
	mu     sync.Mutex
	limits map[string][]time.Time
}

func NewContactHandler(logger *zap.Logger) *ContactHandler {
	h := &ContactHandler{
		logger: logger,
		mailer: mailer.NewMailer(),
		limits: make(map[string][]time.Time),
	}
	go h.cleanupLoop()
	return h
}

func (h *ContactHandler) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-rateLimitWindow)
		h.mu.Lock()
		for ip, timestamps := range h.limits {
			valid := timestamps[:0]
			for _, t := range timestamps {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(h.limits, ip)
			} else {
				h.limits[ip] = valid
			}
		}
		h.mu.Unlock()
	}
}

func (h *ContactHandler) isRateLimited(ip string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	cutoff := time.Now().Add(-rateLimitWindow)
	timestamps := h.limits[ip]
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	h.limits[ip] = valid

	if len(h.limits[ip]) >= maxRequests() {
		return true
	}
	h.limits[ip] = append(h.limits[ip], time.Now())
	return false
}

func (h *ContactHandler) SendContact(c *gin.Context) envelope.Response {
	if h.isRateLimited(c.ClientIP()) {
		return envelope.ErrorResponse(http.StatusTooManyRequests, "contact.rate_limit", core_errors.ErrRateLimitExceeded)
	}

	var dto contact_dto.ContactDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, err.Error(), core_errors.ErrInvalidRequest)
	}

	recipient := os.Getenv("CONTACT_EMAIL")
	if recipient == "" {
		recipient = "pengiservicios@gmail.com"
	}

	if err := h.mailer.SendContactMessage(recipient, dto.Name, dto.Email, dto.Message); err != nil {
		h.logger.Error("failed to send contact email", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "contact.send_error", core_errors.ErrInternal)
	}

	h.logger.Info("contact message sent", zap.String("from", dto.Email), zap.String("ip", c.ClientIP()))
	return envelope.SuccessResponse(nil, "contact.send.success")
}
