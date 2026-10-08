package agenda_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	agenda_dto "pengi-med-saas/features/agenda/dto"
	agenda_models "pengi-med-saas/features/agenda/models"
	agenda_services "pengi-med-saas/features/agenda/services"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 2026-10-19 is a Monday (weekday 1).
const monday = "2026-10-19"

type fixture struct {
	t          *testing.T
	db         *gorm.DB
	h          *AgendaHandler
	own, other tenant_models.Tenant
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &doctor_models.Doctor{}, &clinical_models.Patient{},
		&clinical_models.Appointment{}, &agenda_models.DoctorSchedule{}, &agenda_models.ScheduleBlock{}, &agenda_models.AppointmentType{})
	f := &fixture{t: t, db: db, h: NewAgendaHandler(db, zap.NewNop())}
	now := time.Now().UnixNano()
	for i, tenant := range []*tenant_models.Tenant{&f.own, &f.other} {
		*tenant = tenant_models.Tenant{Name: fmt.Sprintf("Clinic %d", i), Slug: fmt.Sprintf("ag-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-ag-%d-%d", i, now)}
		f.mustCreate(tenant)
	}
	return f
}

func (f *fixture) mustCreate(v any) {
	f.t.Helper()
	if err := f.db.Create(v).Error; err != nil {
		f.t.Fatalf("create %T: %v", v, err)
	}
}

func (f *fixture) doctor(tenant tenant_models.Tenant, name string, userID *uint) doctor_models.Doctor {
	f.t.Helper()
	d := doctor_models.Doctor{TenantID: tenant.ID, UserID: userID, FullName: name, Specialty: "pediatrics", Color: "#2563EB", Active: true}
	f.mustCreate(&d)
	return d
}

func (f *fixture) appointment(tenant tenant_models.Tenant, doctorID *uint, date, start, end, status string) clinical_models.Appointment {
	f.t.Helper()
	p := clinical_models.Patient{TenantID: tenant.ID, FirstName: "Ana", LastName: "Pérez", Institution: "H", Document: fmt.Sprintf("DOC-%d", time.Now().UnixNano())}
	f.mustCreate(&p)
	day, _ := time.Parse(agenda_services.DateLayout, date)
	// Noon UTC: the same calendar day in any session time zone.
	a := clinical_models.Appointment{TenantID: tenant.ID, PatientID: p.ID, Title: "Control", Date: day.Add(12 * time.Hour),
		StartTime: start, EndTime: end, Status: status, DoctorID: doctorID}
	f.mustCreate(&a)
	return a
}

// call runs handler as userID of tenant with an optional JSON body, query
// string and path params (name, value pairs).
func call(handler func(*gin.Context) envelope.Response, tenant tenant_models.Tenant, userID uint, query string, body any, params ...string) envelope.Response {
	c, _ := testutils.NewGinContext(tenant.ID, int64(userID))
	c.Set("user_id", int64(userID))
	c.Set("username", "tester")
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/?"+query, &buf)
	c.Request.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(params); i += 2 {
		c.Params = append(c.Params, gin.Param{Key: params[i], Value: params[i+1]})
	}
	return handler(c)
}

func wantCode(t *testing.T, resp envelope.Response, code int) {
	t.Helper()
	if resp.Code != code {
		t.Fatalf("code = %d, want %d (message %q, data %+v)", resp.Code, code, resp.Message, resp.Data)
	}
}

func wantError(t *testing.T, resp envelope.Response, code int, appErr core_errors.AppError) {
	t.Helper()
	wantCode(t, resp, code)
	got, ok := resp.Data.(core_errors.AppError)
	if !ok || got.ErrorCode != appErr.ErrorCode {
		t.Fatalf("error = %+v, want %s", resp.Data, appErr.ErrorCode)
	}
}

func slot(weekday int, start, end string) map[string]any {
	return map[string]any{"weekday": weekday, "start_time": start, "end_time": end}
}

func id(v uint) string { return fmt.Sprint(v) }

// ─── Weekly schedule ─────────────────────────────────────────────────────────

