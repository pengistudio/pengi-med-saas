package agenda_services

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	agenda_models "pengi-med-saas/features/agenda/models"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	portal_models "pengi-med-saas/features/portal/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

var slotLoc = time.FixedZone("ECT", -5*3600)

type slotFixture struct {
	t        *testing.T
	db       *gorm.DB
	tenantID uint
	doctor   doctor_models.Doctor
	idle     doctor_models.Doctor // no weekly schedule
	patient  clinical_models.Patient
}

func newSlotFixture(t *testing.T) *slotFixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &doctor_models.Doctor{}, &clinical_models.Patient{},
		&clinical_models.Appointment{}, &agenda_models.DoctorSchedule{}, &agenda_models.ScheduleBlock{},
		&portal_models.AppointmentRequest{})
	f := &slotFixture{t: t, db: db}
	now := time.Now().UnixNano()
	tenant := tenant_models.Tenant{Name: "Slots", Slug: fmt.Sprintf("slots-%d", now), DisplayToken: fmt.Sprintf("tok-slots-%d", now)}
	f.create(&tenant)
	f.tenantID = tenant.ID
	f.doctor = doctor_models.Doctor{TenantID: tenant.ID, FullName: "Dr. Slot", Specialty: "general", Active: true}
	f.create(&f.doctor)
	f.idle = doctor_models.Doctor{TenantID: tenant.ID, FullName: "Dr. Idle", Specialty: "general", Active: true}
	f.create(&f.idle)
	f.patient = clinical_models.Patient{TenantID: tenant.ID, FirstName: "P", LastName: "S"}
	f.create(&f.patient)
	// Mondays 08:00-12:00.
	f.create(&agenda_models.DoctorSchedule{TenantID: tenant.ID, DoctorID: f.doctor.ID, Weekday: int(time.Monday), StartTime: "08:00", EndTime: "12:00"})
	return f
}

func (f *slotFixture) create(v any) {
	f.t.Helper()
	if err := tenantdb.System(f.db).Create(v).Error; err != nil {
		f.t.Fatalf("create %T: %v", v, err)
	}
}

func (f *slotFixture) appointment(date, start, end, status string) {
	d, _ := ParseDate(date)
	doctorID := f.doctor.ID
	f.create(&clinical_models.Appointment{
		TenantID: f.tenantID, PatientID: f.patient.ID, Title: "x", Date: time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, slotLoc),
		StartTime: start, EndTime: end, Status: status, DoctorID: &doctorID,
	})
}

func (f *slotFixture) request(date, start, end, status string, expires time.Time) portal_models.AppointmentRequest {
	r := portal_models.AppointmentRequest{
		TenantID: f.tenantID, PatientID: f.patient.ID, DoctorID: f.doctor.ID, AppointmentTypeID: 1,
		Date: date, StartTime: start, EndTime: end, Status: status, ExpiresAt: expires,
	}
	f.create(&r)
	return r
}

func (f *slotFixture) slots(q SlotQuery) FreeSlotMap {
	f.t.Helper()
	if q.DoctorIDs == nil {
		q.DoctorIDs = []uint{f.doctor.ID}
	}
	if q.Duration == 0 {
		q.Duration = 30
	}
	if q.Now.IsZero() {
		q.Now = time.Date(2026, 10, 26, 10, 0, 0, 0, slotLoc)
	}
	q.Loc = slotLoc
	got, err := FreeSlots(tenantdb.ForTenant(f.db, f.tenantID), q)
	if err != nil {
		f.t.Fatal(err)
	}
	return got
}

const monday = "2026-11-02"

func TestFreeSlots_ScheduleGrid(t *testing.T) {
	f := newSlotFixture(t)
	got := f.slots(SlotQuery{From: "2026-11-01", To: "2026-11-03"})
	want := []string{"08:00", "08:30", "09:00", "09:30", "10:00", "10:30", "11:00", "11:30"}
	if !reflect.DeepEqual(got[f.doctor.ID][monday], want) {
		t.Fatalf("monday = %v", got[f.doctor.ID][monday])
	}
	if len(got[f.doctor.ID]) != 1 {
		t.Fatalf("only monday has a schedule: %v", got)
	}
	// 45-minute type: steps of 45 from the range start, ending by 12:00.
	got = f.slots(SlotQuery{From: monday, To: monday, Duration: 45})
	if !reflect.DeepEqual(got[f.doctor.ID][monday], []string{"08:00", "08:45", "09:30", "10:15", "11:00"}) {
		t.Fatalf("45 min = %v", got[f.doctor.ID][monday])
	}
}

