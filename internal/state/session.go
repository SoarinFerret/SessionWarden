package state

import (
	"fmt"
	"time"
)

func (u *User) AddSession(start time.Time, sessionID string) {
	session := SessionRecord{
		StartTime: start,
		EndTime:   time.Time{},
		SessionId: sessionID,
		Segments: []SegmentRecord{
			{
				StartTime: start,
				EndTime:   time.Time{},
				Reason:    "",
			},
		},
	}

	u.Sessions = append(u.Sessions, session)
}

func (u *User) EndSession(sessionID string) error {
	for i, session := range u.Sessions {
		if session.SessionId == sessionID && session.EndTime.IsZero() {
			u.Sessions[i].EndSessionNow()
			return nil
		}
	}
	return fmt.Errorf("session ID %s not found or already ended", sessionID)
}

func (s *SessionRecord) Duration() int64 {
	// if stop is 0, session is ongoing, use current time
	if s.EndTime.IsZero() {
		return time.Now().Unix() - s.StartTime.Unix()
	}

	return s.EndTime.Unix() - s.StartTime.Unix()
}

func (u *User) TotalSessionDuration() int64 {
	var total int64

	// Only sum sessions since 12a
	for _, session := range u.Sessions {
		if session.StartTime.After(time.Now().Truncate(24 * time.Hour)) {
			total += session.Duration()
		}
	}

	return total
}

func (s *SessionRecord) EndSession(stop time.Time) {
	s.EndTime = stop
}

func (s *SessionRecord) EndSessionNow() {
	s.EndTime = time.Now()
}
