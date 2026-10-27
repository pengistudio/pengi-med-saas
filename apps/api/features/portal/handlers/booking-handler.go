package portal_handlers

import (
	"errors"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	agenda_models "pengi-med-saas/features/agenda/models"
	agenda_services "pengi-med-saas/features/agenda/services"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_dto "pengi-med-saas/features/portal/dto"
	portal_middleware "pengi-med-saas/features/portal/middleware"
	portal_models "pengi-med-saas/features/portal/models"
	portal_services "pengi-med-saas/features/portal/services"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// maxReasonRunes bounds the patient's note and the clinic's reject reason.
const maxReasonRunes = 500

// Audit entity types of booking actions.
const (
	auditRequests     = "appointment_requests"
	auditAppointments = "appointments"
)

// CalendarSync mirrors appointments to Google Calendar
// (*clinical_handlers.AppointmentHandler in production).
type CalendarSync interface {
	SyncCreated(tenantID uint, appointment *clinical_models.Appointment)
	SyncStatusChange(tenantID uint, appointment *clinical_models.Appointment)
}

// RequestNotifier tells the patient how a request ended
// (*portal_services.Delivery in production).
type RequestNotifier interface {
	SendRequestNotice(tenantID, requestID uint) string
}

// bookingSide holds what the portal and clinic booking handlers share.
type bookingSide struct {
	db       *gorm.DB
	logger   *zap.Logger
	notices  RequestNotifier
	calendar CalendarSync
	// run starts the after-commit work (notices, calendar, staff
	// notifications); a goroutine in production, inline in tests.
	run func(func())
}

func (b *bookingSide) afterCommit(f func()) {
	if b.run == nil {
		go f()
		return
	}
	b.run(f)
}

// approved runs the after-commit work of an approved request: the calendar
// event and the patient's notice.
func (b *bookingSide) approved(tenantID uint, req portal_models.AppointmentRequest, appt clinical_models.Appointment) {
	b.afterCommit(func() {
		if b.calendar != nil {
			db := tenantdb.ForTenant(b.db, tenantID)
			if err := db.Preload("Patient").Preload("Doctor").First(&appt, appt.ID).Error; err == nil {
				b.calendar.SyncCreated(tenantID, &appt)
			}
		}
		if b.notices != nil {
			b.notices.SendRequestNotice(tenantID, req.ID)
		}
	})
}

// BookingHandler serves booking from the portal: options, free slots, the
// patient's requests and cancelling. Like PatientHandler, every query is
// bound to the session's tenant and patient.
type BookingHandler struct {
	bookingSide
}

func NewBookingHandler(db *gorm.DB, logger *zap.Logger, notices RequestNotifier, calendar CalendarSync) *BookingHandler {
	return &BookingHandler{bookingSide{db: db, logger: logger, notices: notices, calendar: calendar}}
}

// WithSyncAfterCommit runs notices and syncs before answering (tests).
func (h *BookingHandler) WithSyncAfterCommit() *BookingHandler {
	h.run = func(f func()) { f() }
	return h
}

func (h *BookingHandler) session(c *gin.Context) (*gorm.DB, portal_services.Clinic, clinical_models.Patient) {
	clinic := portal_middleware.Clinic(c)
	return tenantdb.ForTenant(h.db, clinic.Tenant.ID), clinic, portal_middleware.Patient(c)
}

func invalidData() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "portal.error.invalid_request", core_errors.ErrPortalInvalidData)
}

func slotTaken() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "portal.error.slot_taken", core_errors.ErrPortalSlotTaken)
}

func notPending() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "portal.error.request_not_pending", core_errors.ErrPortalRequestNotPending)
}

// catalog is the bookable doctors and types of the clinic.
type catalog struct {
	doctors []doctor_models.Doctor
	types   []agenda_models.AppointmentType
}

func loadCatalog(db *gorm.DB) (catalog, error) {
	doctors, err := portal_services.BookableDoctors(db)
	if err != nil {
		return catalog{}, err
	}
	types, err := portal_services.BookableTypes(db)
	if err != nil {
		return catalog{}, err
	}
	return catalog{doctors: doctors, types: types}, nil
}

func (k catalog) typeByID(id uint) (agenda_models.AppointmentType, bool) {
	for _, t := range k.types {
		if t.ID == id {
			return t, true
		}
	}
	return agenda_models.AppointmentType{}, false
}

