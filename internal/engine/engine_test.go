package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeRemaining(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Less than 1 hour",
			duration: 45 * time.Minute,
			expected: "45 minute(s)",
		},
		{
			name:     "Exactly 1 hour",
			duration: 1 * time.Hour,
			expected: "1 hour(s) 0 minute(s)",
		},
		{
			name:     "1 hour 30 minutes",
			duration: 90 * time.Minute,
			expected: "1 hour(s) 30 minute(s)",
		},
		{
			name:     "Multiple hours",
			duration: 2*time.Hour + 15*time.Minute,
			expected: "2 hour(s) 15 minute(s)",
		},
		{
			name:     "Less than 1 minute",
			duration: 30 * time.Second,
			expected: "0 minute(s)",
		},
		{
			name:     "5 minutes",
			duration: 5 * time.Minute,
			expected: "5 minute(s)",
		},
		{
			name:     "10 minutes",
			duration: 10 * time.Minute,
			expected: "10 minute(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeRemaining(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
