package backoffice_models

import (
	"errors"
	"fmt"
	"pengi-med-saas/core/auth"

	"gorm.io/gorm"
)

type BackofficeUser struct {
	gorm.Model
	Name         string `json:"name"`
	UserName     string `json:"user_name"`
	Password     string `json:"-"`
	RefreshToken string `json:"-"`
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
