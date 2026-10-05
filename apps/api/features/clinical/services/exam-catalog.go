// Package clinical_services holds clinical logic shared by handlers, tenant
// provisioning and migrations.
package clinical_services

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"pengi-med-saas/core/tenantdb"
	clinical_data "pengi-med-saas/features/clinical/data"
	clinical_models "pengi-med-saas/features/clinical/models"
)

// CustomExamSortOrder places exams created by the tenant after the seeded
// ones of their category.
const CustomExamSortOrder = 100000

// SeedExamCatalog loads the initial exam catalog and profiles into a tenant
// that has none (counting soft-deleted rows: a tenant that deleted its whole
// catalog uses "restore defaults" instead). db may be a transaction.
func SeedExamCatalog(db *gorm.DB, tenantID uint) error {
	tdb := tenantdb.ForTenant(db, tenantID)
	var count int64
	if err := tdb.Unscoped().Model(&clinical_models.ExamCatalogItem{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count exam catalog: %w", err)
	}
	if count > 0 {
		return nil
	}
	_, _, err := RestoreExamCatalogDefaults(db, tenantID)
	return err
}

// RestoreExamCatalogDefaults re-activates the seeded exams and profiles of a
// tenant, re-creating the ones it deleted, without touching the exams and
// profiles the tenant created. It returns how many rows it created or
// re-activated. db may be a transaction.
func RestoreExamCatalogDefaults(db *gorm.DB, tenantID uint) (items, profiles int, err error) {
	tdb := tenantdb.ForTenant(db, tenantID)

	var existing []clinical_models.ExamCatalogItem
	if err := tdb.Unscoped().Where("is_default = ?", true).Find(&existing).Error; err != nil {
		return 0, 0, fmt.Errorf("load default exams: %w", err)
	}
	byKey := make(map[string]clinical_models.ExamCatalogItem, len(existing))
	for _, item := range existing {
		byKey[item.SeedKey] = item
	}

	var missing []clinical_models.ExamCatalogItem
	for i, seed := range clinical_data.ExamCatalogSeed {
		key := clinical_data.ExamSeedKey(seed.Name)
		item, ok := byKey[key]
		if !ok {
			missing = append(missing, clinical_models.ExamCatalogItem{
				TenantID:           tenantID,
				Name:               seed.Name,
				Category:           seed.Category,
				Subgroup:           seed.Subgroup,
				DefaultIndications: seed.Indications,
				Active:             true,
				IsDefault:          true,
				SeedKey:            key,
				SortOrder:          i + 1,
			})
			continue
		}
		if item.Active && !item.DeletedAt.Valid {
			continue
		}
		if err := tdb.Unscoped().Model(&clinical_models.ExamCatalogItem{}).Where("id = ?", item.ID).
			Updates(map[string]any{"active": true, "deleted_at": nil}).Error; err != nil {
			return 0, 0, fmt.Errorf("restore exam %q: %w", seed.Name, err)
		}
		items++
	}
	if len(missing) > 0 {
		if err := tdb.CreateInBatches(&missing, 100).Error; err != nil {
			return 0, 0, fmt.Errorf("create default exams: %w", err)
		}
		items += len(missing)
		for _, item := range missing {
			byKey[item.SeedKey] = item
		}
	}

	var existingProfiles []clinical_models.ExamProfile
	if err := tdb.Unscoped().Where("is_default = ?", true).Find(&existingProfiles).Error; err != nil {
		return 0, 0, fmt.Errorf("load default profiles: %w", err)
	}
	profileByKey := make(map[string]clinical_models.ExamProfile, len(existingProfiles))
	for _, p := range existingProfiles {
		profileByKey[p.SeedKey] = p
	}

	for _, seed := range clinical_data.ExamProfileSeed {
		key := clinical_data.ExamSeedKey(seed.Name)
		members := make([]uint, 0, len(seed.Exams))
		for _, name := range seed.Exams {
			item, ok := byKey[clinical_data.ExamSeedKey(name)]
			if !ok {
				return 0, 0, fmt.Errorf("profile %q references unknown exam %q", seed.Name, name)
			}
			members = append(members, item.ID)
		}

		profile, ok := profileByKey[key]
		switch {
		case !ok:
			profile = clinical_models.ExamProfile{TenantID: tenantID, Name: seed.Name, Active: true, IsDefault: true, SeedKey: key}
			if err := tdb.Omit("Items").Create(&profile).Error; err != nil {
				return 0, 0, fmt.Errorf("create profile %q: %w", seed.Name, err)
			}
		case profile.Active && !profile.DeletedAt.Valid:
			continue
		case profile.DeletedAt.Valid:
			// Deleted: bring it back as seeded, members included.
			if err := tdb.Unscoped().Model(&clinical_models.ExamProfile{}).Where("id = ?", profile.ID).
				Updates(map[string]any{"active": true, "deleted_at": nil}).Error; err != nil {
				return 0, 0, fmt.Errorf("restore profile %q: %w", seed.Name, err)
			}
			profile.DeletedAt = gorm.DeletedAt{}
		default:
			// Only deactivated: re-activate, keep the tenant's members.
			if err := tdb.Model(&clinical_models.ExamProfile{}).Where("id = ?", profile.ID).
				Update("active", true).Error; err != nil {
				return 0, 0, fmt.Errorf("restore profile %q: %w", seed.Name, err)
			}
			profiles++
			continue
		}
		if err := SetProfileItems(tdb, profile.ID, members); err != nil {
			return 0, 0, fmt.Errorf("set profile %q exams: %w", seed.Name, err)
		}
		profiles++
	}
	return items, profiles, nil
}

// SetProfileItems replaces a profile's exams. It writes the join table
// directly: GORM's association mode would also upsert the exams themselves.
func SetProfileItems(db *gorm.DB, profileID uint, itemIDs []uint) error {
	if err := db.Exec("DELETE FROM exam_profile_items WHERE exam_profile_id = ?", profileID).Error; err != nil {
		return err
	}
	seen := map[uint]bool{}
	rows := make([]map[string]any, 0, len(itemIDs))
	for _, id := range itemIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		rows = append(rows, map[string]any{"exam_profile_id": profileID, "exam_catalog_item_id": id})
	}
	if len(rows) == 0 {
		return nil
	}
	return db.Table("exam_profile_items").Create(&rows).Error
}

// ErrUnknownCatalogItems means some IDs are not catalog exams of the tenant.
var ErrUnknownCatalogItems = errors.New("clinical: unknown exam catalog items")

// CatalogItemsByID loads the tenant's catalog exams with the given IDs (db
// must be tenant-bound), failing with ErrUnknownCatalogItems if any is missing.
func CatalogItemsByID(db *gorm.DB, ids []uint) ([]clinical_models.ExamCatalogItem, error) {
	unique := map[uint]bool{}
	for _, id := range ids {
		unique[id] = true
	}
	if len(unique) == 0 {
		return nil, nil
	}
	var items []clinical_models.ExamCatalogItem
	if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) != len(unique) {
		return nil, ErrUnknownCatalogItems
	}
	return items, nil
}
