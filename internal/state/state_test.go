package state

import (
	"testing"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/session"
)

func makeTestUser(sessionID string, start time.Time) session.User {
	u := session.User{}
	u.AddSession(start, sessionID)
	return u
}

func TestState_GetUser(t *testing.T) {
	st := State{
		Users: map[string]session.User{
			"alice": makeTestUser("sess1", time.Now().Add(-1*time.Hour)),
		},
	}
	user, err := st.GetUser("alice")
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}
	if user == nil {
		t.Fatalf("GetUser returned nil user")
	}
	if len(user.Sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(user.Sessions))
	}

	_, err = st.GetUser("bob")
	if err == nil {
		t.Errorf("expected error for missing user, got nil")
	}
}

func TestState_GetUserBySession(t *testing.T) {
	start := time.Now().Add(-2 * time.Hour)
	st := State{
		Users: map[string]session.User{
			"alice": makeTestUser("sessA", start),
			"bob":   makeTestUser("sessB", start),
		},
	}
	uname, user, err := st.GetUserBySession("sessB")
	if err != nil {
		t.Fatalf("GetUserBySession returned error: %v", err)
	}
	if uname != "bob" {
		t.Errorf("expected username 'bob', got %s", uname)
	}
	if user == nil || user.Sessions[0].SessionId != "sessB" {
		t.Errorf("expected session ID 'sessB', got %+v", user)
	}

	_, _, err = st.GetUserBySession("notfound")
	if err == nil {
		t.Errorf("expected error for missing session ID, got nil")
	}
}

func TestState_EndAllSegmentsAndStartNewSegments(t *testing.T) {
	start := time.Now().Add(-1 * time.Hour)
	st := State{
		Users: map[string]session.User{
			"alice": makeTestUser("sessA", start),
		},
	}
	// End all segments
	st.EndAllSegments("forced")
	for _, user := range st.Users {
		for _, sess := range user.Sessions {
			for _, seg := range sess.Segments {
				if seg.EndTime.IsZero() {
					t.Errorf("Segment EndTime should be set after EndAllSegments")
				}
			}
		}
	}
	// Start new segments for idle sessions
	st.StartNewSegments()
	for _, user := range st.Users {
		for _, sess := range user.Sessions {
			if sess.IsActive() && sess.IsIdle() {
				t.Errorf("Expected new segment to be started for idle session")
			}
		}
	}
}
