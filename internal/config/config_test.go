package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeRangeUnmarshalTOML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"Valid time range", "09:00-17:00", false},
		{"Invalid format", "09:00/17:00", true},
		{"Start time after end time", "17:00-09:00", true},
		{"Invalid time values", "invalid-17:00", true},
		{"Empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tr TimeRange
			err := tr.UnmarshalText([]byte(tt.input))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "09:00", tr.Start.Format("15:04"))
				assert.Equal(t, "17:00", tr.End.Format("15:04"))
			}
		})
	}
}

func TestSetDefault(t *testing.T) {
	defaultConfig := UserConfig{
		DailyLimit: "2h",
		AllowedHours: TimeRange{
			Start: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
			End:   time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
		},
		WeekendHours: TimeRange{
			Start: time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC),
			End:   time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC),
		},
		NotifyBefore: []string{"10m", "5m"},
		LockScreen:   nil,
		Enabled:      nil,
	}

	config := Config{
		Default: defaultConfig,
		Users: map[string]UserConfig{
			"user1": {
				DailyLimit: "",
			},
			"user2": {
				NotifyBefore: []string{"15m"},
			},
		},
	}

	config.SetDefault()

	assert.Equal(t, "2h", config.Users["user1"].DailyLimit)
	assert.Equal(t, defaultConfig.AllowedHours, config.Users["user1"].AllowedHours)
	assert.Equal(t, []string{"15m"}, config.Users["user2"].NotifyBefore)
	assert.Equal(t, defaultConfig.AllowedHours, config.Users["user2"].AllowedHours)
}

func TestLoadConfigFromBytes(t *testing.T) {
	tomlData := `
[default]
daily_limit = "2h"
allowed_hours = "09:00-17:00"
weekend_hours = "10:00-14:00"
notify_before = ["10m", "5m"]
lock_screen = true
enabled = true

[users]
[users.user1]
daily_limit = "3h"
`

	err := LoadConfigFromBytes([]byte(tomlData))
	assert.NoError(t, err)

	assert.Equal(t, "2h", AppConfig.Default.DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Default.AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Default.AllowedHours.End.Format("15:04"))
	assert.Equal(t, true, *AppConfig.Default.LockScreen)
	assert.Equal(t, true, *AppConfig.Default.Enabled)

	assert.Equal(t, "3h", AppConfig.Users["user1"].DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Users["user1"].AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Users["user1"].AllowedHours.End.Format("15:04"))
}

func TestLoadConfigFromFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "config-*.toml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	tomlData := `
[default]
daily_limit = "2h"
allowed_hours = "09:00-17:00"
weekend_hours = "10:00-14:00"
notify_before = ["10m", "5m"]
lock_screen = true
enabled = true

[users]
[users.user1]
daily_limit = "3h"
`
	_, err = tempFile.Write([]byte(tomlData))
	assert.NoError(t, err)
	tempFile.Close()

	err = LoadConfigFromFile(tempFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "2h", AppConfig.Default.DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Default.AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Default.AllowedHours.End.Format("15:04"))
	assert.Equal(t, true, *AppConfig.Default.LockScreen)
	assert.Equal(t, true, *AppConfig.Default.Enabled)

	assert.Equal(t, "3h", AppConfig.Users["user1"].DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Users["user1"].AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Users["user1"].AllowedHours.End.Format("15:04"))
}
