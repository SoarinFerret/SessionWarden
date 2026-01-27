package config

import (
	"encoding/json"
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

func TestDurationUnmarshalTOML(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{"Valid duration", "2h", 2 * time.Hour, false},
		{"Invalid duration", "invalid", 0, true},
		{"Empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalText([]byte(tt.input))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, Duration(tt.expected), d)
			}
		})
	}
}

func TestSetDefault(t *testing.T) {
	defaultConfig := UserConfig{
		DailyLimit: Duration(2 * time.Hour),
		AllowedHours: TimeRange{
			Start: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
			End:   time.Date(0, 1, 1, 17, 0, 0, 0, time.UTC),
		},
		WeekendHours: TimeRange{
			Start: time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC),
			End:   time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC),
		},
		NotifyBefore: []Duration{Duration(10 * time.Minute), Duration(5 * time.Minute)},
		LockScreen:   nil,
		Enabled:      nil,
	}

	config := Config{
		Default: defaultConfig,
		Users: map[string]UserConfig{
			"user1": {
				DailyLimit: 0,
			},
			"user2": {
				NotifyBefore: []Duration{Duration(15 * time.Minute)},
			},
		},
	}

	config.SetDefault()

	assert.Equal(t, Duration(2*time.Hour), config.Users["user1"].DailyLimit)
	assert.Equal(t, defaultConfig.AllowedHours, config.Users["user1"].AllowedHours)
	assert.Equal(t, []Duration{Duration(15 * time.Minute)}, config.Users["user2"].NotifyBefore)
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

	AppConfig, err := LoadConfigFromBytes([]byte(tomlData))
	assert.NoError(t, err)

	assert.Equal(t, Duration(2*time.Hour), AppConfig.Default.DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Default.AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Default.AllowedHours.End.Format("15:04"))
	assert.Equal(t, true, *AppConfig.Default.LockScreen)
	assert.Equal(t, true, *AppConfig.Default.Enabled)

	assert.Equal(t, Duration(3*time.Hour), AppConfig.Users["user1"].DailyLimit)
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

	AppConfig, err := LoadConfigFromBytes([]byte(tomlData))
	assert.NoError(t, err)

	assert.Equal(t, Duration(2*time.Hour), AppConfig.Default.DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Default.AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Default.AllowedHours.End.Format("15:04"))
	assert.Equal(t, true, *AppConfig.Default.LockScreen)
	assert.Equal(t, true, *AppConfig.Default.Enabled)

	assert.Equal(t, Duration(3*time.Hour), AppConfig.Users["user1"].DailyLimit)
	assert.Equal(t, "09:00", AppConfig.Users["user1"].AllowedHours.Start.Format("15:04"))
	assert.Equal(t, "17:00", AppConfig.Users["user1"].AllowedHours.End.Format("15:04"))
}

func TestTimeRangeJSON(t *testing.T) {
	tests := []struct {
		name        string
		timeRange   TimeRange
		expectedJSON string
	}{
		{
			name: "Valid time range",
			timeRange: TimeRange{
				Start: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
				End:   time.Date(0, 1, 1, 17, 30, 0, 0, time.UTC),
			},
			expectedJSON: `"09:00-17:30"`,
		},
		{
			name: "Empty time range",
			timeRange: TimeRange{},
			expectedJSON: `null`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name + " Marshal", func(t *testing.T) {
			data, err := json.Marshal(tt.timeRange)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedJSON, string(data))
		})

		t.Run(tt.name + " Unmarshal", func(t *testing.T) {
			var tr TimeRange
			err := json.Unmarshal([]byte(tt.expectedJSON), &tr)
			assert.NoError(t, err)
			if tt.expectedJSON == `null` {
				assert.True(t, tr.IsEmpty())
			} else {
				assert.Equal(t, tt.timeRange.Start.Hour(), tr.Start.Hour())
				assert.Equal(t, tt.timeRange.Start.Minute(), tr.Start.Minute())
				assert.Equal(t, tt.timeRange.End.Hour(), tr.End.Hour())
				assert.Equal(t, tt.timeRange.End.Minute(), tr.End.Minute())
			}
		})
	}
}

func TestTimeRangeJSONRoundTrip(t *testing.T) {
	// Test with a struct that contains TimeRange (like Override)
	type TestStruct struct {
		AllowedHours TimeRange `json:"allowed_hours"`
	}

	original := TestStruct{
		AllowedHours: TimeRange{
			Start: time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC),
			End:   time.Date(0, 1, 1, 23, 59, 0, 0, time.UTC),
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"09:00-23:59"`)

	// Unmarshal back
	var decoded TestStruct
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, original.AllowedHours.Start.Hour(), decoded.AllowedHours.Start.Hour())
	assert.Equal(t, original.AllowedHours.Start.Minute(), decoded.AllowedHours.Start.Minute())
	assert.Equal(t, original.AllowedHours.End.Hour(), decoded.AllowedHours.End.Hour())
	assert.Equal(t, original.AllowedHours.End.Minute(), decoded.AllowedHours.End.Minute())
}
