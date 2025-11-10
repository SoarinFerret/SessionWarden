package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/session"
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
				Users:     make(map[string]session.User),
				HeartBeat: time.Now(),
				Version:   1,
			}
			if err := m.save(); err != nil {
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
func (m *Manager) save() error {
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
			var updatedSessions []session.SessionRecord
			for _, session := range user.Sessions {
				if session.IsActive() {
					session.End(lastHeartbeat)
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

// CleanupExpiredExceptions removes any expired ones.
func (m *Manager) CleanupExpiredOverrides() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for uname, user := range m.state.Users {
		var validEx []session.Override
		for _, ex := range user.Overrides {
			if ex.ExpiresAt.After(now) {
				validEx = append(validEx, ex)
			}
		}
		user.Overrides = validEx
		m.state.Users[uname] = user
	}
}
