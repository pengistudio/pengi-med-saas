package user_models

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/auth"
	permission_models "pengi-med-saas/features/permissions/models"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UserName          string        `json:"user_name"`
	Password          string        `json:"password"`
	Email             string        `json:"email"`
	Environments      []Environment `json:"environments"`
	RefreshToken      string        `json:"refresh_token"`
	EmailVerified     bool          `gorm:"not null;default:false" json:"email_verified"`
	EmailVerifiedAt   *time.Time    `json:"email_verified_at"`
	MaxOwnedCompanies int           `gorm:"not null;default:3" json:"max_owned_companies"`
	// PasswordChangedAt is set on every password change or reset, and when
	// two-factor authentication is turned on or reset (every session must
	// pass the new second factor); access tokens issued before it are
	// rejected by AuthMiddleware.
	PasswordChangedAt *time.Time `json:"-"`
	// FailedLoginAttempts counts consecutive wrong passwords and second-factor
	// codes (login, change password, change email, 2FA); reaching the limit
	// sets LockedUntil. See user_handlers.accountLockout.
	FailedLoginAttempts int        `gorm:"not null;default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`

	// Two-factor authentication (TOTP), optional per user. TwoFactorSecret is
	// the TOTP secret sealed with core/secretbox (user ID as AAD): pending
	// while TwoFactorEnabled is false (setup started, no code confirmed yet),
	// active once it is true. TwoFactorLastStep is the last accepted TOTP
	// time step: codes at or before it are refused (replay).
	TwoFactorEnabled   bool       `gorm:"not null;default:false" json:"-"`
	TwoFactorSecret    []byte     `json:"-"`
	TwoFactorLastStep  int64      `gorm:"not null;default:0" json:"-"`
	TwoFactorEnabledAt *time.Time `json:"-"`
}

// RecoveryCode is a single-use second factor for when the authenticator is
// lost. Only the SHA-256 of the code is stored; the code is shown once.
type RecoveryCode struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UserID    uint   `gorm:"not null;index"`
	CodeHash  string `gorm:"size:64;not null;index"`
	UsedAt    *time.Time
}

func (RecoveryCode) TableName() string { return "user_recovery_codes" }

type Environment struct {
	gorm.Model
	UserID    uint   `json:"user_id"`
	Name      string `json:"name"`
	RoleID    uint
	Role      Role `json:"role"`
	CompanyID uint `json:"company_id"`
}

type Role struct {
	gorm.Model
	Role        string                         `json:"role"`
	Permissions []permission_models.Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}

func (u *User) Save(db *gorm.DB) error {
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

func (u *User) ValidateCredentials(db *gorm.DB) error {
	var foundUser User
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

func (u *User) Update(db *gorm.DB, dataToUpdate map[string]interface{}) error {
	clean := db.Session(&gorm.Session{NewDB: true}) // limpia scopes/joins previos
	if err := clean.Model(&u).Updates(dataToUpdate).Error; err != nil {
		return fmt.Errorf("failed to update user record: %w", err)
	}
	return nil
}

func (u *User) UpdateRefreshToken(db *gorm.DB, refresh string) error {
	clean := db.Session(&gorm.Session{NewDB: true}) // limpia scopes/joins previos
	if err := clean.
		Model(&User{}). // no uses Model(u) para no duplicar condiciones
		Where("id = ? AND user_name = ?", u.ID, u.UserName).
		Select("refresh_token", "updated_at").
		Updates(map[string]any{
			"refresh_token": refresh,
			"updated_at":    time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}
	return nil
}
