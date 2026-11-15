package auth_middleware

import (
	"sync"
	"time"

	"pengi-med-saas/core/logger"
	"pengi-med-saas/core/tenantdb"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// activeUserTTL is how long a "this user still exists" answer is reused before
// AuthMiddleware asks the database again. Deleting a user in this process
// forgets it at once (ForgetUser); another API instance notices within the TTL.
const activeUserTTL = 30 * time.Second

// activeUsers answers whether the user an access token names still exists
// (users have no active flag: deleting is soft-deleting). A signed token
// outlives its user by up to AUTH_EXP minutes otherwise.
type activeUsers struct {
	mu   sync.RWMutex
	db   *gorm.DB
	seen map[uint]time.Time // user ID -> when it was last found
}

var users = &activeUsers{seen: map[uint]time.Time{}}

// UseUserDB turns on the existence check in AuthMiddleware. RegisterUserRoutes
// calls it once with the API's connection; without it (unit tests that build
// a bare router) the check is skipped.
func UseUserDB(db *gorm.DB) {
	users.mu.Lock()
	defer users.mu.Unlock()
	users.db = db
	users.seen = map[uint]time.Time{}
}

// ForgetUser drops a cached answer, so a user deleted in this process is
// rejected on its next request instead of after activeUserTTL.
func ForgetUser(userID uint) {
	users.mu.Lock()
	defer users.mu.Unlock()
	delete(users.seen, userID)
}

// exists reports whether userID is a live (not deleted) user. Only "not found"
// rejects: on a database error the token, already verified, is let through
// (the request's own queries will fail anyway) rather than signing everyone
// out on a blip.
func (u *activeUsers) exists(userID uint) bool {
	u.mu.RLock()
	db := u.db
	at, cached := u.seen[userID]
	u.mu.RUnlock()
	if db == nil || (cached && time.Since(at) < activeUserTTL) {
		return true
	}

	// users is a global table (no tenant_id): a primary-key lookup that skips
	// soft-deleted rows. By table name: core/audit imports this package, so it
	// can't import user_models (which reaches core/audit through core/database).
	var ids []uint
	err := tenantdb.System(db).Table("users").
		Where("id = ? AND deleted_at IS NULL", userID).
		Limit(1).Pluck("id", &ids).Error
	if err == nil && len(ids) == 0 {
		return false
	}
	if err != nil {
		if logger.Log != nil {
			logger.Log.Warn("auth: user existence check failed", zap.Uint("user_id", userID), zap.Error(err))
		}
		return true
	}

	u.mu.Lock()
	u.seen[userID] = time.Now()
	u.mu.Unlock()
	return true
}
