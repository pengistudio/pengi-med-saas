// Package portal_middleware authenticates patient portal requests.
package portal_middleware

import (
	"errors"
	"net/http"
	"strings"

	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_services "pengi-med-saas/features/portal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Context keys set by RequirePortalSession. They are deliberately not the
// staff keys (tenant_id, user_id): nothing staff-side can mistake a portal
// request for a clinic session.
const (
	ContextClinic  = "portal_clinic"
	ContextPatient = "portal_patient"
)

// Sections of the portal (RequireSection).
const (
	SectionAppointments = "appointments"
	SectionResults      = "results"
	SectionDocuments    = "documents"
	SectionInvoices     = "invoices"
)

func unauthorized() envelope.Response {
	return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.unauthorized", core_errors.ErrPortalUnauthorized)
}

// NotAvailable is the answer when the clinic's portal is closed, the clinic
// does not exist or a section is hidden.
func NotAvailable() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "portal.error.not_available", core_errors.ErrPortalNotAvailable)
}

// RequirePortalSession accepts only portal access tokens (any staff, backoffice,
// refresh or other token is rejected), checks the clinic's portal is still
// open and the patient still exists, and puts both in the context.
func RequirePortalSession(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if !strings.HasPrefix(header, "Bearer ") || token == "" {
			envelope.Abort(c, unauthorized())
			return
		}
		claims, err := auth.ParsePortalAccessToken(token)
		if err != nil {
			envelope.Abort(c, unauthorized())
			return
		}
		clinic, err := portal_services.ClinicByID(db, claims.TenantID)
		if err != nil {
			if errors.Is(err, portal_services.ErrUnavailable) {
				envelope.Abort(c, NotAvailable())
				return
			}
			envelope.Abort(c, envelope.ErrorResponse(http.StatusInternalServerError, "portal.error.internal", core_errors.ErrPortalInternal))
			return
		}
		patient, found, err := portal_services.PatientByID(db, claims.TenantID, claims.PatientID)
		if err != nil {
			envelope.Abort(c, envelope.ErrorResponse(http.StatusInternalServerError, "portal.error.internal", core_errors.ErrPortalInternal))
			return
		}
		if !found {
			envelope.Abort(c, unauthorized())
			return
		}
		c.Set(ContextClinic, clinic)
		c.Set(ContextPatient, patient)
		c.Next()
	}
}

// RequireSection answers 404 E-PORTAL-001 when the clinic hides section.
func RequireSection(section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !SectionEnabled(Clinic(c), section) {
			envelope.Abort(c, NotAvailable())
			return
		}
		c.Next()
	}
}

// SectionEnabled reports whether the clinic shows section.
func SectionEnabled(clinic portal_services.Clinic, section string) bool {
	s := clinic.Settings.Sections
	switch section {
	case SectionAppointments:
		return s.Appointments
	case SectionResults:
		return s.Results
	case SectionDocuments:
		return s.Documents
	case SectionInvoices:
		return s.Invoices
	}
	return false
}

// Clinic is the session's clinic (set by RequirePortalSession).
func Clinic(c *gin.Context) portal_services.Clinic {
	clinic, _ := c.Get(ContextClinic)
	v, _ := clinic.(portal_services.Clinic)
	return v
}

// Patient is the session's patient (set by RequirePortalSession).
func Patient(c *gin.Context) clinical_models.Patient {
	patient, _ := c.Get(ContextPatient)
	v, _ := patient.(clinical_models.Patient)
	return v
}
