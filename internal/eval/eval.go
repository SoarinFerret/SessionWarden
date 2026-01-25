package eval

import (
	"math"
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
		// No specific config for user
		// If default is enabled, use default config; otherwise allow
		if config.Default.Enabled != nil && *config.Default.Enabled {
			userConfig = config.Default
		} else {
			return true
		}
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

	// Check daily limit (with ExtraTime overrides applied)
	todayUsage := userState.GetTimeUsed()
	dailyLimit := time.Duration(userConfig.DailyLimit).Seconds()

	// Apply ExtraTime from active overrides
	for _, override := range userState.Overrides {
		if !override.IsExpired(now) && override.ExtraTime > 0 {
			dailyLimit += float64(override.ExtraTime * 60) // ExtraTime is in minutes
		}
	}

	if dailyLimit > 0 && float64(todayUsage) >= dailyLimit {
		return false
	}

	// Placeholder logic: allow all users to log in
	return true
}

// GetTimeRemaining calculates the time remaining (in seconds) until a user's session
// should be locked, considering both daily time limits and allowed hours restrictions.
// Returns the minimum of:
//   - Time remaining from daily limit (with ExtraTime overrides applied)
//   - Time until end of allowed hours window (with AllowedHours overrides applied)
//
// Returns math.MaxInt64 if there are no restrictions.
func GetTimeRemaining(username string, state state.State, cfg config.Config, now time.Time) int64 {
	if now.IsZero() {
		now = time.Now()
	}

	// Get user config
	userConfig, exists := cfg.Users[username]
	if !exists {
		// No specific config for user
		// If default is enabled, use default config; otherwise unlimited time
		if cfg.Default.Enabled != nil && *cfg.Default.Enabled {
			userConfig = cfg.Default
		} else {
			return math.MaxInt64
		}
	}

	// Get user state
	userState, err := state.GetUser(username)
	if err != nil {
		// User not in state - unlimited time
		return math.MaxInt64
	}

	// Calculate time remaining from daily limit
	var timeRemainingFromLimit int64 = math.MaxInt64
	if userConfig.DailyLimit > 0 {
		timeUsedSeconds := userState.GetTimeUsed()
		dailyLimitSeconds := int64(time.Duration(userConfig.DailyLimit).Seconds())

		// Apply ExtraTime from active overrides
		for _, override := range userState.Overrides {
			if !override.IsExpired(now) && override.ExtraTime > 0 {
				dailyLimitSeconds += int64(override.ExtraTime * 60) // ExtraTime is in minutes
			}
		}

		timeRemainingFromLimit = dailyLimitSeconds - timeUsedSeconds
		if timeRemainingFromLimit < 0 {
			timeRemainingFromLimit = 0
		}
	}

	// Calculate time until end of allowed hours window
	var timeUntilEndOfWindow int64 = math.MaxInt64

	// Check if there's an active AllowedHours override
	var allowedHours config.TimeRange
	hasOverride := false
	for _, override := range userState.Overrides {
		if !override.IsExpired(now) && !override.AllowedHours.IsEmpty() {
			allowedHours = override.AllowedHours
			hasOverride = true
			break
		}
	}

	// If no override, use config-based allowed hours
	if !hasOverride {
		isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday
		if isWeekend && !userConfig.WeekendHours.IsEmpty() {
			allowedHours = userConfig.WeekendHours
		} else if !isWeekend && !userConfig.AllowedHours.IsEmpty() {
			allowedHours = userConfig.AllowedHours
		}
	}

	// Calculate time until end of window if there's a restriction
	if !allowedHours.IsEmpty() {
		endOfWindow := time.Date(now.Year(), now.Month(), now.Day(),
			allowedHours.End.Hour(), allowedHours.End.Minute(), 0, 0, now.Location())

		if now.Before(endOfWindow) {
			timeUntilEndOfWindow = int64(endOfWindow.Sub(now).Seconds())
		} else {
			// Past the allowed hours - no time remaining
			timeUntilEndOfWindow = 0
		}
	}

	// Return the minimum of the two constraints
	if timeRemainingFromLimit < timeUntilEndOfWindow {
		return timeRemainingFromLimit
	}
	return timeUntilEndOfWindow
}

// CheckSendNotification determines if a notification should be sent based on
// the time remaining and configured notification thresholds.
// Returns true if timeRemainingSeconds is within any notification window.
// A notification window is: timeRemaining <= threshold && timeRemaining > (threshold - 1 minute)
func CheckSendNotification(timeRemainingSeconds int64, notifyBefore []config.Duration) bool {
	if len(notifyBefore) == 0 {
		return false
	}

	timeRemaining := time.Duration(timeRemainingSeconds) * time.Second

	for _, notifyDuration := range notifyBefore {
		notifyAt := time.Duration(notifyDuration)

		// Check if we should notify (within 1 minute window to account for check interval)
		if timeRemaining <= notifyAt && timeRemaining > (notifyAt-time.Minute) {
			return true
		}
	}

	return false
}
