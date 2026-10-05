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
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_MEDICAL_REPORT"},
		Name:         "Create Medical Report",
		Category:     "CLINICAL",
		Description:  "Generate, download and email medical reports",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_MEDICAL_CERTIFICATE"},
		Name:         "Create Medical Certificate",
		Category:     "CLINICAL",
		Description:  "Generate, download and email medical certificates",
	},
	{
		BaseStringID: database.BaseStringID{ID: "SIGN_MEDICAL_DOCUMENT"},
		Name:         "Sign Medical Document",
		Category:     "CLINICAL",
		Description:  "Upload an electronic signature (P12) and sign medical reports, certificates and prescriptions",
	},
	{
		BaseStringID: database.BaseStringID{ID: "READ_PATIENT_ATTACHMENT"},
		Name:         "Read Patient Attachment",
		Category:     "CLINICAL",
		Description:  "List, view and download a patient's attached files",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPLOAD_PATIENT_ATTACHMENT"},
		Name:         "Upload Patient Attachment",
		Category:     "CLINICAL",
		Description:  "Upload files (results, images, external reports) to a patient's record",
	},
	{
		BaseStringID: database.BaseStringID{ID: "DELETE_PATIENT_ATTACHMENT"},
		Name:         "Delete Patient Attachment",
		Category:     "CLINICAL",
		Description:  "Delete a patient's attached files with a reason, see the deleted ones and restore them",
	},
	{
		BaseStringID: database.BaseStringID{ID: "READ_EXAM_ORDER"},
		Name:         "Read Exam Orders",
		Category:     "CLINICAL",
		Description:  "View exam orders, their results and the exam catalog",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_EXAM_ORDER"},
		Name:         "Create Exam Orders",
		Category:     "CLINICAL",
		Description:  "Create, edit, void, close, print and email exam orders",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPLOAD_EXAM_RESULTS"},
		Name:         "Upload Exam Results",
		Category:     "CLINICAL",
		Description:  "Upload and delete exam result files",
	},
	{
		BaseStringID: database.BaseStringID{ID: "REVIEW_EXAM_RESULTS"},
		Name:         "Review Exam Results",
		Category:     "CLINICAL",
		Description:  "Mark exam results as reviewed",
	},
	{
		BaseStringID: database.BaseStringID{ID: "MANAGE_EXAM_CATALOG"},
		Name:         "Manage Exam Catalog",
		Category:     "CLINICAL",
		Description:  "Create, edit and restore the exam catalog and profiles",
	},
	{
		BaseStringID: database.BaseStringID{ID: "READ_APPOINTMENT"},
		Name:         "Read Appointments",
		Category:     "CLINICAL",
		Description:  "View the agenda and the waiting room",
	},
	{
		BaseStringID: database.BaseStringID{ID: "MANAGE_APPOINTMENT"},
		Name:         "Manage Appointments",
		Category:     "CLINICAL",
		Description:  "Create, update, change the status of and delete appointments",
	},
	{
		BaseStringID: database.BaseStringID{ID: "RECORD_VITAL_SIGNS"},
		Name:         "Record Vital Signs",
		Category:     "CLINICAL",
		Description:  "Record a consultation's vital signs without editing the medical record",
	},
}

// ExamOrderPermissionIDs are the exam order permissions (part of ClinicalPermissions).
var ExamOrderPermissionIDs = []string{"READ_EXAM_ORDER", "CREATE_EXAM_ORDER", "UPLOAD_EXAM_RESULTS", "REVIEW_EXAM_RESULTS", "MANAGE_EXAM_CATALOG"}

// AppointmentPermissionIDs are the agenda/waiting room and triage permissions
// (part of ClinicalPermissions).
var AppointmentPermissionIDs = []string{"READ_APPOINTMENT", "MANAGE_APPOINTMENT", "RECORD_VITAL_SIGNS"}

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

var TeamPermissions = []permission_models.Permission{
	{
		BaseStringID: database.BaseStringID{ID: "READ_TEAM"},
		Name:         "Read Team",
		Category:     "TEAM",
		Description:  "View teams",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_TEAM"},
		Name:         "Create Team",
		Category:     "TEAM",
		Description:  "Create new teams",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPDATE_TEAM"},
		Name:         "Update Team",
		Category:     "TEAM",
		Description:  "Update team information",
	},
	{
		BaseStringID: database.BaseStringID{ID: "DELETE_TEAM"},
		Name:         "Delete Team",
		Category:     "TEAM",
		Description:  "Delete teams",
	},
	{
		BaseStringID: database.BaseStringID{ID: "MANAGE_TEAM_MEMBERS"},
		Name:         "Manage Team Members",
		Category:     "TEAM",
		Description:  "Add or remove team members",
	},
}

var AuditPermissions = []permission_models.Permission{
	{
		BaseStringID: database.BaseStringID{ID: "READ_AUDIT_LOG"},
		Name:         "Read Audit Log",
		Category:     "AUDIT",
		Description:  "View the compliance audit trail (access/modification history) for clinical records",
	},
}

// DocumentTemplatePermissions guard the tenant's custom templates for printable
// documents (prescription today; every printable document soon). Category
// DOCUMENTS is cross-domain and maps to no nav flag, so the permission can sit
// in both the CLINICAL and BILLING Features without enabling either flag.
var DocumentTemplatePermissions = []permission_models.Permission{
	{
		BaseStringID: database.BaseStringID{ID: "MANAGE_DOCUMENT_TEMPLATES"},
		Name:         "Manage Document Templates",
		Category:     "DOCUMENTS",
		Description:  "Upload, replace or reset the custom templates used to print documents",
	},
}

var KanbanPermissions = []permission_models.Permission{
	{
		BaseStringID: database.BaseStringID{ID: "READ_KANBAN"},
		Name:         "Read Kanban",
		Category:     "KANBAN",
		Description:  "View kanban tasks",
	},
	{
		BaseStringID: database.BaseStringID{ID: "CREATE_KANBAN"},
		Name:         "Create Kanban Task",
		Category:     "KANBAN",
		Description:  "Create new kanban tasks",
	},
	{
		BaseStringID: database.BaseStringID{ID: "UPDATE_KANBAN"},
		Name:         "Update Kanban Task",
		Category:     "KANBAN",
		Description:  "Update and move kanban tasks",
	},
	{
		BaseStringID: database.BaseStringID{ID: "DELETE_KANBAN"},
		Name:         "Delete Kanban Task",
		Category:     "KANBAN",
		Description:  "Delete kanban tasks",
	},
}
