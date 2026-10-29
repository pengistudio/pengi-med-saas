// Package waitlist_handlers serves the appointment waitlist to the clinic.
package waitlist_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/whatsapp"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	waitlist_dto "pengi-med-saas/features/waitlist/dto"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	waitlist_services "pengi-med-saas/features/waitlist/services"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// maxNotesRunes bounds an entry's notes.
const maxNotesRunes = 500

// maxListed bounds GET /agenda/waitlist.
const maxListed = 500

// WaitlistHandler serves /agenda/waitlist: entries (list, add, edit,
// remove), the candidates of a freed slot and offering it.
type WaitlistHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	offers *waitlist_services.Offers
	now    func() time.Time
	loc    *time.Location
}

func NewWaitlistHandler(db *gorm.DB, logger *zap.Logger, client *whatsapp.Client) *WaitlistHandler {
	return &WaitlistHandler{db: db, logger: logger, offers: waitlist_services.NewOffers(db, logger, client),
		now: time.Now, loc: waitlist_services.Location()}
}

// WithClock overrides the clock and time zone (tests).
func (h *WaitlistHandler) WithClock(now func() time.Time, loc *time.Location) *WaitlistHandler {
	h.now, h.loc = now, loc
	h.offers.WithClock(now, loc)
	return h
}

func invalid() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "waitlist.error.invalid_request", core_errors.ErrWaitlistInvalidRequest)
}

func notFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "waitlist.error.not_found", core_errors.ErrWaitlistNotFound)
}

func notActive() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "waitlist.error.not_active", core_errors.ErrWaitlistNotActive)
}

func (h *WaitlistHandler) internal(msg string, err error) envelope.Response {
	h.logger.Error("waitlist: "+msg, zap.Error(err))
	return envelope.ErrorResponse(http.StatusInternalServerError, "waitlist.error.internal", core_errors.ErrWaitlistInternal)
}

// scoped is the request's tenant handle, with the entries that left the
// list on their own already closed.
func (h *WaitlistHandler) scoped(c *gin.Context) (*gorm.DB, error) {
	db := tenantdb.For(c, h.db)
	return db, waitlist_services.Refresh(db, h.now())
}

// List is GET /agenda/waitlist?status=active|waiting|offered|booked|expired|removed|all&doctor_id=<id>|any.
// Default: the active entries (waiting and offered), in signup order.
func (h *WaitlistHandler) List(c *gin.Context) envelope.Response {
	db, err := h.scoped(c)
	if err != nil {
		return h.internal("failed to refresh entries", err)
	}
	q := db.Preload("Patient").Preload("Doctor").Preload("AppointmentType")
	switch status := c.DefaultQuery("status", "active"); status {
	case "active":
		q = q.Where("status IN ?", waitlist_models.ActiveStatuses)
	case "all":
	case waitlist_models.StatusWaiting, waitlist_models.StatusOffered, waitlist_models.StatusBooked,
		waitlist_models.StatusExpired, waitlist_models.StatusRemoved:
		q = q.Where("status = ?", status)
	default:
		return invalid()
	}
	switch doctor := c.Query("doctor_id"); doctor {
	case "":
	case "any":
		q = q.Where("doctor_id IS NULL")
	default:
		id, err := strconv.ParseUint(doctor, 10, 32)
		if err != nil || id == 0 {
			return invalid()
		}
		q = q.Where("doctor_id = ?", id)
	}
	var entries []waitlist_models.WaitlistEntry
	if err := q.Order("created_at ASC, id ASC").Limit(maxListed).Find(&entries).Error; err != nil {
		return h.internal("failed to list entries", err)
	}
	out, err := h.toDTOs(db, entries)
	if err != nil {
		return h.internal("failed to load linked appointments", err)
	}
	return envelope.SuccessResponse(out, "waitlist.list.success")
}

