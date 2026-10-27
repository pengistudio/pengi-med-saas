package portal_handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FirstAppointmentHandler serves first appointments of people without a patient
// record: verifying a contact with a one-time code (public, per clinic) and,
// with the guest token that gives, requesting the appointment. The patient
// record is created or linked only when the clinic approves. Nothing it
// answers depends on whether the document exists in the clinic.
type FirstAppointmentHandler struct {
	bookingSide
	delivery *portal_services.Delivery
	// run starts the code delivery (a goroutine in production, like Login).
	runCode func(func())
	captcha portal_services.CaptchaVerifier
}

func NewFirstAppointmentHandler(db *gorm.DB, logger *zap.Logger, delivery *portal_services.Delivery) *FirstAppointmentHandler {
	return &FirstAppointmentHandler{
		bookingSide: bookingSide{db: db, logger: logger},
		delivery:    delivery,
		runCode:     func(f func()) { go f() },
		captcha:     portal_services.TurnstileFromConfig(),
	}
}

// WithCaptcha replaces the captcha verifier (tests); nil disables it.
func (h *FirstAppointmentHandler) WithCaptcha(v portal_services.CaptchaVerifier) *FirstAppointmentHandler {
	h.captcha = v
	return h
}

// WithSync delivers codes and runs after-commit work before answering (tests).
func (h *FirstAppointmentHandler) WithSync() *FirstAppointmentHandler {
	h.runCode = func(f func()) { f() }
	h.run = func(f func()) { f() }
	return h
}

// openClinic resolves the :slug clinic and checks it takes new patients.
func (h *FirstAppointmentHandler) openClinic(c *gin.Context) (portal_services.Clinic, envelope.Response, bool) {
	clinic, err := portal_services.ClinicBySlug(h.db, c.Param("slug"))
	if err != nil {
		if errors.Is(err, portal_services.ErrUnavailable) {
			return clinic, portal_middleware.NotAvailable(), false
		}
		h.logger.Error("portal: failed to resolve clinic", zap.Error(err))
		return clinic, internalError(), false
	}
	if !clinic.Settings.NewPatientsEnabled() {
		return clinic, portal_middleware.NewPatientsNotAvailable(), false
	}
	return clinic, envelope.Response{}, true
}

// RequestCode is POST /portal/clinics/:slug/new-patient/code. A malformed
// contact, or WhatsApp when the clinic can't send codes by it, answers 400
// E-PORTAL-005; a rejected captcha 400 E-PORTAL-008. Otherwise it always
// answers the same (expires_in): a spent budget or a filled honeypot
// included. The code is issued and sent in the background.
func (h *FirstAppointmentHandler) RequestCode(c *gin.Context) envelope.Response {
	clinic, resp, ok := h.openClinic(c)
	if !ok {
		return resp
	}
	var req portal_dto.NewPatientCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidData()
	}
	contact, err := portal_services.NormalizeContact(req.Channel, req.Contact)
	if err != nil {
		return invalidData()
	}
	if req.Channel == portal_models.ChannelWhatsApp && !h.delivery.WhatsAppReady(clinic.Tenant.ID) {
		return invalidData()
	}
	if h.captcha != nil {
		ok, err := h.captcha.Verify(c.Request.Context(), req.CaptchaToken, c.ClientIP())
		if err != nil {
			h.logger.Warn("portal: captcha verification failed", zap.Uint("tenant_id", clinic.Tenant.ID), zap.Error(err))
		}
		if err != nil || !ok {
			return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.captcha_failed", core_errors.ErrPortalCaptchaFailed)
		}
	}
	answer := envelope.SuccessResponse(portal_dto.LoginResponse{ExpiresIn: int(portal_services.CodeTTL.Seconds())}, "portal.new_patient.code_sent")
	if strings.TrimSpace(req.Website) != "" {
		return answer // honeypot: a bot
	}
	tr := translator(c)
	channel := req.Channel
	h.runCode(func() {
		err := h.delivery.SendContactCode(clinic, channel, contact, tr)
		var limit *portal_services.LimitError
		switch {
		case errors.As(err, &limit) && limit.PatientBudget():
			h.logger.Info("portal: contact code limit reached", zap.Uint("tenant_id", clinic.Tenant.ID), zap.String("budget", limit.Budget))
		case errors.As(err, &limit):
			h.logger.Warn("portal: code budget reached, nothing sent", zap.Uint("tenant_id", clinic.Tenant.ID), zap.String("budget", limit.Budget))
			if limit.TenantBudget() {
				portal_services.NotifyCodeBudget(h.db, h.logger, clinic.Tenant.ID, limit.Budget, timeNow())
			}
		case err != nil:
			h.logger.Warn("portal: contact code not sent", zap.Uint("tenant_id", clinic.Tenant.ID), zap.String("channel", channel), zap.Error(err))
		}
	})
	return answer
}

