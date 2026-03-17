package core_errors

var (
	ErrInternal           AppError = NewAppError("E-INT-001", "Internal server error.")
	ErrInvalidRequest     AppError = NewAppError("E-INT-002", "Invalid request.")
	ErrRateLimitExceeded  AppError = NewAppError("E-INT-003", "Rate limit exceeded.")

	ErrMessagesNotFound AppError = NewAppError("E-MES-001", "Messages not found.")

	ErrCompanyNotFound AppError = NewAppError("E-COMP-001", "Company not found.")

	ErrTenantNotFound AppError = NewAppError("E-TEN-001", "Tenant not found.")

	ErrUserNotFound AppError = NewAppError("E-USR-001", "User not found.")

	// Auth Errors
	ErrAuthInvalidRequest      AppError = NewAppError("E-AUTH-001", "Invalid authentication request.")
	ErrAuthUserCreateError     AppError = NewAppError("E-AUTH-002", "Error creating user.")
	ErrAuthInvalidCredentials  AppError = NewAppError("E-AUTH-003", "Invalid credentials.")
	ErrAuthTokenGenerateError  AppError = NewAppError("E-AUTH-004", "Error generating token.")
	ErrAuthInvalidRefreshToken AppError = NewAppError("E-AUTH-005", "Invalid refresh token.")
	ErrAuthUserInvalidID       AppError = NewAppError("E-AUTH-006", "Invalid user ID.")

	// Clinical Errors
	ErrClinicalPatientNotFound     AppError = NewAppError("E-CLIN-001", "Patient not found.")
	ErrClinicalPatientCreateError  AppError = NewAppError("E-CLIN-002", "Error creating patient.")
	ErrClinicalPatientUpdateError  AppError = NewAppError("E-CLIN-003", "Error updating patient.")
	ErrClinicalPatientDeleteError  AppError = NewAppError("E-CLIN-004", "Error deleting patient.")
	ErrClinicalRecordNotFound      AppError = NewAppError("E-CLIN-005", "Medical record not found.")
	ErrClinicalRecordCreateError   AppError = NewAppError("E-CLIN-006", "Error creating medical record.")
	ErrClinicalRecordUpdateError   AppError = NewAppError("E-CLIN-007", "Error updating medical record.")
	ErrClinicalReportGenerateError    AppError = NewAppError("E-CLIN-008", "Error generating patient report.")
	ErrClinicalAppointmentOverlap     AppError = NewAppError("E-CLIN-009", "Appointment overlaps with an existing one.")
	ErrClinicalInvalidRequest         AppError = NewAppError("E-CLIN-010", "Invalid clinical request.")

	// Permission Errors
	ErrPermissionGetError AppError = NewAppError("E-PERM-001", "Error getting permissions.")

	// Backoffice Errors
	ErrBackofficeInvalidRequest       AppError = NewAppError("E-BO-001", "Invalid backoffice request.")
	ErrBackofficeCompanyCreateError   AppError = NewAppError("E-BO-002", "Error creating company.")
	ErrBackofficeCompanyUpdateError   AppError = NewAppError("E-BO-003", "Error updating company.")
	ErrBackofficeCompanyDeleteError   AppError = NewAppError("E-BO-004", "Error deleting company.")
	ErrBackofficeFeatureNotFound      AppError = NewAppError("E-BO-005", "Feature not found.")
	ErrBackofficePlanNotFound         AppError = NewAppError("E-BO-006", "Plan not found.")
	ErrBackofficeSubscriptionNotFound AppError = NewAppError("E-BO-007", "Subscription not found.")
	ErrPlanLimitUsers                 AppError = NewAppError("E-PLAN-001", "User limit reached for this plan.")
	ErrPlanLimitPatients              AppError = NewAppError("E-PLAN-002", "Patient limit reached for this plan.")

	// Billing Errors
	ErrBillingInvalidRequest       AppError = NewAppError("E-BILL-001", "Invalid billing request.")
	ErrBillingInvoiceNotFound      AppError = NewAppError("E-BILL-002", "Invoice not found.")
	ErrBillingProductNotFound      AppError = NewAppError("E-BILL-003", "Product/Service not found.")
	ErrBillingInvoiceCreateError   AppError = NewAppError("E-BILL-004", "Error creating invoice.")
	ErrBillingInvalidSignatureFile AppError = NewAppError("E-BILL-005", "Incorrect password or invalid signature file.")

	ErrAuthInvalidSignupToken AppError = NewAppError("E-AUTH-007", "Invalid or expired signup token.")
)
