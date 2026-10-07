package whatsapp_services

import (
	"testing"
	"time"

	clinical_models "pengi-med-saas/features/clinical/models"
)

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"0991234567", "593991234567", false},
		{"099 123 4567", "593991234567", false},
		{"(099) 123-4567", "593991234567", false},
		{"+593 99 123 4567", "593991234567", false},
		{"593991234567", "593991234567", false},
		{"991234567", "593991234567", false},
		{"00593991234567", "593991234567", false},
		{"+1 555 010 0199", "15550100199", false},
		{"02 2345678", "59322345678", false}, // landline, still a valid number
		{"", "", true},
		{"abc", "", true},
		{"12345", "", true},
		{"0991234", "", true},          // too short once 593 is added
		{"+59399123456789", "", true},  // Ecuador with too many digits
		{"1234567890123456", "", true}, // over 15 digits
	}
	for _, tc := range cases {
		got, err := NormalizePhone(tc.in)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q, err=%v", tc.in, got, err, tc.want, tc.wantErr)
		}
	}
}

func guayaquil(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Guayaquil")
	if err != nil {
		t.Skip("no tzdata:", err)
	}
	return loc
}

func TestAppointmentStart_UsesClinicTimeZone(t *testing.T) {
	loc := guayaquil(t)
	// The frontend sends local midnight as an ISO instant: 2026-10-06T05:00:00Z.
	appt := clinical_models.Appointment{Date: time.Date(2026, 10, 6, 5, 0, 0, 0, time.UTC), StartTime: "09:30"}
	start, err := AppointmentStart(appt, loc)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 6, 14, 30, 0, 0, time.UTC) // 09:30 at UTC-5
	if !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start.UTC(), want)
	}
	if _, err := AppointmentStart(clinical_models.Appointment{Date: appt.Date, StartTime: "9h"}, loc); err == nil {
		t.Fatal("expected an error for a malformed start time")
	}
}

func TestDueOffset(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	offsets := []int{24, 2}
	cases := []struct {
		name  string
		until time.Duration
		want  int
	}{
		{"past", -time.Minute, 0},
		{"now", 0, 0},
		{"outside every window", 25 * time.Hour, 0},
		{"inside 24h only", 23 * time.Hour, 24},
		{"exactly 24h", 24 * time.Hour, 24},
		{"inside both: smallest wins (late-created)", 90 * time.Minute, 2},
		{"exactly 2h", 2 * time.Hour, 2},
	}
	for _, tc := range cases {
		if got := DueOffset(now.Add(tc.until), now, offsets); got != tc.want {
			t.Errorf("%s: DueOffset = %d, want %d", tc.name, got, tc.want)
		}
	}
	if DueOffset(now.Add(time.Hour), now, nil) != 0 {
		t.Error("no offsets must never be due")
	}
}

func TestNormalizeOffsets(t *testing.T) {
	got, err := NormalizeOffsets([]int{2, 24, 2, 48})
	if err != nil || len(got) != 3 || got[0] != 48 || got[1] != 24 || got[2] != 2 {
		t.Fatalf("got %v, %v", got, err)
	}
	for _, bad := range [][]int{{0}, {169}, {-1}, {1, 2, 3, 4}} {
		if _, err := NormalizeOffsets(bad); err == nil {
			t.Errorf("NormalizeOffsets(%v) should fail", bad)
		}
	}
	if got, err := NormalizeOffsets(nil); err != nil || len(got) != 0 {
		t.Fatalf("empty list: %v, %v", got, err)
	}
}

func TestReminderMessage_ButtonsCarryMessageID(t *testing.T) {
	loc := guayaquil(t)
	msg := ReminderMessage("593991234567", 42, ReminderData{PatientName: "Ana Pérez", ClinicName: "Clínica Sur", Start: time.Date(2026, 10, 5, 10, 30, 0, 0, loc)})
	if msg.Buttons[0].Payload != "confirm:42" || msg.Buttons[1].Payload != "cancel:42" {
		t.Fatalf("buttons = %+v", msg.Buttons)
	}
	want := []string{"Ana Pérez", "Clínica Sur", "lunes 5 de octubre", "10:30"}
	for i, p := range want {
		if msg.BodyParams[i] != p {
			t.Fatalf("param %d = %q, want %q", i, msg.BodyParams[i], p)
		}
	}
}
