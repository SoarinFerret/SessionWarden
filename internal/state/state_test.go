package state

import (
	"fmt"
	"testing"
	"time"
)

func createTestManager(t *testing.T) *Manager {
	tmpDir := t.TempDir()
	tmpFilePath := fmt.Sprintf("%s/state_test.json", tmpDir)

	m, err := NewManager(tmpFilePath)
	if err != nil {
		t.Fatalf("Failed to create Manager: %v", err)
	}
	return m
}

func TestAddSessionAndGetUserBySession(t *testing.T) {
	fmt.Println("here")
	m := createTestManager(t)
	user := "alice"
	sessionID := "sess1"
	start := time.Now()

	m.AddSession(user, sessionID, start)
	foundUser, u, err := m.GetUserBySession(sessionID)
	if err != nil {
		t.Fatalf("GetUserBySession failed: %v", err)
	}
	if foundUser != user {
		t.Errorf("Expected user %q, got %q", user, foundUser)
	}
	if len(u.Sessions) != 1 || u.Sessions[0].SessionId != sessionID {
		t.Errorf("Session not added correctly")
	}
}

func TestManagerSave(t *testing.T) {
	m := createTestManager(t)
	user := "charlie"
	sessionID := "sess3"
	start := time.Now()

	m.AddSession(user, sessionID, start)
	err := m.Save()
	if err != nil {
		t.Fatalf("Manager Save failed: %v", err)
	}

	// Reload manager to verify save
	m2, err := NewManager(m.path)
	if err != nil {
		t.Fatalf("Failed to reload Manager: %v", err)
	}
	foundUser, u, err := m2.GetUserBySession(sessionID)
	if err != nil {
		t.Fatalf("GetUserBySession after reload failed: %v", err)
	}
	if foundUser != user {
		t.Errorf("Expected user %q after reload, got %q", user, foundUser)
	}
	if len(u.Sessions) != 1 || u.Sessions[0].SessionId != sessionID {
		t.Errorf("Session not persisted correctly after reload")
	}
}

func TestHandleLogin(t *testing.T) {
	m := createTestManager(t)
	user := "dave"
	sessionID := "sess4"

	m.HandleLogin(user, sessionID)
	foundUser, u, err := m.GetUserBySession(sessionID)
	if err != nil || foundUser != user {
		t.Fatalf("HandleLogin/GetUserBySession failed: %v", err)
	}
	if len(u.Sessions) != 1 || u.Sessions[0].SessionId != sessionID {
		t.Errorf("Session not added correctly on login")
	}
}

func TestHandleLogout(t *testing.T) {
	m := createTestManager(t)
	user := "eve"
	sessionID := "sess5"

	m.HandleLogin(user, sessionID)
	m.HandleLogout(sessionID)
	// After logout, session should have EndTime set
	_, u, err := m.GetUserBySession(sessionID)
	if err != nil {
		t.Fatalf("GetUserBySession after logout failed: %v", err)
	}
	if u.Sessions[0].EndTime.IsZero() {
		t.Errorf("Session EndTime not set after logout")
	}
}