func TestReplaceSchedule_ValidatesStepsAndOverlaps(t *testing.T) {
	f := newFixture(t)
	d := f.doctor(f.own, "Dra. Ana", nil)
	put := func(slots ...map[string]any) envelope.Response {
		return call(f.h.ReplaceDoctorSchedule, f.own, 1, "", map[string]any{"slots": slots}, "id", id(d.ID))
	}

	wantError(t, put(slot(1, "08:03", "12:00")), http.StatusBadRequest, core_errors.ErrAgendaInvalidTime)
	wantError(t, put(slot(1, "12:00", "08:00")), http.StatusBadRequest, core_errors.ErrAgendaInvalidTime)
	wantError(t, put(slot(1, "08:00", "24:05")), http.StatusBadRequest, core_errors.ErrAgendaInvalidTime)
	wantError(t, put(slot(7, "08:00", "12:00")), http.StatusBadRequest, core_errors.ErrAgendaInvalidRequest)
	wantError(t, put(slot(1, "08:00", "12:00"), slot(1, "11:55", "14:00")), http.StatusBadRequest, core_errors.ErrAgendaScheduleOverlap)

	// Touching ranges and the same range on other days are fine; "24:00" closes the day.
	resp := put(slot(1, "08:00", "12:00"), slot(1, "12:00", "14:00"), slot(2, "08:00", "12:00"), slot(5, "18:00", "24:00"))
	wantCode(t, resp, http.StatusOK)
	if got := len(resp.Data.([]agenda_models.DoctorSchedule)); got != 4 {
		t.Fatalf("saved %d slots, want 4", got)
	}

	// Replacing drops the previous week.
	wantCode(t, put(slot(3, "09:00", "10:00")), http.StatusOK)
	var rows []agenda_models.DoctorSchedule
	f.db.Unscoped().Where("doctor_id = ?", d.ID).Find(&rows)
	if len(rows) != 1 || rows[0].Weekday != 3 || rows[0].TenantID != f.own.ID {
		t.Fatalf("rows after replace = %+v", rows)
	}
}

func TestMySchedule_EditsOnlyTheLinkedDoctor(t *testing.T) {
	f := newFixture(t)
	userID := uint(77)
	mine := f.doctor(f.own, "Dr. Mío", &userID)
	colleague := f.doctor(f.own, "Dra. Colega", nil)
	f.mustCreate(&agenda_models.DoctorSchedule{TenantID: f.own.ID, DoctorID: colleague.ID, Weekday: 1, StartTime: "08:00", EndTime: "12:00"})

	resp := call(f.h.ReplaceMySchedule, f.own, userID, "", map[string]any{"slots": []any{slot(2, "14:00", "18:00")}})
	wantCode(t, resp, http.StatusOK)

	var mineRows, colleagueRows []agenda_models.DoctorSchedule
	f.db.Where("doctor_id = ?", mine.ID).Find(&mineRows)
	f.db.Where("doctor_id = ?", colleague.ID).Find(&colleagueRows)
	if len(mineRows) != 1 || mineRows[0].Weekday != 2 {
		t.Fatalf("own schedule = %+v", mineRows)
	}
	if len(colleagueRows) != 1 || colleagueRows[0].Weekday != 1 {
		t.Fatalf("colleague schedule changed: %+v", colleagueRows)
	}

	// A user without a doctor profile has no "my schedule".
	wantError(t, call(f.h.ReplaceMySchedule, f.own, 999, "", map[string]any{"slots": []any{}}),
		http.StatusNotFound, core_errors.ErrDoctorProfileNotFound)
	// Own blocks land on the linked doctor only.
	resp = call(f.h.CreateMyBlock, f.own, userID, "", map[string]any{"start_date": monday, "end_date": monday})
	wantCode(t, resp, http.StatusOK)
	if b := resp.Data.(agenda_dto.BlockResponse).Block; b.DoctorID == nil || *b.DoctorID != mine.ID {
		t.Fatalf("own block doctor = %v, want %d", b.DoctorID, mine.ID)
	}
	// …and a colleague's block can't be edited through /me.
	theirs := agenda_models.ScheduleBlock{TenantID: f.own.ID, DoctorID: &colleague.ID, StartDate: monday, EndDate: monday}
	f.mustCreate(&theirs)
	wantError(t, call(f.h.DeleteMyBlock, f.own, userID, "", nil, "blockId", id(theirs.ID)), http.StatusNotFound, core_errors.ErrAgendaBlockNotFound)
}