// Create is POST /agenda/waitlist.
func (h *WaitlistHandler) Create(c *gin.Context) envelope.Response {
	var body waitlist_dto.EntryBody
	if err := c.ShouldBindJSON(&body); err != nil || body.PatientID == 0 {
		return invalid()
	}
	db, err := h.scoped(c)
	if err != nil {
		return h.internal("failed to refresh entries", err)
	}
	entry := waitlist_models.WaitlistEntry{
		TenantID: tenantdb.TenantID(c), PatientID: body.PatientID, Status: waitlist_models.StatusWaiting,
		Source: waitlist_models.SourceStaff, CreatedByUserID: userID(c),
	}
	if resp, ok := h.apply(db, &entry, body); !ok {
		return resp
	}
	if err := db.Create(&entry).Error; err != nil {
		return h.internal("failed to create entry", err)
	}
	return h.one(db, entry.ID, http.StatusCreated, "waitlist.create.success")
}

// Update is PUT /agenda/waitlist/:id: an active entry's doctor, type,
// range, appointment and notes (its patient stays).
func (h *WaitlistHandler) Update(c *gin.Context) envelope.Response {
	var body waitlist_dto.EntryBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return invalid()
	}
	db, err := h.scoped(c)
	if err != nil {
		return h.internal("failed to refresh entries", err)
	}
	entry, resp, ok := h.load(c, db)
	if !ok {
		return resp
	}
	if !entry.Active() {
		return notActive()
	}
	body.PatientID = entry.PatientID
	if resp, ok := h.apply(db, &entry, body); !ok {
		return resp
	}
	if err := db.Model(&waitlist_models.WaitlistEntry{}).Where("id = ?", entry.ID).Updates(map[string]any{
		"doctor_id": entry.DoctorID, "appointment_type_id": entry.AppointmentTypeID, "date_from": entry.DateFrom,
		"date_to": entry.DateTo, "appointment_id": entry.AppointmentID, "notes": entry.Notes,
		"expires_at": entry.ExpiresAt.UTC(), "updated_at": h.now().UTC(),
	}).Error; err != nil {
		return h.internal("failed to update entry", err)
	}
	return h.one(db, entry.ID, http.StatusOK, "waitlist.update.success")
}

// Remove is DELETE /agenda/waitlist/:id: the entry leaves the list
// (removed; kept for the history).
func (h *WaitlistHandler) Remove(c *gin.Context) envelope.Response {
	db, err := h.scoped(c)
	if err != nil {
		return h.internal("failed to refresh entries", err)
	}
	entry, resp, ok := h.load(c, db)
	if !ok {
		return resp
	}
	now := h.now().UTC()
	res := db.Model(&waitlist_models.WaitlistEntry{}).Where("id = ? AND status IN ?", entry.ID, waitlist_models.ActiveStatuses).
		Updates(map[string]any{"status": waitlist_models.StatusRemoved, "closed_at": now, "updated_at": now})
	if res.Error != nil {
		return h.internal("failed to remove entry", res.Error)
	}
	if res.RowsAffected == 0 {
		return notActive()
	}
	return h.one(db, entry.ID, http.StatusOK, "waitlist.remove.success")
}

// Candidates is GET /agenda/waitlist/candidates?date&start_time&end_time&doctor_id&appointment_type_id:
// the entries the slot matches, in signup order, and whether it can be
// offered by WhatsApp.
func (h *WaitlistHandler) Candidates(c *gin.Context) envelope.Response {
	slot, ok := slotFromQuery(c)
	if !ok {
		return invalid()
	}
	db, err := h.scoped(c)
	if err != nil {
		return h.internal("failed to refresh entries", err)
	}
	entries, err := waitlist_services.Matching(db, slot, h.now(), h.loc)
	if err != nil {
		return h.internal("failed to match entries", err)
	}
	list, err := h.toDTOs(db, entries)
	if err != nil {
		return h.internal("failed to load linked appointments", err)
	}
	entryIDs := make([]uint, 0, len(entries))
	for _, e := range entries {
		entryIDs = append(entryIDs, e.ID)
	}
	offered, err := waitlist_services.OfferedSlots(db, entryIDs, slot)
	if err != nil {
		return h.internal("failed to load the offer history", err)
	}
	for i, e := range entries {
		list[i].CanWhatsApp = waitlist_services.CanReceiveWhatsApp(e.Patient)
		list[i].OfferedThisSlot = offered[e.ID]
	}
	return envelope.SuccessResponse(waitlist_dto.Candidates{
		WhatsAppAvailable: waitlist_services.WhatsAppReady(h.db, tenantdb.TenantID(c)),
		Doctor:            waitlist_services.DoctorName(db, slot.DoctorID),
		Candidates:        list,
	}, "waitlist.candidates.success")
}

