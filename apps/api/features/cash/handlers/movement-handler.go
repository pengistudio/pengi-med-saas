package cash_handlers

import (
	"net/http"
	"strings"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	cash_dto "pengi-med-saas/features/cash/dto"
	cash_models "pengi-med-saas/features/cash/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// movementTypes are the directions each category allows: expenses and
// withdrawals take cash out, deposits put it in, adjustments go either way.
var movementTypes = map[string][]string{
	cash_models.CategoryExpense:    {cash_models.MovementOut},
	cash_models.CategoryWithdrawal: {cash_models.MovementOut},
	cash_models.CategoryDeposit:    {cash_models.MovementIn},
	cash_models.CategoryAdjustment: {cash_models.MovementIn, cash_models.MovementOut},
}

// movementPermission is what each direction requires: putting cash in is part
// of running the session, taking it out is as sensitive as voiding.
var movementPermission = map[string]string{
	cash_models.MovementIn:  "MANAGE_CASH_SESSION",
	cash_models.MovementOut: "VOID_PAYMENT",
}

func validMovement(typ, category string) bool {
	for _, t := range movementTypes[category] {
		if t == typ {
			return true
		}
	}
	return false
}

// CreateMovement records cash put into or taken out of the drawer of the open
// session, with a required reason.
func (h *CashHandler) CreateMovement(c *gin.Context) envelope.Response {
	var req cash_dto.CreateMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalid("cash.movement.error.invalid_payload")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return invalid("cash.movement.error.reason_required")
	}
	if !validMovement(req.Type, req.Category) {
		return invalid("cash.movement.error.invalid_type")
	}
	if cash_models.Cents(req.Amount) <= 0 {
		return invalid("cash.movement.error.invalid_amount")
	}
	if !h.can(c, movementPermission[req.Type]) {
		return envelope.ErrorResponse(http.StatusForbidden, "permission.error.insufficient", core_errors.ErrPermissionDenied)
	}

	db := tenantdb.For(c, h.db)
	userID, _ := currentUser(c)
	var movement cash_models.CashMovement
	err := db.Transaction(func(tx *gorm.DB) error {
		session, found, err := openSession(tx)
		if err != nil {
			return err
		}
		if !found {
			return fail(http.StatusConflict, "cash.session.error.no_open", core_errors.ErrCashNoOpenSession)
		}
		movement = cash_models.CashMovement{
			TenantID:      session.TenantID,
			CashSessionID: session.ID,
			Type:          req.Type,
			Category:      req.Category,
			Amount:        cash_models.Round2(req.Amount),
			Reason:        reason,
			CreatedBy:     userID,
		}
		return tx.Create(&movement).Error
	})
	if resp, ok := asResponse(err); ok {
		return resp
	}
	if err != nil {
		return h.internal("failed to create cash movement", err)
	}
	return envelope.SuccessResponse(movement, "cash.movement.create.success")
}
