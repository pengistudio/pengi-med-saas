package agenda_services

import (
	"fmt"
	"sort"
	"time"

	agenda_dto "pengi-med-saas/features/agenda/dto"
)

// DateLayout is the format of every agenda date ("YYYY-MM-DD").
const DateLayout = "2006-01-02"

// dayMinutes is the end of the day, "24:00".
const dayMinutes = 24 * 60

// ParseClock parses "HH:MM" into minutes since midnight. "24:00" is accepted
// (as an end time); with step > 0 the minutes must be a multiple of it.
func ParseClock(s string, step int) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	for _, ch := range s[:2] + s[3:] {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if m > 59 || h > 24 || (h == 24 && m != 0) {
		return 0, false
	}
	total := h*60 + m
	if step > 0 && total%step != 0 {
		return 0, false
	}
	return total, true
}

// ValidRange reports whether start-end is a valid range of one day, in steps
// of step minutes (0 = any minute).
func ValidRange(start, end string, step int) (int, int, bool) {
	s, ok1 := ParseClock(start, step)
	e, ok2 := ParseClock(end, step)
	if !ok1 || !ok2 || s >= e || e > dayMinutes {
		return 0, 0, false
	}
	return s, e, true
}

// FormatClock formats minutes since midnight as "HH:MM".
func FormatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// ParseDate parses an agenda date "YYYY-MM-DD".
func ParseDate(s string) (time.Time, bool) {
	t, err := time.Parse(DateLayout, s)
	return t, err == nil
}

// span is a time range in minutes.
type span struct{ start, end int }

// mergeRanges sorts ranges and joins the ones that touch or overlap, so
// 08:00-12:00 + 12:00-14:00 reads as one working range.
func mergeRanges(ranges []span) []span {
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	var out []span
	for _, r := range ranges {
		if n := len(out); n > 0 && r.start <= out[n-1].end {
			if r.end > out[n-1].end {
				out[n-1].end = r.end
			}
			continue
		}
		out = append(out, r)
	}
	return out
}

func toTimeRanges(spans []span) []agenda_dto.TimeRange {
	out := make([]agenda_dto.TimeRange, 0, len(spans))
	for _, s := range spans {
		out = append(out, agenda_dto.TimeRange{StartTime: FormatClock(s.start), EndTime: FormatClock(s.end)})
	}
	return out
}
