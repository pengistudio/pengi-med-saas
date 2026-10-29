package waitlist_services_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	company_models "pengi-med-saas/features/companies/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	notifications_models "pengi-med-saas/features/notifications/models"
	permission_models "pengi-med-saas/features/permissions/models"
	portal_models "pengi-med-saas/features/portal/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"
	waitlist_models "pengi-med-saas/features/waitlist/models"
	waitlist_services "pengi-med-saas/features/waitlist/services"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var loc = time.FixedZone("ECT", -5*3600)

// now is Monday 2 November 2026, 09:00 clinic time.
var now = time.Date(2026, 11, 2, 9, 0, 0, 0, loc)

type fx struct {
	t          *testing.T
	db         *gorm.DB
	own, other uint
	seq        int
}

func newFx(t *testing.T) *fx {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &company_models.Company{}, &user_models.Environment{},
		&user_models.Role{}, &permission_models.Permission{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&doctor_models.Doctor{}, &agenda_models.AppointmentType{}, &waitlist_models.WaitlistEntry{},
		&notifications_models.Notification{})
	f := &fx{t: t, db: db}
	stamp := time.Now().UnixNano()
	for i, id := range []*uint{&f.own, &f.other} {
		slug := fmt.Sprintf("wl-%d-%d", i, stamp)
		tenant := tenant_models.Tenant{Name: slug, Slug: slug, DisplayToken: "tok-" + slug}
		if err := db.Create(&tenant).Error; err != nil {
			t.Fatal(err)
		}
		*id = tenant.ID
		if err := db.Create(&company_models.Company{LegalName: "C", TradeName: "C", PlanCode: "P", TenantID: tenant.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *fx) sys() *gorm.DB { return tenantdb.System(f.db) }

func (f *fx) patient(tenantID uint) clinical_models.Patient {
	f.t.Helper()
	f.seq++
	p := clinical_models.Patient{TenantID: tenantID, FirstName: fmt.Sprintf("P%d", f.seq), LastName: "Mora",
		Phone: "0991234567", Document: fmt.Sprintf("D%d-%d", f.seq, time.Now().UnixNano())}
	if err := f.sys().Create(&p).Error; err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fx) doctor(tenantID uint, name string) doctor_models.Doctor {
	f.t.Helper()
	d := doctor_models.Doctor{TenantID: tenantID, FullName: name, Specialty: "general"}
	if err := f.sys().Create(&d).Error; err != nil {
		f.t.Fatal(err)
	}
	return d
}

func (f *fx) apptType(tenantID uint, name string) agenda_models.AppointmentType {
	f.t.Helper()
	at := agenda_models.AppointmentType{TenantID: tenantID, Name: name, DurationMinutes: 30, Active: true}
	if err := f.sys().Create(&at).Error; err != nil {
		f.t.Fatal(err)
	}
	return at
}

// entry adds a waiting entry of patientID; mod adjusts it before insert.
// Entries are inserted in call order, one second apart (signup order).
func (f *fx) entry(tenantID, patientID uint, mod func(*waitlist_models.WaitlistEntry)) waitlist_models.WaitlistEntry {
	f.t.Helper()
	f.seq++
	e := waitlist_models.WaitlistEntry{TenantID: tenantID, PatientID: patientID, Status: waitlist_models.StatusWaiting,
		Source: waitlist_models.SourceStaff, ExpiresAt: now.Add(30 * 24 * time.Hour)}
	e.CreatedAt = now.Add(-48*time.Hour + time.Duration(f.seq)*time.Second)
	if mod != nil {
		mod(&e)
	}
	if err := f.sys().Create(&e).Error; err != nil {
		f.t.Fatal(err)
	}
	return e
}

func (f *fx) appointment(tenantID, patientID uint, doctorID *uint, day, start, status string, createdAt time.Time) clinical_models.Appointment {
	f.t.Helper()
	d, _ := time.ParseInLocation("2006-01-02", day, loc)
	a := clinical_models.Appointment{TenantID: tenantID, PatientID: patientID, DoctorID: doctorID, Title: "Cita",
		Date: d, StartTime: start, EndTime: "23:59", Status: status}
	a.CreatedAt = createdAt
	if err := f.sys().Create(&a).Error; err != nil {
		f.t.Fatal(err)
	}
	return a
}

func (f *fx) grant(tenantID uint, userIDs []uint, permissionID string) {
	f.t.Helper()
	var company company_models.Company
	if err := f.sys().Where("tenant_id = ?", tenantID).First(&company).Error; err != nil {
		f.t.Fatal(err)
	}
	perm := permission_models.Permission{BaseStringID: database.BaseStringID{ID: permissionID}, Name: permissionID}
	if err := f.db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
		f.t.Fatal(err)
	}
	role := user_models.Role{Role: fmt.Sprintf("r-%d", time.Now().UnixNano()), Permissions: []permission_models.Permission{perm}}
	if err := f.db.Create(&role).Error; err != nil {
		f.t.Fatal(err)
	}
	for _, uid := range userIDs {
		if err := f.db.Create(&user_models.Environment{UserID: uid, CompanyID: company.ID, RoleID: role.ID}).Error; err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fx) notifications(userID uint) []notifications_models.Notification {
	var out []notifications_models.Notification
	f.sys().Where("user_id = ? AND type = ?", userID, waitlist_services.NotificationTypeSlotFreed).Order("id").Find(&out)
	return out
}

func (f *fx) status(id uint) string {
	var e waitlist_models.WaitlistEntry
	f.sys().First(&e, id)
	return e.Status
}

func ids(entries []waitlist_models.WaitlistEntry) []uint {
	out := make([]uint, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.ID)
	}
	return out
}

func TestMatching_Rules(t *testing.T) {
	f := newFx(t)
	d1, d2 := f.doctor(f.own, "Dra. Ana"), f.doctor(f.own, "Dr. Luis")
	t1, t2 := f.apptType(f.own, "Consulta"), f.apptType(f.own, "Control")
	freer := f.patient(f.own)
	slot := waitlist_services.Slot{DoctorID: &d1.ID, AppointmentTypeID: &t1.ID, Date: "2026-11-04", StartTime: "10:00", EndTime: "10:30", PatientID: freer.ID}

	sameDoctor := f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.DoctorID = &d1.ID })
	anyDoctor := f.entry(f.own, f.patient(f.own).ID, nil)
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.DoctorID = &d2.ID })
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.AppointmentTypeID = &t2.ID })
	sameType := f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.AppointmentTypeID = &t1.ID })
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.DateTo = "2026-11-03" })
	inRange := f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) {
		e.DateFrom, e.DateTo = "2026-11-04", "2026-11-04"
	})
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.DateFrom = "2026-11-05" })

	// Move earlier: only a slot before the appointment; a cancelled one no longer constrains.
	later := f.patient(f.own)
	laterAppt := f.appointment(f.own, later.ID, &d1.ID, "2026-11-20", "09:00", "scheduled", now.Add(-72*time.Hour))
	moveEarlier := f.entry(f.own, later.ID, func(e *waitlist_models.WaitlistEntry) { e.AppointmentID = &laterAppt.ID })
	earlier := f.patient(f.own)
	earlierAppt := f.appointment(f.own, earlier.ID, &d1.ID, "2026-11-04", "08:00", "scheduled", now.Add(-72*time.Hour))
	f.entry(f.own, earlier.ID, func(e *waitlist_models.WaitlistEntry) { e.AppointmentID = &earlierAppt.ID })
	gone := f.patient(f.own)
	goneAppt := f.appointment(f.own, gone.ID, &d1.ID, "2026-11-03", "08:00", "cancelled", now.Add(-72*time.Hour))
	cancelledLink := f.entry(f.own, gone.ID, func(e *waitlist_models.WaitlistEntry) { e.AppointmentID = &goneAppt.ID })

	// Closed or expired entries, the freeing patient and other clinics never match.
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.Status = waitlist_models.StatusRemoved })
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.ExpiresAt = now.Add(-time.Minute) })
	f.entry(f.own, freer.ID, nil)
	f.entry(f.other, f.patient(f.other).ID, nil)
	offered := f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.Status = waitlist_models.StatusOffered })

	got, err := waitlist_services.Matching(tenantdb.ForTenant(f.db, f.own), slot, now, loc)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint{sameDoctor.ID, anyDoctor.ID, sameType.ID, inRange.ID, moveEarlier.ID, cancelledLink.ID, offered.ID}
	if fmt.Sprint(ids(got)) != fmt.Sprint(want) {
		t.Fatalf("matching = %v, want %v (signup order)", ids(got), want)
	}

	// A slot without a doctor only matches any-doctor entries; an untyped
	// slot matches every type.
	noDoctor := slot
	noDoctor.DoctorID, noDoctor.AppointmentTypeID = nil, nil
	got, err = waitlist_services.Matching(tenantdb.ForTenant(f.db, f.own), noDoctor, now, loc)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range got {
		if e.DoctorID != nil {
			t.Fatalf("entry %d with a doctor matched a slot without one", e.ID)
		}
	}

	ok, err := waitlist_services.Matches(tenantdb.ForTenant(f.db, f.own), sameDoctor, slot, now, loc)
	if err != nil || !ok {
		t.Fatalf("Matches(sameDoctor) = %v, %v", ok, err)
	}
	other := slot
	other.DoctorID = &d2.ID
	if ok, _ := waitlist_services.Matches(tenantdb.ForTenant(f.db, f.own), sameDoctor, other, now, loc); ok {
		t.Fatal("an entry for Dra. Ana matched Dr. Luis's slot")
	}
}

