package portal_handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	agenda_models "pengi-med-saas/features/agenda/models"
	agenda_services "pengi-med-saas/features/agenda/services"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// maxListedRequests bounds GET /appointment-requests?status=all.
const maxListedRequests = 200

// RequestHandler is the clinic's inbox of portal appointment requests
// (MANAGE_APPOINTMENT_REQUESTS): list, count, approve, reject.
type RequestHandler struct {
	bookingSide
}

func NewRequestHandler(db *gorm.DB, logger *zap.Logger, notices RequestNotifier, calendar CalendarSync) *RequestHandler {
	return &RequestHandler{bookingSide{db: db, logger: logger, notices: notices, calendar: calendar}}
}

// WithSyncAfterCommit runs notices and syncs before answering (tests).
func (h *RequestHandler) WithSyncAfterCommit() *RequestHandler {
	h.run = func(f func()) { f() }
	return h
}

func requestNotFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "portal.error.not_found", core_errors.ErrPortalNotFound)
}

// staffUserID is the signed-in user (AuthMiddleware stores it as int64).
func staffUserID(c *gin.Context) *uint {
	raw, ok := c.Get("user_id")
	if !ok {
		return nil
	}
	var id uint
	switch v := raw.(type) {
	case int64:
		id = uint(v)
	case uint:
		id = v
	case int:
		id = uint(v)
	}
	if id == 0 {
		return nil
	}
	return &id
}

// pendingScope is the requests still waiting for a decision.
func pendingScope(db *gorm.DB, now time.Time) *gorm.DB {
	return db.Where("status = ? AND expires_at > ?", portal_models.RequestPending, now.UTC())
}

// ListRequests is GET /appointment-requests?status=pending|all. Pending
// (default): the undecided, unexpired requests, soonest slot first. All: the
// latest maxListedRequests requests, newest first.
func (h *RequestHandler) ListRequests(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	now := timeNow()
	q := db.Model(&portal_models.AppointmentRequest{})
	switch c.DefaultQuery("status", "pending") {
	case "pending":
		q = pendingScope(q, now).Order("date ASC").Order("start_time ASC").Order("id ASC")
	case "all":
		q = q.Order("created_at DESC").Order("id DESC").Limit(maxListedRequests)
	default:
		return invalidData()
	}
	var reqs []portal_models.AppointmentRequest
	if err := q.Find(&reqs).Error; err != nil {
		h.logger.Error("appointment requests: failed to list", zap.Error(err))
		return internalError()
	}
	items, err := h.clinicItems(db, reqs, now)
	if err != nil {
		h.logger.Error("appointment requests: failed to load details", zap.Error(err))
		return internalError()
	}
	return envelope.SuccessResponse(items, "appointment_requests.list.success")
}

// PendingCount is GET /appointment-requests/pending-count.
func (h *RequestHandler) PendingCount(c *gin.Context) envelope.Response {
	var count int64
	if err := pendingScope(tenantdb.For(c, h.db).Model(&portal_models.AppointmentRequest{}), timeNow()).Count(&count).Error; err != nil {
		h.logger.Error("appointment requests: failed to count", zap.Error(err))
		return internalError()
	}
	return envelope.SuccessResponse(portal_dto.PendingCountResponse{Count: count}, "appointment_requests.count.success")
}