// Offer is POST /agenda/waitlist/:id/offer: offers a future slot the entry
// matches, by WhatsApp (sends the slot offer template) or phone (records
// the call). The same slot is never offered twice to the same entry.
func (h *WaitlistHandler) Offer(c *gin.Context) envelope.Response {
	var body waitlist_dto.OfferBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return invalid()
	}
	if body.Channel != waitlist_models.ChannelWhatsApp && body.Channel != waitlist_models.ChannelPhone {
		return invalid()
	}
	slot := waitlist_services.Slot{
		DoctorID: positive(body.DoctorID), AppointmentTypeID: positive(body.AppointmentTypeID),
		Date: body.Date, StartTime: body.StartTime, EndTime: body.EndTime, PatientID: body.PatientID,
	}
	if !validSlot(slot) {
		return invalid()
	}
	if start, err := slot.Start(h.loc); err != nil || !start.After(h.now()) {
		return invalid()
	}
	db, err := h.scoped(c)
	if err != nil {
		return h.internal("failed to refresh entries", err)
	}
	entry, resp, ok := h.load(c, db)
	if !ok {
		return resp
	}
	if !entry.Active() {
		return notActive()
	}
	matches, err := waitlist_services.Matches(db, entry, slot, h.now(), h.loc)
	if err != nil {
		return h.internal("failed to match entry", err)
	}
	if !matches {
		return envelope.ErrorResponse(http.StatusConflict, "waitlist.error.slot_mismatch", core_errors.ErrWaitlistSlotMismatch)
	}
	updated, err := h.offers.Offer(tenantdb.TenantID(c), entry, slot, body.Channel, userID(c))
	if err != nil {
		return h.offerError(err)
	}
	out, err := h.toDTOs(db, []waitlist_models.WaitlistEntry{updated})
	if err != nil {
		return h.internal("failed to load linked appointments", err)
	}
	key := "waitlist.offer.whatsapp.success"
	if body.Channel == waitlist_models.ChannelPhone {
		key = "waitlist.offer.phone.success"
	}
	return envelope.SuccessResponse(out[0], key)
}

func (h *WaitlistHandler) offerError(err error) envelope.Response {
	switch {
	case errors.Is(err, waitlist_services.ErrAlreadyOffered):
		return envelope.ErrorResponse(http.StatusConflict, "waitlist.error.already_offered", core_errors.ErrWaitlistAlreadyOffered)
	case errors.Is(err, waitlist_services.ErrNotActive):
		return notActive()
	case errors.Is(err, waitlist_services.ErrOwnSlot):
		return envelope.ErrorResponse(http.StatusConflict, "waitlist.error.slot_mismatch", core_errors.ErrWaitlistSlotMismatch)
	case errors.Is(err, waitlist_services.ErrWhatsAppNotInPlan):
		return envelope.ErrorResponse(http.StatusForbidden, "waitlist.error.whatsapp_unavailable", core_errors.ErrWaitlistWhatsAppUnavailable)
	case errors.Is(err, waitlist_services.ErrNotConnected):
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.not_connected", core_errors.ErrWhatsAppNotConnected)
	case errors.Is(err, waitlist_services.ErrTemplateUnavailable):
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.template_unavailable", core_errors.ErrWhatsAppTemplateUnavailable)
	case errors.Is(err, waitlist_services.ErrNoOptIn):
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.no_opt_in", core_errors.ErrWhatsAppNoOptIn)
	case errors.Is(err, waitlist_services.ErrInvalidPhone):
		return envelope.ErrorResponse(http.StatusConflict, "whatsapp.error.invalid_phone", core_errors.ErrWhatsAppInvalidPhone)
	case errors.Is(err, waitlist_services.ErrMonthlyLimit):
		return envelope.ErrorResponse(http.StatusForbidden, "plan.limit.whatsapp_messages", core_errors.ErrWhatsAppMonthlyLimit)
	case errors.Is(err, waitlist_services.ErrSendFailed):
		return envelope.ErrorResponse(http.StatusBadGateway, "whatsapp.error.send_failed", core_errors.ErrWhatsAppSendFailed)
	}
	return h.internal("failed to offer slot", err)
}

