package core_errors

var (
	ErrInternal          AppError = NewAppError("E-INT-001", "Internal server error.")
	ErrInvalidRequest    AppError = NewAppError("E-INT-002", "Invalid request.")
	ErrRateLimitExceeded AppError = NewAppError("E-INT-003", "Rate limit exceeded.")

	ErrMessagesNotFound AppError = NewAppError("E-MES-001", "Messages not found.")

	ErrCompanyNotFound       AppError = NewAppError("E-COMP-001", "Company not found.")
	ErrCompanyOwnershipLimit AppError = NewAppError("E-COMP-002", "Owned company limit reached for this user.")

	ErrTenantNotFound            AppError = NewAppError("E-TEN-001", "Tenant not found.")
	ErrTenantInvalidDisplayToken AppError = NewAppError("E-TEN-002", "Invalid or missing display token.")
	ErrTenantInvalidLogoFile     AppError = NewAppError("E-TEN-003", "Invalid logo file. Only PNG or JPG images are allowed.")
	ErrTenantLogoNotFound        AppError = NewAppError("E-TEN-004", "No logo has been uploaded for this tenant.")
	ErrTenantForbidden           AppError = NewAppError("E-TEN-005", "User has no role in this tenant.")

	ErrUserNotFound AppError = NewAppError("E-USR-001", "User not found.")

	// Auth Errors
	ErrAuthInvalidRequest      AppError = NewAppError("E-AUTH-001", "Invalid authentication request.")
	ErrAuthUserCreateError     AppError = NewAppError("E-AUTH-002", "Error creating user.")
	ErrAuthInvalidCredentials  AppError = NewAppError("E-AUTH-003", "Invalid credentials.")
	ErrAuthTokenGenerateError  AppError = NewAppError("E-AUTH-004", "Error generating token.")
	ErrAuthInvalidRefreshToken AppError = NewAppError("E-AUTH-005", "Invalid refresh token.")
	ErrAuthUserInvalidID       AppError = NewAppError("E-AUTH-006", "Invalid user ID.")

	// Clinical Errors
	ErrClinicalPatientNotFound         AppError = NewAppError("E-CLIN-001", "Patient not found.")
	ErrClinicalPatientCreateError      AppError = NewAppError("E-CLIN-002", "Error creating patient.")
	ErrClinicalPatientUpdateError      AppError = NewAppError("E-CLIN-003", "Error updating patient.")
	ErrClinicalPatientDeleteError      AppError = NewAppError("E-CLIN-004", "Error deleting patient.")
	ErrClinicalRecordNotFound          AppError = NewAppError("E-CLIN-005", "Medical record not found.")
	ErrClinicalRecordCreateError       AppError = NewAppError("E-CLIN-006", "Error creating medical record.")
	ErrClinicalRecordUpdateError       AppError = NewAppError("E-CLIN-007", "Error updating medical record.")
	ErrClinicalReportGenerateError     AppError = NewAppError("E-CLIN-008", "Error generating patient report.")
	ErrClinicalAppointmentOverlap      AppError = NewAppError("E-CLIN-009", "Appointment overlaps with an existing one.")
	ErrClinicalInvalidRequest          AppError = NewAppError("E-CLIN-010", "Invalid clinical request.")
	ErrClinicalDraftError              AppError = NewAppError("E-CLIN-011", "Error processing medical record draft.")
	ErrClinicalMedicalReportError      AppError = NewAppError("E-CLIN-012", "Error generating/saving the medical report.")
	ErrClinicalMedicalCertificateError AppError = NewAppError("E-CLIN-013", "Error generating/saving the medical certificate.")
	ErrClinicalDocumentEmailError      AppError = NewAppError("E-CLIN-014", "Error sending the document by email.")

	// Electronic Signature Errors
	ErrSignatureInvalidFile    AppError = NewAppError("E-SIGN-001", "Invalid signature file or password.")
	ErrSignatureExpired        AppError = NewAppError("E-SIGN-002", "The signature certificate is expired.")
	ErrSignatureNotConfigured  AppError = NewAppError("E-SIGN-003", "You have not uploaded an electronic signature.")
	ErrSignatureAlreadySigned  AppError = NewAppError("E-SIGN-004", "The document is already signed.")
	ErrSignatureSignFailed     AppError = NewAppError("E-SIGN-005", "Error signing the document.")
	ErrSignatureKeyUnavailable AppError = NewAppError("E-SIGN-006", "Electronic signature is not available on this server.")
	ErrSignatureWrongPassword  AppError = NewAppError("E-SIGN-007", "Incorrect signature password.")
	ErrSignatureNotYetValid    AppError = NewAppError("E-SIGN-008", "The signature certificate is not valid yet.")
	ErrSignatureRucMismatch    AppError = NewAppError("E-SIGN-009", "The signature does not belong to the company's RUC.")

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
	ErrBackofficePaymentCreateFailed  AppError = NewAppError("E-BO-008", "Payment creation failed.")
	ErrBackofficePaymentNotFound      AppError = NewAppError("E-BO-009", "Payment not found.")
	ErrBackofficeWebhookInvalidSig    AppError = NewAppError("E-BO-010", "Invalid webhook signature.")
	ErrPlanLimitUsers                 AppError = NewAppError("E-PLAN-001", "User limit reached for this plan.")
	ErrPlanLimitPatients              AppError = NewAppError("E-PLAN-002", "Patient limit reached for this plan.")

	// Billing Errors
	ErrBillingInvalidRequest        AppError = NewAppError("E-BILL-001", "Invalid billing request.")
	ErrBillingInvoiceNotFound       AppError = NewAppError("E-BILL-002", "Invoice not found.")
	ErrBillingProductNotFound       AppError = NewAppError("E-BILL-003", "Product/Service not found.")
	ErrBillingInvoiceCreateError    AppError = NewAppError("E-BILL-004", "Error creating invoice.")
	ErrBillingInvalidSignatureFile  AppError = NewAppError("E-BILL-005", "Incorrect password or invalid signature file.")
	ErrBillingCreditNoteNotFound    AppError = NewAppError("E-BILL-006", "Credit note not found.")
	ErrBillingCreditNoteCreateError AppError = NewAppError("E-BILL-007", "Error creating credit note.")
	ErrBillingInvoiceNotAuthorized  AppError = NewAppError("E-BILL-008", "Invoice must be authorized by the SRI before it can be credited.")
	ErrBillingDebitNoteNotFound     AppError = NewAppError("E-BILL-009", "Debit note not found.")
	ErrBillingDebitNoteCreateError  AppError = NewAppError("E-BILL-010", "Error creating debit note.")
	ErrBillingInvoiceRideNotReady   AppError = NewAppError("E-BILL-011", "Invoice must be authorized before the RIDE can be downloaded.")
	ErrBillingInvoiceRideGenerate   AppError = NewAppError("E-BILL-012", "Error generating the RIDE PDF.")
	ErrBillingSignatureExpired      AppError = NewAppError("E-BILL-013", "The SRI electronic signature has expired. Upload a valid one to issue documents.")

	ErrAuthInvalidSignupToken        AppError = NewAppError("E-AUTH-007", "Invalid or expired signup token.")
	ErrAuthInvalidPasswordResetToken AppError = NewAppError("E-AUTH-008", "Invalid or expired password reset token.")
	ErrAuthEmailTaken                AppError = NewAppError("E-AUTH-009", "Email already in use.")
	ErrAuthUsernameTaken             AppError = NewAppError("E-AUTH-010", "Username already in use.")
	ErrAuthEmailNotVerified          AppError = NewAppError("E-AUTH-011", "Email not verified.")
	ErrAuthInvalidVerificationToken  AppError = NewAppError("E-AUTH-012", "Invalid or expired verification token.")
	ErrAuthRefreshTokenReused        AppError = NewAppError("E-AUTH-013", "Refresh token reuse detected; session revoked.")
	ErrAuthSessionRevoked            AppError = NewAppError("E-AUTH-014", "Session has been revoked. Please log in again.")

	// Integration Errors
	ErrIntegrationNotConfigured AppError = NewAppError("E-INT-004", "Integration not configured.")
	ErrIntegrationNotFound      AppError = NewAppError("E-INT-005", "Integration not found.")

	// Audit Errors
	ErrAuditInvalidRequest AppError = NewAppError("E-AUDIT-001", "Invalid audit log query.")
	ErrAuditFetchError     AppError = NewAppError("E-AUDIT-002", "Error fetching audit logs.")

	// Notification Errors
	ErrNotificationNotFound    AppError = NewAppError("E-NOTIF-001", "Notification not found.")
	ErrNotificationFetchError  AppError = NewAppError("E-NOTIF-002", "Error fetching notifications.")
	ErrNotificationUpdateError AppError = NewAppError("E-NOTIF-003", "Error updating notification.")
	ErrNotificationDeleteError AppError = NewAppError("E-NOTIF-004", "Error deleting notification.")

	// Announcement Errors
	ErrAnnouncementNotFound       AppError = NewAppError("E-ANN-001", "Announcement not found.")
	ErrAnnouncementInvalidTarget  AppError = NewAppError("E-ANN-002", "Invalid announcement target.")
	ErrAnnouncementNotCancellable AppError = NewAppError("E-ANN-003", "Only scheduled announcements can be cancelled.")
	ErrAnnouncementDispatchError  AppError = NewAppError("E-ANN-004", "Error sending announcement.")
	ErrAnnouncementInvalidRequest AppError = NewAppError("E-ANN-005", "Invalid announcement data.")

	// Team Errors
	ErrTeamEnvironmentNotFound AppError = NewAppError("E-TEAM-001", "Team member not found in this company.")
	ErrTeamInvalidRole         AppError = NewAppError("E-TEAM-002", "Invalid or unsupported role.")
	ErrTeamLastAdmin           AppError = NewAppError("E-TEAM-003", "Cannot remove the last admin from this company.")
	ErrTeamAlreadyMember       AppError = NewAppError("E-TEAM-004", "User already belongs to this company.")
)
