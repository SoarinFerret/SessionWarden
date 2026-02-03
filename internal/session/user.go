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

// GetSessionsForDay returns all sessions that started on the given day
func (u *User) GetSessionsForDay(day time.Time) []SessionRecord {
	var sessions []SessionRecord
	for _, session := range u.Sessions {
		// Check if session started on the same day (ignoring time)
		if isSameDay(session.StartTime, day) {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// GetTimeUsed returns the total time used for sessions that started today
func (u *User) GetTimeUsed() int64 {
	return u.GetTimeUsedForDay(time.Now())
}

// GetTimeUsedForDay returns the total time used for sessions that started on the given day
func (u *User) GetTimeUsedForDay(day time.Time) int64 {
	var totalDuration int64
	for _, session := range u.Sessions {
		// Only count sessions from the specified day
		if isSameDay(session.StartTime, day) {
			totalDuration += session.Duration()
		}
	}
	return totalDuration
}

// isSameDay checks if two times are on the same calendar day
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
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

// RemoveOldSessions removes all sessions that did not start today
func (u *User) RemoveOldSessions(now time.Time) {
	var currentSessions []SessionRecord
	for _, session := range u.Sessions {
		if isSameDay(session.StartTime, now) {
			currentSessions = append(currentSessions, session)
		}
	}
	u.Sessions = currentSessions
}
