package clinical_workers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type fakeCalendar struct{ synced []clinical_models.Appointment }

func (f *fakeCalendar) SyncStatusChange(_ uint, a *clinical_models.Appointment) {
	f.synced = append(f.synced, *a)
}

type noShowFixture struct {
	t   *testing.T
	db  *gorm.DB
	loc *time.Location
	seq int
}

func (f *noShowFixture) tenant(settings *tenant_models.UISettings) tenant_models.Tenant {
	f.t.Helper()
	f.seq++
	tenant := tenant_models.Tenant{Name: "Clinic", Slug: fmt.Sprintf("noshow-%d-%d", f.seq, time.Now().UnixNano()), DisplayToken: fmt.Sprintf("tok-noshow-%d-%d", f.seq, time.Now().UnixNano())}
	if settings != nil {
		raw, _ := json.Marshal(settings)
		tenant.UISettings = string(raw)
	}
	if err := f.db.Create(&tenant).Error; err != nil {
		f.t.Fatalf("create tenant: %v", err)
	}
	return tenant
}

func (f *noShowFixture) patient(tenantID uint) clinical_models.Patient {
	f.t.Helper()
	f.seq++
	p := clinical_models.Patient{TenantID: tenantID, FirstName: "P", LastName: "X", Document: fmt.Sprintf("DOC-NS-%d-%d", f.seq, time.Now().UnixNano())}
	if err := f.db.Create(&p).Error; err != nil {
		f.t.Fatalf("create patient: %v", err)
	}
	return p
}

func (f *noShowFixture) doctor(tenantID uint) doctor_models.Doctor {
	f.t.Helper()
	d := doctor_models.Doctor{TenantID: tenantID, FullName: "Dra.", Specialty: "general_medicine", Active: true}
	if err := f.db.Create(&d).Error; err != nil {
		f.t.Fatalf("create doctor: %v", err)
	}
	return d
}

// appt creates an appointment on day (local midnight, as the frontend sends it).
func (f *noShowFixture) appt(tenantID, patientID uint, doctorID *uint, day time.Time, start, end, status, eventID string) clinical_models.Appointment {
	f.t.Helper()
	a := clinical_models.Appointment{
		TenantID: tenantID, PatientID: patientID, DoctorID: doctorID, Title: "Control",
		Date:      time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, f.loc),
		StartTime: start, EndTime: end, Status: status, GoogleEventID: eventID,
	}
	if err := f.db.Create(&a).Error; err != nil {
		f.t.Fatalf("create appointment: %v", err)
	}
	return a
}

func (f *noShowFixture) record(tenantID, patientID uint, doctorID, appointmentID *uint, at time.Time) {
	f.t.Helper()
	r := clinical_models.MedicalRecord{TenantID: tenantID, PatientID: patientID, DoctorID: doctorID, AppointmentID: appointmentID, Date: at}
	if err := f.db.Create(&r).Error; err != nil {
		f.t.Fatalf("create record: %v", err)
	}
}

func (f *noShowFixture) status(id uint) string {
	f.t.Helper()
	var a clinical_models.Appointment
	if err := f.db.First(&a, id).Error; err != nil {
		f.t.Fatalf("reload appointment: %v", err)
	}
	return a.Status
}

func agendaOn(since time.Time, minutes int) *tenant_models.UISettings {
	s := tenant_models.DefaultUISettings()
	s.Agenda = tenant_models.AgendaSettings{AutoNoShow: true, NoShowAfterMinutes: minutes, AutoNoShowSince: &since}
	return &s
}