func TestRefresh_ExpiresAndBooks(t *testing.T) {
	f := newFx(t)
	d1, d2 := f.doctor(f.own, "Dra. Ana"), f.doctor(f.own, "Dr. Luis")

	expired := f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.ExpiresAt = now.Add(-time.Second) })

	pBooked := f.patient(f.own)
	booked := f.entry(f.own, pBooked.ID, func(e *waitlist_models.WaitlistEntry) { e.DoctorID = &d1.ID })
	f.appointment(f.own, pBooked.ID, &d1.ID, "2026-11-10", "10:00", "scheduled", booked.CreatedAt.Add(time.Hour))

	pAny := f.patient(f.own)
	anyBooked := f.entry(f.own, pAny.ID, nil)
	f.appointment(f.own, pAny.ID, &d2.ID, "2026-11-10", "11:00", "scheduled", anyBooked.CreatedAt.Add(time.Hour))

	pOtherDoctor := f.patient(f.own)
	otherDoctor := f.entry(f.own, pOtherDoctor.ID, func(e *waitlist_models.WaitlistEntry) { e.DoctorID = &d1.ID })
	f.appointment(f.own, pOtherDoctor.ID, &d2.ID, "2026-11-10", "12:00", "scheduled", otherDoctor.CreatedAt.Add(time.Hour))

	pCancelled := f.patient(f.own)
	cancelled := f.entry(f.own, pCancelled.ID, nil)
	f.appointment(f.own, pCancelled.ID, &d1.ID, "2026-11-10", "13:00", "cancelled", cancelled.CreatedAt.Add(time.Hour))

	pBefore := f.patient(f.own)
	before := f.entry(f.own, pBefore.ID, nil)
	f.appointment(f.own, pBefore.ID, &d1.ID, "2026-11-20", "09:00", "scheduled", before.CreatedAt.Add(-time.Hour))

	// Another clinic's refresh leaves this one alone.
	if err := waitlist_services.Refresh(tenantdb.ForTenant(f.db, f.other), now); err != nil {
		t.Fatal(err)
	}
	if s := f.status(expired.ID); s != waitlist_models.StatusWaiting {
		t.Fatalf("other tenant's refresh changed the entry to %s", s)
	}

	if err := waitlist_services.Refresh(tenantdb.ForTenant(f.db, f.own), now); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		id   uint
		want string
	}{
		{"expired", expired.ID, waitlist_models.StatusExpired},
		{"booked with its doctor", booked.ID, waitlist_models.StatusBooked},
		{"any doctor, booked with any", anyBooked.ID, waitlist_models.StatusBooked},
		{"booked with another doctor", otherDoctor.ID, waitlist_models.StatusWaiting},
		{"cancelled appointment", cancelled.ID, waitlist_models.StatusWaiting},
		{"appointment from before joining", before.ID, waitlist_models.StatusWaiting},
	} {
		if s := f.status(c.id); s != c.want {
			t.Errorf("%s: status = %s, want %s", c.name, s, c.want)
		}
	}
}