// candidates are the bookable doctors a request may go to: the chosen doctor
// (doctor mode) or every doctor of the specialty (specialty mode), always
// attending typ.
func (k catalog) candidates(mode string, typ agenda_models.AppointmentType, doctorID uint, specialty, other string) []doctor_models.Doctor {
	var out []doctor_models.Doctor
	key := portal_services.SpecialtyKey(specialty, other)
	for _, d := range k.doctors {
		if !typ.AllowsDoctor(d.ID) {
			continue
		}
		if mode == tenant_models.BookingChooseBySpecialty {
			if specialty != "" && portal_services.SpecialtyKey(d.Specialty, d.SpecialtyOther) == key {
				out = append(out, d)
			}
		} else if d.ID == doctorID {
			out = append(out, d)
		}
	}
	return out
}

// GetOptions is GET /portal/booking/options.
func (h *BookingHandler) GetOptions(c *gin.Context) envelope.Response {
	db, clinic, _ := h.session(c)
	k, err := loadCatalog(db)
	if err != nil {
		h.logger.Error("portal booking: failed to load options", zap.Error(err))
		return internalError()
	}
	booking := clinic.Settings.Booking
	minDate, maxDate := portal_services.Window(timeNow(), booking)
	out := portal_dto.BookingOptions{
		ChooseBy: booking.ChooseBy, MinDate: minDate, MaxDate: maxDate,
		Doctors: []portal_dto.BookingDoctor{}, Specialties: []portal_dto.BookingSpecialty{}, Types: []portal_dto.BookingType{},
	}
	usedTypes := map[uint]bool{}
	specialtyIndex := map[string]int{}
	for _, d := range k.doctors {
		typeIDs := []uint{}
		for _, t := range k.types {
			if t.AllowsDoctor(d.ID) {
				typeIDs = append(typeIDs, t.ID)
				usedTypes[t.ID] = true
			}
		}
		if len(typeIDs) == 0 {
			continue
		}
		if booking.ChooseBy != tenant_models.BookingChooseBySpecialty {
			out.Doctors = append(out.Doctors, portal_dto.BookingDoctor{
				ID: d.ID, FullName: d.FullName, Specialty: d.Specialty, SpecialtyOther: d.SpecialtyOther, TypeIDs: typeIDs,
			})
		}
		key := portal_services.SpecialtyKey(d.Specialty, d.SpecialtyOther)
		i, ok := specialtyIndex[key]
		if !ok {
			other := ""
			if d.Specialty == "other" {
				other = strings.TrimSpace(d.SpecialtyOther)
			}
			out.Specialties = append(out.Specialties, portal_dto.BookingSpecialty{Specialty: d.Specialty, SpecialtyOther: other, TypeIDs: []uint{}})
			i = len(out.Specialties) - 1
			specialtyIndex[key] = i
		}
		out.Specialties[i].TypeIDs = mergeIDs(out.Specialties[i].TypeIDs, typeIDs)
	}
	for _, t := range k.types {
		if usedTypes[t.ID] {
			out.Types = append(out.Types, portal_dto.BookingType{ID: t.ID, Name: t.Name, DurationMinutes: t.DurationMinutes})
		}
	}
	return envelope.SuccessResponse(out, "portal.booking.options.success")
}

func mergeIDs(a, b []uint) []uint {
	seen := map[uint]bool{}
	for _, id := range a {
		seen[id] = true
	}
	for _, id := range b {
		if !seen[id] {
			a = append(a, id)
			seen[id] = true
		}
	}
	sort.Slice(a, func(i, j int) bool { return a[i] < a[j] })
	return a
}

