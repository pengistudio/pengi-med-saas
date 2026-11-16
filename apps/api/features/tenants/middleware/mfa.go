package tenant_middleware

import (
	"pengi-med-saas/core/tenantdb"

	"gorm.io/gorm"
)

// HasMFA reports whether the user has a second factor: TOTP two-factor
// turned on, or at least one passkey. One query on global tables (users,
// user_passkeys: no tenant_id), both looked up by indexed keys.
//
// It is read fresh on every call (no cache), so enabling or disabling 2FA and
// adding or deleting a passkey take effect on the next request without any
// invalidation.
func HasMFA(db *gorm.DB, userID uint) (bool, error) {
	var has bool
	err := tenantdb.System(db).Raw(
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = ? AND two_factor_enabled AND deleted_at IS NULL)
		     OR EXISTS (SELECT 1 FROM user_passkeys WHERE user_id = ?)`,
		userID, userID,
	).Scan(&has).Error
	return has, err
}
