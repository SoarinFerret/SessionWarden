package session

import (
	"fmt"
	"time"
)

func (u *User) AddSession(start time.Time, sessionID string) {
	newSession := SessionRecord{
		SessionId: sessionID,
		StartTime: time.Time{},
		Segments:  []SegmentRecord{},
	}
	newSession.Start(start)

	u.Sessions = append(u.Sessions, newSession)
}

func (u *User) EndSession(end time.Time, sessionID string) error {
	for i := range u.Sessions {
		if u.Sessions[i].SessionId == sessionID && u.Sessions[i].IsActive() {
			u.Sessions[i].End(end)
			return nil
		}
	}
	return fmt.Errorf("no active session found with ID %s", sessionID)
}

func (u *User) EndAllSegments(reason string) {
	now := time.Now()
	for i := range u.Sessions {
		u.Sessions[i].EndSegment(now, reason)
	}
}

func (u *User) StartNewSegments() {
	now := time.Now()
	for i := range u.Sessions {
		if u.Sessions[i].IsActive() && u.Sessions[i].IsIdle() {
			u.Sessions[i].AddSegment(now)
		}
	}
}

func (u *User) GetActiveSession() *SessionRecord {
	for i := len(u.Sessions) - 1; i >= 0; i-- {
		if u.Sessions[i].IsActive() {
			return &u.Sessions[i]
		}
	}
	return nil
}

func (u *User) GetSessionByID(sessionID string) (*SessionRecord, error) {
	for i := range u.Sessions {
		if u.Sessions[i].SessionId == sessionID {
			return &u.Sessions[i], nil
		}
	}
	return nil, fmt.Errorf("session ID %s not found", sessionID)
}

func (u *User) IsSessionActive(sessionID string) bool {
	for _, session := range u.Sessions {
		if session.SessionId == sessionID {
			return session.IsActive()
		}
	}
	return false
}

func (u *User) Pause() {
	u.Paused = true
}

func (u *User) Resume() {
	u.Paused = false
}

func (u *User) GetTimeUsed() int64 {
	var totalDuration int64
	for _, session := range u.Sessions {
		totalDuration += session.Duration()
	}
	return totalDuration
}

func (u *User) AllowedHoursOverrideIsSet() bool {
	// check all overrides for AllowedHours set
	for _, override := range u.Overrides {
		if override.AllowedHours.IsEmpty() == false {
			return true
		}
	}
	return false
}

func (u *User) AllowedHoursOverrideWithinRange(now time.Time) bool {
	// evaluate all overrides for AllowedHours
	for _, override := range u.Overrides {
		if override.IsExpired(now) {
			continue
		}
		allow, err := override.EvalAllowedHours(now)
		if err != nil {
			continue
		}
		if allow == false {
			return false
		}
	}
	return true
}

func (u *User) AddOverride(o Override) {
	u.Overrides = append(u.Overrides, o)
}
