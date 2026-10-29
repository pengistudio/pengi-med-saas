package waitlist_services

import (
	"encoding/json"
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	portal_models "pengi-med-saas/features/portal/models"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	whatsapp_models "pengi-med-saas/features/whatsapp/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Staff notifications of the waitlist.
const (
	// NotificationTypeSlotFreed: a slot freed and the waitlist has
	// candidates; it leads to the waitlist view filtered for the slot.
	NotificationTypeSlotFreed = "waitlist.slot_freed"
	SlotFreedKey              = "notification.waitlist.slot_freed"
	SlotFreedNoDoctorKey      = "notification.waitlist.slot_freed_no_doctor"
	// NotificationTypeOfferAccepted: the patient tapped "Lo quiero" and the
	// clinic has no online booking to send them to.
	NotificationTypeOfferAccepted = "waitlist.offer_accepted"
	OfferAcceptedKey              = "notification.waitlist.offer_accepted"
	OfferAcceptedNoSlotKey        = "notification.waitlist.offer_accepted_no_slot"

	SlotResourceType  = "waitlist_slot"
	EntryResourceType = "waitlist_entry"

	// ManagePermission: who adds entries, offers slots and is told a slot freed.
	ManagePermission = "MANAGE_APPOINTMENT"
	inboxPermission  = "USE_WHATSAPP_INBOX"
)

// Slots reacts to freed slots: when a future appointment is cancelled or
// deleted, or a pending portal request releases the slot it held, and the
// waitlist has entries the slot matches, every user with MANAGE_APPOINTMENT
// is told ("Se liberó mar 10:00 con Dra. X — 3 en lista de espera"). Nothing
// is offered on its own. Never fails the caller: problems are logged.
type Slots struct {
	db     *gorm.DB
	logger *zap.Logger
	now    func() time.Time
	loc    *time.Location
}

func NewSlots(db *gorm.DB, logger *zap.Logger) *Slots {
	return &Slots{db: db, logger: logger, now: time.Now, loc: Location()}
}

// WithClock overrides the clock and time zone (tests).
func (s *Slots) WithClock(now func() time.Time, loc *time.Location) *Slots {
	s.now, s.loc = now, loc
	return s
}

// AppointmentFreed handles appt (status before the change: active, i.e. not
// cancelled, completed or no-show) cancelled or deleted. Only future slots
// count. It returns the number of matching entries.
func (s *Slots) AppointmentFreed(tenantID uint, appt clinical_models.Appointment) int {
	slot, err := SlotFromAppointment(appt, s.loc)
	if err != nil {
		return 0
	}
	return s.Freed(tenantID, slot, appt.ID)
}

// RequestReleased handles a pending portal request rejected, cancelled or
// expired: the slot it held is free again.
func (s *Slots) RequestReleased(tenantID uint, req portal_models.AppointmentRequest) int {
	doctorID, typeID := req.DoctorID, req.AppointmentTypeID
	slot := Slot{Date: req.Date, StartTime: req.StartTime, EndTime: req.EndTime, PatientID: req.PatientID}
	if doctorID != 0 {
		slot.DoctorID = &doctorID
	}
	if typeID != 0 {
		slot.AppointmentTypeID = &typeID
	}
	return s.Freed(tenantID, slot, req.ID)
}

// Freed notifies the staff when slot, just freed, matches waitlist entries,
// and returns how many. A slot already started is ignored.
func (s *Slots) Freed(tenantID uint, slot Slot, resourceID uint) int {
	if s == nil || tenantID == 0 {
		return 0
	}
	now := s.now()
	start, err := slot.Start(s.loc)
	if err != nil || !start.After(now) {
		return 0
	}
	db := tenantdb.ForTenant(s.db, tenantID)
	if err := Refresh(db, now); err != nil {
		s.logger.Error("waitlist: failed to refresh entries", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return 0
	}
	entries, err := Matching(db, slot, now, s.loc)
	if err != nil {
		s.logger.Error("waitlist: failed to match a freed slot", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return 0
	}
	if len(entries) == 0 {
		return 0
	}
	s.notifyFreed(db, tenantID, slot, len(entries), resourceID)
	return len(entries)
}

func (s *Slots) notifyFreed(db *gorm.DB, tenantID uint, slot Slot, count int, resourceID uint) {
	userIDs, err := whatsapp_services.UsersWithPermission(s.db, tenantID, ManagePermission)
	if err != nil {
		s.logger.Error("waitlist: failed to resolve recipients", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		return
	}
	doctor := DoctorName(db, slot.DoctorID)
	key := SlotFreedKey
	if doctor == "" {
		key = SlotFreedNoDoctorKey
	}
	params, _ := json.Marshal(map[string]any{
		"date": slot.Date, "time": slot.StartTime, "end_time": slot.EndTime, "doctor": doctor, "count": count,
	})
	rows := make([]notifications_models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		rows = append(rows, notifications_models.Notification{
			TenantID: tenantID, UserID: uid, Type: NotificationTypeSlotFreed, ResourceType: SlotResourceType,
			ResourceID: resourceID, MessageKey: key, Params: params, ActionURL: slot.URL(),
			Level: notifications_models.NotificationLevelInfo,
		})
	}
	if err := db.Create(&rows).Error; err != nil {
		s.logger.Error("waitlist: failed to create slot notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}

// DoctorName is the doctor's name (db bound to the tenant; "" for none).
func DoctorName(db *gorm.DB, doctorID *uint) string {
	if doctorID == nil {
		return ""
	}
	var doctor doctor_models.Doctor
	if err := db.Unscoped().Select("id", "full_name").Where("id = ?", *doctorID).Limit(1).Find(&doctor).Error; err != nil {
		return ""
	}
	return doctor.FullName
}

// OfferAccepted records that the patient tapped "Lo quiero" on the offer
// message msgID (db bound to the tenant) and returns its entry (ID 0 when
// the entry was offered another slot since, or is gone).
func OfferAccepted(db *gorm.DB, msgID uint, now time.Time) (waitlist_models.WaitlistEntry, error) {
	var entry waitlist_models.WaitlistEntry
	if err := db.Preload("Patient").Where("offer_message_id = ?", msgID).Limit(1).Find(&entry).Error; err != nil || entry.ID == 0 {
		return entry, err
	}
	at := now.UTC()
	if err := db.Model(&waitlist_models.WaitlistEntry{}).Where("id = ?", entry.ID).
		Updates(map[string]any{"offer_accepted_at": at, "updated_at": at}).Error; err != nil {
		return entry, err
	}
	entry.OfferAcceptedAt = &at
	return entry, nil
}

// NotifyOfferAccepted tells the users with MANAGE_APPOINTMENT or
// USE_WHATSAPP_INBOX that the patient of conv wants the slot offered to them
// (entry's last offer; entry.ID 0 when unknown). It leads to the
// conversation, to answer and book. Params: patient_name, date, time, doctor.
func NotifyOfferAccepted(db *gorm.DB, logger *zap.Logger, tenantID uint, conv whatsapp_models.WhatsAppConversation, entry waitlist_models.WaitlistEntry) {
	recipients := map[uint]bool{}
	for _, perm := range []string{ManagePermission, inboxPermission} {
		ids, err := whatsapp_services.UsersWithPermission(db, tenantID, perm)
		if err != nil {
			logger.Error("waitlist: failed to resolve recipients", zap.Uint("tenant_id", tenantID), zap.Error(err))
			return
		}
		for _, id := range ids {
			recipients[id] = true
		}
	}
	if len(recipients) == 0 {
		return
	}
	tdb := tenantdb.ForTenant(db, tenantID)
	name := "+" + conv.Phone
	if entry.Patient != nil {
		if n := strings.TrimSpace(entry.Patient.FirstName + " " + entry.Patient.LastName); n != "" {
			name = n
		}
	}
	key, resourceType, resourceID := OfferAcceptedNoSlotKey, whatsapp_services.ConversationResourceType, conv.ID
	params := map[string]any{"patient_name": name, "conversation_id": conv.ID}
	if entry.ID != 0 && entry.OfferedDate != "" {
		key = OfferAcceptedKey
		params["date"], params["time"] = entry.OfferedDate, entry.OfferedStartTime
		params["doctor"] = DoctorName(tdb, entry.OfferedDoctorID)
	}
	raw, _ := json.Marshal(params)
	rows := make([]notifications_models.Notification, 0, len(recipients))
	for uid := range recipients {
		rows = append(rows, notifications_models.Notification{
			TenantID: tenantID, UserID: uid, Type: NotificationTypeOfferAccepted, ResourceType: resourceType,
			ResourceID: resourceID, MessageKey: key, Params: raw,
			ActionURL: whatsapp_services.MessageNotificationURL(conv.ID), Level: notifications_models.NotificationLevelInfo,
		})
	}
	if err := tdb.Create(&rows).Error; err != nil {
		logger.Error("waitlist: failed to create offer notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}