func TestFreeSlots_Step(t *testing.T) {
	f := newSlotFixture(t)
	// 20-minute type offered every 60: whole hours, the last one ends by 12:00.
	got := f.slots(SlotQuery{From: monday, To: monday, Duration: 20, Step: 60})
	if !reflect.DeepEqual(got[f.doctor.ID][monday], []string{"08:00", "09:00", "10:00", "11:00"}) {
		t.Fatalf("step 60 = %v", got[f.doctor.ID][monday])
	}
	// 30-minute type every 15: a start is offered only if the whole 30 fit;
	// 09:00-09:30 taken drops 08:45 and 09:15 too.
	f.appointment(monday, "09:00", "09:30", "scheduled")
	got = f.slots(SlotQuery{From: monday, To: monday, Duration: 30, Step: 15})
	want := []string{"08:00", "08:15", "08:30", "09:30", "09:45", "10:00", "10:15", "10:30", "10:45", "11:00", "11:15", "11:30"}
	if !reflect.DeepEqual(got[f.doctor.ID][monday], want) {
		t.Fatalf("step 15 = %v", got[f.doctor.ID][monday])
	}
}

func TestFreeSlots_NoScheduleNothing(t *testing.T) {
	f := newSlotFixture(t)
	got := f.slots(SlotQuery{DoctorIDs: []uint{f.idle.ID}, From: monday, To: monday})
	if len(got) != 0 {
		t.Fatalf("doctor without schedule = %v", got)
	}
}

func TestFreeSlots_Blocks(t *testing.T) {
	f := newSlotFixture(t)
	doctorID := f.doctor.ID
	// Partial doctor block on monday, clinic full-day block the next monday.
	f.create(&agenda_models.ScheduleBlock{TenantID: f.tenantID, DoctorID: &doctorID, StartDate: monday, EndDate: monday, StartTime: "09:15", EndTime: "10:00"})
	f.create(&agenda_models.ScheduleBlock{TenantID: f.tenantID, StartDate: "2026-11-09", EndDate: "2026-11-09"})
	got := f.slots(SlotQuery{From: monday, To: "2026-11-09"})
	if !reflect.DeepEqual(got[f.doctor.ID][monday], []string{"08:00", "08:30", "10:00", "10:30", "11:00", "11:30"}) {
		t.Fatalf("partial block = %v", got[f.doctor.ID][monday])
	}
	if _, ok := got[f.doctor.ID]["2026-11-09"]; ok {
		t.Fatal("clinic holiday must offer nothing")
	}
}

func TestFreeSlots_AppointmentsAndHeldRequests(t *testing.T) {
	f := newSlotFixture(t)
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, slotLoc)
	f.appointment(monday, "10:00", "10:45", "scheduled")
	f.appointment(monday, "08:00", "08:30", "cancelled") // does not take the slot
	held := f.request(monday, "11:00", "11:30", portal_models.RequestPending, now.Add(time.Hour))
	f.request(monday, "09:00", "09:30", portal_models.RequestPending, now.Add(-time.Minute)) // expired hold
	f.request(monday, "09:30", "10:00", portal_models.RequestRejected, now.Add(time.Hour))

	got := f.slots(SlotQuery{From: monday, To: monday, Now: now})
	want := []string{"08:00", "08:30", "09:00", "09:30", "11:30"}
	if !reflect.DeepEqual(got[f.doctor.ID][monday], want) {
		t.Fatalf("taken = %v", got[f.doctor.ID][monday])
	}
	if !got.Has(f.doctor.ID, monday, "09:00") || got.Has(f.doctor.ID, monday, "11:00") {
		t.Fatal("Has disagrees with the map")
	}
	// The request being re-checked does not hold against itself.
	got = f.slots(SlotQuery{From: monday, To: monday, Now: now, IgnoreRequestID: held.ID})
	if !got.Has(f.doctor.ID, monday, "11:00") {
		t.Fatalf("ignored request still holds: %v", got[f.doctor.ID][monday])
	}
	overlaps, err := AppointmentOverlaps(tenantdb.ForTenant(f.db, f.tenantID), f.doctor.ID, monday, "10:30", "11:00", slotLoc)
	if err != nil || !overlaps {
		t.Fatalf("overlap = %v %v", overlaps, err)
	}
	overlaps, _ = AppointmentOverlaps(tenantdb.ForTenant(f.db, f.tenantID), f.doctor.ID, monday, "08:00", "08:30", slotLoc)
	if overlaps {
		t.Fatal("a cancelled appointment does not overlap")
	}
}

func TestFreeSlots_MinNotice(t *testing.T) {
	f := newSlotFixture(t)
	now := time.Date(2026, 11, 2, 8, 10, 0, 0, slotLoc)
	got := f.slots(SlotQuery{From: monday, To: monday, Now: now, MinNotice: 2 * time.Hour})
	if !reflect.DeepEqual(got[f.doctor.ID][monday], []string{"10:30", "11:00", "11:30"}) {
		t.Fatalf("min notice = %v", got[f.doctor.ID][monday])
	}
}
