package session

import (
	"fmt"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
)

func NewExtraTimeOverride(reason string, extraMinutes int, expiresAt time.Time) Override {
	if expiresAt.IsZero() {
		// expire at eod today
		expiresAt = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	}
	return Override{
		Reason:    reason,
		ExtraTime: extraMinutes,
		ExpiresAt: expiresAt,
	}
}

func NewAllowedHoursOverride(reason string, allowedHours config.TimeRange, expiresAt time.Time) Override {
	if expiresAt.IsZero() {
		// expire at eod today
		expiresAt = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	}
	return Override{
		Reason:       reason,
		AllowedHours: allowedHours,
		ExpiresAt:    expiresAt,
	}
}

func (o Override) IsExpired(now time.Time) bool {
	if now.IsZero() {
		now = time.Now()
	}
	return now.After(o.ExpiresAt)
}

// Eval
func (o Override) EvalAllowedHours(now time.Time) (bool, error) {
	if now.IsZero() {
		now = time.Now()
	}

	if o.AllowedHours.IsEmpty() {
		return false, fmt.Errorf("allowed hours override is empty")
	} else {
		return o.AllowedHours.WithinRange(now), nil
	}
}
