package portal_services

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"pengi-med-saas/core/tenantdb"
	agenda_models "pengi-med-saas/features/agenda/models"
	agenda_services "pengi-med-saas/features/agenda/services"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	portal_models "pengi-med-saas/features/portal/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	whatsapp_services "pengi-med-saas/features/whatsapp/services"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Booking limits.
const (
	// MaxPendingRequests is how many pending requests a patient may have.
	MaxPendingRequests = 3
	// MaxSlotDays is the longest date range GET /portal/booking/slots answers.
	MaxSlotDays = 14
	// RecentRequestDays is how long ended requests stay listed to the patient.
	RecentRequestDays = 30
	// ManageRequestsPermission lets staff see and decide portal requests.
	ManageRequestsPermission = "MANAGE_APPOINTMENT_REQUESTS"
)

// Staff notifications of booking from the portal.
const (
	NotificationTypeBookingRequest   = "portal.booking.request"
	NotificationTypeBookingBooked    = "portal.booking.booked"
	NotificationTypeBookingCancelled = "portal.booking.appointment_cancelled"
	RequestResourceType              = "appointment_request"
	AppointmentResourceType          = "appointment"
	RequestsURL                      = "/appointment-requests"
	AppointmentsURL                  = "/clinical/appointments"
)

// ErrNotPending: the request was decided, cancelled or expired meanwhile.
var ErrNotPending = errors.New("appointment request is not pending")

// Location is the clinic time zone of every booking date and time.
func Location() *time.Location { return whatsapp_services.ClinicLocation() }

// Window is the booking window on now: from today to today + MaxDaysAhead
// (clinic dates, "YYYY-MM-DD").
func Window(now time.Time, b tenant_models.PortalBooking) (minDate, maxDate string) {
	today := now.In(Location())
	return today.Format(agenda_services.DateLayout), today.AddDate(0, 0, b.MaxDaysAhead).Format(agenda_services.DateLayout)
}

// SlotStart is the instant date ("YYYY-MM-DD") at start ("HH:MM") in the
// clinic's time zone.
func SlotStart(date, start string) (time.Time, bool) {
	d, ok := agenda_services.ParseDate(date)
	if !ok {
		return time.Time{}, false
	}
	m, ok := agenda_services.ParseClock(start, 0)
	if !ok {
		return time.Time{}, false
	}
	return time.Date(d.Year(), d.Month(), d.Day(), m/60, m%60, 0, 0, Location()), true
}

// ExpiresAt is when a request made at now for a slot starting at start
// stops holding it: after expiryHours, or at the slot's start if sooner.
func ExpiresAt(now, start time.Time, expiryHours int) time.Time {
	exp := now.Add(time.Duration(expiryHours) * time.Hour)
	if start.Before(exp) {
		exp = start
	}
	return exp.UTC()
}

// BookableDoctors are the tenant's doctors offered in the portal (db bound
// to the tenant): active, marked bookable and with a weekly schedule; by id.
func BookableDoctors(db *gorm.DB) ([]doctor_models.Doctor, error) {
	var doctors []doctor_models.Doctor
	err := db.Where("active = ? AND portal_bookable = ?", true, true).
		Where("EXISTS (SELECT 1 FROM doctor_schedules s WHERE s.doctor_id = doctors.id AND s.deleted_at IS NULL)").
		Order("id").Find(&doctors).Error
	return doctors, err
}

// BookableTypes are the tenant's appointment types offered in the portal:
// active and marked bookable, with their Doctors loaded; by name.
func BookableTypes(db *gorm.DB) ([]agenda_models.AppointmentType, error) {
	var types []agenda_models.AppointmentType
	err := db.Preload("Doctors").Where("active = ? AND portal_bookable = ?", true, true).Order("name, id").Find(&types).Error
	return types, err
}

// SpecialtyKey identifies a specialty: the code, plus the free-text name
// (trimmed, lowercase) when the code is "other".
func SpecialtyKey(specialty, other string) string {
	if specialty != "other" {
		return specialty
	}
	return specialty + ":" + strings.ToLower(strings.TrimSpace(other))
}

