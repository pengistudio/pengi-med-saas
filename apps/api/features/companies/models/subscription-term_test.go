package company_models

import (
	"testing"
	"time"
)

func TestParseExpiry_ADateMeansTheEndOfThatDayInEcuador(t *testing.T) {
	got, err := ParseExpiry("2026-11-16")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 11, 17, 4, 59, 59, 0, time.UTC) // 16/11 23:59:59 at UTC-5
	if !got.Equal(want) {
		t.Fatalf("ParseExpiry(2026-11-16) = %s, want %s", got.UTC(), want)
	}
}

func TestParseExpiry_AcceptsRFC3339(t *testing.T) {
	got, err := ParseExpiry("2026-11-16T15:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(time.Date(2026, 11, 16, 15, 4, 5, 0, time.UTC)) {
		t.Fatalf("got %s", got)
	}
}

func TestParseExpiry_RejectsGarbage(t *testing.T) {
	for _, s := range []string{"", "16/11/2026", "2026-13-01"} {
		if _, err := ParseExpiry(s); err == nil {
			t.Fatalf("ParseExpiry(%q) accepted", s)
		}
	}
}

func TestAddMonths_ClampsToTheLastDayOfTheMonth(t *testing.T) {
	at := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 23, 59, 59, 0, SubscriptionLocation) }
	cases := []struct {
		from   time.Time
		months int
		want   time.Time
	}{
		{at(2026, time.January, 31), 1, at(2026, time.February, 28)},
		{at(2028, time.January, 31), 1, at(2028, time.February, 29)},
		{at(2026, time.March, 31), 1, at(2026, time.April, 30)},
		{at(2026, time.August, 31), 6, at(2027, time.February, 28)},
		{at(2026, time.January, 15), 12, at(2027, time.January, 15)},
		{at(2026, time.November, 30), 3, at(2027, time.February, 28)},
	}
	for _, c := range cases {
		if got := AddMonths(c.from, c.months); !got.Equal(c.want) {
			t.Errorf("AddMonths(%s, %d) = %s, want %s", c.from.Format("2006-01-02"), c.months, got.Format("2006-01-02 15:04"), c.want.Format("2006-01-02 15:04"))
		}
	}
}
