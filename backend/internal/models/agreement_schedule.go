package models

import (
	"fmt"
	"math/bits"
	"time"
)

// Visit-date logic for service agreements (T167).
//
// Visit dates are calendar dates, represented as midnight UTC and stepped
// with AddDate(0, 0, 1) in UTC, so time zones and DST never move, skip or
// repeat a date. Only the visit's start instant depends on the tenant's
// time zone (VisitStart).

// CivilDate returns the calendar date of t in loc, as midnight UTC.
func CivilDate(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

// ParseDate parses a YYYY-MM-DD calendar date (midnight UTC).
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation(DateLayout, s, time.UTC)
}

// normalizeDate drops any time of day (DATE columns arrive as midnight UTC,
// but be safe with values from other drivers).
func normalizeDate(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

// defaultSpread are the weekday masks for N visits a week.
var defaultSpread = map[int]int{
	1: maskOf(time.Monday),
	2: maskOf(time.Monday, time.Thursday),
	3: maskOf(time.Monday, time.Wednesday, time.Friday),
	4: maskOf(time.Monday, time.Tuesday, time.Thursday, time.Friday),
	5: maskOf(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday),
	6: maskOf(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday),
	7: 0x7f,
}

func maskOf(days ...time.Weekday) int {
	m := 0
	for _, d := range days {
		m |= 1 << uint(d)
	}
	return m
}

// WeekdayMaskOf turns weekday numbers (0 = Sunday … 6) into a mask.
func WeekdayMaskOf(days []int) int {
	m := 0
	for _, d := range days {
		if d >= 0 && d <= 6 {
			m |= 1 << uint(d)
		}
	}
	return m
}

// WeekdaysOf turns a mask back into sorted weekday numbers.
func WeekdaysOf(mask int) []int {
	out := []int{}
	for d := 0; d <= 6; d++ {
		if mask&(1<<uint(d)) != 0 {
			out = append(out, d)
		}
	}
	return out
}

// daysIn returns the number of days in the month of t.
func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// ValidateSchedule checks the frequency fields; the returned map names the
// bad JSON fields.
func (a *ServiceAgreement) ValidateSchedule() map[string]string {
	fields := map[string]string{}
	switch a.Frequency {
	case FrequencyWeekdays:
	case FrequencyWeekly:
		if a.WeekdayMask == 0 {
			fields["weekdays"] = "pick at least one weekday"
		}
	case FrequencyTimesPerWeek:
		if a.TimesPerWeek == nil || *a.TimesPerWeek < 1 || *a.TimesPerWeek > 7 {
			fields["times_per_week"] = "must be 1 to 7"
		} else if a.WeekdayMask != 0 && bits.OnesCount(uint(a.WeekdayMask)) != *a.TimesPerWeek {
			fields["weekdays"] = "give exactly times_per_week weekdays, or none for the default days"
		}
	case FrequencyMonthly:
		if a.DayOfMonth == nil || *a.DayOfMonth < 1 || *a.DayOfMonth > 31 {
			fields["day_of_month"] = "must be 1 to 31"
		}
	default:
		fields["frequency"] = "must be weekdays, weekly, times_per_week or monthly"
	}
	if a.EndDate != nil && normalizeDate(*a.EndDate).Before(normalizeDate(a.StartDate)) {
		fields["end_date"] = "must be on or after start_date"
	}
	if a.VisitStartMinutes < 0 || a.VisitStartMinutes >= 24*60 {
		fields["visit_start_time"] = "must be HH:MM"
	}
	if a.VisitDurationMinutes < 1 || a.VisitDurationMinutes > 24*60 {
		fields["visit_duration_minutes"] = "must be 1 to 1440"
	}
	return fields
}

// effectiveMask is the weekday mask of a weekly-style frequency.
func (a *ServiceAgreement) effectiveMask() int {
	switch a.Frequency {
	case FrequencyWeekdays:
		return defaultSpread[5]
	case FrequencyWeekly:
		return a.WeekdayMask
	case FrequencyTimesPerWeek:
		if a.WeekdayMask != 0 {
			return a.WeekdayMask
		}
		if a.TimesPerWeek != nil {
			return defaultSpread[*a.TimesPerWeek]
		}
	}
	return 0
}

// HasVisitOn reports whether the schedule has a visit on calendar date day
// (ignoring start/end dates).
func (a *ServiceAgreement) HasVisitOn(day time.Time) bool {
	day = normalizeDate(day)
	if a.Frequency == FrequencyMonthly {
		if a.DayOfMonth == nil {
			return false
		}
		want := *a.DayOfMonth
		if last := daysIn(day.Year(), day.Month()); want > last {
			want = last
		}
		return day.Day() == want
	}
	return a.effectiveMask()&(1<<uint(day.Weekday())) != 0
}

// VisitDates returns the visit dates from from to to (inclusive calendar
// dates), within the agreement's start and end dates, in order.
func (a *ServiceAgreement) VisitDates(from, to time.Time) []time.Time {
	from, to = normalizeDate(from), normalizeDate(to)
	if start := normalizeDate(a.StartDate); from.Before(start) {
		from = start
	}
	if a.EndDate != nil {
		if end := normalizeDate(*a.EndDate); to.After(end) {
			to = end
		}
	}
	var out []time.Time
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		if a.HasVisitOn(d) {
			out = append(out, d)
		}
	}
	return out
}

