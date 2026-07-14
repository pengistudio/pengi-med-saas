package role_data

// Canonical global role names. Roles are a fixed catalog shared by every
// tenant — not created per-company. See migrations DB20260708_1/DB20260708_2.
const (
	RoleAdmin         = "admin"
	RoleDoctor        = "doctor"
	RoleRecepcionista = "recepcionista"
	RoleContador      = "contador"
)

// CanonicalRoles lists every role that should be selectable/assignable.
// Used to filter out legacy "moderator"/"user" placeholder rows wherever
// roles are listed or validated (team invites, role reassignment).
var CanonicalRoles = []string{RoleAdmin, RoleDoctor, RoleRecepcionista, RoleContador}

// RolePermissionMatrix maps each non-admin canonical role to the permission
// IDs it should hold. admin is intentionally omitted — it always gets every
// Permission row that exists, matching the existing repo convention that new
// permission catalogs auto-wire to admin.
var RolePermissionMatrix = map[string][]string{
	RoleDoctor: {
		"READ_PATIENT", "CREATE_PATIENT", "UPDATE_PATIENT",
		"READ_MEDICAL_RECORD", "CREATE_MEDICAL_RECORD", "UPDATE_MEDICAL_RECORD",
		"UPDATE_PRESCRIPTION", "DOWNLOAD_PATIENT_REPORT",
		"READ_KANBAN", "CREATE_KANBAN", "UPDATE_KANBAN",
	},
	RoleRecepcionista: {
		"READ_PATIENT", "CREATE_PATIENT", "UPDATE_PATIENT",
		"READ_BILLING", "CREATE_BILLING", "UPDATE_BILLING",
		"READ_KANBAN", "CREATE_KANBAN", "UPDATE_KANBAN",
	},
	RoleContador: {
		"READ_BILLING", "CREATE_BILLING", "UPDATE_BILLING", "DELETE_BILLING", "MANAGE_SRI_SETTINGS",
		"READ_PATIENT",
	},
}

// IsCanonicalRole reports whether roleName is one of the fixed catalog roles.
func IsCanonicalRole(roleName string) bool {
	for _, r := range CanonicalRoles {
		if r == roleName {
			return true
		}
	}
	return false
}
