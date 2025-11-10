package session

import "time"

func NewOverride(reason string, extraHours int, expiresAt time.Time) Override {
	if expiresAt.IsZero() {
		// expire at eod today
		expiresAt = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	}
	return Override{
		Reason:    reason,
		ExtraTime: extraHours,
		ExpiresAt: expiresAt,
	}
}
