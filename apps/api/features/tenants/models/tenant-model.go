package tenant_models

import "gorm.io/gorm"

type Tenant struct {
	gorm.Model
	Name string `gorm:"not null" json:"name"`
	Slug string `gorm:"not null;unique" json:"slug"`
}

func NewTenant(name string) *Tenant {
	return &Tenant{
		Name: name,
	}
}

func (t *Tenant) Save(db *gorm.DB) error {
	return db.Save(t).Error
}
