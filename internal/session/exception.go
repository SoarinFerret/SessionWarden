package session

import "time"

func NewException(reason string, extraHours int, expiresAt time.Time) Exception {
	if expiresAt.IsZero() {
		// expire at eod today
		expiresAt = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	}
	return Exception{
		Reason:    reason,
		ExtraTime: extraHours,
		ExpiresAt: expiresAt,
	}
}
