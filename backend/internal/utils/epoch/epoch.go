package epoch

import "time"

// EpochToTime converts epoch seconds to time.Time (UTC).
func EpochToTime(epoch int64) time.Time {
	return time.Unix(epoch, 0).UTC()
}

// NowEpoch returns the current time as epoch seconds.
func NowEpoch() int64 {
	return time.Now().Unix()
}