// GetSlots is GET /portal/booking/slots?type_id=&doctor_id= (or
// &specialty=&specialty_other=) &from=&to=: the free start times, at most
// MaxSlotDays days, clamped to the booking window. In specialty mode the
// times of every matching doctor are merged.
func (h *BookingHandler) GetSlots(c *gin.Context) envelope.Response {
	db, clinic, _ := h.session(c)
	booking := clinic.Settings.Booking
	k, err := loadCatalog(db)
	if err != nil {
		h.logger.Error("portal booking: failed to load options", zap.Error(err))
		return internalError()
	}
	typeID, _ := parseUint(c.Query("type_id"))
	typ, ok := k.typeByID(typeID)
	if !ok {
		return invalidData()
	}
	doctorID, _ := parseUint(c.Query("doctor_id"))
	doctors := k.candidates(booking.ChooseBy, typ, doctorID, c.Query("specialty"), c.Query("specialty_other"))
	if len(doctors) == 0 {
		return invalidData()
	}

	now := timeNow()
	minDate, maxDate := portal_services.Window(now, booking)
	from, to := c.DefaultQuery("from", minDate), c.Query("to")
	if _, ok := agenda_services.ParseDate(from); !ok {
		return invalidData()
	}
	if from < minDate {
		from = minDate
	}
	f, _ := agenda_services.ParseDate(from)
	limit := f.AddDate(0, 0, portal_services.MaxSlotDays-1).Format(agenda_services.DateLayout)
	if to == "" || to > limit {
		to = limit
	} else if _, ok := agenda_services.ParseDate(to); !ok {
		return invalidData()
	}
	if to > maxDate {
		to = maxDate
	}

	out := portal_dto.SlotsResponse{Days: []portal_dto.SlotDay{}}
	if from > to {
		return envelope.SuccessResponse(out, "portal.booking.slots.success")
	}
	ids := make([]uint, 0, len(doctors))
	for _, d := range doctors {
		ids = append(ids, d.ID)
	}
	free, err := agenda_services.FreeSlots(db, agenda_services.SlotQuery{
		DoctorIDs: ids, Duration: typ.DurationMinutes, Step: booking.SlotIntervalMinutes, From: from, To: to, Now: now, Loc: portal_services.Location(),
		MinNotice: time.Duration(booking.MinNoticeHours) * time.Hour,
	})
	if err != nil {
		h.logger.Error("portal booking: failed to compute slots", zap.Error(err))
		return internalError()
	}
	byDate := map[string]map[string]bool{}
	for _, days := range free {
		for date, times := range days {
			if byDate[date] == nil {
				byDate[date] = map[string]bool{}
			}
			for _, t := range times {
				byDate[date][t] = true
			}
		}
	}
	for date, set := range byDate {
		times := make([]string, 0, len(set))
		for t := range set {
			times = append(times, t)
		}
		sort.Strings(times)
		out.Days = append(out.Days, portal_dto.SlotDay{Date: date, Times: times})
	}
	sort.Slice(out.Days, func(i, j int) bool { return out.Days[i].Date < out.Days[j].Date })
	return envelope.SuccessResponse(out, "portal.booking.slots.success")
}

// errSlotTaken / errTooMany end the create transaction with a 409.
var (
	errSlotTaken = errors.New("slot taken")
	errTooMany   = errors.New("too many pending requests")
)

// slotRequest is a request for a slot checked against the clinic's catalog
// and booking window, ready to be placed.
type slotRequest struct {
	typ     agenda_models.AppointmentType
	doctors []doctor_models.Doctor // candidates, any order
	req     portal_models.AppointmentRequest
}

// prepareRequest checks body against the clinic's window and catalog (db
// bound to the tenant) and drafts the pending request. ok is false (answer
// invalidData, or internalError when resp says so) when it can't be placed.
func (b *bookingSide) prepareRequest(db *gorm.DB, booking tenant_models.PortalBooking, tenantID uint, body portal_dto.CreateRequestBody, now time.Time) (slotRequest, envelope.Response, bool) {
	body.Reason = strings.TrimSpace(body.Reason)
	if utf8.RuneCountInString(body.Reason) > maxReasonRunes {
		return slotRequest{}, invalidData(), false
	}
	minDate, maxDate := portal_services.Window(now, booking)
	if _, ok := agenda_services.ParseDate(body.Date); !ok || body.Date < minDate || body.Date > maxDate {
		return slotRequest{}, invalidData(), false
	}
	startMin, ok := agenda_services.ParseClock(body.StartTime, 5)
	if !ok {
		return slotRequest{}, invalidData(), false
	}
	k, err := loadCatalog(db)
	if err != nil {
		b.logger.Error("portal booking: failed to load options", zap.Error(err))
		return slotRequest{}, internalError(), false
	}
	typ, ok := k.typeByID(body.TypeID)
	if !ok {
		return slotRequest{}, invalidData(), false
	}
	endMin := startMin + typ.DurationMinutes
	if endMin > 24*60 {
		return slotRequest{}, invalidData(), false
	}
	doctors := k.candidates(booking.ChooseBy, typ, body.DoctorID, body.Specialty, body.SpecialtyOther)
	if len(doctors) == 0 {
		return slotRequest{}, invalidData(), false
	}
	start, _ := portal_services.SlotStart(body.Date, body.StartTime)
	return slotRequest{typ: typ, doctors: doctors, req: portal_models.AppointmentRequest{
		TenantID: tenantID, AppointmentTypeID: typ.ID, Date: body.Date,
		StartTime: body.StartTime, EndTime: agenda_services.FormatClock(endMin), Reason: body.Reason,
		Status: portal_models.RequestPending, ExpiresAt: portal_services.ExpiresAt(now, start, booking.RequestExpiryHours),
	}}, envelope.Response{}, true
}

