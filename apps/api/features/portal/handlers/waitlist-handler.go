package portal_handlers

import (
	"net/http"

	"pengi-med-saas/core/envelope"
	portal_services "pengi-med-saas/features/portal/services"
	waitlist_dto "pengi-med-saas/features/waitlist/dto"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	waitlist_services "pengi-med-saas/features/waitlist/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// The waitlist from the portal: when the slot picker has no time for the
// chosen doctor (or specialty) and type, the signed-in patient asks to be
// told when one frees ("Avísame si se libera un turno"). Only patients with
// a record (a portal session, never a new patient's guest token) and only
// while the clinic takes bookings (the booking routes' gating). The entry is
// like reception's: any doctor in specialty mode, no date range (30 days).

// ListWaitlist is GET /portal/waitlist: the patient's active entries.
func (h *BookingHandler) ListWaitlist(c *gin.Context) envelope.Response {
	db, _, patient := h.session(c)
	if err := waitlist_services.Refresh(db, timeNow()); err != nil {
		h.logger.Error("portal waitlist: failed to refresh entries", zap.Error(err))
		return internalError()
	}
	var entries []waitlist_models.WaitlistEntry
	if err := db.Preload("Doctor").Preload("AppointmentType").
		Where("patient_id = ? AND status IN ?", patient.ID, waitlist_models.ActiveStatuses).
		Order("id").Find(&entries).Error; err != nil {
		h.logger.Error("portal waitlist: failed to list entries", zap.Error(err))
		return internalError()
	}
	out := make([]waitlist_dto.PortalEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, portalEntry(e))
	}
	return envelope.SuccessResponse(out, "portal.waitlist.list.success")
}

// JoinWaitlist is POST /portal/waitlist {type_id, doctor_id?}: a type the
// portal books and, when given, a bookable doctor attending it. Joining
// again for the same doctor and type returns the entry already there.
func (h *BookingHandler) JoinWaitlist(c *gin.Context) envelope.Response {
	db, clinic, patient := h.session(c)
	var body waitlist_dto.PortalEntryBody
	if err := c.ShouldBindJSON(&body); err != nil {
		return invalidData()
	}
	k, err := loadCatalog(db)
	if err != nil {
		h.logger.Error("portal waitlist: failed to load the booking catalog", zap.Error(err))
		return internalError()
	}
	typ, ok := k.typeByID(body.TypeID)
	if !ok {
		return invalidData()
	}
	var doctorID *uint
	if body.DoctorID != nil && *body.DoctorID != 0 {
		id := *body.DoctorID
		bookable := false
		for _, d := range k.doctors {
			if d.ID == id && typ.AllowsDoctor(id) {
				bookable = true
			}
		}
		if !bookable {
			return invalidData()
		}
		doctorID = &id
	}
	now := timeNow()
	if err := waitlist_services.Refresh(db, now); err != nil {
		h.logger.Error("portal waitlist: failed to refresh entries", zap.Error(err))
		return internalError()
	}
	typeID := typ.ID
	existing, err := waitlist_services.ActiveDuplicate(db, patient.ID, doctorID, &typeID, 0)
	if err != nil {
		h.logger.Error("portal waitlist: failed to check duplicates", zap.Error(err))
		return internalError()
	}
	id := existing.ID
	status := http.StatusOK
	if id == 0 {
		entry := waitlist_models.WaitlistEntry{
			TenantID: clinic.Tenant.ID, PatientID: patient.ID, DoctorID: doctorID, AppointmentTypeID: &typeID,
			Source: waitlist_models.SourcePortal, Status: waitlist_models.StatusWaiting,
			ExpiresAt: waitlist_services.ExpiresAt(now, "", "", portal_services.Location()),
		}
		if err := db.Create(&entry).Error; err != nil {
			h.logger.Error("portal waitlist: failed to create entry", zap.Error(err))
			return internalError()
		}
		id, status = entry.ID, http.StatusCreated
	}
	entry, ok := h.ownEntry(db, patient.ID, id)
	if !ok {
		return internalError()
	}
	resp := envelope.SuccessResponse(portalEntry(entry), "portal.waitlist.join.success")
	resp.Code = status
	return resp
}

// LeaveWaitlist is DELETE /portal/waitlist/:id: the patient's own entry
// leaves the list (removed). Leaving an entry already closed changes nothing.
func (h *BookingHandler) LeaveWaitlist(c *gin.Context) envelope.Response {
	db, _, patient := h.session(c)
	id, ok := paramID(c, "id")
	if !ok {
		return notFound()
	}
	entry, ok := h.ownEntry(db, patient.ID, id)
	if !ok {
		return notFound()
	}
	if entry.Active() {
		now := timeNow().UTC()
		if err := db.Model(&waitlist_models.WaitlistEntry{}).
			Where("id = ? AND patient_id = ? AND status IN ?", entry.ID, patient.ID, waitlist_models.ActiveStatuses).
			Updates(map[string]any{"status": waitlist_models.StatusRemoved, "closed_at": now, "updated_at": now}).Error; err != nil {
			h.logger.Error("portal waitlist: failed to remove entry", zap.Error(err))
			return internalError()
		}
		entry.Status = waitlist_models.StatusRemoved
	}
	return envelope.SuccessResponse(portalEntry(entry), "portal.waitlist.leave.success")
}

// ownEntry loads the patient's entry id with its doctor and type.
func (h *BookingHandler) ownEntry(db *gorm.DB, patientID, id uint) (waitlist_models.WaitlistEntry, bool) {
	var entry waitlist_models.WaitlistEntry
	if err := db.Preload("Doctor").Preload("AppointmentType").
		Where("id = ? AND patient_id = ?", id, patientID).Limit(1).Find(&entry).Error; err != nil {
		h.logger.Error("portal waitlist: failed to load entry", zap.Error(err))
		return entry, false
	}
	return entry, entry.ID != 0
}

func portalEntry(e waitlist_models.WaitlistEntry) waitlist_dto.PortalEntry {
	out := waitlist_dto.PortalEntry{
		ID: e.ID, Status: e.Status, DoctorID: e.DoctorID, TypeID: e.AppointmentTypeID,
		CreatedAt: e.CreatedAt, ExpiresAt: e.ExpiresAt,
	}
	if e.Doctor != nil {
		out.DoctorName = e.Doctor.FullName
	}
	if e.AppointmentType != nil {
		out.TypeName = e.AppointmentType.Name
	}
	return out
}