func (h *RequestHandler) clinicItems(db *gorm.DB, reqs []portal_models.AppointmentRequest, now time.Time) ([]portal_dto.ClinicRequest, error) {
	out := make([]portal_dto.ClinicRequest, 0, len(reqs))
	if len(reqs) == 0 {
		return out, nil
	}
	patientIDs, doctorIDs, typeIDs := []uint{}, []uint{}, []uint{}
	for _, r := range reqs {
		patientIDs = append(patientIDs, r.PatientID)
		doctorIDs = append(doctorIDs, r.DoctorID)
		typeIDs = append(typeIDs, r.AppointmentTypeID)
	}
	var patients []clinical_models.Patient
	if err := db.Unscoped().Select("id", "first_name", "last_name", "document", "phone").Where("id IN ?", patientIDs).Find(&patients).Error; err != nil {
		return nil, err
	}
	var doctors []doctor_models.Doctor
	if err := db.Unscoped().Where("id IN ?", doctorIDs).Find(&doctors).Error; err != nil {
		return nil, err
	}
	var types []agenda_models.AppointmentType
	if err := db.Unscoped().Select("id", "name").Where("id IN ?", typeIDs).Find(&types).Error; err != nil {
		return nil, err
	}
	pByID := map[uint]clinical_models.Patient{}
	for _, p := range patients {
		pByID[p.ID] = p
	}
	forms, matches, err := newPatientDetails(db, reqs)
	if err != nil {
		return nil, err
	}
	dByID := map[uint]doctor_models.Doctor{}
	for _, d := range doctors {
		dByID[d.ID] = d
	}
	tByID := map[uint]string{}
	for _, t := range types {
		tByID[t.ID] = t.Name
	}
	for _, r := range reqs {
		d := dByID[r.DoctorID]
		status := r.Status
		if status == portal_models.RequestPending && !r.ExpiresAt.After(now) {
			status = portal_models.RequestExpired
		}
		var patient *portal_dto.RequestPatient
		if p, ok := pByID[r.PatientID]; ok && r.PatientID != 0 {
			patient = &portal_dto.RequestPatient{ID: p.ID, FirstName: p.FirstName, LastName: p.LastName, Document: p.Document, Phone: p.Phone}
		}
		item := portal_dto.ClinicRequest{
			ID: r.ID, Status: status, Date: r.Date, StartTime: r.StartTime, EndTime: r.EndTime,
			Reason: r.Reason, RejectReason: r.RejectReason, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt,
			DecidedAt: r.DecidedAt, AppointmentID: r.AppointmentID, AppointmentTypeID: r.AppointmentTypeID,
			TypeName: tByID[r.AppointmentTypeID],
			Patient:  patient,
			Doctor:   portal_dto.ClinicRequestDoctor{ID: d.ID, FullName: d.FullName, Specialty: d.Specialty, SpecialtyOther: d.SpecialtyOther},
		}
		if form, ok := forms[r.ID]; ok {
			item.NewPatient = &portal_dto.NewPatientInfo{
				FirstName: form.FirstName, LastName: form.LastName, DocumentType: form.DocumentType, Document: form.Document,
				BirthDate: form.BirthDate, Phone: form.Phone, Email: form.Email, ContactChannel: r.ContactChannel,
			}
			if m, ok := matches[form.Document]; ok && r.PatientID == 0 {
				item.MatchingPatient = &portal_dto.MatchingPatient{ID: m.ID, FirstName: m.FirstName, LastName: m.LastName}
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// newPatientDetails decodes the forms of reqs' new-patient requests (by
// request id) and finds, for those not approved yet, the tenant's existing
// patient with the same document (by document; the oldest, like
// PatientByDocument).
func newPatientDetails(db *gorm.DB, reqs []portal_models.AppointmentRequest) (map[uint]portal_dto.NewPatientForm, map[string]clinical_models.Patient, error) {
	forms := map[uint]portal_dto.NewPatientForm{}
	matches := map[string]clinical_models.Patient{}
	documents := []string{}
	for _, r := range reqs {
		if !r.NewPatient {
			continue
		}
		var form portal_dto.NewPatientForm
		if err := json.Unmarshal([]byte(r.NewPatientData), &form); err != nil {
			continue
		}
		forms[r.ID] = form
		if r.PatientID == 0 && form.Document != "" {
			documents = append(documents, form.Document)
		}
	}
	if len(documents) == 0 {
		return forms, matches, nil
	}
	var patients []clinical_models.Patient
	if err := db.Select("id", "first_name", "last_name", "document").Where("document IN ?", documents).Order("id ASC").Find(&patients).Error; err != nil {
		return nil, nil, err
	}
	for _, p := range patients {
		if _, ok := matches[p.Document]; !ok {
			matches[p.Document] = p
		}
	}
	return forms, matches, nil
}

// item answers one request with message (the i18n key of what was done).
func (h *RequestHandler) item(db *gorm.DB, req portal_models.AppointmentRequest, message string) envelope.Response {
	items, err := h.clinicItems(db, []portal_models.AppointmentRequest{req}, timeNow())
	if err != nil {
		h.logger.Error("appointment requests: failed to load details", zap.Error(err))
		return internalError()
	}
	return envelope.SuccessResponse(items[0], message)
}

// load returns the tenant's request :id.
func (h *RequestHandler) load(c *gin.Context, db *gorm.DB) (portal_models.AppointmentRequest, *envelope.Response) {
	var req portal_models.AppointmentRequest
	id, ok := paramID(c, "id")
	if !ok {
		resp := requestNotFound()
		return req, &resp
	}
	if err := db.Where("id = ?", id).Limit(1).Find(&req).Error; err != nil {
		h.logger.Error("appointment requests: failed to load", zap.Error(err))
		resp := internalError()
		return req, &resp
	}
	if req.ID == 0 {
		resp := requestNotFound()
		return req, &resp
	}
	return req, nil
}

var errOverlap = errors.New("appointment overlap")

// Approve is POST /appointment-requests/:id/approve: the request must be
// pending and not expired, and the slot still free of appointments (staff
// may have booked over it: 409 E-CLIN-009). Creates the appointment. A
// new-patient request is first linked to the tenant's patient with the same
// document, or a patient is created from its form (ResolveNewPatient).
func (h *RequestHandler) Approve(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	tenantID := tenantdb.TenantID(c)
	req, errResp := h.load(c, db)
	if errResp != nil {
		return *errResp
	}
	now := timeNow()
	if req.Status != portal_models.RequestPending || !req.ExpiresAt.After(now) {
		return notPending()
	}
	var appt clinical_models.Appointment
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := portal_services.LockSlot(tx, tenantID, req.DoctorID, req.Date); err != nil {
			return err
		}
		overlaps, err := agenda_services.AppointmentOverlaps(tx, req.DoctorID, req.Date, req.StartTime, req.EndTime, portal_services.Location())
		if err != nil {
			return err
		}
		if overlaps {
			return errOverlap
		}
		if req.NewPatient && req.PatientID == 0 {
			if req.PatientID, err = portal_services.ResolveNewPatient(tx, req, now); err != nil {
				return err
			}
		}
		if appt, err = portal_services.CreateAppointment(tx, req, portal_services.TypeName(tx, req.AppointmentTypeID)); err != nil {
			return err
		}
		decided := now.UTC()
		res := pendingScope(tx.Model(&req), now).Updates(map[string]any{
			"status": portal_models.RequestApproved, "appointment_id": appt.ID, "patient_id": req.PatientID,
			"decided_by_user_id": staffUserID(c), "decided_at": decided, "updated_at": decided,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return portal_services.ErrNotPending
		}
		return nil
	})
	switch {
	case errors.Is(err, errOverlap):
		return envelope.ErrorResponse(http.StatusConflict, "appointments.overlap.error", core_errors.ErrClinicalAppointmentOverlap)
	case errors.Is(err, portal_services.ErrNotPending):
		return notPending()
	case err != nil:
		h.logger.Error("appointment requests: failed to approve", zap.Uint("id", req.ID), zap.Error(err))
		return internalError()
	}
	if err := db.Where("id = ?", req.ID).First(&req).Error; err != nil {
		h.logger.Error("appointment requests: failed to reload", zap.Error(err))
		return internalError()
	}
	h.approved(tenantID, req, appt)
	return h.item(db, req, "appointment_requests.approve.success")
}

// Reject is POST /appointment-requests/:id/reject {reason?}: the request
// must still be pending. The patient is told, with the reason.
func (h *RequestHandler) Reject(c *gin.Context) envelope.Response {
	var body portal_dto.RejectRequestBody
	if c.Request != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			return invalidData()
		}
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if utf8.RuneCountInString(body.Reason) > maxReasonRunes {
		return invalidData()
	}
	db := tenantdb.For(c, h.db)
	tenantID := tenantdb.TenantID(c)
	req, errResp := h.load(c, db)
	if errResp != nil {
		return *errResp
	}
	now := timeNow()
	if req.Status != portal_models.RequestPending || !req.ExpiresAt.After(now) {
		return notPending()
	}
	// pendingScope: a request past its expiry is the worker's to mark expired
	// (the patient gets the "expired" notice, not "rejected").
	res := pendingScope(db.Model(&req), now).Updates(map[string]any{
		"status": portal_models.RequestRejected, "reject_reason": body.Reason,
		"decided_by_user_id": staffUserID(c), "decided_at": now.UTC(), "updated_at": now.UTC(),
	})
	if res.Error != nil {
		h.logger.Error("appointment requests: failed to reject", zap.Uint("id", req.ID), zap.Error(res.Error))
		return internalError()
	}
	if res.RowsAffected == 0 {
		return notPending()
	}
	if err := db.Where("id = ?", req.ID).First(&req).Error; err != nil {
		h.logger.Error("appointment requests: failed to reload", zap.Error(err))
		return internalError()
	}
	if h.notices != nil {
		h.afterCommit(func() { h.notices.SendRequestNotice(tenantID, req.ID) })
	}
	return h.item(db, req, "appointment_requests.reject.success")
}