// placeRequest stores sr.req in a transaction: first limits (the caller's
// locks and caps; its error ends the transaction), then it locks (tenant,
// doctor, date) of every candidate, re-checks the slot is free and assigns
// the first free doctor (errSlotTaken if none). With autoConfirm the request
// is stored approved with its appointment.
func (b *bookingSide) placeRequest(db *gorm.DB, booking tenant_models.PortalBooking, sr *slotRequest, autoConfirm bool, now time.Time,
	limits func(tx *gorm.DB) error) (clinical_models.Appointment, error) {
	req := &sr.req
	var appt clinical_models.Appointment
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := limits(tx); err != nil {
			return err
		}
		ranked := sr.doctors
		if len(ranked) > 1 {
			var err error
			if ranked, err = portal_services.RankBySchedule(tx, sr.doctors, req.Date); err != nil {
				return err
			}
		}
		// Lock every candidate in id order before checking any, so two
		// requests that rank the doctors differently can't deadlock.
		lockIDs := make([]uint, 0, len(ranked))
		for _, d := range ranked {
			lockIDs = append(lockIDs, d.ID)
		}
		slices.Sort(lockIDs)
		for _, id := range lockIDs {
			if err := portal_services.LockSlot(tx, req.TenantID, id, req.Date); err != nil {
				return err
			}
		}
		for _, d := range ranked {
			free, err := agenda_services.FreeSlots(tx, agenda_services.SlotQuery{
				DoctorIDs: []uint{d.ID}, Duration: sr.typ.DurationMinutes, Step: booking.SlotIntervalMinutes, From: req.Date, To: req.Date, Now: now,
				Loc: portal_services.Location(), MinNotice: time.Duration(booking.MinNoticeHours) * time.Hour,
			})
			if err != nil {
				return err
			}
			if free.Has(d.ID, req.Date, req.StartTime) {
				req.DoctorID = d.ID
				break
			}
		}
		if req.DoctorID == 0 {
			return errSlotTaken
		}
		if autoConfirm {
			created, err := portal_services.CreateAppointment(tx, *req, sr.typ.Name)
			if err != nil {
				return err
			}
			appt = created
			decided := now.UTC()
			req.Status, req.AppointmentID, req.DecidedAt = portal_models.RequestApproved, &appt.ID, &decided
		}
		return tx.Create(req).Error
	})
	return appt, err
}

// CreateRequest is POST /portal/booking/requests. Inside a transaction it
// locks (tenant, doctor, date), re-checks the slot is free and stores the
// request: pending (holding the slot until it expires), or approved with its
// appointment when the clinic auto-confirms.
func (h *BookingHandler) CreateRequest(c *gin.Context) envelope.Response {
	db, clinic, patient := h.session(c)
	booking := clinic.Settings.Booking
	tenantID := clinic.Tenant.ID
	var body portal_dto.CreateRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return invalidData()
	}
	now := timeNow()
	sr, resp, ok := h.prepareRequest(db, booking, tenantID, body, now)
	if !ok {
		return resp
	}
	sr.req.PatientID = patient.ID
	appt, err := h.placeRequest(db, booking, &sr, booking.AutoConfirm, now, func(tx *gorm.DB) error {
		if err := portal_services.LockPatient(tx, tenantID, patient.ID); err != nil {
			return err
		}
		var pending int64
		if err := tx.Model(&portal_models.AppointmentRequest{}).
			Where("patient_id = ? AND status = ? AND expires_at > ?", patient.ID, portal_models.RequestPending, now.UTC()).
			Count(&pending).Error; err != nil {
			return err
		}
		if pending >= portal_services.MaxPendingRequests {
			return errTooMany
		}
		return nil
	})
	switch {
	case errors.Is(err, errTooMany):
		return envelope.ErrorResponse(http.StatusConflict, "portal.error.too_many_requests", core_errors.ErrPortalTooManyRequests)
	case errors.Is(err, errSlotTaken):
		return slotTaken()
	case err != nil:
		h.logger.Error("portal booking: failed to create request", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return internalError()
	}
	req := sr.req

	audit.RecordPortalChange(h.db, tenantID, patient.ID, "CREATE", auditRequests, req.ID)
	if appt.ID != 0 {
		audit.RecordPortalChange(h.db, tenantID, patient.ID, "CREATE", auditAppointments, appt.ID)
		h.approved(tenantID, req, appt)
		h.afterCommit(func() {
			portal_services.NotifyStaff(h.db, h.logger, tenantID, portal_services.NotificationTypeBookingBooked,
				"notification.portal.booking.booked", portal_services.AppointmentResourceType, appt.ID,
				portal_services.AppointmentsURL, patient, req.Date, req.StartTime)
		})
	} else {
		h.afterCommit(func() {
			portal_services.NotifyStaff(h.db, h.logger, tenantID, portal_services.NotificationTypeBookingRequest,
				"notification.portal.booking.request", portal_services.RequestResourceType, req.ID,
				portal_services.RequestsURL, patient, req.Date, req.StartTime)
		})
	}
	return envelope.SuccessResponse(h.requestItem(db, req, now), "portal.booking.request.create.success")
}