// load finds the :id entry of the tenant.
func (h *WaitlistHandler) load(c *gin.Context, db *gorm.DB) (waitlist_models.WaitlistEntry, envelope.Response, bool) {
	var entry waitlist_models.WaitlistEntry
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		return entry, notFound(), false
	}
	if err := db.Preload("Patient").Where("id = ?", id).Limit(1).Find(&entry).Error; err != nil {
		return entry, h.internal("failed to load entry", err), false
	}
	if entry.ID == 0 {
		return entry, notFound(), false
	}
	return entry, envelope.Response{}, true
}

// one answers with the entry id, reloaded.
func (h *WaitlistHandler) one(db *gorm.DB, id uint, status int, key string) envelope.Response {
	var entry waitlist_models.WaitlistEntry
	if err := db.Preload("Patient").Preload("Doctor").Preload("AppointmentType").Where("id = ?", id).First(&entry).Error; err != nil {
		return h.internal("failed to reload entry", err)
	}
	out, err := h.toDTOs(db, []waitlist_models.WaitlistEntry{entry})
	if err != nil {
		return h.internal("failed to load linked appointments", err)
	}
	resp := envelope.SuccessResponse(out[0], key)
	resp.Code = status
	return resp
}

// apply validates body and writes it into entry (db bound to the tenant):
// the patient, doctor, type and appointment belong to the clinic, the
// appointment is the patient's and still ahead, the range is valid and not
// over, and the patient has no other active entry for the same doctor and
// type. ExpiresAt follows the range.
func (h *WaitlistHandler) apply(db *gorm.DB, entry *waitlist_models.WaitlistEntry, body waitlist_dto.EntryBody) (envelope.Response, bool) {
	now := h.now()
	today := now.In(h.loc).Format(waitlist_services.DateLayout)
	body.DateFrom, body.DateTo = strings.TrimSpace(body.DateFrom), strings.TrimSpace(body.DateTo)
	body.Notes = strings.TrimSpace(body.Notes)
	if !validDate(body.DateFrom) || !validDate(body.DateTo) || utf8.RuneCountInString(body.Notes) > maxNotesRunes {
		return invalid(), false
	}
	if body.DateFrom != "" && body.DateTo != "" && body.DateFrom > body.DateTo {
		return invalid(), false
	}
	if body.DateTo != "" && body.DateTo < today {
		return invalid(), false
	}
	doctorID, typeID, apptID := positive(body.DoctorID), positive(body.AppointmentTypeID), positive(body.AppointmentID)
	if !exists(db, &clinical_models.Patient{}, body.PatientID) ||
		(doctorID != nil && !exists(db, &doctor_models.Doctor{}, *doctorID)) ||
		(typeID != nil && !exists(db, &agenda_models.AppointmentType{}, *typeID)) {
		return invalid(), false
	}
	if apptID != nil {
		var appt clinical_models.Appointment
		if err := db.Where("id = ? AND patient_id = ?", *apptID, body.PatientID).Limit(1).Find(&appt).Error; err != nil {
			return h.internal("failed to load appointment", err), false
		}
		start, err := whatsapp_services.AppointmentStart(appt, h.loc)
		if appt.ID == 0 || err != nil || clinical_models.IsClosedStatus(appt.Status) || !start.After(now) {
			return invalid(), false
		}
	}
	dup, err := waitlist_services.ActiveDuplicate(db, body.PatientID, doctorID, typeID, entry.ID)
	if err != nil {
		return h.internal("failed to check duplicates", err), false
	}
	if dup.ID != 0 {
		return envelope.ErrorResponse(http.StatusConflict, "waitlist.error.duplicate", core_errors.ErrWaitlistDuplicate), false
	}
	entry.DoctorID, entry.AppointmentTypeID, entry.AppointmentID = doctorID, typeID, apptID
	entry.DateFrom, entry.DateTo, entry.Notes = body.DateFrom, body.DateTo, body.Notes
	entry.ExpiresAt = waitlist_services.ExpiresAt(now, body.DateFrom, body.DateTo, h.loc)
	return envelope.Response{}, true
}

