package backoffice_models

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/auth"
	"time"

	"gorm.io/gorm"
)

type BackofficeUser struct {
	gorm.Model
	Name         string `json:"name"`
	UserName     string `json:"user_name"`
	Password     string `json:"-"`
	RefreshToken string `json:"-"`
	// RefreshFamily is the sid of the login RefreshToken descends from: a
	// rotated-out token of the same family presented again is reuse (the
	// session is revoked); one of an older login is merely stale.
	RefreshFamily string `gorm:"size:64;not null;default:''" json:"-"`

	// Second factor, mandatory for admins (TOTP and/or passkeys). The secret
	// is sealed (secretbox, AAD bound to the admin); while TwoFactorEnabled is
	// false a stored secret is a pending setup. TwoFactorLastStep is the last
	// accepted TOTP step (each code works once).
	TwoFactorEnabled   bool       `gorm:"not null;default:false" json:"two_factor_enabled"`
	TwoFactorSecret    []byte     `json:"-"`
	TwoFactorLastStep  int64      `gorm:"not null;default:0" json:"-"`
	TwoFactorEnabledAt *time.Time `json:"-"`

	// FailedLoginAttempts counts consecutive wrong second-factor codes and
	// current-password checks; reaching AUTH_LOCKOUT_MAX_ATTEMPTS sets
	// LockedUntil (AUTH_LOCKOUT_MINUTES).
	FailedLoginAttempts int        `gorm:"not null;default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`

	// PasskeyCount is filled by the user list/detail handlers.
	PasskeyCount int64 `gorm:"-" json:"passkey_count"`
}

func (u *BackofficeUser) Save(db *gorm.DB) error {
	hashPassword, err := auth.HashPassword(u.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = hashPassword

	err = db.Create(&u).Error
	if err != nil {
		return fmt.Errorf("failed to create user record: %w", err)
	}

	return db.Save(&u).Error
}

func (u *BackofficeUser) ValidateCredentials(db *gorm.DB) error {
	var foundUser BackofficeUser
	err := db.Where("user_name = ?", u.UserName).First(&foundUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("incorrect username or password")
		}
		return fmt.Errorf("failed to retrieve user record: %w", err)
	}

	isPassword := auth.CompareHashAndPassword(foundUser.Password, u.Password)
	if !isPassword {
		return errors.New("incorrect username or password")
	}

	*u = foundUser
	return nil
}

func (u *BackofficeUser) UpdateRefreshToken(db *gorm.DB, refresh string) error {
	clean := db.Session(&gorm.Session{NewDB: true})
	if err := clean.
		Model(&BackofficeUser{}).
		Where("id = ? AND user_name = ?", u.ID, u.UserName).
		Select("refresh_token", "updated_at").
		Updates(map[string]any{
			"refresh_token": refresh,
			"updated_at":    db.NowFunc(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}
	return nil
}