// Verify is POST /portal/clinics/:slug/new-patient/verify: a valid code
// gives a guest token (30 minutes) for that contact. Every failure answers
// 401 E-PORTAL-002.
func (h *FirstAppointmentHandler) Verify(c *gin.Context) envelope.Response {
	clinic, resp, ok := h.openClinic(c)
	if !ok {
		return resp
	}
	var req portal_dto.NewPatientVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return invalidCode()
	}
	contact, err := portal_services.NormalizeContact(req.Channel, req.Contact)
	if err != nil {
		return invalidCode()
	}
	if err := portal_services.VerifyContactCode(h.db, clinic.Tenant.ID, req.Channel, contact, req.Code, timeNow()); err != nil {
		if !errors.Is(err, portal_services.ErrInvalidCode) {
			h.logger.Error("portal: failed to verify contact code", zap.Error(err))
			return internalError()
		}
		return invalidCode()
	}
	token, err := auth.GeneratePortalGuestToken(clinic.Tenant.ID, req.Channel, contact)
	if err != nil {
		h.logger.Error("portal: failed to sign guest token", zap.Error(err))
		return internalError()
	}
	return envelope.SuccessResponse(portal_dto.GuestTokenResponse{
		GuestToken: token, ExpiresIn: int(auth.PortalGuestTTL.Seconds()),
	}, "portal.new_patient.verify.success")
}

// errAlreadyPending / errClinicFull end a new-patient request's transaction.
var (
	errAlreadyPending = errors.New("new patient already has a pending request")
	errClinicFull     = errors.New("too many pending new-patient requests")
)

// CreateRequest is POST /portal/new-patient/requests (guest token). The
// person's data is validated (400 E-PORTAL-016; the verified contact must be
// in it) and the slot placed like POST /portal/booking/requests, always
// pending: one pending new-patient request per verified contact and per
// document (409 E-PORTAL-017), MaxPendingNewPatientRequests per clinic (429
// E-PORTAL-018).
func (h *FirstAppointmentHandler) CreateRequest(c *gin.Context) envelope.Response {
	clinic := portal_middleware.Clinic(c)
	guest, ok := portal_middleware.Guest(c)
	if !ok {
		return envelope.ErrorResponse(http.StatusUnauthorized, "portal.error.unauthorized", core_errors.ErrPortalUnauthorized)
	}
	tenantID := clinic.Tenant.ID
	booking := clinic.Settings.Booking
	db := tenantdb.ForTenant(h.db, tenantID)
	var body portal_dto.NewPatientRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return invalidData()
	}
	now := timeNow()
	form, err := portal_services.NormalizeNewPatientForm(body.Patient, guest.Channel, guest.Contact, now)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.invalid_personal_data", core_errors.ErrPortalInvalidPersonalData)
	}
	sr, resp, ok := h.prepareRequest(db, booking, tenantID, body.CreateRequestBody, now)
	if !ok {
		return resp
	}
	data, err := json.Marshal(form)
	if err != nil {
		return internalError()
	}
	sr.req.PatientID = 0
	sr.req.NewPatient, sr.req.NewPatientData = true, string(data)
	sr.req.ContactChannel, sr.req.ContactValue = guest.Channel, guest.Contact

	// Always pending, even when the clinic auto-confirms: the clinic checks
	// who the person is before a record is created.
	_, err = h.placeRequest(db, booking, &sr, false, now, func(tx *gorm.DB) error {
		if err := portal_services.LockNewPatients(tx, tenantID); err != nil {
			return err
		}
		pending, forms, err := portal_services.PendingNewPatientForms(tx, now)
		if err != nil {
			return err
		}
		for i, r := range pending {
			if (r.ContactChannel == guest.Channel && r.ContactValue == guest.Contact) || forms[i].Document == form.Document {
				return errAlreadyPending
			}
		}
		if len(pending) >= portal_services.MaxPendingNewPatientRequests {
			return errClinicFull
		}
		return nil
	})
	switch {
	case errors.Is(err, errAlreadyPending):
		return envelope.ErrorResponse(http.StatusConflict, "portal.error.request_already_pending", core_errors.ErrPortalRequestAlreadyPending)
	case errors.Is(err, errClinicFull):
		h.logger.Warn("portal: new-patient request cap reached", zap.Uint("tenant_id", tenantID), zap.Int("max", portal_services.MaxPendingNewPatientRequests))
		return envelope.ErrorResponse(http.StatusTooManyRequests, "portal.error.too_many_new_patient_requests", core_errors.ErrPortalTooManyNewPatients)
	case errors.Is(err, errSlotTaken):
		return slotTaken()
	case err != nil:
		h.logger.Error("portal: failed to create new-patient request", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return internalError()
	}
	req := sr.req

	audit.RecordPortalGuestChange(h.db, tenantID, "CREATE", auditRequests, req.ID)
	named := clinical_models.Patient{FirstName: form.FirstName, LastName: form.LastName}
	h.afterCommit(func() {
		portal_services.NotifyStaff(h.db, h.logger, tenantID, portal_services.NotificationTypeBookingRequest,
			"notification.portal.booking.new_patient_request", portal_services.RequestResourceType, req.ID,
			portal_services.RequestsURL, named, req.Date, req.StartTime)
	})
	item := (&BookingHandler{h.bookingSide}).requestItem(db, req, now)
	return envelope.SuccessResponse(portal_dto.NewPatientRequestResponse{
		ID: item.ID, Status: item.Status, Date: item.Date, StartTime: item.StartTime, EndTime: item.EndTime,
		Doctor: item.Doctor, TypeName: item.TypeName, ExpiresAt: item.ExpiresAt, ContactChannel: req.ContactChannel,
	}, "portal.new_patient.request.create.success")
}
