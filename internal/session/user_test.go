package session

import (
	"testing"
	"time"
)

func TestAddSession(t *testing.T) {
	user := User{Username: "testuser"}

	startTime := time.Now()
	user.AddSession(startTime)

	if len(user.Sessions) != 1 {
		t.Errorf("expected 1 session, got %d", len(user.Sessions))
	}

	if !user.Sessions[0].start.Equal(startTime) {
		t.Errorf("expected session start time to be %v, got %v", startTime, user.Sessions[0].start)
	}

	if !user.Sessions[0].stop.IsZero() {
		t.Errorf("expected session stop time to be zero, got %v", user.Sessions[0].stop)
	}
}

func TestSessionDuration(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour) // 1 hour ago
	session := Session{start: startTime}

	// Test ongoing session
	duration := session.Duration()
	expectedDuration := int64(3600) // 1 hour in seconds
	if duration < expectedDuration || duration > expectedDuration+1 {
		t.Errorf("expected duration to be around %d seconds, got %d", expectedDuration, duration)
	}

	// Test ended session
	stopTime := startTime.Add(2 * time.Hour) // 2 hours after start
	session.EndSession(stopTime)
	duration = session.Duration()
	expectedDuration = int64(7200) // 2 hours in seconds
	if duration != expectedDuration {
		t.Errorf("expected duration to be %d seconds, got %d", expectedDuration, duration)
	}
}

func TestEndSession(t *testing.T) {
	startTime := time.Now()
	session := Session{start: startTime}

	stopTime := startTime.Add(30 * time.Minute) // 30 minutes after start
	session.EndSession(stopTime)

	if !session.stop.Equal(stopTime) {
		t.Errorf("expected stop time to be %v, got %v", stopTime, session.stop)
	}
}

func TestEndSessionNow(t *testing.T) {
	startTime := time.Now()
	session := Session{start: startTime}

	session.EndSessionNow()

	if session.stop.IsZero() {
		t.Errorf("expected stop time to be set, got zero value")
	}

	if session.stop.Before(startTime) {
		t.Errorf("expected stop time to be after start time, got %v", session.stop)
	}
}

func TestTotalSessionDuration(t *testing.T) {
	user := User{Username: "testuser"}

	// Add a session from yesterday (should not be included in total)
	yesterday := time.Now().Add(-24 * time.Hour)
	user.AddSession(yesterday)
	user.Sessions[0].EndSession(yesterday.Add(1 * time.Hour)) // 1 hour session

	// Add a session from today
	todayStart := time.Now().Add(-2 * time.Hour) // 2 hours ago
	user.AddSession(todayStart)
	user.Sessions[1].EndSession(todayStart.Add(1 * time.Hour)) // 1 hour session

	// Add an ongoing session from today
	ongoingStart := time.Now().Add(-30 * time.Minute) // 30 minutes ago
	user.AddSession(ongoingStart)

	totalDuration := user.TotalSessionDuration()
	expectedDuration := int64(3600 + 1800) // 1 hour + 30 minutes in seconds

	if totalDuration < expectedDuration || totalDuration > expectedDuration+1 {
		t.Errorf("expected total duration to be around %d seconds, got %d", expectedDuration, totalDuration)
	}
}
