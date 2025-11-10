package eval

import (
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
	"github.com/SoarinFerret/SessionWarden/internal/state"
)

func PermitLogin(username string, state state.State, config config.Config, now time.Time) bool {
	if now.IsZero() {
		now = time.Now()
	}

	// get user config
	userConfig, exists := config.Users[username]
	if !exists {
		// No specific config for user, apply default policy
		return true
	}

	// check allowed hours
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		if !userConfig.WeekendHours.WithinRange(now) {
			return false
		}
	} else {
		if !userConfig.AllowedHours.WithinRange(now) {
			return false
		}
	}

	userState, err := state.GetUser(username)
	if err != nil {
		// User not found, apply default policy
		return true
	}

	if userState.Paused {
		return false
	}

	// Check daily limit
	todayUsage := userState.GetTimeUsed()
	dailyLimit := time.Duration(userConfig.DailyLimit).Seconds()
	if dailyLimit > 0 && float64(todayUsage) >= dailyLimit {
		return false
	}

	// Placeholder logic: allow all users to log in
	return true
}
