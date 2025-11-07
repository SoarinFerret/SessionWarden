package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

// Manager handles reading and writing state.json safely.
type Manager struct {
	path  string
	mu    sync.Mutex
	state *State
}

// NewManager loads or initializes a new state manager.
func NewManager(path string) (*Manager, error) {
	m := &Manager{path: path}

	if err := m.load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.state = &State{
				Users:     make(map[string]User),
				HeartBeat: time.Now(),
				Version:   1,
			}
			if err := m.Save(); err != nil {
				return nil, err
			}
			return m, nil
		}
		return nil, err
	} else {
		m.startUpChecks()
	}

	return m, nil
}

// load reads the state file into memory.
func (m *Manager) load() error {
	var s State

	// read mtime of file to set heartbeat
	info, err := os.Stat(m.path)
	if err != nil {
		return err
	}
	s.HeartBeat = info.ModTime()

	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	m.state = &s
	return nil
}

func (m *Manager) Heartbeat() {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := time.Now()
	os.Chtimes(m.path, t, t)
	m.state.HeartBeat = t
}

// Save atomically writes the state file to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tmp := m.path + ".tmp"
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, m.path)
}

// StartUpChecks checks for power outages and cleans up sessions
func (m *Manager) startUpChecks() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get uptime
	uptime, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return
	}

	var upSeconds float64
	_, err = fmt.Sscanf(string(uptime), "%f", &upSeconds)
	if err != nil {
		return
	}

	lastHeartbeat := m.state.HeartBeat
	now := time.Now()
	if now.Sub(lastHeartbeat) > time.Duration(upSeconds)*time.Second {
		// system was down, clean up sessions
		for uname, user := range m.state.Users {
			var updatedSessions []SessionRecord
			for _, session := range user.Sessions {
				if session.EndTime.IsZero() {
					session.EndTime = lastHeartbeat
				}
				updatedSessions = append(updatedSessions, session)
			}
			user.Sessions = updatedSessions
			m.state.Users[uname] = user
		}
	}

	// update heartbeat
	m.state.HeartBeat = now
}

// AddOrUpdateSession updates a user's session usage.
func (m *Manager) AddSession(user string, sessionID string, start time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u, ok := m.state.Users[user]; ok {
		u.AddSession(start, sessionID)
		m.state.Users[user] = u
	} else {
		newUser := User{
			Sessions:   []SessionRecord{},
			Exceptions: []Exception{},
			Paused:     false,
		}
		newUser.AddSession(start, sessionID)
		m.state.Users[user] = newUser
	}
}

func (m *Manager) GetUserBySession(sessionID string) (string, *User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for uname, user := range m.state.Users {
		for _, session := range user.Sessions {
			if session.SessionId == sessionID {
				return uname, &user, nil
			}
		}
	}

	return "", nil, fmt.Errorf("session ID %s not found", sessionID)
}

func (m *Manager) EndAllSegments() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for uname, user := range m.state.Users {
		var updatedSessions []SessionRecord
		for _, session := range user.Sessions {
			if len(session.Segments) > 0 {
				lastSegment := &session.Segments[len(session.Segments)-1]
				if lastSegment.EndTime.IsZero() {
					lastSegment.EndTime = now
				}
			}
			updatedSessions = append(updatedSessions, session)
		}
		user.Sessions = updatedSessions
		m.state.Users[uname] = user
	}
}

func (m *Manager) StartNewSegments() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for uname, user := range m.state.Users {
		var updatedSessions []SessionRecord
		for _, session := range user.Sessions {
			if len(session.Segments) > 0 {
				lastSegment := &session.Segments[len(session.Segments)-1]
				if !lastSegment.EndTime.IsZero() {
					newSegment := SegmentRecord{
						StartTime: now,
						EndTime:   time.Time{},
						Reason:    "",
					}
					session.Segments = append(session.Segments, newSegment)
				}
			}
			updatedSessions = append(updatedSessions, session)
		}
		user.Sessions = updatedSessions
		m.state.Users[uname] = user
	}
}

// AddException adds a temporary exception for a user.
func (m *Manager) AddException(u string, ex Exception) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user, ok := m.state.Users[u]; ok {
		user.Exceptions = append(user.Exceptions, ex)
		m.state.Users[u] = user
	}
}

// CleanupExpiredExceptions removes any expired ones.
func (m *Manager) CleanupExpiredExceptions() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for uname, user := range m.state.Users {
		var validEx []Exception
		for _, ex := range user.Exceptions {
			if ex.ExpiresAt.After(now) {
				validEx = append(validEx, ex)
			}
		}
		user.Exceptions = validEx
		m.state.Users[uname] = user
	}
}

// GetState returns a snapshot of the current state.
func (m *Manager) GetState() *State {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *m.state
	return &copy
}

func (m *Manager) StopSegmentForSession(user string, sessionID string, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()

	if u, ok := m.state.Users[user]; ok {
		sessionFound := false
		for si, session := range u.Sessions {
			if session.SessionId == sessionID {
				if len(session.Segments) > 0 {
					lastSegment := &u.Sessions[si].Segments[len(session.Segments)-1]
					if lastSegment.EndTime.IsZero() {
						lastSegment.EndTime = now
						lastSegment.Reason = reason
					}
				}
				sessionFound = true
				break
			}
		}
		if !sessionFound {
			return fmt.Errorf("session ID %s not found for user %s", sessionID, user)
		}
		m.state.Users[user] = u
		return nil
	} else {
		return fmt.Errorf("user %s not found", user)
	}
}

func (m *Manager) StartSegment(user string, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()

	if u, ok := m.state.Users[user]; ok {
		sessionFound := false
		for si, session := range u.Sessions {
			if session.SessionId == sessionID {
				newSegment := SegmentRecord{
					StartTime: now,
					EndTime:   time.Time{},
					Reason:    "",
				}
				u.Sessions[si].Segments = append(u.Sessions[si].Segments, newSegment)
				sessionFound = true
				break
			}
		}
		if !sessionFound {
			return fmt.Errorf("session ID %s not found for user %s", sessionID, user)
		}
		m.state.Users[user] = u
		return nil
	} else {
		return fmt.Errorf("user %s not found", user)
	}
}
