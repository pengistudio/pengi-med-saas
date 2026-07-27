package company_models

import (
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

type Company struct {
	gorm.Model
	LegalName     string                    `gorm:"not null" json:"legal_name"`
	TradeName     string                    `gorm:"not null" json:"trade_name"`
	PlanCode      string                    `gorm:"not null" json:"plan_code"`
	Subscriptions []Subscription            `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	TenantID      uint                      `gorm:"not null" json:"tenant_id"`
	Tenant        tenant_models.Tenant      `gorm:"foreignKey:TenantID;references:ID" json:"tenant"`
	Environments  []user_models.Environment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	OwnerUserID   uint                      `gorm:"index" json:"owner_user_id"`
}
