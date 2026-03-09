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

var BillingPermissions = []permission_models.Permission{
	{
		BaseStringID: database.BaseStringID{ID: "READ_BILLING"},
		Name:         "Read Billing",
		Category:     "BILLING",
		Description:  "View invoices and billing data",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_BILLING"},
		Name:         "Create Billing",
		Category:     "BILLING",
		Description:  "Create new invoices",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPDATE_BILLING"},
		Name:         "Update Billing",
		Category:     "BILLING",
		Description:  "Update existing invoices",
	},
	{
		BaseStringID: database.BaseStringID{ID: "DELETE_BILLING"},
		Name:         "Delete Billing",
		Category:     "BILLING",
		Description:  "Delete invoices",
	},
	{
		BaseStringID: database.BaseStringID{ID: "MANAGE_SRI_SETTINGS"},
		Name:         "Manage SRI Settings",
		Category:     "BILLING",
		Description:  "Configure electronic signature and SRI environment",
	},
}
