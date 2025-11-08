package state

import (
	"path/filepath"
	"testing"
	"time"
)

func tempManager(t *testing.T) *Manager {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	m, err := NewManager(path)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	return m
}

func TestHandleLoginAndLogout(t *testing.T) {
	m := tempManager(t)
	user := "alice"
	sessionID := "sess1"

	m.HandleLogin(user, sessionID)
	u, err := m.state.GetUser(user)
	if err != nil {
		t.Fatalf("user not found after login: %v", err)
	}
	if len(u.Sessions) != 1 || u.Sessions[0].SessionId != sessionID {
		t.Errorf("session not added on login")
	}

	m.HandleLogout(sessionID)
	u, _ = m.state.GetUser(user)
	if u.Sessions[0].IsActive() {
		t.Errorf("session should not be active after logout")
	}
}

func TestHandleSleepAndWake(t *testing.T) {
	m := tempManager(t)
	user := "bob"
	sessionID := "sess2"
	m.HandleLogin(user, sessionID)
	u, _ := m.state.GetUser(user)
	s, _ := u.GetSessionByID(sessionID)
	s.AddSegment(time.Now().Add(-10 * time.Minute))
	m.state.Users[user] = *u

	m.HandleSleep()
	u, _ = m.state.GetUser(user)
	for _, sess := range u.Sessions {
		for _, seg := range sess.Segments {
			if seg.EndTime.IsZero() {
				t.Errorf("segment should be ended after sleep")
			}
		}
	}

	m.HandleWake()
	u, _ = m.state.GetUser(user)
	foundActive := false
	for _, sess := range u.Sessions {
		if sess.IsActive() && !sess.IsIdle() {
			foundActive = true
		}
	}
	if !foundActive {
		t.Errorf("expected new segment to be started after wake")
	}
}

func TestHandleLockAndUnlock(t *testing.T) {
	m := tempManager(t)
	user := "carol"
	sessionID := "sess3"
	m.HandleLogin(user, sessionID)
	u, _ := m.state.GetUser(user)
	s, _ := u.GetSessionByID(sessionID)
	s.AddSegment(time.Now().Add(-5 * time.Minute))
	m.state.Users[user] = *u

	m.HandleLock(user, sessionID)
	u, _ = m.state.GetUser(user)
	s, _ = u.GetSessionByID(sessionID)
	if s.Segments[len(s.Segments)-1].EndTime.IsZero() {
		t.Errorf("segment should be ended after lock")
	}

	m.HandleUnlock(user, sessionID)
	u, _ = m.state.GetUser(user)
	s, _ = u.GetSessionByID(sessionID)
	if s.Segments[len(s.Segments)-1].EndTime.IsZero() {
		// Last segment should be active (not ended)
	} else {
		t.Errorf("segment should be active after unlock")
	}
}
