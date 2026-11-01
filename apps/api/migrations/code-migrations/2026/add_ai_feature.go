package y2026

import (
	"errors"
	"fmt"

	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const addAIFeatureID = "DB20261031_1"

func init() {
	// Creates USE_AI_SUMMARY and USE_AI_SCRIBE, gives them to the admin and
	// doctor roles and puts them in a new AI Feature that backoffice adds to
	// the top plan. No plan gets the Feature here, and each clinic must also
	// opt in from its settings (off by default). Monthly quotas are plan
	// properties (max_ai_*); absent means the default, never unlimited.
	database.GlobalDBMap[addAIFeatureID] = database.DBExecute{
		ID:      addAIFeatureID,
		Execute: addAIFeature,
	}
}

func addAIFeature(db *gorm.DB) error {
	var roles []*user_models.Role
	for _, name := range []string{role_data.RoleAdmin, role_data.RoleDoctor} {
		var role user_models.Role
		switch err := db.Where(user_models.Role{Role: name}).First(&role).Error; {
		case err == nil:
			roles = append(roles, &role)
		case errors.Is(err, gorm.ErrRecordNotFound):
			fmt.Printf("⚠️  %s role not found, skipping its AI permissions.\n", name)
		default:
			return fmt.Errorf("failed to find %s role: %w", name, err)
		}
	}

	feature := company_models.Feature{Code: "AI", Name: "Inteligencia artificial"}
	if err := db.Where(company_models.Feature{Code: feature.Code}).FirstOrCreate(&feature).Error; err != nil {
		return fmt.Errorf("failed to create AI feature: %w", err)
	}

	for _, perm := range permission_data.AIPermissions {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		for _, role := range roles {
			if err := db.Model(role).Association("Permissions").Append(&perm); err != nil {
				return fmt.Errorf("failed to assign permission '%s' to role '%s': %w", perm.ID, role.Role, err)
			}
		}
		if err := db.Model(&feature).Association("Permissions").Append(&perm); err != nil {
			return fmt.Errorf("failed to associate '%s' with AI feature: %w", perm.ID, err)
		}
		fmt.Printf("✅ Permission '%s' created, assigned to admin, doctor and the AI feature.\n", perm.ID)
	}
	return nil
}
