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
	// No user config and default.enabled = false, should permit
	if !PermitLogin("bobby", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login for user with no config when default.enabled = false")
	}
}

func TestPermitLogin_DefaultPolicyEnabled(t *testing.T) {
	st := state.State{Users: map[string]session.User{}}
	tomlData := `
[default]
daily_limit = "2h"
allowed_hours = "09:00-17:00"
weekend_hours = "10:00-14:00"
enabled = true
`
	cfg, _ := config.LoadConfigFromBytes([]byte(tomlData))

	// User not in config but default.enabled = true, should apply default restrictions
	// Monday at 10:00 - within allowed hours, should permit
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC)
	if !PermitLogin("bobby", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login during allowed hours with default config")
	}

	// Monday at 23:00 - outside allowed hours, should deny
	now = time.Date(2024, 6, 3, 23, 0, 0, 0, time.UTC)
	if PermitLogin("bobby", st, cfg, now) {
		t.Errorf("expected PermitLogin to deny login outside allowed hours with default config")
	}

	// Saturday at 11:00 - within weekend hours, should permit
	now = time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC)
	if !PermitLogin("bobby", st, cfg, now) {
		t.Errorf("expected PermitLogin to allow login during weekend hours with default config")
	}

	// Saturday at 15:00 - outside weekend hours, should deny
	now = time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC)
	if PermitLogin("bobby", st, cfg, now) {
		t.Errorf("expected PermitLogin to deny login outside weekend hours with default config")
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

func TestGetTimeRemaining_NoPolicy(t *testing.T) {
	st := state.State{Users: map[string]session.User{}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00

	// User with no config and default.enabled = false should have unlimited time
	remaining := GetTimeRemaining("bobby", st, cfg, now)
	if remaining != 9223372036854775807 { // math.MaxInt64
		t.Errorf("expected unlimited time for user with no config, got %d", remaining)
	}
}

func TestGetTimeRemaining_DefaultPolicyEnabled(t *testing.T) {
	st := state.State{Users: map[string]session.User{"bobby": {}}}
	tomlData := `
[default]
daily_limit = "2h"
allowed_hours = "09:00-17:00"
weekend_hours = "10:00-14:00"
enabled = true
`
	cfg, _ := config.LoadConfigFromBytes([]byte(tomlData))

	// User not explicitly in config but default.enabled = true
	// Should apply default 2h daily limit
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00, allowed hours 09:00-17:00
	remaining := GetTimeRemaining("bobby", st, cfg, now)
	expected := int64(2 * 60 * 60) // 2 hours in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining with default config, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_NoUsage(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00, allowed hours 09:00-17:00

	// Alice has 3h daily limit, no usage
	// Should be limited by daily limit (3 hours) since it's less than time until end of window (7 hours)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(3 * 60 * 60) // 3 hours in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_DailyLimitConstraint(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00

	// Alice has used 2.5 hours of her 3h limit
	alice := st.Users["alice"]
	start := time.Now().Add(-3 * time.Hour)
	alice.AddSession(start, "sess1")
	alice.EndSession(start.Add(2*time.Hour+30*time.Minute), "sess1")
	st.Users["alice"] = alice

	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(30 * 60) // 30 minutes in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_AllowedHoursConstraint(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 16, 30, 0, 0, time.UTC) // Monday at 16:30, allowed until 17:00

	// Alice has 3h daily limit, no usage
	// Should be limited by allowed hours (30 minutes until 17:00)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(30 * 60) // 30 minutes in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_PastAllowedHours(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 18, 0, 0, 0, time.UTC) // Monday at 18:00, past allowed hours

	// Past allowed hours should return 0
	remaining := GetTimeRemaining("alice", st, cfg, now)
	if remaining != 0 {
		t.Errorf("expected 0 seconds remaining past allowed hours, got %d", remaining)
	}
}

func TestGetTimeRemaining_WeekendHours(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC) // Saturday at 11:00, weekend hours 10:00-14:00

	// Should be limited by weekend hours (3 hours until 14:00)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(3 * 60 * 60) // 3 hours in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_ExtraTimeOverride(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00

	// Alice has used 2.5 hours of her 3h limit
	alice := st.Users["alice"]
	start := time.Now().Add(-3 * time.Hour)
	alice.AddSession(start, "sess1")
	alice.EndSession(start.Add(2*time.Hour+30*time.Minute), "sess1")

	// Add 60 minute ExtraTime override
	override := session.NewExtraTimeOverride("Extra work", 60, now.Add(24*time.Hour))
	alice.AddOverride(override)
	st.Users["alice"] = alice

	// Should now have 1h 30m remaining (30m original + 60m override)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(90 * 60) // 90 minutes in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining with override, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_AllowedHoursOverride(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 16, 30, 0, 0, time.UTC) // Monday at 16:30

	// Alice has override allowing until 20:00 instead of 17:00
	alice := st.Users["alice"]
	allowedHours, _ := config.ParseTimeRange("08:00-20:00")
	override := session.NewAllowedHoursOverride("Working late", allowedHours, now.Add(24*time.Hour))
	alice.AddOverride(override)
	st.Users["alice"] = alice

	// Should be limited by daily limit (3 hours) since it's less than time until end of window (3.5 hours)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(3 * 60 * 60) // 3 hours in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining with override, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_ExpiredOverride(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00

	// Alice has used 2.5 hours of her 3h limit
	alice := st.Users["alice"]
	start := time.Now().Add(-3 * time.Hour)
	alice.AddSession(start, "sess1")
	alice.EndSession(start.Add(2*time.Hour+30*time.Minute), "sess1")

	// Add expired ExtraTime override (should be ignored)
	override := session.NewExtraTimeOverride("Extra work", 60, now.Add(-1*time.Hour))
	alice.AddOverride(override)
	st.Users["alice"] = alice

	// Should have 30m remaining (override should be ignored)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(30 * 60) // 30 minutes in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining (expired override should be ignored), got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_MultipleOverrides(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00

	// Alice has used 2.5 hours of her 3h limit
	alice := st.Users["alice"]
	start := time.Now().Add(-3 * time.Hour)
	alice.AddSession(start, "sess1")
	alice.EndSession(start.Add(2*time.Hour+30*time.Minute), "sess1")

	// Add two ExtraTime overrides
	override1 := session.NewExtraTimeOverride("Extra work 1", 30, now.Add(24*time.Hour))
	override2 := session.NewExtraTimeOverride("Extra work 2", 30, now.Add(24*time.Hour))
	alice.AddOverride(override1)
	alice.AddOverride(override2)
	st.Users["alice"] = alice

	// Should have 90m remaining (30m original + 30m + 30m)
	remaining := GetTimeRemaining("alice", st, cfg, now)
	expected := int64(90 * 60) // 90 minutes in seconds
	if remaining != expected {
		t.Errorf("expected %d seconds remaining with multiple overrides, got %d", expected, remaining)
	}
}

func TestGetTimeRemaining_NegativeTimeRemaining(t *testing.T) {
	st := state.State{Users: map[string]session.User{"alice": {}}}
	cfg := exampleConfig()
	now := time.Date(2024, 6, 3, 10, 0, 0, 0, time.UTC) // Monday at 10:00

	// Alice has used 3.5 hours, exceeding her 3h limit
	alice := st.Users["alice"]
	start := time.Now().Add(-4 * time.Hour)
	alice.AddSession(start, "sess1")
	alice.EndSession(start.Add(3*time.Hour+30*time.Minute), "sess1")
	st.Users["alice"] = alice

	// Should return 0, not negative
	remaining := GetTimeRemaining("alice", st, cfg, now)
	if remaining != 0 {
		t.Errorf("expected 0 seconds remaining when limit exceeded, got %d", remaining)
	}
}

func TestCheckSendNotification_EmptyThresholds(t *testing.T) {
	notifyBefore := []config.Duration{}
	timeRemaining := int64(10 * 60) // 10 minutes

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if result {
		t.Errorf("expected false with empty notifyBefore list, got true")
	}
}

func TestCheckSendNotification_ExactlyAtThreshold(t *testing.T) {
	notifyBefore := []config.Duration{config.Duration(10 * time.Minute)}
	timeRemaining := int64(10 * 60) // 10 minutes

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if !result {
		t.Errorf("expected true when time remaining exactly at threshold, got false")
	}
}

func TestCheckSendNotification_WithinWindow(t *testing.T) {
	notifyBefore := []config.Duration{config.Duration(10 * time.Minute)}
	timeRemaining := int64(9*60 + 30) // 9 minutes 30 seconds

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if !result {
		t.Errorf("expected true when time remaining within window, got false")
	}
}

func TestCheckSendNotification_AtWindowEdge(t *testing.T) {
	notifyBefore := []config.Duration{config.Duration(10 * time.Minute)}
	timeRemaining := int64(9*60 + 1) // 9 minutes 1 second (just inside window)

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if !result {
		t.Errorf("expected true when time remaining at edge of window, got false")
	}
}

func TestCheckSendNotification_OutsideWindow(t *testing.T) {
	notifyBefore := []config.Duration{config.Duration(10 * time.Minute)}
	timeRemaining := int64(8*60 + 59) // 8 minutes 59 seconds (outside window)

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if result {
		t.Errorf("expected false when time remaining outside window, got true")
	}
}

func TestCheckSendNotification_AboveAllThresholds(t *testing.T) {
	notifyBefore := []config.Duration{
		config.Duration(10 * time.Minute),
		config.Duration(5 * time.Minute),
	}
	timeRemaining := int64(20 * 60) // 20 minutes

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if result {
		t.Errorf("expected false when time remaining above all thresholds, got true")
	}
}

func TestCheckSendNotification_MultipleThresholds(t *testing.T) {
	notifyBefore := []config.Duration{
		config.Duration(10 * time.Minute),
		config.Duration(5 * time.Minute),
	}

	// Test matching first threshold
	timeRemaining1 := int64(9*60 + 30) // 9 minutes 30 seconds
	result1 := CheckSendNotification(timeRemaining1, notifyBefore)
	if !result1 {
		t.Errorf("expected true when matching first threshold, got false")
	}

	// Test matching second threshold
	timeRemaining2 := int64(4*60 + 30) // 4 minutes 30 seconds
	result2 := CheckSendNotification(timeRemaining2, notifyBefore)
	if !result2 {
		t.Errorf("expected true when matching second threshold, got false")
	}

	// Test between thresholds (not in any window)
	timeRemaining3 := int64(7 * 60) // 7 minutes
	result3 := CheckSendNotification(timeRemaining3, notifyBefore)
	if result3 {
		t.Errorf("expected false when between thresholds, got true")
	}
}

func TestCheckSendNotification_ZeroTimeRemaining(t *testing.T) {
	notifyBefore := []config.Duration{config.Duration(5 * time.Minute)}
	timeRemaining := int64(0)

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if result {
		t.Errorf("expected false when time remaining is 0, got true")
	}
}

func TestCheckSendNotification_NegativeTimeRemaining(t *testing.T) {
	notifyBefore := []config.Duration{config.Duration(5 * time.Minute)}
	timeRemaining := int64(-60) // -1 minute

	result := CheckSendNotification(timeRemaining, notifyBefore)
	if result {
		t.Errorf("expected false when time remaining is negative, got true")
	}
}