// RankBySchedule orders doctors for a specialty request on date: fewest
// appointments that day first, then lowest id.
func RankBySchedule(db *gorm.DB, doctors []doctor_models.Doctor, date string) ([]doctor_models.Doctor, error) {
	ids := make([]uint, 0, len(doctors))
	for _, d := range doctors {
		ids = append(ids, d.ID)
	}
	appts, err := agenda_services.DoctorAppointments(db, ids, date, date, Location())
	if err != nil {
		return nil, err
	}
	load := map[uint]int{}
	for _, a := range appts {
		if a.DoctorID != nil {
			load[*a.DoctorID]++
		}
	}
	out := append([]doctor_models.Doctor(nil), doctors...)
	sort.SliceStable(out, func(i, j int) bool {
		if load[out[i].ID] != load[out[j].ID] {
			return load[out[i].ID] < load[out[j].ID]
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// LockSlot serializes bookings of one doctor and date until tx ends
// (Postgres advisory lock; a no-op on other databases).
func LockSlot(tx *gorm.DB, tenantID, doctorID uint, date string) error {
	return advisoryLock(tx, fmt.Sprintf("portal-booking:%d:%d:%s", tenantID, doctorID, date))
}

// LockPatient serializes one patient's requests until tx ends.
func LockPatient(tx *gorm.DB, tenantID, patientID uint) error {
	return advisoryLock(tx, fmt.Sprintf("portal-booking-patient:%d:%d", tenantID, patientID))
}

func advisoryLock(tx *gorm.DB, key string) error {
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(h.Sum64())).Error
}

// CreateAppointment creates, on tx (bound to the tenant), the appointment of
// an approved request: status scheduled, titled with the type's name.
func CreateAppointment(tx *gorm.DB, req portal_models.AppointmentRequest, typeName string) (clinical_models.Appointment, error) {
	d, ok := agenda_services.ParseDate(req.Date)
	if !ok {
		return clinical_models.Appointment{}, fmt.Errorf("invalid request date %q", req.Date)
	}
	doctorID, typeID := req.DoctorID, req.AppointmentTypeID
	appt := clinical_models.Appointment{
		TenantID:          req.TenantID,
		PatientID:         req.PatientID,
		Title:             typeName,
		Date:              time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, Location()),
		StartTime:         req.StartTime,
		EndTime:           req.EndTime,
		Notes:             req.Reason,
		Status:            "scheduled",
		DoctorID:          &doctorID,
		AppointmentTypeID: &typeID,
	}
	err := tx.Omit("Patient", "Doctor", "VitalSigns").Create(&appt).Error
	return appt, err
}

// TypeName is the name of appointment type id, deleted or not ("" if none).
func TypeName(db *gorm.DB, id uint) string {
	var t agenda_models.AppointmentType
	if err := db.Unscoped().Select("id", "name").Where("id = ?", id).Limit(1).Find(&t).Error; err != nil {
		return ""
	}
	return t.Name
}

// PatientName is how staff notifications name a patient.
func PatientName(p clinical_models.Patient) string {
	if p.FullName != nil && strings.TrimSpace(*p.FullName) != "" {
		return strings.TrimSpace(*p.FullName)
	}
	return strings.TrimSpace(p.FirstName + " " + p.LastName)
}

// NotifyStaff tells the tenant's users with MANAGE_APPOINTMENT_REQUESTS about
// a booking event (one row per user, no dedupe: each event happens once).
// Params: patient_name, date (YYYY-MM-DD), time (HH:MM). Failures are logged.
func NotifyStaff(db *gorm.DB, logger *zap.Logger, tenantID uint, notifType, key, resourceType string, resourceID uint,
	actionURL string, patient clinical_models.Patient, date, start string) {
	userIDs, err := whatsapp_services.UsersWithPermission(db, tenantID, ManageRequestsPermission)
	if err != nil {
		logger.Error("portal booking: failed to resolve staff to notify", zap.Uint("tenant_id", tenantID), zap.Error(err))
		return
	}
	if len(userIDs) == 0 {
		return
	}
	params, _ := json.Marshal(map[string]string{"patient_name": PatientName(patient), "date": date, "time": start})
	rows := make([]notifications_models.Notification, 0, len(userIDs))
	for _, uid := range userIDs {
		rows = append(rows, notifications_models.Notification{
			TenantID: tenantID, UserID: uid, Type: notifType, ResourceType: resourceType, ResourceID: resourceID,
			MessageKey: key, Params: params, ActionURL: actionURL, Level: notifications_models.NotificationLevelInfo,
		})
	}
	if err := tenantdb.ForTenant(db, tenantID).Create(&rows).Error; err != nil {
		logger.Error("portal booking: failed to create staff notifications", zap.Uint("tenant_id", tenantID), zap.Error(err))
	}
}

// CanCancel reports whether the patient may cancel appointment a from the
// portal at now: booking is on, the appointment is scheduled or confirmed and
// it starts more than the clinic's cancel notice from now.
func CanCancel(a clinical_models.Appointment, s tenant_models.PortalSettings, now time.Time) bool {
	if !s.BookingEnabled() || (a.Status != "scheduled" && a.Status != "confirmed") {
		return false
	}
	start, err := whatsapp_services.AppointmentStart(a, Location())
	if err != nil {
		return false
	}
	return start.After(now.Add(time.Duration(s.Booking.CancelNoticeHours) * time.Hour))
}