// toDTOs maps entries (with Patient, Doctor and AppointmentType loaded).
func (h *WaitlistHandler) toDTOs(db *gorm.DB, entries []waitlist_models.WaitlistEntry) ([]waitlist_dto.Entry, error) {
	var apptIDs []uint
	for _, e := range entries {
		if e.AppointmentID != nil {
			apptIDs = append(apptIDs, *e.AppointmentID)
		}
	}
	appts := map[uint]clinical_models.Appointment{}
	if len(apptIDs) > 0 {
		var rows []clinical_models.Appointment
		if err := db.Unscoped().Where("id IN ?", apptIDs).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, a := range rows {
			appts[a.ID] = a
		}
	}
	out := make([]waitlist_dto.Entry, 0, len(entries))
	for _, e := range entries {
		item := waitlist_dto.Entry{
			ID: e.ID, Status: e.Status, Source: e.Source, DateFrom: e.DateFrom, DateTo: e.DateTo,
			Notes: e.Notes, CreatedAt: e.CreatedAt, ExpiresAt: e.ExpiresAt,
		}
		if p := e.Patient; p != nil {
			item.Patient = waitlist_dto.Person{ID: p.ID, FirstName: p.FirstName, LastName: p.LastName,
				Document: p.Document, Phone: p.Phone, WhatsAppOptIn: p.WhatsAppOptIn}
		} else {
			item.Patient = waitlist_dto.Person{ID: e.PatientID}
		}
		if d := e.Doctor; d != nil {
			item.Doctor = &waitlist_dto.Doctor{ID: d.ID, FullName: d.FullName}
		}
		if t := e.AppointmentType; t != nil {
			item.AppointmentType = &waitlist_dto.Type{ID: t.ID, Name: t.Name, DurationMinutes: t.DurationMinutes}
		}
		if e.AppointmentID != nil {
			if a, ok := appts[*e.AppointmentID]; ok {
				appt := &waitlist_dto.Appointment{ID: a.ID, StartTime: a.StartTime, Status: a.Status}
				if !a.DeletedAt.Valid {
					appt.Date = a.Date.In(h.loc).Format(waitlist_services.DateLayout)
					if start, err := whatsapp_services.AppointmentStart(a, h.loc); err == nil {
						appt.Date = start.Format(waitlist_services.DateLayout)
					}
				} else {
					appt.Status = "deleted"
				}
				item.Appointment = appt
			}
		}
		if e.OfferedAt != nil {
			item.Offer = &waitlist_dto.Offer{
				DoctorID: e.OfferedDoctorID, Date: e.OfferedDate, StartTime: e.OfferedStartTime, EndTime: e.OfferedEndTime,
				Channel: e.OfferChannel, OfferedAt: e.OfferedAt, AcceptedAt: e.OfferAcceptedAt,
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// slotFromQuery reads the slot of GET /candidates.
func slotFromQuery(c *gin.Context) (waitlist_services.Slot, bool) {
	slot := waitlist_services.Slot{Date: c.Query("date"), StartTime: c.Query("start_time"), EndTime: c.Query("end_time")}
	for _, p := range []struct {
		name string
		dst  **uint
	}{{"doctor_id", &slot.DoctorID}, {"appointment_type_id", &slot.AppointmentTypeID}} {
		raw := c.Query(p.name)
		if raw == "" {
			continue
		}
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return slot, false
		}
		if id > 0 {
			v := uint(id)
			*p.dst = &v
		}
	}
	if raw := c.Query("patient_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return slot, false
		}
		slot.PatientID = uint(id)
	}
	return slot, validSlot(slot)
}

func validSlot(s waitlist_services.Slot) bool {
	if s.Date == "" || !validDate(s.Date) || !validClock(s.StartTime) {
		return false
	}
	return s.EndTime == "" || (validClock(s.EndTime) && s.EndTime > s.StartTime)
}

func validDate(s string) bool {
	if s == "" {
		return true
	}
	_, err := time.Parse(waitlist_services.DateLayout, s)
	return err == nil
}

func validClock(s string) bool {
	if len(s) != 5 {
		return false
	}
	_, err := time.Parse("15:04", s)
	return err == nil
}

func positive(p *uint) *uint {
	if p == nil || *p == 0 {
		return nil
	}
	v := *p
	return &v
}

// exists reports whether model id is the tenant's (db bound to it).
func exists(db *gorm.DB, model any, id uint) bool {
	var n int64
	return db.Model(model).Where("id = ?", id).Count(&n).Error == nil && n == 1
}

func userID(c *gin.Context) *uint {
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
