package company_models

import (
	"errors"
	"time"
	_ "time/tzdata" // the production image (alpine) ships no zoneinfo
)

// SubscriptionLocation is the calendar subscriptions are sold in: "expires on
// the 16th" means until the end of the 16th in Ecuador.
var SubscriptionLocation = mustLoadLocation("America/Guayaquil")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

var ErrInvalidExpiry = errors.New("invalid expiry: use YYYY-MM-DD or RFC3339")

// ParseExpiry reads a subscription expiry. A date ("2026-11-16") means the end
// of that day in SubscriptionLocation; an RFC3339 timestamp is taken as is.
func ParseExpiry(s string) (time.Time, error) {
	if day, err := time.ParseInLocation(time.DateOnly, s, SubscriptionLocation); err == nil {
		return day.Add(24*time.Hour - time.Second), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, ErrInvalidExpiry
}

// AddMonths moves t forward by months in SubscriptionLocation, keeping the time
// of day. A day that doesn't exist in the target month becomes its last day
// (31 Jan + 1 month = 28/29 Feb), unlike time.AddDate, which overflows into
// the next month (3 Mar).
func AddMonths(t time.Time, months int) time.Time {
	local := t.In(SubscriptionLocation)
	firstOfTarget := time.Date(local.Year(), local.Month()+time.Month(months), 1, 0, 0, 0, 0, SubscriptionLocation)
	lastDay := firstOfTarget.AddDate(0, 1, -1).Day()
	day := min(local.Day(), lastDay)
	return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), day,
		local.Hour(), local.Minute(), local.Second(), local.Nanosecond(), SubscriptionLocation)
}
