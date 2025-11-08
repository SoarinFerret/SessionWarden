package session

import (
	"testing"
	"time"
)

func TestSessionRecord_StartAndEnd(t *testing.T) {
	var s SessionRecord
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now().Add(-1 * time.Hour)

	s.Start(start)
	if !s.StartTime.Equal(start) {
		t.Errorf("StartTime = %v, want %v", s.StartTime, start)
	}
	if len(s.Segments) != 1 {
		t.Fatalf("expected 1 segment after Start, got %d", len(s.Segments))
	}
	if !s.Segments[0].StartTime.Equal(start) {
		t.Errorf("Segment StartTime = %v, want %v", s.Segments[0].StartTime, start)
	}

	s.End(end)
	if !s.EndTime.Equal(end) {
		t.Errorf("EndTime = %v, want %v", s.EndTime, end)
	}
	if s.Segments[0].EndTime != end {
		t.Errorf("Segment EndTime = %v, want %v", s.Segments[0].EndTime, end)
	}
}

func TestSessionRecord_IsActive(t *testing.T) {
	s := SessionRecord{}
	if !s.IsActive() {
		t.Errorf("IsActive() = false, want true when EndTime is zero")
	}
	s.EndTime = time.Now()
	if s.IsActive() {
		t.Errorf("IsActive() = true, want false when EndTime is set")
	}
}

func TestSessionRecord_AddAndEndSegment(t *testing.T) {
	s := SessionRecord{}
	start := time.Now().Add(-30 * time.Minute)
	s.AddSegment(start)
	if len(s.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(s.Segments))
	}
	if !s.Segments[0].StartTime.Equal(start) {
		t.Errorf("Segment StartTime = %v, want %v", s.Segments[0].StartTime, start)
	}
	reason := "break"
	end := time.Now()
	s.EndSegment(end, reason)
	if !s.Segments[0].EndTime.Equal(end) {
		t.Errorf("Segment EndTime = %v, want %v", s.Segments[0].EndTime, end)
	}
	if s.Segments[0].Reason != reason {
		t.Errorf("Segment Reason = %v, want %v", s.Segments[0].Reason, reason)
	}
}

func TestSessionRecord_IsIdle(t *testing.T) {
	s := SessionRecord{}
	if s.IsIdle() {
		t.Errorf("IsIdle() = true, want false when no segments")
	}
	s.AddSegment(time.Now())
	if s.IsIdle() {
		t.Errorf("IsIdle() = true, want false when segment is active")
	}
	s.EndSegment(time.Now(), "idle")
	if !s.IsIdle() {
		t.Errorf("IsIdle() = false, want true when last segment is ended")
	}
}

func TestSessionRecord_Duration(t *testing.T) {
	s := SessionRecord{}
	start1 := time.Now().Add(-2 * time.Hour)
	end1 := start1.Add(1 * time.Hour)
	s.AddSegment(start1)
	s.EndSegment(end1, "first")

	start2 := end1.Add(10 * time.Minute)
	end2 := start2.Add(30 * time.Minute)
	s.AddSegment(start2)
	s.EndSegment(end2, "second")

	want := int64(60*60 + 30*60) // 1h + 30m in seconds
	got := s.Duration()
	if got < want || got > want+2 {
		t.Errorf("Duration() = %d, want around %d", got, want)
	}
}
