package auth_middleware

import (
	"sync"
	"time"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/tenantdb"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// activeUserTTL is how long a "this user still exists" answer is reused before
// AuthMiddleware asks the database again. Deleting a user or changing their
// password in this process forgets it at once (ForgetUser); another API
// instance notices within the TTL.
const activeUserTTL = 30 * time.Second

// tokenStatus is what the user check says about a verified token.
type tokenStatus int

const (
	tokenOK      tokenStatus = iota
	tokenNoUser              // the user was deleted
	tokenRevoked             // issued before the user's last password change
)

// userState is a cached lookup of the user an access token names.
type userState struct {
	at                time.Time  // when it was read
	passwordChangedAt *time.Time // nil: never changed
}

// activeUsers answers whether the user an access token names still exists
// (users have no active flag: deleting is soft-deleting) and whether the token
// predates their last password change. A signed token outlives both by up to
// AUTH_EXP minutes otherwise.
type activeUsers struct {
	mu   sync.RWMutex
	db   *gorm.DB
	seen map[uint]userState
}

var users = &activeUsers{seen: map[uint]userState{}}

// UseUserDB turns on the user check in AuthMiddleware. RegisterUserRoutes
// calls it once with the API's connection; without it (unit tests that build
// a bare router) the check is skipped.
func UseUserDB(db *gorm.DB) {
	users.mu.Lock()
	defer users.mu.Unlock()
	users.db = db
	users.seen = map[uint]userState{}
}

// ForgetUser drops a cached answer, so a user deleted (or whose password
// changed) in this process is checked again on its next request instead of
// after activeUserTTL.
func ForgetUser(userID uint) {
	users.mu.Lock()
	defer users.mu.Unlock()
	delete(users.seen, userID)
}

// check validates a verified token against its user: deleted user or token
// issued before the last password change.
func (u *activeUsers) check(userID uint, claims jwt.MapClaims) tokenStatus {
	state, found := u.lookup(userID)
	if !found {
		return tokenNoUser
	}
	if state.passwordChangedAt != nil && auth.IssuedBefore(claims, *state.passwordChangedAt) {
		return tokenRevoked
	}
	return tokenOK
}

// lookup returns the user's cached or fresh state. Only "not found" reports
// false: on a database error the token, already verified, is let through
// (the request's own queries will fail anyway) rather than signing everyone
// out on a blip.
func (u *activeUsers) lookup(userID uint) (userState, bool) {
	u.mu.RLock()
	db := u.db
	state, cached := u.seen[userID]
	u.mu.RUnlock()
	if db == nil || (cached && time.Since(state.at) < activeUserTTL) {
		return state, true
	}

	// users is a global table (no tenant_id): a primary-key lookup that skips
	// soft-deleted rows. By table name: core/audit imports this package, so it
	// can't import user_models (which reaches core/audit through core/database).
	var rows []struct {
		ID                uint
		PasswordChangedAt *time.Time
	}
	err := tenantdb.System(db).Table("users").
		Select("id", "password_changed_at").
		Where("id = ? AND deleted_at IS NULL", userID).
		Limit(1).Scan(&rows).Error
	if err != nil {
		if logger.Log != nil {
			logger.Log.Warn("auth: user check failed", zap.Uint("user_id", userID), zap.Error(err))
		}
		return userState{}, true
	}
	if len(rows) == 0 {
		return userState{}, false
	}

	state = userState{at: time.Now(), passwordChangedAt: rows[0].PasswordChangedAt}
	u.mu.Lock()
	u.seen[userID] = state
	u.mu.Unlock()
	return state, true
}
