package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
)

func TestUser_AddSessionAndGetSessionByID(t *testing.T) {
	u := &User{}
	start := time.Now().Add(-1 * time.Hour)
	sessionID := "abc123"

	u.AddSession(start, sessionID)
	if len(u.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(u.Sessions))
	}
	s, err := u.GetSessionByID(sessionID)
	if err != nil {
		t.Fatalf("GetSessionByID returned error: %v", err)
	}
	if s.SessionId != sessionID {
		t.Errorf("SessionId = %v, want %v", s.SessionId, sessionID)
	}
	if !s.StartTime.Equal(start) {
		t.Errorf("StartTime = %v, want %v", s.StartTime, start)
	}
}

func TestUser_EndSession(t *testing.T) {
	u := &User{}
	start := time.Now().Add(-2 * time.Hour)
	sessionID := "endme"
	u.AddSession(start, sessionID)

	err := u.EndSession(time.Now(), sessionID)
	if err != nil {
		t.Errorf("EndSession returned error: %v", err)
	}
	s, _ := u.GetSessionByID(sessionID)
	if s.IsActive() {
		t.Errorf("Session should not be active after EndSession")
	}

	// Try to end again, should error
	err = u.EndSession(time.Now(), sessionID)
	if err == nil {
		t.Errorf("Expected error when ending already ended session")
	}
}

func TestUser_EndAllSegmentsAndStartNewSegments(t *testing.T) {
	u := &User{}
	start := time.Now().Add(-1 * time.Hour)
	sessionID := "segtest"
	u.AddSession(start, sessionID)

	u.EndAllSegments("forced")
	for _, sess := range u.Sessions {
		for _, seg := range sess.Segments {
			if seg.EndTime.IsZero() {
				t.Errorf("Segment EndTime should be set after EndAllSegments")
			}
		}
	}

	u.StartNewSegments()
	for _, sess := range u.Sessions {
		fmt.Println(sess.IsIdle())
		if sess.IsActive() && sess.IsIdle() {
			t.Errorf("Expected new segment to be started for idle session")
		}
	}
}

func TestUser_DontStartNewSegmentForActiveSegment(t *testing.T) {
	u := &User{}
	start := time.Now().Add(-1 * time.Hour)
	sessionID := "activeSegTest"
	u.AddSession(start, sessionID)

	// Add a segment and do not end it
	sess, _ := u.GetSessionByID(sessionID)
	sess.AddSegment(time.Now().Add(-30 * time.Minute))

	u.StartNewSegments()
	for _, sess := range u.Sessions {
		if sess.IsActive() && len(sess.Segments) != 1 {
			t.Errorf("No new segment should be started if there's an active segment")
		}
	}
}

func TestUser_GetActiveSession(t *testing.T) {
	u := &User{}
	u.AddSession(time.Now().Add(-1*time.Hour), "a1")
	u.AddSession(time.Now().Add(-30*time.Minute), "a2")
	u.EndSession(time.Now(), "a2")
	active := u.GetActiveSession()
	if active == nil || active.SessionId != "a1" {
		t.Errorf("GetActiveSession returned wrong session: %+v", active)
	}
}

func TestUser_IsSessionActive(t *testing.T) {
	u := &User{}
	u.AddSession(time.Now(), "foo")
	if !u.IsSessionActive("foo") {
		t.Errorf("IsSessionActive = false, want true")
	}
	u.EndSession(time.Now(), "foo")
	if u.IsSessionActive("foo") {
		t.Errorf("IsSessionActive = true, want false")
	}
}

func TestUser_PauseResume(t *testing.T) {
	u := &User{}
	u.Pause()
	if !u.Paused {
		t.Errorf("Pause() did not set Paused to true")
	}
	u.Resume()
	if u.Paused {
		t.Errorf("Resume() did not set Paused to false")
	}
}

func TestUser_GetTimeUsed(t *testing.T) {
	u := &User{}
	start := time.Now().Add(-2 * time.Hour)
	u.AddSession(start, "t1")
	u.EndSession(start.Add(1*time.Hour), "t1")
	if u.GetTimeUsed() < 3600 {
		t.Errorf("GetTimeUsed = %d, want at least 3600", u.GetTimeUsed())
	}
}

