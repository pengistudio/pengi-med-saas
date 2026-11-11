package cash_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	cash_dto "pengi-med-saas/features/cash/dto"
	cash_models "pengi-med-saas/features/cash/models"
	cash_services "pengi-med-saas/features/cash/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCurrentSession returns the open session with its detail, or null when
// the desk is closed.
func (h *CashHandler) GetCurrentSession(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	var session cash_models.CashSession
	if err := db.Where("status = ?", cash_models.SessionOpen).Limit(1).Find(&session).Error; err != nil {
		return h.internal("failed to fetch current cash session", err)
	}
	if session.ID == 0 {
		return envelope.SuccessResponse(nil, "cash.session.current.fetch.success")
	}
	detail, err := h.detail(db, session)
	if err != nil {
		return h.internal("failed to build cash session detail", err)
	}
	return envelope.SuccessResponse(detail, "cash.session.current.fetch.success")
}

// OpenSession opens the desk with an opening float. Only one session per
// tenant may be open (checked here and by a partial unique index).
func (h *CashHandler) OpenSession(c *gin.Context) envelope.Response {
	var req cash_dto.OpenSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalid("cash.session.error.invalid_payload")
	}
	float := 0.0
	if req.OpeningFloat != nil {
		float = cash_models.Round2(*req.OpeningFloat)
	}
	if float < 0 {
		return invalid("cash.session.error.invalid_float")
	}

	db := tenantdb.For(c, h.db)
	userID, userName := currentUser(c)
	session := cash_models.CashSession{
		TenantID:     tenantdb.TenantID(c),
		Status:       cash_models.SessionOpen,
		OpenedBy:     userID,
		OpenedByName: userName,
		OpenedAt:     time.Now().UTC(),
		OpeningFloat: float,
	}
	alreadyOpen := fail(http.StatusConflict, "cash.session.error.already_open", core_errors.ErrCashSessionAlreadyOpen)
	err := db.Transaction(func(tx *gorm.DB) error {
		if _, found, err := openSession(tx); err != nil {
			return err
		} else if found {
			return alreadyOpen
		}
		return tx.Create(&session).Error
	})
	if resp, ok := asResponse(err); ok {
		return resp
	}
	if err != nil {
		// A concurrent open won the race and hit the unique index.
		if _, found, _ := openSession(db); found {
			resp, _ := asResponse(alreadyOpen)
			return resp
		}
		return h.internal("failed to open cash session", err)
	}
	detail, err := h.detail(db, session)
	if err != nil {
		return h.internal("failed to build cash session detail", err)
	}
	return envelope.SuccessResponse(detail, "cash.session.open.success")
}

// CloseSession is the blind close: the counted cash is compared with the
// expected cash (opening float + cash payments − cash refunds + in − out). A
// difference needs a note; the close then stores both and the session
// becomes immutable. Only the response reveals the expected amount.
func (h *CashHandler) CloseSession(c *gin.Context) envelope.Response {
	id, ok := paramID(c)
	if !ok {
		return invalid("cash.session.error.invalid_id")
	}
	var req cash_dto.CloseSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalid("cash.session.error.invalid_payload")
	}
	counted := cash_models.Round2(*req.CountedCash)
	if counted < 0 {
		return invalid("cash.session.error.invalid_counted")
	}
	note := strings.TrimSpace(req.Note)

	db := tenantdb.For(c, h.db)
	userID, userName := currentUser(c)
	var session cash_models.CashSession
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(forUpdate).First(&session, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fail(http.StatusNotFound, "cash.session.error.not_found", core_errors.ErrCashSessionNotFound)
			}
			return err
		}
		if session.Status != cash_models.SessionOpen {
			return fail(http.StatusConflict, "cash.session.error.closed", core_errors.ErrCashSessionClosed)
		}
		totals, err := cash_services.ComputeTotals(tx, session)
		if err != nil {
			return err
		}
		expected := totals.Expected()
		difference := cash_models.Round2(counted - expected)
		if cash_models.Cents(difference) != 0 && note == "" {
			return fail(http.StatusUnprocessableEntity, "cash.session.error.note_required", core_errors.ErrCashCloseNoteRequired)
		}
		now := time.Now().UTC()
		session.Status = cash_models.SessionClosed
		session.ClosedBy = &userID
		session.ClosedByName = userName
		session.ClosedAt = &now
		session.CountedCash = &counted
		session.ExpectedCash = &expected
		session.Difference = &difference
		session.CloseNote = note
		return tx.Save(&session).Error
	})
	if resp, ok := asResponse(err); ok {
		return resp
	}
	if err != nil {
		return h.internal("failed to close cash session", err)
	}
	detail, err := h.detail(db, session)
	if err != nil {
		return h.internal("failed to build cash session detail", err)
	}
	return envelope.SuccessResponse(detail, "cash.session.close.success")
}

// GetSessions is the session history, newest first.
func (h *CashHandler) GetSessions(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	query := db.Model(&cash_models.CashSession{})
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return h.internal("failed to count cash sessions", err)
	}
	sessions := []cash_models.CashSession{}
	if err := query.Order("opened_at DESC, id DESC").Limit(limit).Offset((page - 1) * limit).Find(&sessions).Error; err != nil {
		return h.internal("failed to fetch cash sessions", err)
	}
	return envelope.PagedSuccessResponse(sessions, int(total), page, limit, "cash.sessions.fetch.success")
}

// GetSession returns one session with its payments, movements and totals.
func (h *CashHandler) GetSession(c *gin.Context) envelope.Response {
	id, ok := paramID(c)
	if !ok {
		return invalid("cash.session.error.invalid_id")
	}
	db := tenantdb.For(c, h.db)
	var session cash_models.CashSession
	if err := db.First(&session, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return envelope.ErrorResponse(http.StatusNotFound, "cash.session.error.not_found", core_errors.ErrCashSessionNotFound)
		}
		return h.internal("failed to fetch cash session", err)
	}
	detail, err := h.detail(db, session)
	if err != nil {
		return h.internal("failed to build cash session detail", err)
	}
	return envelope.SuccessResponse(detail, "cash.session.fetch.success")
}
