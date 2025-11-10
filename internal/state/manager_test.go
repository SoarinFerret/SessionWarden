package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/session"
)

func tempStateFile(t *testing.T) (string, func()) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	return path, func() { os.RemoveAll(dir) }
}

func TestNewManager_CreatesFileIfNotExist(t *testing.T) {
	path, cleanup := tempStateFile(t)
	defer cleanup()

	m, err := NewManager(path)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	if m.state == nil {
		t.Fatalf("state should not be nil")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}

func TestManager_SaveAndLoad(t *testing.T) {
	path, cleanup := tempStateFile(t)
	defer cleanup()

	m, _ := NewManager(path)
	m.state.Users["alice"] = session.User{}
	if err := m.save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	m2 := &Manager{path: path}
	if err := m2.load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if _, ok := m2.state.Users["alice"]; !ok {
		t.Errorf("user alice not found after load")
	}
}

func TestManager_Heartbeat(t *testing.T) {
	path, cleanup := tempStateFile(t)
	defer cleanup()
	m, _ := NewManager(path)
	oldTime := m.state.HeartBeat
	time.Sleep(10 * time.Millisecond)
	m.Heartbeat()
	if !m.state.HeartBeat.After(oldTime) {
		t.Errorf("Heartbeat did not update HeartBeat time")
	}
}

func TestManager_CleanupExpiredExceptions(t *testing.T) {
	path, cleanup := tempStateFile(t)
	defer cleanup()
	m, _ := NewManager(path)
	expired := session.Override{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	valid := session.Override{ExpiresAt: time.Now().Add(1 * time.Hour)}
	user := session.User{Overrides: []session.Override{expired, valid}}
	m.state.Users["bob"] = user

	m.CleanupExpiredOverrides()
	exs := m.state.Users["bob"].Overrides
	if len(exs) != 1 || !exs[0].ExpiresAt.Equal(valid.ExpiresAt) {
		t.Errorf("CleanupExpiredExceptions did not filter expired exceptions")
	}
}
