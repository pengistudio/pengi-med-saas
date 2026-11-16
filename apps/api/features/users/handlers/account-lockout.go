package user_handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/config"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	user_models "pengi-med-saas/features/users/models"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

var (
	// errAccountLocked is returned by accountLockout.check while the account
	// is locked; the password was not checked.
	errAccountLocked = errors.New("account locked")
	// errWrongPassword is returned by accountLockout.check when verify fails.
	errWrongPassword = errors.New("wrong password")
)

// accountLockout locks an account after maxAttempts consecutive wrong
// passwords, for every place that checks one (login, change password, change
// email), so rotating IPs or holding a stolen session doesn't buy more
// guesses. State lives on the user row, so it holds across API instances and
// restarts.
type accountLockout struct {
	maxAttempts int // <= 0 disables the lockout
	duration    time.Duration
}

func newAccountLockout() accountLockout {
	cfg := config.Get()
	return accountLockout{
		maxAttempts: cfg.AuthLockoutMaxAttempts,
		duration:    time.Duration(cfg.AuthLockoutMinutes) * time.Minute,
	}
}

func (l accountLockout) enabled() bool { return l.maxAttempts > 0 }

// unknownUserLimiter allows logins to usernames that don't exist the same
// budget as a real account (maxAttempts per lock window), so they also end
// in the locked answer. Approximate: a token bucket refills gradually where a
// real lock lifts all at once.
func (l accountLockout) unknownUserLimiter() *auth.LoginLimiter {
	if !l.enabled() || l.duration <= 0 {
		return auth.NewDefaultLoginLimiter()
	}
	return auth.NewLoginLimiter(rate.Every(l.duration/time.Duration(l.maxAttempts)), l.maxAttempts)
}

// check runs verify (the password comparison) for userID under the lockout.
// It returns nil when verify succeeds, errAccountLocked when the account is
// locked (verify is not called, so a locked account is no password oracle),
// errWrongPassword when verify fails, or a database error.
//
// Each attempt is counted before verify runs, in one conditional UPDATE that
// skips locked rows: concurrent requests can't all slip past the lock check,
// so at most maxAttempts guesses are ever checked per lock window. The
// attempt that reaches the limit sets locked_until and restarts the counter;
// a correct password clears both, unless the user has two-factor
// authentication: then only a correct second factor (checkSecondFactor)
// clears them, so knowing the password never buys fresh code guesses.
func (l accountLockout) check(db *gorm.DB, logger *zap.Logger, userID uint, verify func() bool) error {
	return l.attempt(db, logger, userID, verify, clearPasswordLockout)
}

// checkSecondFactor is check for a TOTP or recovery code: same counter, and a
// correct code always clears it.
func (l accountLockout) checkSecondFactor(db *gorm.DB, logger *zap.Logger, userID uint, verify func() bool) error {
	return l.attempt(db, logger, userID, verify, clearLockout)
}

func (l accountLockout) attempt(db *gorm.DB, logger *zap.Logger, userID uint, verify func() bool, clear func(*gorm.DB, uint) error) error {
	if !l.enabled() {
		if verify() {
			return nil
		}
		return errWrongPassword
	}

	now := time.Now().UTC()
	res := db.Model(&user_models.User{}).
		Where("id = ? AND (locked_until IS NULL OR locked_until <= ?)", userID, now).
		UpdateColumns(map[string]any{
			"failed_login_attempts": gorm.Expr("CASE WHEN failed_login_attempts + 1 >= ? THEN 0 ELSE failed_login_attempts + 1 END", l.maxAttempts),
			// ELSE keeps the column (NULL or an expired lock) and lets Postgres
			// type the parameter as a timestamp.
			"locked_until": gorm.Expr("CASE WHEN failed_login_attempts + 1 >= ? THEN ? ELSE locked_until END", l.maxAttempts, now.Add(l.duration)),
		})
	if res.Error != nil {
		return fmt.Errorf("count password attempt: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errAccountLocked
	}

	if verify() {
		if err := clear(db, userID); err != nil {
			logger.Error("Failed to reset failed password attempts", zap.Uint("user_id", userID), zap.Error(err))
		}
		return nil
	}

	var row struct{ LockedUntil *time.Time }
	if err := db.Model(&user_models.User{}).Select("locked_until").Where("id = ?", userID).Scan(&row).Error; err == nil &&
		row.LockedUntil != nil && row.LockedUntil.After(now) {
		logger.Warn("Account locked after repeated failed password checks",
			zap.Uint("user_id", userID), zap.Time("locked_until", *row.LockedUntil))
	}
	return errWrongPassword
}

// clearLockout forgets userID's failed attempts and lifts any lock. Called
// after a correct second factor and by the backoffice 2FA reset.
func clearLockout(db *gorm.DB, userID uint) error {
	return db.Model(&user_models.User{}).Where("id = ?", userID).UpdateColumns(map[string]any{
		"failed_login_attempts": 0,
		"locked_until":          nil,
	}).Error
}

// clearPasswordLockout is clearLockout after a password proof (correct
// password, password change or reset): it does nothing for a user with
// two-factor authentication, whose counter only a correct code clears.
func clearPasswordLockout(db *gorm.DB, userID uint) error {
	return db.Model(&user_models.User{}).Where("id = ? AND two_factor_enabled = ?", userID, false).UpdateColumns(map[string]any{
		"failed_login_attempts": 0,
		"locked_until":          nil,
	}).Error
}

// accountLockedResponse is the answer while an account is locked.
func accountLockedResponse() envelope.Response {
	return envelope.ErrorResponse(http.StatusTooManyRequests, "auth.account_locked", core_errors.ErrAuthAccountLocked)
}