// ─── Tenant isolation ────────────────────────────────────────────────────────

func TestAgenda_TenantIsolation(t *testing.T) {
	f := newFixture(t)
	ownDoctor := f.doctor(f.own, "Dra. Propia", nil)
	foreign := f.doctor(f.other, "Dr. Ajeno", nil)
	f.mustCreate(&agenda_models.DoctorSchedule{TenantID: f.other.ID, DoctorID: foreign.ID, Weekday: 1, StartTime: "08:00", EndTime: "12:00"})
	foreignBlock := agenda_models.ScheduleBlock{TenantID: f.other.ID, StartDate: monday, EndDate: monday, Reason: "Feriado ajeno"}
	f.mustCreate(&foreignBlock)
	foreignType := agenda_models.AppointmentType{TenantID: f.other.ID, Name: "Ajeno", DurationMinutes: 30, Active: true}
	f.mustCreate(&foreignType)

	wantError(t, call(f.h.GetDoctorSchedule, f.own, 1, "", nil, "id", id(foreign.ID)), http.StatusNotFound, core_errors.ErrDoctorNotFound)
	wantError(t, call(f.h.ReplaceDoctorSchedule, f.own, 1, "", map[string]any{"slots": []any{}}, "id", id(foreign.ID)), http.StatusNotFound, core_errors.ErrDoctorNotFound)
	wantError(t, call(f.h.CreateDoctorBlock, f.own, 1, "", map[string]any{"start_date": monday, "end_date": monday}, "id", id(foreign.ID)), http.StatusNotFound, core_errors.ErrDoctorNotFound)
	var n int64
	f.db.Model(&agenda_models.DoctorSchedule{}).Where("doctor_id = ?", foreign.ID).Count(&n)
	if n != 1 {
		t.Fatalf("foreign schedule rows = %d, want 1 (untouched)", n)
	}

	// Clinic blocks: not listed, not editable, not deletable, and they don't block our doctors.
	resp := call(f.h.ListClinicBlocks, f.own, 1, "from=2026-01-01", nil)
	wantCode(t, resp, http.StatusOK)
	if blocks := resp.Data.([]agenda_models.ScheduleBlock); len(blocks) != 0 {
		t.Fatalf("listed foreign clinic blocks: %+v", blocks)
	}
	wantError(t, call(f.h.UpdateClinicBlock, f.own, 1, "", map[string]any{"start_date": monday, "end_date": monday}, "id", id(foreignBlock.ID)), http.StatusNotFound, core_errors.ErrAgendaBlockNotFound)
	wantError(t, call(f.h.DeleteClinicBlock, f.own, 1, "", nil, "id", id(foreignBlock.ID)), http.StatusNotFound, core_errors.ErrAgendaBlockNotFound)
	resp = call(f.h.GetAvailability, f.own, 1, "doctor_id="+id(ownDoctor.ID)+"&date="+monday+"&start_time=09:00&end_time=09:30", nil)
	wantCode(t, resp, http.StatusOK)
	if got := resp.Data.(agenda_dto.AvailabilityResponse).Status; got != agenda_services.StatusNoSchedule {
		t.Fatalf("status = %q, want no_schedule (foreign clinic block must not apply)", got)
	}
	wantError(t, call(f.h.GetAvailability, f.own, 1, "doctor_id="+id(foreign.ID)+"&date="+monday+"&start_time=09:00&end_time=09:30", nil), http.StatusNotFound, core_errors.ErrDoctorNotFound)

	// Agenda range drops foreign doctor ids.
	resp = call(f.h.GetAgendaRange, f.own, 1, "from="+monday+"&to="+monday+"&doctor_ids="+id(ownDoctor.ID)+","+id(foreign.ID), nil)
	wantCode(t, resp, http.StatusOK)
	if docs := resp.Data.(agenda_dto.AgendaRangeResponse).Doctors; len(docs) != 1 || docs[0].DoctorID != ownDoctor.ID {
		t.Fatalf("agenda doctors = %+v", docs)
	}

	// Appointment types: foreign ones are invisible, foreign doctors can't be attached.
	resp = call(f.h.ListAppointmentTypes, f.own, 1, "", nil)
	wantCode(t, resp, http.StatusOK)
	if types := resp.Data.([]agenda_models.AppointmentType); len(types) != 0 {
		t.Fatalf("listed foreign types: %+v", types)
	}
	wantError(t, call(f.h.UpdateAppointmentType, f.own, 1, "", map[string]any{"name": "X"}, "id", id(foreignType.ID)), http.StatusNotFound, core_errors.ErrAgendaTypeNotFound)
	wantError(t, call(f.h.DeleteAppointmentType, f.own, 1, "", nil, "id", id(foreignType.ID)), http.StatusNotFound, core_errors.ErrAgendaTypeNotFound)
	wantError(t, call(f.h.CreateAppointmentType, f.own, 1, "", map[string]any{"name": "Control", "duration_minutes": 30, "doctor_ids": []uint{foreign.ID}}),
		http.StatusBadRequest, core_errors.ErrDoctorNotFound)
}

