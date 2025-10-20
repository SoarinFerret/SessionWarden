package session

import "time"

type Session struct {
	start time.Time
	stop  time.Time
}

type User struct {
	Username string
	Sessions []Session
}

func (u *User) AddSession(start time.Time) {
	session := Session{
		start: start,
		stop:  time.Time{},
	}
	u.Sessions = append(u.Sessions, session)
}

func (s *Session) Duration() int64 {
	// if stop is 0, session is ongoing, use current time
	if s.stop.IsZero() {
		return time.Now().Unix() - s.start.Unix()
	}

	return s.stop.Unix() - s.start.Unix()
}

func (u *User) TotalSessionDuration() int64 {
	var total int64

	// Only sum sessions since 12a
	for _, session := range u.Sessions {
		if session.start.After(time.Now().Truncate(24 * time.Hour)) {
			total += session.Duration()
		}
	}

	return total
}

func (s *Session) EndSession(stop time.Time) {
	s.stop = stop
}

func (s *Session) EndSessionNow() {
	s.stop = time.Now()
}
