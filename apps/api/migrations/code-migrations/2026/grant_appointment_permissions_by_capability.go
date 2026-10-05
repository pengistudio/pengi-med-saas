package y2026

import (
	"fmt"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	permission_models "pengi-med-saas/features/permissions/models"
	role_data "pengi-med-saas/features/users/data"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

const grantAppointmentPermissionsByCapabilityID = "DB20261004_3"

// capabilityGrant gives Grants to every role that already holds any of Holds,
// except the roles named in Except.
type capabilityGrant struct {
	Holds  []string
	Grants []string
	Except []string
}

// appointmentCapabilityGrants: before the appointment permissions existed the
// agenda needed none, so DB20261004_1 (which grants by canonical role name)
// would take it away from any other role. A role that reads patients or
// medical records keeps the agenda; one that edits medical records keeps
// recording vital signs. contador reads patients only to pick one on an
// invoice, so it stays without the agenda.
var appointmentCapabilityGrants = []capabilityGrant{
	{
		Holds:  []string{"READ_PATIENT", "READ_MEDICAL_RECORD"},
		Grants: []string{"READ_APPOINTMENT", "MANAGE_APPOINTMENT"},
		Except: []string{role_data.RoleContador},
	},
	{
		Holds:  []string{"UPDATE_MEDICAL_RECORD"},
		Grants: []string{"RECORD_VITAL_SIGNS"},
	},
}

// grantAppointmentPermissionsByCapability applies appointmentCapabilityGrants
// to every Role row. Roles are global (shared by every tenant and company, see
// role_data), so there is no per-tenant pass. Idempotent: appending a
// permission a role already holds is a no-op.
func grantAppointmentPermissionsByCapability(db *gorm.DB) error {
	db = tenantdb.System(db).Session(&gorm.Session{})

	perms := map[string]permission_models.Permission{}
	for _, perm := range appointmentPermissions() {
		if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
			return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
		}
		perms[perm.ID] = perm
	}

	for _, g := range appointmentCapabilityGrants {
		holders := db.Table("role_permissions").Select("role_id").Where("permission_id IN ?", g.Holds)
		query := db.Where("id IN (?)", holders)
		if len(g.Except) > 0 {
			query = query.Where("role NOT IN ?", g.Except)
		}
		var roles []user_models.Role
		if err := query.Order("id").Find(&roles).Error; err != nil {
			return fmt.Errorf("failed to list roles holding %v: %w", g.Holds, err)
		}
		for i := range roles {
			for _, id := range g.Grants {
				perm, ok := perms[id]
				if !ok {
					return fmt.Errorf("permission '%s' missing from the clinical catalog", id)
				}
				if err := db.Model(&roles[i]).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign '%s' to role %d (%s): %w", id, roles[i].ID, roles[i].Role, err)
				}
			}
			fmt.Printf("✅ Role %d (%s) holds %v.\n", roles[i].ID, roles[i].Role, g.Grants)
		}
	}
	return nil
}

func init() {
	database.GlobalDBMap[grantAppointmentPermissionsByCapabilityID] = database.DBExecute{
		ID:      grantAppointmentPermissionsByCapabilityID,
		Execute: grantAppointmentPermissionsByCapability,
	}
}