func TestUser_OverrideAllowedHours(t *testing.T) {
	u := &User{}

	// Create a fixed time range: 09:00-17:00 (9 AM to 5 PM)
	// Parse times to get proper time.Time values with hour/minute set
	start, _ := time.Parse("15:04", "09:00")
	end, _ := time.Parse("15:04", "17:00")

	// Set expiration to tomorrow (won't expire during test)
	expiresAt := time.Now().Add(24 * time.Hour)

	u.AddOverride(NewAllowedHoursOverride("", config.TimeRange{Start: start, End: end}, expiresAt))

	if len(u.Overrides) != 1 {
		t.Errorf("Expected 1 override, got %d", len(u.Overrides))
	}

	if u.AllowedHoursOverrideIsSet() != true {
		t.Errorf("Expected AllowedHoursOverrideIsSet to be true")
	}

	// Test time within range (12:00 PM / noon)
	testTimeInRange := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	if u.AllowedHoursOverrideWithinRange(testTimeInRange) != true {
		t.Errorf("Expected AllowedHoursOverrideWithinRange to be true for 12:00 PM")
	}

	// Test time outside range (20:00 / 8 PM)
	testTimeOutOfRange := time.Date(2024, 1, 1, 20, 0, 0, 0, time.UTC)
	if u.AllowedHoursOverrideWithinRange(testTimeOutOfRange) == true {
		t.Errorf("Expected AllowedHoursOverrideWithinRange to be false for 8:00 PM")
	}

}

func TestUser_OverrideDuration(t *testing.T) {
	u := &User{}
	u.AddOverride(
		NewExtraTimeOverride(
			"",
			3600,
			time.Time{}),
	)

	if len(u.Overrides) != 1 {
		t.Errorf("Expected 1 override, got %d", len(u.Overrides))
	}


}

func TestUser_GetTimeUsedOnlyCountsTodaySessions(t *testing.T) {
	u := &User{}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Add a session from yesterday (1 hour duration)
	u.AddSession(yesterday, "yesterday1")
	u.EndSession(yesterday.Add(1*time.Hour), "yesterday1")

	// Add a session from today (30 minutes duration)
	u.AddSession(now.Add(-30*time.Minute), "today1")
	u.EndSession(now, "today1")

	// GetTimeUsed should only count today's session (30 minutes = 1800 seconds)
	timeUsed := u.GetTimeUsed()
	if timeUsed < 1700 || timeUsed > 1900 { // Allow small margin
		t.Errorf("GetTimeUsed = %d, want ~1800 (30 minutes)", timeUsed)
	}

	// GetTimeUsedForDay for yesterday should count only yesterday's session
	timeUsedYesterday := u.GetTimeUsedForDay(yesterday)
	if timeUsedYesterday < 3500 || timeUsedYesterday > 3700 { // Allow small margin
		t.Errorf("GetTimeUsedForDay(yesterday) = %d, want ~3600 (1 hour)", timeUsedYesterday)
	}
}

func TestUser_RemoveOldSessions(t *testing.T) {
	u := &User{}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Add sessions from different days
	u.AddSession(twoDaysAgo, "twoDaysAgo")
	u.AddSession(yesterday, "yesterday")
	u.AddSession(now.Add(-1*time.Hour), "today1")
	u.AddSession(now.Add(-30*time.Minute), "today2")

	// Should have 4 sessions
	if len(u.Sessions) != 4 {
		t.Fatalf("Expected 4 sessions before cleanup, got %d", len(u.Sessions))
	}

	// Remove old sessions
	u.RemoveOldSessions(now)

	// Should only have today's sessions left
	if len(u.Sessions) != 2 {
		t.Errorf("Expected 2 sessions after cleanup, got %d", len(u.Sessions))
	}

	// Verify the remaining sessions are from today
	for _, session := range u.Sessions {
		if !isSameDay(session.StartTime, now) {
			t.Errorf("Found session from wrong day: %v", session.StartTime)
		}
	}
}

func TestUser_GetSessionsForDay(t *testing.T) {
	u := &User{}
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Add sessions from different days
	u.AddSession(yesterday, "yesterday1")
	u.AddSession(yesterday.Add(-1*time.Hour), "yesterday2")
	u.AddSession(now.Add(-1*time.Hour), "today1")

	// Get yesterday's sessions
	yesterdaySessions := u.GetSessionsForDay(yesterday)
	if len(yesterdaySessions) != 2 {
		t.Errorf("Expected 2 sessions for yesterday, got %d", len(yesterdaySessions))
	}

	// Get today's sessions
	todaySessions := u.GetSessionsForDay(now)
	if len(todaySessions) != 1 {
		t.Errorf("Expected 1 session for today, got %d", len(todaySessions))
	}
}
