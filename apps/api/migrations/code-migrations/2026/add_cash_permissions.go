package y2026

import (
	"errors"
	"fmt"

	"pengi-med-saas/core/database"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const addCashPermissionsID = "DB20261104_1"

func init() {
	// Creates the cash desk permissions (READ_CASH, CREATE_PAYMENT,
	// MANAGE_CASH_SESSION, VOID_PAYMENT) and gives them to the existing roles:
	// admin all four, recepcionista and contador what RolePermissionMatrix
	// lists for them, doctor none. DB20261104_2 puts them in the CASH Feature.
	database.GlobalDBMap[addCashPermissionsID] = database.DBExecute{
		ID:      addCashPermissionsID,
		Execute: addCashPermissions,
	}
}

func addCashPermissions(db *gorm.DB) error {
	cash := map[string]bool{}
	for _, perm := range permission_data.CashPermissions {
		cash[perm.ID] = true
	}
	grants := map[string][]string{role_data.RoleAdmin: nil}
	for _, perm := range permission_data.CashPermissions {
		grants[role_data.RoleAdmin] = append(grants[role_data.RoleAdmin], perm.ID)
	}
	for _, name := range []string{role_data.RoleRecepcionista, role_data.RoleContador, role_data.RoleDoctor} {
		for _, id := range role_data.RolePermissionMatrix[name] {
			if cash[id] {
				grants[name] = append(grants[name], id)
			}
		}
	}

	perms := map[string]*permission_models.Permission{}
	for _, perm := range permission_data.CashPermissions {
		p := perm
		if err := db.Where(permission_models.Permission{BaseStringID: p.BaseStringID}).FirstOrCreate(&p).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", p.ID, err)
		}
		perms[p.ID] = &p
	}

	for _, name := range []string{role_data.RoleAdmin, role_data.RoleRecepcionista, role_data.RoleContador, role_data.RoleDoctor} {
		if len(grants[name]) == 0 {
			continue
		}
		var role user_models.Role
		switch err := db.Where(user_models.Role{Role: name}).First(&role).Error; {
		case errors.Is(err, gorm.ErrRecordNotFound):
			fmt.Printf("⚠️  %s role not found, skipping its cash permissions.\n", name)
			continue
		case err != nil:
			return fmt.Errorf("failed to find %s role: %w", name, err)
		}
		for _, id := range grants[name] {
			if err := db.Model(&role).Association("Permissions").Append(perms[id]); err != nil {
				return fmt.Errorf("failed to assign permission '%s' to role '%s': %w", id, name, err)
			}
		}
		fmt.Printf("✅ Cash permissions %v assigned to %s.\n", grants[name], name)
	}
	return nil
}
