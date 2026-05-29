package settings_models

import "gorm.io/gorm"

type SystemSetting struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex;not null"`
	Value string `gorm:"not null"`
}
