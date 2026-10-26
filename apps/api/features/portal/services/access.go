// Package portal_services holds the patient portal's rules: when a clinic's
// portal is open, the one-time sign-in codes and their delivery.
package portal_services

import (
	"errors"
	"strings"

	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	"gorm.io/gorm"
)

// ErrUnavailable: the clinic does not exist, its plan does not include the
// patient portal, or it did not enable it. Callers answer all three the same.
var ErrUnavailable = errors.New("patient portal not available")

// Clinic is a tenant whose patient portal is open.
type Clinic struct {
	Tenant   tenant_models.Tenant
	Settings tenant_models.PortalSettings
}

// HasLogo reports whether the clinic uploaded a logo.
func (c Clinic) HasLogo() bool { return c.Tenant.LogoPath != nil && *c.Tenant.LogoPath != "" }

// ClinicBySlug returns the clinic with slug if its portal is open.
func ClinicBySlug(db *gorm.DB, slug string) (Clinic, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Clinic{}, ErrUnavailable
	}
	return openClinic(db, "slug = ?", slug)
}

// ClinicByID returns the tenant id's clinic if its portal is open.
func ClinicByID(db *gorm.DB, tenantID uint) (Clinic, error) {
	if tenantID == 0 {
		return Clinic{}, ErrUnavailable
	}
	return openClinic(db, "id = ?", tenantID)
}

func openClinic(db *gorm.DB, where string, arg any) (Clinic, error) {
	var tenant tenant_models.Tenant
	if err := tenantdb.System(db).Where(where, arg).Limit(1).Find(&tenant).Error; err != nil {
		return Clinic{}, err
	}
	if tenant.ID == 0 {
		return Clinic{}, ErrUnavailable
	}
	settings := tenant.ParsePortalSettings()
	if !settings.Enabled {
		return Clinic{}, ErrUnavailable
	}
	included, err := PlanIncludesPortal(db, tenant.ID)
	if err != nil {
		return Clinic{}, err
	}
	if !included {
		return Clinic{}, ErrUnavailable
	}
	return Clinic{Tenant: tenant, Settings: settings}, nil
}

// PlanIncludesPortal reports whether the tenant's active subscription (grace
// period included, like SubscriptionMiddleware) has a plan whose Features
// carry a PATIENT_PORTAL permission. No company or subscription: false.
func PlanIncludesPortal(db *gorm.DB, tenantID uint) (bool, error) {
	var company company_models.Company
	if err := tenantdb.ForTenant(db, tenantID).Limit(1).Find(&company).Error; err != nil {
		return false, err
	}
	if company.ID == 0 {
		return false, nil
	}
	var subscription company_models.Subscription
	err := company_services.ActiveSubscriptionQuery(tenantdb.System(db), company.ID).
		Preload("Plan.Features.Permissions").
		First(&subscription).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return company_services.CalculateEnabledFeaturesFromFeatures(subscription.Plan.Features).PatientPortal, nil
}

// PatientByDocument finds the tenant's patient with exactly this document
// (trimmed). Duplicated records resolve to the oldest one, the same for login
// and verification. Found is false for an empty document.
func PatientByDocument(db *gorm.DB, tenantID uint, document string) (clinical_models.Patient, bool, error) {
	var patient clinical_models.Patient
	document = strings.TrimSpace(document)
	if document == "" {
		return patient, false, nil
	}
	err := tenantdb.ForTenant(db, tenantID).Where("document = ?", document).Order("id ASC").Limit(1).Find(&patient).Error
	if err != nil {
		return patient, false, err
	}
	return patient, patient.ID != 0, nil
}

// PatientByID loads the tenant's patient id (soft-deleted ones are not found).
func PatientByID(db *gorm.DB, tenantID, patientID uint) (clinical_models.Patient, bool, error) {
	var patient clinical_models.Patient
	err := tenantdb.ForTenant(db, tenantID).Where("id = ?", patientID).Limit(1).Find(&patient).Error
	return patient, err == nil && patient.ID != 0, err
}
