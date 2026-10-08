package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	doctor_models "pengi-med-saas/features/doctors/models"

	"gorm.io/gorm"
)

const backfillSingleDoctorID = "DB20261008_3"

// singleDoctorTables have tenant_id and doctor_id: in a tenant with only one
// doctor, every row without doctor belongs to that doctor. Prescriptions have
// no tenant_id and are reached through their medical record.
var singleDoctorTables = []string{
	"patients",
	"appointments",
	"medical_records",
	"medical_certificates",
	"medical_reports",
	"exam_orders",
}

func init() {
	// Runs after DB20261008_2 created the first doctors: tenants with exactly
	// one doctor get it set on their existing records (cabecera included).
	// Tenants with several are left alone (the doctor can't be guessed).
	database.GlobalDBMap[backfillSingleDoctorID] = database.DBExecute{
		ID:      backfillSingleDoctorID,
		Execute: backfillSingleDoctor,
	}
}

func backfillSingleDoctor(db *gorm.DB) error {
	db = tenantdb.System(db)
	var rows []struct {
		TenantID uint
		DoctorID uint
	}
	if err := db.Model(&doctor_models.Doctor{}).
		Select("tenant_id, MIN(id) AS doctor_id").
		Group("tenant_id").Having("COUNT(*) = 1").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("find single-doctor tenants: %w", err)
	}
	var updated int64
	for _, r := range rows {
		for _, table := range singleDoctorTables {
			res := db.Table(table).Where("tenant_id = ? AND doctor_id IS NULL", r.TenantID).Update("doctor_id", r.DoctorID)
			if res.Error != nil {
				return fmt.Errorf("tenant %d: backfill %s: %w", r.TenantID, table, res.Error)
			}
			updated += res.RowsAffected
		}
		res := db.Table("prescriptions").
			Where("doctor_id IS NULL AND id IN (?)",
				db.Table("medical_records").Select("prescription_id").Where("tenant_id = ? AND prescription_id IS NOT NULL", r.TenantID)).
			Update("doctor_id", r.DoctorID)
		if res.Error != nil {
			return fmt.Errorf("tenant %d: backfill prescriptions: %w", r.TenantID, res.Error)
		}
		updated += res.RowsAffected
	}
	fmt.Printf("✅ Set the only doctor on %d rows of %d single-doctor tenants.\n", updated, len(rows))
	return nil
}
