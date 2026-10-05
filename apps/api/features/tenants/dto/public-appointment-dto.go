package tenant_dto

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// PublicAppointment is what the public waiting-room TV receives for each of
// today's appointments. The endpoint needs no login, so it carries only what
// the screen renders: never the patient record, ids that link to it, the
// appointment's title or notes, or anything clinical.
type PublicAppointment struct {
	ID          uint   `json:"id"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
	PatientName string `json:"patient_name"`
}

// PatientDisplayName shortens a patient's name for a public screen: the first
// word of the first name plus the initial of the last name ("Juan P.").
func PatientDisplayName(firstName, lastName string) string {
	first := ""
	if fields := strings.Fields(firstName); len(fields) > 0 {
		first = fields[0]
	}
	initial := ""
	if last := strings.TrimSpace(lastName); last != "" {
		r, _ := utf8.DecodeRuneInString(last)
		initial = string(unicode.ToUpper(r)) + "."
	}
	return strings.TrimSpace(first + " " + initial)
}
