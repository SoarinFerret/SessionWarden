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
	u.AddOverride(NewAllowedHoursOverride("", config.TimeRange{Start: time.Now().Add(-1 * time.Hour), End: time.Now().Add(1 * time.Hour)}, time.Time{}))

	if len(u.Overrides) != 1 {
		t.Errorf("Expected 1 override, got %d", len(u.Overrides))
	}

	if u.AllowedHoursOverrideIsSet() != true {
		t.Errorf("Expected AllowedHoursOverrideIsSet to be true")
	}

	if u.AllowedHoursOverrideWithinRange(time.Now()) != true {
		t.Errorf("Expected AllowedHoursOverrideWithinRange to be true")
	}

	if u.AllowedHoursOverrideWithinRange(time.Now().Add(12*time.Hour)) == true {
		t.Errorf("Expected AllowedHoursOverrideWithinRange to be false")
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
