package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T167: property tests for agreement visit dates over month ends, leap days
// and both DST transitions (America/New_York), proving no duplicate or
// missing visits.

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func intp(v int) *int { return &v }

// schedules covers every frequency, including month-end days of month.
func schedules() map[string]*ServiceAgreement {
	base := func(f AgreementFrequency) *ServiceAgreement {
		return &ServiceAgreement{Frequency: f, StartDate: date(2020, 1, 1), VisitStartMinutes: 18 * 60, VisitDurationMinutes: 60}
	}
	out := map[string]*ServiceAgreement{"weekdays": base(FrequencyWeekdays)}
	weekly := base(FrequencyWeekly)
	weekly.WeekdayMask = WeekdayMaskOf([]int{2, 5}) // Tue/Fri
	out["weekly Tue/Fri"] = weekly
	for n := 1; n <= 7; n++ {
		a := base(FrequencyTimesPerWeek)
		a.TimesPerWeek = intp(n)
		out[fmt.Sprintf("%dx weekly", n)] = a
	}
	mwf := base(FrequencyTimesPerWeek)
	mwf.TimesPerWeek, mwf.WeekdayMask = intp(3), WeekdayMaskOf([]int{1, 3, 5})
	out["3x weekly M/W/F"] = mwf
	for _, d := range []int{1, 15, 28, 29, 30, 31} {
		a := base(FrequencyMonthly)
		a.DayOfMonth = intp(d)
		out[fmt.Sprintf("monthly on %d", d)] = a
	}
	return out
}

// ranges span month ends, leap days (2024, 2028), a non-leap February
// (2027) and both 2026 DST changes in the US (Mar 8, Nov 1).
var ranges = []struct{ from, to time.Time }{
	{date(2024, 1, 25), date(2024, 3, 10)},   // leap February
	{date(2027, 1, 25), date(2027, 3, 10)},   // non-leap February
	{date(2028, 2, 20), date(2028, 3, 5)},    // leap day 2028
	{date(2026, 2, 20), date(2026, 3, 20)},   // spring forward
	{date(2026, 10, 20), date(2026, 11, 15)}, // fall back
	{date(2025, 12, 20), date(2026, 1, 10)},  // year end
	{date(2026, 1, 1), date(2026, 12, 31)},   // a whole year
}

// expected walks the range by building each date with time.Date (not
// AddDate) and asks the rule directly.
func expected(a *ServiceAgreement, from, to time.Time) []time.Time {
	var out []time.Time
	for i := 0; ; i++ {
		d := time.Date(from.Year(), from.Month(), from.Day()+i, 0, 0, 0, 0, time.UTC)
		if d.After(to) {
			return out
		}
		if a.HasVisitOn(d) {
			out = append(out, d)
		}
	}
}

func TestVisitDates_NoDuplicatesNoGaps(t *testing.T) {
	for name, a := range schedules() {
		for _, r := range ranges {
			got := a.VisitDates(r.from, r.to)
			assert.Equal(t, expected(a, r.from, r.to), got, "%s %s..%s", name, r.from.Format(DateLayout), r.to.Format(DateLayout))
			for i := 1; i < len(got); i++ {
				require.True(t, got[i].After(got[i-1]), "%s: dates strictly increasing", name)
			}
		}
	}
}

// The generator runs every hour with a sliding 14-day window. However the
// windows overlap, their union is exactly the visits of the whole span.
func TestVisitDates_SlidingWindowsCoverEveryVisitOnce(t *testing.T) {
	for name, a := range schedules() {
		for _, r := range ranges {
			seen := map[time.Time]int{}
			for d := r.from; !d.After(r.to); d = d.AddDate(0, 0, 1) {
				end := d.AddDate(0, 0, 13)
				if end.After(r.to) {
					end = r.to
				}
				for _, v := range a.VisitDates(d, end) {
					seen[v]++
				}
			}
			want := expected(a, r.from, r.to)
			assert.Len(t, seen, len(want), "%s: every visit appears", name)
			for _, v := range want {
				assert.Positive(t, seen[v], "%s: visit %s missing", name, v.Format(DateLayout))
			}
		}
	}
}

func TestVisitDates_MonthlyIsOncePerMonthClampedToMonthEnd(t *testing.T) {
	a := &ServiceAgreement{Frequency: FrequencyMonthly, DayOfMonth: intp(31), StartDate: date(2020, 1, 1)}
	got := a.VisitDates(date(2027, 1, 1), date(2028, 12, 31))
	require.Len(t, got, 24, "one visit a month")
	for i, d := range got {
		y, m := 2027+i/12, time.Month(i%12+1)
		assert.Equal(t, date(y, m, daysIn(y, m)), d, "last day of %d-%02d", y, m)
	}
	assert.Contains(t, got, date(2028, 2, 29), "leap day")
	assert.Contains(t, got, date(2027, 2, 28), "non-leap February")

	a.DayOfMonth = intp(29)
	feb := a.VisitDates(date(2027, 2, 1), date(2027, 3, 31))
	assert.Equal(t, []time.Time{date(2027, 2, 28), date(2027, 3, 29)}, feb)
}