// ─── Blocks ──────────────────────────────────────────────────────────────────

func TestBlocks_ValidateDatesAndTimes(t *testing.T) {
	f := newFixture(t)
	create := func(body map[string]any) envelope.Response { return call(f.h.CreateClinicBlock, f.own, 1, "", body) }

	wantError(t, create(map[string]any{"start_date": "2026-10-20", "end_date": "2026-10-19"}), http.StatusBadRequest, core_errors.ErrAgendaInvalidDateRange)
	wantError(t, create(map[string]any{"start_date": "20-10-2026", "end_date": "2026-10-19"}), http.StatusBadRequest, core_errors.ErrAgendaInvalidDateRange)
	wantError(t, create(map[string]any{"start_date": monday, "end_date": monday, "start_time": "10:00"}), http.StatusBadRequest, core_errors.ErrAgendaInvalidTime)
	wantError(t, create(map[string]any{"start_date": monday, "end_date": monday, "start_time": "10:00", "end_time": "10:07"}), http.StatusBadRequest, core_errors.ErrAgendaInvalidTime)
	resp := create(map[string]any{"start_date": monday, "end_date": "2026-10-21", "start_time": "13:00", "end_time": "15:00", "reason": "Congreso"})
	wantCode(t, resp, http.StatusOK)
	block := resp.Data.(agenda_dto.BlockResponse).Block
	if block.DoctorID != nil || block.TenantID != f.own.ID || block.Reason != "Congreso" {
		t.Fatalf("clinic block = %+v", block)
	}

	// Update keeps the owner, turns it into a full-day block.
	resp = call(f.h.UpdateClinicBlock, f.own, 1, "", map[string]any{"start_date": monday, "end_date": monday}, "id", id(block.ID))
	wantCode(t, resp, http.StatusOK)
	if b := resp.Data.(agenda_dto.BlockResponse).Block; !b.FullDay() || b.EndDate != monday {
		t.Fatalf("updated block = %+v", b)
	}
	wantCode(t, call(f.h.DeleteClinicBlock, f.own, 1, "", nil, "id", id(block.ID)), http.StatusOK)
}

