package backoffice_handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	backoffice_models "pengi-med-saas/features/backoffice/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	errAdminLocked = errors.New("admin locked")
	errAdminWrong  = errors.New("wrong password or code")
)

// adminLockout is the clinic's account lockout (user_handlers.accountLockout)
// for backoffice admins: AUTH_LOCKOUT_MAX_ATTEMPTS consecutive wrong
// second-factor codes or current-password checks lock the admin for
// AUTH_LOCKOUT_MINUTES. The password step of the login keeps its in-memory
// per-username limiter; this counter lives on the row, so a known password
// never buys fresh code guesses, across instances and restarts.
type adminLockout struct {
	maxAttempts int // <= 0 disables it
	duration    time.Duration
}

func newAdminLockout() adminLockout {
	cfg := config.Get()
	return adminLockout{
		maxAttempts: cfg.AuthLockoutMaxAttempts,
		duration:    time.Duration(cfg.AuthLockoutMinutes) * time.Minute,
	}
}

// check runs verify for admin id under the lockout. Each attempt is counted
// before verify runs, in one conditional UPDATE that skips locked rows, so at
// most maxAttempts guesses are checked per lock window. A correct second
// factor (clear=true) resets the counter; a correct password (clear=false)
// only refunds its own attempt, so a known password never resets earlier
// wrong guesses but repeated legitimate checks never lock the admin.
// Returns nil, errAdminLocked (verify not called), errAdminWrong, or a
// database error.
func (l adminLockout) check(db *gorm.DB, logger *zap.Logger, id uint, clear bool, verify func() (bool, error)) error {
	if l.maxAttempts <= 0 {
		ok, err := verify()
		if err != nil {
			return err
		}
		if !ok {
			return errAdminWrong
		}
		return nil
	}

	now := time.Now().UTC()
	res := db.Model(&backoffice_models.BackofficeUser{}).
		Where("id = ? AND (locked_until IS NULL OR locked_until <= ?)", id, now).
		UpdateColumns(map[string]any{
			"failed_login_attempts": gorm.Expr("CASE WHEN failed_login_attempts + 1 >= ? THEN 0 ELSE failed_login_attempts + 1 END", l.maxAttempts),
			"locked_until":          gorm.Expr("CASE WHEN failed_login_attempts + 1 >= ? THEN ? ELSE locked_until END", l.maxAttempts, now.Add(l.duration)),
		})
	if res.Error != nil {
		return fmt.Errorf("count admin attempt: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errAdminLocked
	}

	ok, err := verify()
	if err != nil {
		return err
	}
	if ok {
		if clear {
			if err := clearAdminLockout(db, id); err != nil {
				logger.Error("Failed to reset backoffice failed attempts", zap.Uint("user_id", id), zap.Error(err))
			}
		} else if err := refundAdminAttempt(db, id, now); err != nil {
			logger.Error("Failed to refund backoffice attempt", zap.Uint("user_id", id), zap.Error(err))
		}
		return nil
	}
	logger.Warn("Backoffice: wrong password or second-factor code", zap.Uint("user_id", id))
	return errAdminWrong
}

// refundAdminAttempt gives back the attempt that check counted for a correct
// password, so legitimate re-authentications (set up 2FA, add a passkey)
// never add up to a lockout. Wrong guesses stay counted, and an attempt that
// already tripped the lock is not refunded.
func refundAdminAttempt(db *gorm.DB, id uint, now time.Time) error {
	return db.Model(&backoffice_models.BackofficeUser{}).
		Where("id = ? AND failed_login_attempts > 0 AND (locked_until IS NULL OR locked_until <= ?)", id, now).
		UpdateColumn("failed_login_attempts", gorm.Expr("failed_login_attempts - 1")).Error
}

func clearAdminLockout(db *gorm.DB, id uint) error {
	return db.Model(&backoffice_models.BackofficeUser{}).Where("id = ?", id).UpdateColumns(map[string]any{
		"failed_login_attempts": 0,
		"locked_until":          nil,
	}).Error
}

func accountLockedResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusTooManyRequests, "auth.account_locked", core_errors.ErrAuthAccountLocked)
}

func internalResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
}

// lockoutResponse maps a check error to the answer: wrong is the response for
// errAdminWrong (differs between login and signed-in endpoints).
func (h *BackofficeUserHandler) lockoutResponse(err error, userID uint, wrong envelope.Response) *envelope.Response {
	var resp envelope.Response
	switch {
	case err == nil:
		return nil
	case errors.Is(err, errAdminLocked):
		resp = accountLockedResponse()
	case errors.Is(err, errAdminWrong):
		resp = wrong
	default:
		h.logger.Error("Backoffice: failed to check password or code", zap.Uint("user_id", userID), zap.Error(err))
		resp = internalResponse()
	}
	return &resp
}
