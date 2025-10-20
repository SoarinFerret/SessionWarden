package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func (tr *TimeRange) UnmarshalText(text []byte) error {
	str := string(text)
	parts := strings.Split(str, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid time range format: expected 'HH:MM-HH:MM'")
	}

	layout := "15:04"
	start, err1 := time.Parse(layout, parts[0])
	end, err2 := time.Parse(layout, parts[1])
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid time values: %v, %v", err1, err2)
	}

	// Optional: Adjust to same date to compare time of day
	now := time.Now()
	s := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, time.UTC)
	e := time.Date(now.Year(), now.Month(), now.Day(), end.Hour(), end.Minute(), 0, 0, time.UTC)

	if !s.Before(e) {
		return fmt.Errorf("start time %s must be before end time %s", parts[0], parts[1])
	}

	tr.Start = start
	tr.End = end

	return nil
}

type UserConfig struct {
	DailyLimit   string    `toml:"daily_limit"`
	AllowedHours TimeRange `toml:"allowed_hours"`
	WeekendHours TimeRange `toml:"weekend_hours"`
	NotifyBefore []string  `toml:"notify_before"`
	LockScreen   *bool     `toml:"lock_screen"`
	Enabled      *bool     `toml:"enabled"`
}

type Config struct {
	Default UserConfig            `toml:"default"`
	Users   map[string]UserConfig `toml:"users"`
}

// SetDefault sets default configuration values for each user based on the Default config.
func (c *Config) SetDefault() {
	if c.Default.Enabled == nil {
		defaultVal := false
		c.Default.Enabled = &defaultVal
	}
	if c.Default.LockScreen == nil {
		defaultVal := false
		c.Default.LockScreen = &defaultVal
	}

	if c.Users != nil {
		for username, userConfig := range c.Users {
			if userConfig.DailyLimit == "" {
				userConfig.DailyLimit = c.Default.DailyLimit
			}
			if userConfig.AllowedHours == (TimeRange{}) {
				userConfig.AllowedHours = c.Default.AllowedHours
			}
			if userConfig.WeekendHours == (TimeRange{}) {
				userConfig.WeekendHours = c.Default.WeekendHours
			}
			if userConfig.NotifyBefore == nil {
				userConfig.NotifyBefore = c.Default.NotifyBefore
			}
			if userConfig.LockScreen == nil {
				userConfig.LockScreen = c.Default.LockScreen
			}
			if userConfig.Enabled == nil {
				userConfig.Enabled = c.Default.Enabled
			}
			c.Users[username] = userConfig
		}
	}
}

var AppConfig Config

func LoadConfigFromFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := toml.NewDecoder(file)
	var config Config
	err = decoder.Decode(&config)
	if err != nil {
		return err
	}
	AppConfig = config
	AppConfig.SetDefault()
	return nil
}

func LoadConfigFromBytes(data []byte) error {
	var config Config
	err := toml.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	AppConfig = config
	AppConfig.SetDefault()
	return nil
}