func TestCreateBlock_ListsAffectedAppointments(t *testing.T) {
	f := newFixture(t)
	a := f.doctor(f.own, "Dra. A", nil)
	b := f.doctor(f.own, "Dr. B", nil)
	morning := f.appointment(f.own, &a.ID, monday, "09:00", "09:30", "scheduled")
	afternoon := f.appointment(f.own, &a.ID, monday, "15:00", "15:30", "confirmed")
	f.appointment(f.own, &a.ID, monday, "10:00", "10:30", "cancelled")
	f.appointment(f.own, &a.ID, "2026-10-20", "09:00", "09:30", "scheduled") // outside the dates
	other := f.appointment(f.own, &b.ID, monday, "09:00", "09:30", "scheduled")
	f.appointment(f.other, nil, monday, "09:00", "09:30", "scheduled") // another clinic

	ids := func(resp envelope.Response) []uint {
		t.Helper()
		wantCode(t, resp, http.StatusOK)
		var out []uint
		for _, ap := range resp.Data.(agenda_dto.BlockResponse).AffectedAppointments {
			out = append(out, ap.ID)
		}
		return out
	}

	// Doctor A, morning only: just A's morning appointment; still created.
	got := ids(call(f.h.CreateDoctorBlock, f.own, 1, "", map[string]any{"start_date": monday, "end_date": monday, "start_time": "08:00", "end_time": "12:00"}, "id", id(a.ID)))
	if fmt.Sprint(got) != fmt.Sprint([]uint{morning.ID}) {
		t.Fatalf("doctor morning block affected %v, want [%d]", got, morning.ID)
	}
	// Whole clinic, full day: every doctor's active appointments that day.
	resp := call(f.h.CreateClinicBlock, f.own, 1, "", map[string]any{"start_date": monday, "end_date": monday, "reason": "Feriado"})
	got = ids(resp)
	want := map[uint]bool{morning.ID: true, afternoon.ID: true, other.ID: true}
	if len(got) != len(want) {
		t.Fatalf("clinic block affected %v, want %v", got, want)
	}
	for _, g := range got {
		if !want[g] {
			t.Fatalf("clinic block affected %v, want %v", got, want)
		}
	}
	if name := resp.Data.(agenda_dto.BlockResponse).AffectedAppointments[0].PatientName; name != "Ana Pérez" {
		t.Fatalf("patient name = %q", name)
	}
	var stored int64
	f.db.Model(&agenda_models.ScheduleBlock{}).Where("tenant_id = ?", f.own.ID).Count(&stored)
	if stored != 2 {
		t.Fatalf("blocks stored = %d, want 2 (blocks over appointments are allowed)", stored)
	}
}

// ─── Availability and agenda range ───────────────────────────────────────────

func TestAvailability_Statuses(t *testing.T) {
	f := newFixture(t)
	d := f.doctor(f.own, "Dra. Ana", nil)
	free := f.doctor(f.own, "Dr. Sin horario", nil)
	check := func(doctorID uint, date, start, end string) agenda_dto.AvailabilityResponse {
		t.Helper()
		resp := call(f.h.GetAvailability, f.own, 1, fmt.Sprintf("doctor_id=%d&date=%s&start_time=%s&end_time=%s", doctorID, date, start, end), nil)
		wantCode(t, resp, http.StatusOK)
		return resp.Data.(agenda_dto.AvailabilityResponse)
	}

	if got := check(d.ID, monday, "09:00", "09:30").Status; got != agenda_services.StatusNoSchedule {
		t.Fatalf("no schedule yet: %q", got)
	}
	wantCode(t, call(f.h.ReplaceDoctorSchedule, f.own, 1, "", map[string]any{"slots": []any{
		slot(1, "08:00", "12:00"), slot(1, "12:00", "14:00"), slot(2, "08:00", "12:00"),
	}}, "id", id(d.ID)), http.StatusOK)

	cases := []struct{ date, start, end, want string }{
		{monday, "09:00", "09:30", agenda_services.StatusInside},
		{monday, "11:30", "12:30", agenda_services.StatusInside}, // across touching ranges
		{monday, "13:45", "14:15", agenda_services.StatusOutside},
		{monday, "07:00", "07:30", agenda_services.StatusOutside},
		{"2026-10-21", "09:00", "09:30", agenda_services.StatusOutside}, // Wednesday: no ranges
		{monday, "09:07", "09:22", agenda_services.StatusInside},        // any minute is accepted here
	}
	for _, tc := range cases {
		if got := check(d.ID, tc.date, tc.start, tc.end); got.Status != tc.want {
			t.Errorf("%s %s-%s = %q, want %q", tc.date, tc.start, tc.end, got.Status, tc.want)
		}
	}
	if r := check(d.ID, monday, "09:00", "09:30").Ranges; len(r) != 1 || r[0].StartTime != "08:00" || r[0].EndTime != "14:00" {
		t.Fatalf("merged ranges = %+v", r)
	}

	// A doctor block over part of the slot.
	f.mustCreate(&agenda_models.ScheduleBlock{TenantID: f.own.ID, DoctorID: &d.ID, StartDate: monday, EndDate: monday, StartTime: "10:00", EndTime: "11:00"})
	if got := check(d.ID, monday, "10:30", "11:30"); got.Status != agenda_services.StatusBlocked || len(got.Blocks) != 1 {
		t.Fatalf("doctor block: %+v", got)
	}
	if got := check(d.ID, monday, "11:00", "11:30").Status; got != agenda_services.StatusInside {
		t.Fatalf("right after the block = %q, want inside", got)
	}

	// A clinic-wide full-day block blocks every doctor, even one without schedule.
	f.mustCreate(&agenda_models.ScheduleBlock{TenantID: f.own.ID, StartDate: "2026-10-20", EndDate: "2026-10-20", Reason: "Feriado"})
	if got := check(d.ID, "2026-10-20", "09:00", "09:30").Status; got != agenda_services.StatusBlocked {
		t.Fatalf("clinic block on scheduled doctor = %q", got)
	}
	if got := check(free.ID, "2026-10-20", "09:00", "09:30").Status; got != agenda_services.StatusBlocked {
		t.Fatalf("clinic block on doctor without schedule = %q", got)
	}
	if got := check(free.ID, monday, "09:00", "09:30").Status; got != agenda_services.StatusNoSchedule {
		t.Fatalf("doctor without schedule = %q", got)
	}

	// Bad input.
	wantError(t, call(f.h.GetAvailability, f.own, 1, "doctor_id="+id(d.ID)+"&date="+monday+"&start_time=10:00&end_time=09:00", nil), http.StatusBadRequest, core_errors.ErrAgendaInvalidTime)
	wantError(t, call(f.h.GetAvailability, f.own, 1, "doctor_id="+id(d.ID)+"&date=x&start_time=09:00&end_time=10:00", nil), http.StatusBadRequest, core_errors.ErrAgendaInvalidDateRange)
}

