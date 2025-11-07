package state

import "time"

type User struct {
	Sessions   []SessionRecord `json:"sessions"`
	Exceptions []Exception     `json:"exceptions"`
	Paused     bool            `json:"paused"`
}

// Exception represents a temporary rule override for a user.
type Exception struct {
	Reason    string    `json:"reason,omitempty"`
	AddedBy   string    `json:"added_by,omitempty"`
	ExtraTime int       `json:"extra_hours,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
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

// State is the top-level structure stored in the state.json file.
type State struct {
	Users     map[string]User `json:"users"`
	Version   int             `json:"version"`
	HeartBeat time.Time       `json:"-"` // not stored in JSON
}

func NewException(reason, addedBy string, extraHours int, expiresAt time.Time) Exception {
	if expiresAt.IsZero() {
		// expire at eod today
		expiresAt = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	}
	return Exception{
		Reason:    reason,
		AddedBy:   addedBy,
		ExtraTime: extraHours,
		ExpiresAt: expiresAt,
	}
}