// VisitStart is the instant a visit on calendar date day starts, at the
// agreement's local start time in loc. A start time that doesn't exist that
// day (inside a spring-forward gap, e.g. 02:30) moves forward by the gap
// (03:30), never earlier; an ambiguous one (fall-back) is whichever
// occurrence time.Date picks — still one instant on the right date.
func (a *ServiceAgreement) VisitStart(day time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	day = normalizeDate(day)
	t := time.Date(day.Year(), day.Month(), day.Day(), a.VisitStartMinutes/60, a.VisitStartMinutes%60, 0, 0, loc)
	// time.Date doesn't say which side of a gap it resolves to; if the wall
	// clock came out earlier than asked, push it forward by the difference.
	l := t.In(loc)
	if got := l.Hour()*60 + l.Minute(); l.Day() == day.Day() && got < a.VisitStartMinutes {
		t = t.Add(time.Duration(a.VisitStartMinutes-got) * time.Minute)
	}
	return t
}

// FormatClock formats minutes after midnight as HH:MM.
func FormatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// ParseClock parses HH:MM (24-hour) into minutes after midnight.
func ParseClock(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

// ToResponse converts an agreement to its API form.
func (a *ServiceAgreement) ToResponse() AgreementResponse {
	resp := AgreementResponse{
		ID: a.ID, CustomerID: a.CustomerID, SiteID: a.SiteID, Title: a.Title,
		Frequency: a.Frequency, Weekdays: WeekdaysOf(a.WeekdayMask), TimesPerWeek: a.TimesPerWeek, DayOfMonth: a.DayOfMonth,
		StartDate:            normalizeDate(a.StartDate).Format(DateLayout),
		VisitStartTime:       FormatClock(a.VisitStartMinutes),
		VisitDurationMinutes: a.VisitDurationMinutes,
		PricePerPeriod:       a.PricePerPeriod, BillingPeriod: a.BillingPeriod,
		AssignedTechnicianID: a.AssignedTechnicianID, JobType: a.JobType, Status: a.Status, Notes: a.Notes,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
	// Show the days visits actually fall on for the implicit frequencies.
	if a.Frequency == FrequencyWeekdays || (a.Frequency == FrequencyTimesPerWeek && a.WeekdayMask == 0) {
		resp.Weekdays = WeekdaysOf(a.effectiveMask())
	}
	if a.EndDate != nil {
		s := normalizeDate(*a.EndDate).Format(DateLayout)
		resp.EndDate = &s
	}
	return resp
}