// requestItem is req as the patient sees it. A pending request past its
// expiry reads as expired before the scheduler marks it.
func (h *BookingHandler) requestItem(db *gorm.DB, req portal_models.AppointmentRequest, now time.Time) portal_dto.BookingRequest {
	items := h.requestItems(db, []portal_models.AppointmentRequest{req}, now)
	return items[0]
}

func (h *BookingHandler) requestItems(db *gorm.DB, reqs []portal_models.AppointmentRequest, now time.Time) []portal_dto.BookingRequest {
	doctorIDs, typeIDs := []uint{}, []uint{}
	for _, r := range reqs {
		doctorIDs = append(doctorIDs, r.DoctorID)
		typeIDs = append(typeIDs, r.AppointmentTypeID)
	}
	doctors := map[uint]doctor_models.Doctor{}
	types := map[uint]string{}
	if len(reqs) > 0 {
		var ds []doctor_models.Doctor
		_ = db.Unscoped().Where("id IN ?", doctorIDs).Find(&ds).Error
		for _, d := range ds {
			doctors[d.ID] = d
		}
		var ts []agenda_models.AppointmentType
		_ = db.Unscoped().Select("id", "name").Where("id IN ?", typeIDs).Find(&ts).Error
		for _, t := range ts {
			types[t.ID] = t.Name
		}
	}
	out := make([]portal_dto.BookingRequest, 0, len(reqs))
	for _, r := range reqs {
		status := r.Status
		if status == portal_models.RequestPending && !r.ExpiresAt.After(now) {
			status = portal_models.RequestExpired
		}
		d := doctors[r.DoctorID]
		out = append(out, portal_dto.BookingRequest{
			ID: r.ID, Status: status, Date: r.Date, StartTime: r.StartTime, EndTime: r.EndTime,
			Doctor:   portal_dto.RequestDoctor{FullName: d.FullName, Specialty: d.Specialty, SpecialtyOther: d.SpecialtyOther},
			TypeName: types[r.AppointmentTypeID], Reason: r.Reason, RejectReason: r.RejectReason, ExpiresAt: r.ExpiresAt,
			AppointmentID: r.AppointmentID,
		})
	}
	return out
}

// ListRequests is GET /portal/booking/requests: the patient's pending
// requests plus those rejected, expired or cancelled in the last
// RecentRequestDays days, newest first.
func (h *BookingHandler) ListRequests(c *gin.Context) envelope.Response {
	db, _, patient := h.session(c)
	now := timeNow()
	since := now.AddDate(0, 0, -portal_services.RecentRequestDays)
	var reqs []portal_models.AppointmentRequest
	err := db.Where("patient_id = ?", patient.ID).
		Where("status = ? OR (status IN ? AND updated_at >= ?)", portal_models.RequestPending,
			[]string{portal_models.RequestRejected, portal_models.RequestExpired, portal_models.RequestCancelled}, since.UTC()).
		Order("created_at DESC").Order("id DESC").Find(&reqs).Error
	if err != nil {
		h.logger.Error("portal booking: failed to list requests", zap.Error(err))
		return internalError()
	}
	return envelope.SuccessResponse(h.requestItems(db, reqs, now), "portal.booking.requests.list.success")
}

