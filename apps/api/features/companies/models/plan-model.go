package company_models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Plan struct {
	gorm.Model
	Name       string            `gorm:"not null" json:"name"`
	Code       string            `gorm:"not null;unique" json:"code"`
	Tier       int               `gorm:"not null;default:1" json:"tier"`
	CanRenew   bool              `gorm:"not null;default:true" json:"can_renew"`
	Features   []Feature         `gorm:"many2many:plan_features;"`
	Price      float64           `gorm:"not null" json:"price"`
	Properties datatypes.JSONMap `gorm:"type:jsonb;default:'{}'"`
	Pricings   []PlanPricing     `gorm:"foreignKey:PlanID" json:"pricings"`
	// StorageQuotaMB is how much space the tenant's patient attachments may
	// take, in MB (1 MB = 1024*1024 bytes). 0 means no attachments allowed
	// (plans without the clinical module).
	StorageQuotaMB int64 `gorm:"not null;default:0" json:"storage_quota_mb"`
}

// StorageQuotaBytes is the attachment quota in bytes.
func (p *Plan) StorageQuotaBytes() int64 {
	return p.StorageQuotaMB << 20
}

func (p *Plan) Save(db *gorm.DB) error {
	return db.Save(p).Error
}