func TestExpiresAt(t *testing.T) {
	if got := waitlist_services.ExpiresAt(now, "", "2026-11-10", loc); !got.Equal(time.Date(2026, 11, 11, 0, 0, 0, 0, loc)) {
		t.Fatalf("with an end date: %v", got)
	}
	if got := waitlist_services.ExpiresAt(now, "", "", loc); !got.Equal(now.Add(30 * 24 * time.Hour)) {
		t.Fatalf("open: %v", got)
	}
	from := time.Date(2027, 1, 10, 0, 0, 0, 0, loc)
	if got := waitlist_services.ExpiresAt(now, "2027-01-10", "", loc); !got.Equal(from.Add(30 * 24 * time.Hour)) {
		t.Fatalf("starting later: %v", got)
	}
}

func TestSlotsFreed_NotifiesAppointmentManagers(t *testing.T) {
	f := newFx(t)
	f.grant(f.own, []uint{100}, waitlist_services.ManagePermission)
	f.grant(f.own, []uint{101}, "READ_PATIENT")
	f.grant(f.other, []uint{110}, waitlist_services.ManagePermission)
	d1 := f.doctor(f.own, "Dra. Ana")
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.DoctorID = &d1.ID })
	f.entry(f.own, f.patient(f.own).ID, nil)
	f.entry(f.other, f.patient(f.other).ID, nil)
	slots := waitlist_services.NewSlots(f.db, zap.NewNop()).WithClock(func() time.Time { return now }, loc)

	freer := f.patient(f.own)
	appt := f.appointment(f.own, freer.ID, &d1.ID, "2026-11-04", "10:00", "cancelled", now.Add(-time.Hour))
	appt.EndTime = "10:30"
	if n := slots.AppointmentFreed(f.own, appt); n != 2 {
		t.Fatalf("matches = %d, want 2", n)
	}
	ns := f.notifications(100)
	if len(ns) != 1 {
		t.Fatalf("notifications of the manager = %d", len(ns))
	}
	n := ns[0]
	var params map[string]any
	_ = json.Unmarshal(n.Params, &params)
	if n.MessageKey != waitlist_services.SlotFreedKey || params["count"] != float64(2) || params["doctor"] != "Dra. Ana" ||
		params["date"] != "2026-11-04" || params["time"] != "10:00" || n.TenantID != f.own {
		t.Fatalf("notification = %+v params = %v", n, params)
	}
	if !strings.HasPrefix(n.ActionURL, "/clinical/waitlist?") || !strings.Contains(n.ActionURL, "date=2026-11-04") ||
		!strings.Contains(n.ActionURL, fmt.Sprintf("doctor_id=%d", d1.ID)) || !strings.Contains(n.ActionURL, "start_time=10%3A00") ||
		!strings.Contains(n.ActionURL, fmt.Sprintf("patient_id=%d", freer.ID)) {
		t.Fatalf("action url = %s", n.ActionURL)
	}
	if len(f.notifications(101)) != 0 || len(f.notifications(110)) != 0 {
		t.Fatal("only the clinic's MANAGE_APPOINTMENT users are told")
	}

	// A slot nobody waits for, or already started, tells nobody.
	d2 := f.doctor(f.own, "Dr. Luis")
	if err := f.sys().Where("tenant_id = ? AND doctor_id IS NULL", f.own).Delete(&waitlist_models.WaitlistEntry{}).Error; err != nil {
		t.Fatal(err)
	}
	other := f.appointment(f.own, freer.ID, &d2.ID, "2026-11-04", "11:00", "cancelled", now)
	if n := slots.AppointmentFreed(f.own, other); n != 0 {
		t.Fatalf("matches for an unwanted slot = %d", n)
	}
	past := f.appointment(f.own, freer.ID, &d1.ID, "2026-11-02", "08:00", "cancelled", now)
	if n := slots.AppointmentFreed(f.own, past); n != 0 {
		t.Fatalf("matches for a past slot = %d", n)
	}
	if len(f.notifications(100)) != 1 {
		t.Fatalf("notifications = %d, want still 1", len(f.notifications(100)))
	}
}

func TestSlotsFreed_RequestReleased(t *testing.T) {
	f := newFx(t)
	f.grant(f.own, []uint{100}, waitlist_services.ManagePermission)
	d1 := f.doctor(f.own, "Dra. Ana")
	t1 := f.apptType(f.own, "Consulta")
	f.entry(f.own, f.patient(f.own).ID, func(e *waitlist_models.WaitlistEntry) { e.AppointmentTypeID = &t1.ID })
	requester := f.patient(f.own)
	f.entry(f.own, requester.ID, nil) // the requester never gets their own slot
	slots := waitlist_services.NewSlots(f.db, zap.NewNop()).WithClock(func() time.Time { return now }, loc)

	req := portal_models.AppointmentRequest{TenantID: f.own, PatientID: requester.ID, DoctorID: d1.ID, AppointmentTypeID: t1.ID,
		Date: "2026-11-05", StartTime: "09:00", EndTime: "09:30"}
	if n := slots.RequestReleased(f.own, req); n != 1 {
		t.Fatalf("matches = %d, want 1", n)
	}
	if len(f.notifications(100)) != 1 {
		t.Fatal("the released slot must be notified")
	}
}
