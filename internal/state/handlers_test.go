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
	initialSegmentCount := len(s.Segments)

	m.HandleSleep()
	u, _ = m.state.GetUser(user)
	for _, sess := range u.Sessions {
		for _, seg := range sess.Segments {
			if seg.EndTime.IsZero() {
				t.Errorf("segment should be ended after sleep")
			}
		}
	}

	// Wake doesn't create segments - just logs
	m.HandleWake()
	u, _ = m.state.GetUser(user)
	s, _ = u.GetSessionByID(sessionID)
	if !s.IsIdle() {
		t.Errorf("session should still be idle after wake (no automatic segment creation)")
	}

	// User unlocks screen - this should create a new segment
	m.HandleUnlock(user, sessionID)
	u, _ = m.state.GetUser(user)
	s, _ = u.GetSessionByID(sessionID)
	if len(s.Segments) != initialSegmentCount+1 {
		t.Errorf("expected new segment after unlock, got %d segments, expected %d", len(s.Segments), initialSegmentCount+1)
	}
	if s.IsIdle() {
		t.Errorf("session should not be idle after unlock")
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

func TestHandleLoginDuplicateSession(t *testing.T) {
	m := tempManager(t)
	user := "dave"
	sessionID := "sess4"

	// Initial login
	m.HandleLogin(user, sessionID)
	u, err := m.state.GetUser(user)
	if err != nil {
		t.Fatalf("user not found after login: %v", err)
	}
	if len(u.Sessions) != 1 {
		t.Errorf("expected 1 session after initial login, got %d", len(u.Sessions))
	}

	// System wakes from sleep and systemd-logind re-emits SessionNew
	// This should NOT create a duplicate session
	m.HandleLogin(user, sessionID)
	u, _ = m.state.GetUser(user)
	if len(u.Sessions) != 1 {
		t.Errorf("expected 1 session after duplicate login signal, got %d", len(u.Sessions))
	}
	if u.Sessions[0].SessionId != sessionID {
		t.Errorf("session ID mismatch")
	}
}

func TestSleepWakeWithSessionNewReemission(t *testing.T) {
	m := tempManager(t)
	user := "eve"
	sessionID := "sess5"

	// User logs in
	m.HandleLogin(user, sessionID)
	u, _ := m.state.GetUser(user)
	if len(u.Sessions) != 1 {
		t.Fatalf("expected 1 session after login, got %d", len(u.Sessions))
	}
	initialSegmentCount := len(u.Sessions[0].Segments)

	// System goes to sleep - ends all segments
	m.HandleSleep()
	u, _ = m.state.GetUser(user)
	s, _ := u.GetSessionByID(sessionID)
	if len(s.Segments) != initialSegmentCount {
		t.Errorf("sleep should not change segment count, got %d", len(s.Segments))
	}
	// Verify last segment is ended
	if s.Segments[len(s.Segments)-1].EndTime.IsZero() {
		t.Errorf("last segment should be ended after sleep")
	}

	// System wakes up - does NOT create segments automatically
	m.HandleWake()
	u, _ = m.state.GetUser(user)
	s, _ = u.GetSessionByID(sessionID)
	if len(s.Segments) != initialSegmentCount {
		t.Errorf("wake should not create segment yet, expected %d segments, got %d", initialSegmentCount, len(s.Segments))
	}

	// systemd-logind re-emits SessionNew after wake (or user interacts)
	// This SHOULD create a new segment since session is idle
	m.HandleLogin(user, sessionID)
	u, _ = m.state.GetUser(user)
	if len(u.Sessions) != 1 {
		t.Errorf("HandleLogin after wake should not create duplicate session, expected 1 session, got %d", len(u.Sessions))
	}
	s, _ = u.GetSessionByID(sessionID)
	if len(s.Segments) != initialSegmentCount+1 {
		t.Errorf("HandleLogin after wake should create new segment, expected %d segments, got %d", initialSegmentCount+1, len(s.Segments))
	}
	// Verify new segment is active
	if !s.Segments[len(s.Segments)-1].IsActive() {
		t.Errorf("new segment should be active after login")
	}

	// Another SessionNew signal should NOT create another segment (already active)
	m.HandleLogin(user, sessionID)
	u, _ = m.state.GetUser(user)
	s, _ = u.GetSessionByID(sessionID)
	if len(s.Segments) != initialSegmentCount+1 {
		t.Errorf("second HandleLogin should not create another segment, expected %d segments, got %d", initialSegmentCount+1, len(s.Segments))
	}
}
