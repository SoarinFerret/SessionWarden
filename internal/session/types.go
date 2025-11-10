package session

import (
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
)

type User struct {
	Sessions  []SessionRecord `json:"sessions"`
	Overrides []Override      `json:"exceptions"`
	Paused    bool            `json:"paused"`
}

// Exception represents a temporary rule override for a user.
type Override struct {
	Reason       string           `json:"reason,omitempty"`
	ExtraTime    int              `json:"extra_hours,omitempty"`
	AllowedHours config.TimeRange `json:"allowed_hours,omitempty"`
	ExpiresAt    time.Time        `json:"expires_at"`
}

// SegmentRecord represents a time segment for tracking.
type SegmentRecord struct {
	StartTime time.Time `json:"start"`
	EndTime   time.Time `json:"stop"`
	Reason    string    `json:"reason,omitempty"`
}

// SessionRecord tracks a user's daily session usage.
type SessionRecord struct {
	StartTime time.Time       `json:"start"`
	EndTime   time.Time       `json:"end"`
	SessionId string          `json:"session_id,omitempty"`
	Segments  []SegmentRecord `json:"segments,omitempty"`
}
