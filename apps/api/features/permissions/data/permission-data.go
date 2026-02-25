package permission_data

import (
	"pengi-med-saas/core/database"
	permission_models "pengi-med-saas/features/permissions/models"
)

var ClinicalPermissions = []permission_models.Permission{
	{
		BaseStringID: database.BaseStringID{ID: "READ_PATIENT"},
		Name:         "Read Patient",
		Category:     "CLINICAL",
		Description:  "View patient records",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_PATIENT"},
		Name:         "Create Patient",
		Category:     "CLINICAL",
		Description:  "Create new patients",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPDATE_PATIENT"},
		Name:         "Update Patient",
		Category:     "CLINICAL",
		Description:  "Update patient records",
	},
	{
		BaseStringID: database.BaseStringID{ID: "DELETE_PATIENT"},
		Name:         "Delete Patient",
		Category:     "CLINICAL",
		Description:  "Delete patients",
	},
	{
		BaseStringID: database.BaseStringID{ID: "READ_MEDICAL_RECORD"},
		Name:         "Read Medical Record",
		Category:     "CLINICAL",
		Description:  "View medical records",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_MEDICAL_RECORD"},
		Name:         "Create Medical Record",
		Category:     "CLINICAL",
		Description:  "Create new medical records",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPDATE_MEDICAL_RECORD"},
		Name:         "Update Medical Record",
		Category:     "CLINICAL",
		Description:  "Update medical records",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPDATE_PRESCRIPTION"},
		Name:         "Update Prescription",
		Category:     "CLINICAL",
		Description:  "Update prescriptions",
	},
	{
		BaseStringID: database.BaseStringID{ID: "DOWNLOAD_PATIENT_REPORT"},
		Name:         "Download Patient Report",
		Category:     "CLINICAL",
		Description:  "Download patient reports",
	},
}