func TestAgendaRange_RangesAndBlocksPerDay(t *testing.T) {
	f := newFixture(t)
	d := f.doctor(f.own, "Dra. Ana", nil)
	inactive := f.doctor(f.own, "Dr. Inactivo", nil)
	f.db.Model(&inactive).Update("active", false)
	f.mustCreate(&agenda_models.DoctorSchedule{TenantID: f.own.ID, DoctorID: d.ID, Weekday: 1, StartTime: "08:00", EndTime: "12:00"})
	f.mustCreate(&agenda_models.DoctorSchedule{TenantID: f.own.ID, DoctorID: inactive.ID, Weekday: 1, StartTime: "08:00", EndTime: "12:00"})
	f.mustCreate(&agenda_models.ScheduleBlock{TenantID: f.own.ID, DoctorID: &d.ID, StartDate: monday, EndDate: monday, StartTime: "09:00", EndTime: "10:00"})
	f.mustCreate(&agenda_models.ScheduleBlock{TenantID: f.own.ID, StartDate: "2026-10-20", EndDate: "2026-10-20", Reason: "Feriado"})

	resp := call(f.h.GetAgendaRange, f.own, 1, "from="+monday+"&to=2026-10-20", nil)
	wantCode(t, resp, http.StatusOK)
	data := resp.Data.(agenda_dto.AgendaRangeResponse)
	if len(data.Doctors) != 1 || data.Doctors[0].DoctorID != d.ID || !data.Doctors[0].HasSchedule {
		t.Fatalf("default doctors = %+v (inactive must be left out)", data.Doctors)
	}
	days := data.Doctors[0].Days
	if len(days) != 2 || days[0].Date != monday || len(days[0].Ranges) != 1 || len(days[0].Blocks) != 1 {
		t.Fatalf("monday = %+v", days)
	}
	if len(days[1].Ranges) != 0 || len(days[1].Blocks) != 1 || days[1].Blocks[0].DoctorID != nil {
		t.Fatalf("tuesday = %+v (clinic block expected)", days[1])
	}
	if len(data.Clinic) != 2 || len(data.Clinic[0].Blocks) != 0 || len(data.Clinic[1].Blocks) != 1 {
		t.Fatalf("clinic days = %+v", data.Clinic)
	}

	// Explicit ids include inactive doctors.
	resp = call(f.h.GetAgendaRange, f.own, 1, "from="+monday+"&to="+monday+"&doctor_ids="+id(inactive.ID), nil)
	wantCode(t, resp, http.StatusOK)
	if docs := resp.Data.(agenda_dto.AgendaRangeResponse).Doctors; len(docs) != 1 || docs[0].Active {
		t.Fatalf("explicit inactive doctor = %+v", docs)
	}
	wantError(t, call(f.h.GetAgendaRange, f.own, 1, "from=2026-01-01&to=2026-06-01", nil), http.StatusBadRequest, core_errors.ErrAgendaRangeTooLong)
}