// CancelRequest is POST /portal/booking/requests/:id/cancel: only the
// patient's own pending requests.
func (h *BookingHandler) CancelRequest(c *gin.Context) envelope.Response {
	db, clinic, patient := h.session(c)
	id, ok := paramID(c, "id")
	if !ok {
		return notFound()
	}
	var req portal_models.AppointmentRequest
	if err := db.Where("id = ? AND patient_id = ?", id, patient.ID).Limit(1).Find(&req).Error; err != nil {
		h.logger.Error("portal booking: failed to load request", zap.Error(err))
		return internalError()
	}
	if req.ID == 0 {
		return notFound()
	}
	now := timeNow()
	res := db.Model(&portal_models.AppointmentRequest{}).
		Where("id = ? AND patient_id = ? AND status = ?", req.ID, patient.ID, portal_models.RequestPending).
		Updates(map[string]any{"status": portal_models.RequestCancelled, "updated_at": now.UTC()})
	if res.Error != nil {
		h.logger.Error("portal booking: failed to cancel request", zap.Error(res.Error))
		return internalError()
	}
	if res.RowsAffected == 0 {
		return notPending()
	}
	req.Status = portal_models.RequestCancelled
	audit.RecordPortalChange(h.db, clinic.Tenant.ID, patient.ID, "UPDATE", auditRequests, req.ID)
	return envelope.SuccessResponse(h.requestItem(db, req, now), "portal.booking.request.cancel.success")
}

// CancelAppointment is POST /portal/appointments/:id/cancel: the patient's
// own appointment, scheduled or confirmed, more than the clinic's cancel
// notice before it starts. Later, the portal says to call the clinic (409).
func (h *BookingHandler) CancelAppointment(c *gin.Context) envelope.Response {
	db, clinic, patient := h.session(c)
	tenantID := clinic.Tenant.ID
	id, ok := paramID(c, "id")
	if !ok {
		return notFound()
	}
	var appt clinical_models.Appointment
	if err := db.Where("id = ? AND patient_id = ?", id, patient.ID).Limit(1).Find(&appt).Error; err != nil {
		h.logger.Error("portal booking: failed to load appointment", zap.Error(err))
		return internalError()
	}
	if appt.ID == 0 {
		return notFound()
	}
	if !portal_services.CanCancel(appt, clinic.Settings, timeNow()) {
		return envelope.ErrorResponse(http.StatusConflict, "portal.error.cancel_too_late", core_errors.ErrPortalCancelTooLate)
	}
	res := db.Model(&clinical_models.Appointment{}).
		Where("id = ? AND patient_id = ? AND status IN ?", appt.ID, patient.ID, []string{"scheduled", "confirmed"}).
		Update("status", "cancelled")
	if res.Error != nil {
		h.logger.Error("portal booking: failed to cancel appointment", zap.Error(res.Error))
		return internalError()
	}
	if res.RowsAffected == 0 {
		return envelope.ErrorResponse(http.StatusConflict, "portal.error.cancel_too_late", core_errors.ErrPortalCancelTooLate)
	}
	appt.Status = "cancelled"
	audit.RecordPortalChange(h.db, tenantID, patient.ID, "UPDATE", auditAppointments, appt.ID)
	date := appt.Date.In(portal_services.Location()).Format(agenda_services.DateLayout)
	h.afterCommit(func() {
		if h.calendar != nil {
			full := appt
			if err := tenantdb.ForTenant(h.db, tenantID).Preload("Patient").Preload("Doctor").First(&full, appt.ID).Error; err == nil {
				h.calendar.SyncStatusChange(tenantID, &full)
			}
		}
		portal_services.NotifyStaff(h.db, h.logger, tenantID, portal_services.NotificationTypeBookingCancelled,
			"notification.portal.booking.appointment_cancelled", portal_services.AppointmentResourceType, appt.ID,
			portal_services.AppointmentsURL, patient, date, appt.StartTime)
	})
	return envelope.SuccessResponse(portal_dto.Appointment{
		ID: appt.ID, Date: date, StartTime: appt.StartTime, EndTime: appt.EndTime, Status: appt.Status, Location: appt.Location,
	}, "portal.booking.appointment.cancel.success")
}

func parseUint(s string) (uint, bool) {
	id, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	return uint(id), err == nil && id > 0
}
