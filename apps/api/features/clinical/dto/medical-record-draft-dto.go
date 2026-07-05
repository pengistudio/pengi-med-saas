package dto

import "gorm.io/datatypes"

type SaveMedicalRecordDraftDTO struct {
	Data datatypes.JSON `json:"data" binding:"required"`
}