// ─── Appointment types ───────────────────────────────────────────────────────

func TestAppointmentTypes_CRUDAndDoctorFilter(t *testing.T) {
	f := newFixture(t)
	a := f.doctor(f.own, "Dra. A", nil)
	b := f.doctor(f.own, "Dr. B", nil)

	for _, minutes := range []int{0, 7, 485, -5} {
		wantError(t, call(f.h.CreateAppointmentType, f.own, 1, "", map[string]any{"name": "X", "duration_minutes": minutes}),
			http.StatusBadRequest, core_errors.ErrAgendaInvalidDuration)
	}
	wantError(t, call(f.h.CreateAppointmentType, f.own, 1, "", map[string]any{"name": "X", "duration_minutes": 30, "color": "red"}),
		http.StatusBadRequest, core_errors.ErrDoctorInvalidColor)

	resp := call(f.h.CreateAppointmentType, f.own, 1, "", map[string]any{"name": "Procedimiento", "duration_minutes": 60, "color": "#16a34a", "doctor_ids": []uint{a.ID}})
	wantCode(t, resp, http.StatusOK)
	onlyA := resp.Data.(agenda_models.AppointmentType)
	if onlyA.Color != "#16A34A" || fmt.Sprint(onlyA.DoctorIDs) != fmt.Sprint([]uint{a.ID}) || !onlyA.Active {
		t.Fatalf("created type = %+v", onlyA)
	}
	resp = call(f.h.CreateAppointmentType, f.own, 1, "", map[string]any{"name": "Control", "duration_minutes": 20, "active": false})
	wantCode(t, resp, http.StatusOK)
	inactive := resp.Data.(agenda_models.AppointmentType)
	if inactive.Active || len(inactive.DoctorIDs) != 0 {
		t.Fatalf("inactive type = %+v", inactive)
	}

	list := func(query string) []string {
		t.Helper()
		resp := call(f.h.ListAppointmentTypes, f.own, 1, query, nil)
		wantCode(t, resp, http.StatusOK)
		var names []string
		for _, ty := range resp.Data.([]agenda_models.AppointmentType) {
			names = append(names, ty.Name)
		}
		return names
	}
	if got := fmt.Sprint(list("doctor_id=" + id(b.ID))); got != "[Control]" {
		t.Fatalf("types for B = %s", got)
	}
	if got := fmt.Sprint(list("active=true&doctor_id=" + id(a.ID))); got != "[Procedimiento]" {
		t.Fatalf("active types for A = %s", got)
	}

	// Update: open to every doctor, activate; then delete.
	resp = call(f.h.UpdateAppointmentType, f.own, 1, "", map[string]any{"doctor_ids": []uint{}, "duration_minutes": 45}, "id", id(onlyA.ID))
	wantCode(t, resp, http.StatusOK)
	if u := resp.Data.(agenda_models.AppointmentType); len(u.DoctorIDs) != 0 || u.DurationMinutes != 45 {
		t.Fatalf("updated type = %+v", u)
	}
	wantCode(t, call(f.h.UpdateAppointmentType, f.own, 1, "", map[string]any{"active": true}, "id", id(inactive.ID)), http.StatusOK)
	if got := fmt.Sprint(list("active=true&doctor_id=" + id(b.ID))); got != "[Control Procedimiento]" {
		t.Fatalf("after update = %s", got)
	}
	wantCode(t, call(f.h.DeleteAppointmentType, f.own, 1, "", nil, "id", id(inactive.ID)), http.StatusOK)
	if got := fmt.Sprint(list("")); got != "[Procedimiento]" {
		t.Fatalf("after delete = %s", got)
	}
}