func TestNoShowScheduler_MarksOnlyEligibleAppointments(t *testing.T) {
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatal(err)
	}
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&clinical_models.MedicalRecord{}, &clinical_models.VitalSigns{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{})
	f := &noShowFixture{t: t, db: db, loc: loc}

	now := time.Date(2026, 10, 28, 12, 0, 0, 0, loc)
	today := now
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)
	since := time.Date(2026, 10, 27, 0, 0, 0, 0, loc)

	on := f.tenant(agendaOn(since, 30))
	off := f.tenant(nil) // no agenda section: off
	docA, docB := f.doctor(on.ID), f.doctor(on.ID)

	// Marked.
	plain := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, today, "09:00", "09:30", "scheduled", "evt-1")
	confirmed := f.appt(on.ID, f.patient(on.ID).ID, nil, today, "10:00", "10:30", "confirmed", "")
	fromYesterday := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, yesterday, "16:00", "16:30", "scheduled", "")
	otherDoctorPatient := f.patient(on.ID)
	otherDoctorRecord := f.appt(on.ID, otherDoctorPatient.ID, &docA.ID, today, "08:00", "08:30", "scheduled", "")
	f.record(on.ID, otherDoctorPatient.ID, &docB.ID, nil, time.Date(2026, 10, 28, 8, 15, 0, 0, loc))
	otherDayPatient := f.patient(on.ID)
	otherDayRecord := f.appt(on.ID, otherDayPatient.ID, &docA.ID, today, "08:00", "08:30", "scheduled", "")
	f.record(on.ID, otherDayPatient.ID, &docA.ID, nil, time.Date(2026, 10, 27, 23, 30, 0, 0, loc))

	// Untouched.
	withinGrace := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, today, "11:00", "11:40", "scheduled", "")
	beforeSince := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, twoDaysAgo, "09:00", "09:30", "scheduled", "")
	sameDoctorPatient := f.patient(on.ID)
	sameDoctor := f.appt(on.ID, sameDoctorPatient.ID, &docA.ID, today, "08:00", "08:30", "scheduled", "")
	f.record(on.ID, sameDoctorPatient.ID, &docA.ID, nil, time.Date(2026, 10, 28, 8, 20, 0, 0, loc))
	legacyRecordPatient := f.patient(on.ID)
	legacyRecord := f.appt(on.ID, legacyRecordPatient.ID, &docA.ID, today, "08:00", "08:30", "scheduled", "")
	f.record(on.ID, legacyRecordPatient.ID, nil, nil, time.Date(2026, 10, 28, 9, 0, 0, 0, loc))
	linkedPatient := f.patient(on.ID)
	linked := f.appt(on.ID, linkedPatient.ID, &docA.ID, today, "07:00", "07:30", "scheduled", "")
	f.record(on.ID, linkedPatient.ID, &docB.ID, &linked.ID, time.Date(2026, 10, 25, 9, 0, 0, 0, loc)) // linked, any day
	noDoctorPatient := f.patient(on.ID)
	noDoctor := f.appt(on.ID, noDoctorPatient.ID, nil, today, "08:00", "08:30", "confirmed", "")
	f.record(on.ID, noDoctorPatient.ID, &docB.ID, nil, time.Date(2026, 10, 28, 10, 0, 0, 0, loc))
	arrived := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, today, "08:00", "08:30", "arrived", "")
	inConsultation := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, today, "08:00", "08:30", "in_consultation", "")
	completed := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, today, "08:00", "08:30", "completed", "")
	cancelled := f.appt(on.ID, f.patient(on.ID).ID, &docA.ID, today, "08:00", "08:30", "cancelled", "")
	otherTenant := f.appt(off.ID, f.patient(off.ID).ID, nil, today, "08:00", "08:30", "scheduled", "evt-off")

	calendar := &fakeCalendar{}
	s := NewNoShowScheduler(db, zap.NewNop(), calendar).WithClock(func() time.Time { return now }, loc)

	if marked := s.Tick(); marked != 5 {
		t.Errorf("first tick marked %d, want 5", marked)
	}
	for name, a := range map[string]clinical_models.Appointment{
		"plain": plain, "confirmed": confirmed, "from yesterday": fromYesterday,
		"record by another doctor": otherDoctorRecord, "record another day": otherDayRecord,
	} {
		if got := f.status(a.ID); got != "no_show" {
			t.Errorf("%s: status = %q, want no_show", name, got)
		}
	}
	for name, a := range map[string]clinical_models.Appointment{
		"within grace": withinGrace, "before since": beforeSince, "same-doctor record": sameDoctor,
		"record without doctor": legacyRecord, "linked record": linked, "no doctor + record": noDoctor,
		"arrived": arrived, "in consultation": inConsultation, "completed": completed, "cancelled": cancelled,
		"setting off": otherTenant,
	} {
		if got := f.status(a.ID); got != a.Status {
			t.Errorf("%s: status = %q, want %q", name, got, a.Status)
		}
	}

	// Only the appointment with a Google event is synced, as no_show.
	if len(calendar.synced) != 1 || calendar.synced[0].ID != plain.ID || calendar.synced[0].Status != "no_show" {
		t.Errorf("calendar synced %+v, want only appointment %d as no_show", calendar.synced, plain.ID)
	}

	// Idempotent: a second run marks nothing and syncs nothing more.
	if marked := s.Tick(); marked != 0 {
		t.Errorf("second tick marked %d, want 0", marked)
	}
	if len(calendar.synced) != 1 {
		t.Errorf("second tick synced again: %d calls", len(calendar.synced))
	}

	// Once the grace period is over, the late one is marked too.
	later := now.Add(30 * time.Minute)
	s.WithClock(func() time.Time { return later }, loc)
	if marked := s.Tick(); marked != 1 || f.status(withinGrace.ID) != "no_show" {
		t.Errorf("after grace: marked %d, status %q", marked, f.status(withinGrace.ID))
	}
}

func TestNoShowScheduler_OffOrWithoutSinceMarksNothing(t *testing.T) {
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Fatal(err)
	}
	db := testutils.SetupTestDB(t, &doctor_models.Doctor{}, &tenant_models.Tenant{}, &clinical_models.Patient{}, &clinical_models.Appointment{},
		&clinical_models.MedicalRecord{}, &clinical_models.VitalSigns{}, &clinical_models.SOAPRecord{}, &clinical_models.Prescription{})
	f := &noShowFixture{t: t, db: db, loc: loc}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, loc)

	explicitOff := tenant_models.DefaultUISettings()
	explicitOff.Agenda.AutoNoShowSince = &now // stale since, flag off
	noSince := agendaOn(now, 30)
	noSince.Agenda.AutoNoShowSince = nil

	for _, settings := range []*tenant_models.UISettings{nil, &explicitOff, noSince} {
		tenant := f.tenant(settings)
		f.appt(tenant.ID, f.patient(tenant.ID).ID, nil, now.AddDate(0, 0, -1), "09:00", "09:30", "scheduled", "")
	}
	s := NewNoShowScheduler(db, zap.NewNop(), nil).WithClock(func() time.Time { return now }, loc)
	if marked := s.Tick(); marked != 0 {
		t.Fatalf("marked %d, want 0", marked)
	}
}
