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
	UserName     string        `json:"user_name"`
	Password     string        `json:"password"`
	Email        string        `json:"email"`
	Environments []Environment `json:"environments"`
	RefreshToken string        `json:"refresh_token"`
}

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
