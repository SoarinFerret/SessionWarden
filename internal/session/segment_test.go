package session

import (
	"testing"
	"time"
)

func TestSegmentRecord_Duration(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantMin int64
		wantMax int64
	}{
		{
			name:    "active segment (EndTime zero)",
			start:   now.Add(-10 * time.Minute),
			end:     time.Time{},
			wantMin: 600, // 10 minutes in seconds
			wantMax: 605, // allow a few seconds for test execution
		},
		{
			name:    "ended segment",
			start:   now.Add(-20 * time.Minute),
			end:     now.Add(-10 * time.Minute),
			wantMin: 600,
			wantMax: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := SegmentRecord{StartTime: tt.start, EndTime: tt.end}
			got := seg.Duration()
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Duration() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSegmentRecord_IsActive(t *testing.T) {
	segActive := SegmentRecord{StartTime: time.Now(), EndTime: time.Time{}}
	segEnded := SegmentRecord{StartTime: time.Now(), EndTime: time.Now()}

	if !segActive.IsActive() {
		t.Errorf("IsActive() = false, want true for active segment")
	}
	if segEnded.IsActive() {
		t.Errorf("IsActive() = true, want false for ended segment")
	}
}
