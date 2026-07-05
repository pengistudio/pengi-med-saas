package clinical_models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MedicalRecordDraft struct {
	gorm.Model
	TenantID  uint           `json:"tenant_id" gorm:"uniqueIndex:idx_draft_tenant_user_patient"`
	UserID    uint           `json:"user_id" gorm:"uniqueIndex:idx_draft_tenant_user_patient"`
	PatientID uint           `json:"patient_id" gorm:"uniqueIndex:idx_draft_tenant_user_patient"`
	Data      datatypes.JSON `json:"data" gorm:"type:jsonb;default:'{}'"`
}

func (MedicalRecordDraft) IsAuditable() bool { return false }
