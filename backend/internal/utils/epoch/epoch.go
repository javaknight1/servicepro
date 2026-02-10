package epoch

import "time"

// EpochToTime converts epoch seconds to time.Time (UTC).
func EpochToTime(epoch int64) time.Time {
	return time.Unix(epoch, 0).UTC()
}

// TimeToEpoch converts time.Time to epoch seconds.
func TimeToEpoch(t time.Time) int64 {
	return t.Unix()
}

// PtrEpochToTime converts *int64 epoch to *time.Time. Returns nil if input is nil.
func PtrEpochToTime(epoch *int64) *time.Time {
	if epoch == nil {
		return nil
	}
	t := EpochToTime(*epoch)
	return &t
}

// PtrTimeToEpoch converts *time.Time to *int64 epoch. Returns nil if input is nil.
func PtrTimeToEpoch(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	e := TimeToEpoch(*t)
	return &e
}

// NowEpoch returns the current time as epoch seconds.
func NowEpoch() int64 {
	return time.Now().Unix()
}

// EpochWeekday returns the weekday for an epoch timestamp.
func EpochWeekday(epoch int64) time.Weekday {
	return EpochToTime(epoch).Weekday()
}

// EpochHour returns the hour (0-23) for an epoch timestamp in UTC.
func EpochHour(epoch int64) int {
	return EpochToTime(epoch).Hour()
}

// SecondsFromMidnightToTime converts seconds from midnight to hour and minute.
func SecondsFromMidnightToTime(secs int) (hour, minute int) {
	hour = secs / 3600
	minute = (secs % 3600) / 60
	return
}
