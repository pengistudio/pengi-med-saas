// Package doctor_services holds doctor logic shared by handlers and other
// domains.
package doctor_services

import (
	"fmt"

	"gorm.io/gorm"
)

// ReferencingTables lists every table that points to a doctor through a
// doctor_id column. A doctor referenced from any of them can only be
// deactivated, never deleted. Add a table here when it gains doctor_id.
// Tables that don't have the column yet are skipped, so the list can name
// them ahead of the migration that adds it.
var ReferencingTables = []string{
	"patients",
	"appointments",
	"medical_records",
	"prescriptions",
	"medical_certificates",
	"medical_reports",
	"exam_orders",
}

// IsReferenced reports whether any row of ReferencingTables (soft-deleted
// rows included) points to doctorID. Doctor IDs are global, so no tenant
// filter is needed.
func IsReferenced(db *gorm.DB, doctorID uint) (bool, error) {
	for _, table := range ReferencingTables {
		if !db.Migrator().HasColumn(table, "doctor_id") {
			continue
		}
		var n int64
		if err := db.Table(table).Where("doctor_id = ?", doctorID).Limit(1).Count(&n).Error; err != nil {
			return false, fmt.Errorf("check %s.doctor_id: %w", table, err)
		}
		if n > 0 {
			return true, nil
		}
	}
	return false, nil
}
