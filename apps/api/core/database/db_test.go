package database

import "testing"

// Migrations run by date (YYYYMMDD), then by sequence: a 2027 migration comes
// after every 2026 one.
func TestParseFileName_OrdersByDateThenSequence(t *testing.T) {
	ordered := []string{"DB20260304_BACKOFFICE_USERS", "DB20260708_2", "DB20260708_10", "DB20261231_1", "DB20270101_1"}
	for i := 1; i < len(ordered); i++ {
		prevDate, prevID := ParseFileName(ordered[i-1])
		date, id := ParseFileName(ordered[i])
		if prevDate > date || (prevDate == date && prevID >= id) {
			t.Errorf("%s must run before %s", ordered[i-1], ordered[i])
		}
	}
}
