package audit

import (
	"reflect"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuditLog represents a record of a change made to an Auditable entity
type AuditLog struct {
	ID         uint           `gorm:"primaryKey"`
	TenantID   uint           `json:"tenant_id" gorm:"index"`
	UserID     uint           `json:"user_id" gorm:"index"`
	Action     string         `json:"action"`                   // "CREATE", "UPDATE", "DELETE"
	EntityType string         `json:"entity_type" gorm:"index"` // "Patient", "MedicalRecord", etc.
	EntityID   uint           `json:"entity_id" gorm:"index"`
	OldValues  datatypes.JSON `json:"old_values"`
	NewValues  datatypes.JSON `json:"new_values"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Auditable is an interface that models must implement to be tracked by the audit trail
type Auditable interface {
	IsAuditable() bool
}

// AuditableID allows models to override how their primary key is extracted if not standard "ID"
type AuditableID interface {
	GetAuditID() uint
}

func getEntityID(db *gorm.DB) uint {
	if i, ok := db.Statement.Dest.(AuditableID); ok {
		return i.GetAuditID()
	}

	val := db.Statement.ReflectValue
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		idField := val.FieldByName("ID")
		if idField.IsValid() {
			switch idField.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return uint(idField.Uint())
			}
		}
	}
	return 0
}