func TestVisitDates_WeekdaysAreMondayToFriday(t *testing.T) {
	a := schedules()["weekdays"]
	got := a.VisitDates(date(2026, 1, 1), date(2026, 12, 31))
	assert.Len(t, got, 261, "2026 has 261 weekdays")
	for _, d := range got {
		assert.NotContains(t, []time.Weekday{time.Saturday, time.Sunday}, d.Weekday())
	}
}

func TestVisitDates_RespectsStartAndEnd(t *testing.T) {
	a := schedules()["weekdays"]
	a.StartDate = date(2026, 3, 4)
	end := date(2026, 3, 10)
	a.EndDate = &end
	assert.Equal(t, []time.Time{date(2026, 3, 4), date(2026, 3, 5), date(2026, 3, 6), date(2026, 3, 9), date(2026, 3, 10)},
		a.VisitDates(date(2026, 3, 1), date(2026, 3, 31)))
}

func TestVisitStart_KeepsLocalDateAndTimeAcrossDST(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	daily := schedules()["7x weekly"]

	for _, clock := range []int{0, 90, 150, 18 * 60, 23*60 + 59} { // 00:00, 01:30, 02:30, 18:00, 23:59
		daily.VisitStartMinutes = clock
		var prev time.Time
		for _, d := range daily.VisitDates(date(2026, 1, 1), date(2026, 12, 31)) {
			start := daily.VisitStart(d, ny)
			local := start.In(ny)
			assert.Equal(t, d, CivilDate(start, ny), "visit stays on its date (%s at %s)", d.Format(DateLayout), FormatClock(clock))
			if !(clock == 150 && d.Equal(date(2026, 3, 8))) { // 02:30 doesn't exist on Mar 8
				assert.Equal(t, clock, local.Hour()*60+local.Minute(), "local start time on %s", d.Format(DateLayout))
			}
			if !prev.IsZero() {
				gap := start.Sub(prev)
				assert.True(t, gap >= 23*time.Hour && gap <= 25*time.Hour, "one visit per day around %s (gap %s)", d.Format(DateLayout), gap)
			}
			prev = start
		}
	}

	daily.VisitStartMinutes = 150
	gapDay := daily.VisitStart(date(2026, 3, 8), ny).In(ny)
	assert.Equal(t, "03:30", gapDay.Format("15:04"), "02:30 in the spring-forward gap moves to 03:30")
}

// The generator's "today" is CivilDate(now, tenant zone). Ticking hourly
// through both DST days, today never skips or repeats a date out of order.
func TestCivilDate_HourlyTicksNeverSkipADay(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	for _, span := range []struct{ from, to time.Time }{
		{time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)},
	} {
		prev := CivilDate(span.from, ny)
		for now := span.from; now.Before(span.to); now = now.Add(time.Hour) {
			today := CivilDate(now, ny)
			diff := today.Sub(prev)
			assert.True(t, diff == 0 || diff == 24*time.Hour, "%s: today moved by %s", now, diff)
			prev = today
		}
	}
}

func TestValidateSchedule(t *testing.T) {
	a := &ServiceAgreement{Frequency: FrequencyWeekly, StartDate: date(2026, 1, 1), VisitDurationMinutes: 60}
	assert.Contains(t, a.ValidateSchedule(), "weekdays")
	a = &ServiceAgreement{Frequency: FrequencyTimesPerWeek, TimesPerWeek: intp(2), WeekdayMask: WeekdayMaskOf([]int{1}), StartDate: date(2026, 1, 1), VisitDurationMinutes: 60}
	assert.Contains(t, a.ValidateSchedule(), "weekdays")
	a = &ServiceAgreement{Frequency: FrequencyMonthly, StartDate: date(2026, 1, 1), VisitDurationMinutes: 60}
	assert.Contains(t, a.ValidateSchedule(), "day_of_month")
	end := date(2025, 1, 1)
	a = &ServiceAgreement{Frequency: FrequencyWeekdays, StartDate: date(2026, 1, 1), EndDate: &end, VisitDurationMinutes: 60}
	assert.Contains(t, a.ValidateSchedule(), "end_date")
	a = &ServiceAgreement{Frequency: FrequencyWeekdays, StartDate: date(2026, 1, 1), VisitDurationMinutes: 60}
	assert.Empty(t, a.ValidateSchedule())
}
