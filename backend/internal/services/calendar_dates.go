package services

import "time"

// Calendar-date helpers (T127).
//
// Invoice issue and due dates are Postgres DATE values, which arrive in Go as
// midnight UTC. Converting them with .In(loc) moved them to the previous day
// in every US time zone, and Hours()/24 day counts were off by one across DST
// changes. These helpers compare whole calendar days instead.

// calendarDay returns the calendar date of t as midnight UTC.
//
// A value that is exactly midnight UTC is taken to be a DATE and kept as-is;
// any other instant (e.g. "now") is first converted to loc, so the tenant's
// local date is used.
func calendarDay(t time.Time, loc *time.Location) time.Time {
	u := t.UTC()
	if u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0 {
		return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
	}
	if loc == nil {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

// calendarDaysBetween returns the number of calendar days from a to b
// (negative when b is before a). Both are reduced to calendar days first.
func calendarDaysBetween(a, b time.Time, loc *time.Location) int {
	return int(calendarDay(b, loc).Sub(calendarDay(a, loc)).Hours() / 24)
}

// loadLocationOrUTC resolves an IANA time zone name, falling back to UTC.
func loadLocationOrUTC(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}
