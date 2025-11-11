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

	userNotFound := false
	userState, err := state.GetUser(username)
	if err != nil {
		userNotFound = true // process this later
	}

	if !userNotFound && userState.AllowedHoursOverrideIsSet() {
		if !userState.AllowedHoursOverrideWithinRange(now) {
			return false
		}
	} else {
		// check allowed hours
		if now.Weekday() == time.Friday || now.Weekday() == time.Saturday {
			if !userConfig.WeekendHours.WithinRange(now) {
				return false
			}
		} else {
			if !userConfig.AllowedHours.WithinRange(now) {
				return false
			}
		}
	}

	if userNotFound {
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
