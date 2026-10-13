package y2026

import (
	"fmt"
	"strings"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	company_models "pengi-med-saas/features/companies/models"
	doctor_data "pengi-med-saas/features/doctors/data"
	doctor_models "pengi-med-saas/features/doctors/models"
	signature_models "pengi-med-saas/features/signatures/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const createDoctorsFromUsersID = "DB20261008_2"

func init() {
	// Gives every tenant its first doctor profiles, flagged needs_review:
	//   - one per user whose role in the tenant is `doctor` (name from their
	//     signature certificate, else their user name);
	//   - if there are none, one for the company owner (name from their
	//     signature, else the most frequent patients.medic, else user name).
	// Linking existing records to the doctor (doctor_id) is a later migration.
	database.GlobalDBMap[createDoctorsFromUsersID] = database.DBExecute{
		ID:      createDoctorsFromUsersID,
		Execute: createDoctorsFromUsers,
	}
}

func createDoctorsFromUsers(db *gorm.DB) error {
	db = tenantdb.System(db)
	var tenantIDs []uint
	if err := db.Model(&tenant_models.Tenant{}).Order("id").Pluck("id", &tenantIDs).Error; err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}
	created := 0
	for _, tenantID := range tenantIDs {
		n, err := createTenantDoctors(db, tenantID)
		if err != nil {
			return fmt.Errorf("tenant %d: %w", tenantID, err)
		}
		created += n
	}
	fmt.Printf("✅ Created %d doctor profiles from existing users.\n", created)
	return nil
}

// createTenantDoctors creates the missing doctor profiles of one tenant and
// returns how many it created.
func createTenantDoctors(db *gorm.DB, tenantID uint) (int, error) {
	var doctorUserIDs []uint
	if err := db.Table("environments").
		Joins("JOIN companies ON companies.id = environments.company_id AND companies.deleted_at IS NULL").
		Joins("JOIN roles ON roles.id = environments.role_id").
		Joins("JOIN users ON users.id = environments.user_id AND users.deleted_at IS NULL").
		Where("companies.tenant_id = ? AND roles.role = ? AND environments.deleted_at IS NULL", tenantID, role_data.RoleDoctor).
		Distinct("environments.user_id").Order("environments.user_id").
		Pluck("environments.user_id", &doctorUserIDs).Error; err != nil {
		return 0, fmt.Errorf("list doctor-role users: %w", err)
	}

	var colors []string
	if err := db.Model(&doctor_models.Doctor{}).Where("tenant_id = ?", tenantID).Pluck("color", &colors).Error; err != nil {
		return 0, fmt.Errorf("list doctor colors: %w", err)
	}

	created := 0
	add := func(userID *uint, name, email string) error {
		doctor := doctor_models.Doctor{
			TenantID:    tenantID,
			UserID:      userID,
			FullName:    name,
			Specialty:   doctor_data.DefaultSpecialty,
			Email:       email,
			Color:       doctor_data.NextColor(colors),
			Active:      true,
			NeedsReview: true,
		}
		if err := db.Create(&doctor).Error; err != nil {
			return fmt.Errorf("create doctor %q: %w", name, err)
		}
		colors = append(colors, doctor.Color)
		created++
		return nil
	}

	if len(doctorUserIDs) > 0 {
		for _, userID := range doctorUserIDs {
			var existing int64
			if err := db.Model(&doctor_models.Doctor{}).Where("tenant_id = ? AND user_id = ?", tenantID, userID).Count(&existing).Error; err != nil {
				return created, err
			}
			if existing > 0 {
				continue
			}
			user, err := migrationUser(db, userID)
			if err != nil {
				return created, err
			}
			name, err := signatureName(db, tenantID, userID)
			if err != nil {
				return created, err
			}
			if name == "" {
				name = user.UserName
			}
			id := userID
			if err := add(&id, name, user.Email); err != nil {
				return created, err
			}
		}
		return created, nil
	}

	// No doctor-role users: the owner is the clinic's doctor, unless the
	// tenant already has one (a re-run, or created by hand).
	var existing int64
	if err := db.Model(&doctor_models.Doctor{}).Where("tenant_id = ?", tenantID).Count(&existing).Error; err != nil {
		return created, err
	}
	if existing > 0 {
		return created, nil
	}
	var company company_models.Company
	if err := db.Where("tenant_id = ?", tenantID).Order("id").Limit(1).Find(&company).Error; err != nil {
		return created, fmt.Errorf("find company: %w", err)
	}
	medic, err := mostFrequentMedic(db, tenantID)
	if err != nil {
		return created, err
	}
	if company.OwnerUserID == 0 {
		// No owner to link: keep the name the clinic already printed, if any.
		if medic != "" {
			return created, add(nil, medic, "")
		}
		return created, nil
	}
	owner, err := migrationUser(db, company.OwnerUserID)
	if err != nil {
		return created, err
	}
	name, err := signatureName(db, tenantID, company.OwnerUserID)
	if err != nil {
		return created, err
	}
	if name == "" {
		name = medic
	}
	if name == "" {
		name = owner.UserName
	}
	ownerID := company.OwnerUserID
	return created, add(&ownerID, name, owner.Email)
}

func migrationUser(db *gorm.DB, userID uint) (user_models.User, error) {
	var user user_models.User
	if err := db.Unscoped().Select("id", "user_name", "email").First(&user, userID).Error; err != nil {
		return user, fmt.Errorf("find user %d: %w", userID, err)
	}
	return user, nil
}

// signatureName is the holder name of the user's signature certificate in the
// tenant, or "".
func signatureName(db *gorm.DB, tenantID, userID uint) (string, error) {
	var names []string
	if err := db.Model(&signature_models.UserSignature{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Limit(1).Pluck("subject_name", &names).Error; err != nil {
		return "", fmt.Errorf("find signature of user %d: %w", userID, err)
	}
	if len(names) == 0 {
		return "", nil
	}
	return strings.TrimSpace(names[0]), nil
}

// mostFrequentMedic is the doctor name most patients of the tenant carry in
// the legacy free-text medic field, or "".
func mostFrequentMedic(db *gorm.DB, tenantID uint) (string, error) {
	var rows []struct {
		Medic string
		Total int64
	}
	if err := db.Table("patients").
		Select("TRIM(medic) AS medic, COUNT(*) AS total").
		Where("tenant_id = ? AND deleted_at IS NULL AND TRIM(COALESCE(medic, '')) <> ''", tenantID).
		Group("TRIM(medic)").
		Order("total DESC, medic").
		Limit(1).
		Scan(&rows).Error; err != nil {
		return "", fmt.Errorf("find most frequent patients.medic: %w", err)
	}
	if len(rows) == 0 {
		return "", nil
	}
	return rows[0].Medic, nil
}
