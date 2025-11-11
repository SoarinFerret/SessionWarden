package eval

import (
	"testing"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
	"github.com/SoarinFerret/SessionWarden/internal/session"
	"github.com/SoarinFerret/SessionWarden/internal/state"
)

func exampleConfig() config.Config {
	tomlData := `
[default]
daily_limit = "2h"
allowed_hours = "09:00-17:00"
weekend_hours = "10:00-14:00"
notify_before = ["10m", "5m"]
lock_screen = true
enabled = false

[users]
[users.alice]
daily_limit = "3h"
enabled = true
[users.bob]
enabled = true
[users.steve]
enabled = false
`
	cfg, err := config.LoadConfigFromBytes([]byte(tomlData))
	if err != nil {
		panic(err)
	}
	return cfg
}

func TestPermitLogin_DefaultPolicy(t *testing.T) {
	st := state.State{Users: map[string]session.User{}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00
	// No user config, should permit
	if !PermitLogin("bobby", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login for user with no config")
	}
}

func TestPermitLogin_AllowedHours(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00
	if !PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login during allowed hours")
	}

	now = time.Date(2024, 6, 3, 23, 0, 0, 0, time.UTC) // Monday at 23:00
	if PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to deny login outside allowed hours")
	}

	now = time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC) // Saturday at 11:00
	if !PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login during weekend hours")
	}

	now = time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC) // Saturday at 15:00
	if PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to deny login outside weekend hours")
	}
}

func TestPermitLogin_PausedUser(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {Paused: true}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00
	if PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to deny login for paused user")
	}
}

func TestPermitLogin_DailyLimit(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	// Simulate user has used 2.5 hours today
	alice := st.Users["alice"]
	start := time.Now().Add(-3 * time.Hour)
	alice.AddSession(start, "sess1")
	alice.EndSession(start.Add(2*time.Hour+30*time.Minute), "sess1")
	st.Users["alice"] = alice
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00
	if !PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login when daily limit has not been reached")
	}

	// Simulate user has used 3.5 hours today
	// Add another session
	start = time.Now().Add(1 * time.Hour)
	alice.AddSession(start, "sess2")
	alice.EndSession(start.Add(1*time.Hour), "sess2")
	st.Users["alice"] = alice

	if PermitLogin("alice", st, cfg, now) {
		t.Errorf("expected PermitLogin to deny login when daily limit has been reached")
	}
}
