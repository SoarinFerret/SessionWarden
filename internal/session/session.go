package session

import (
	"fmt"
	"time"
)

func (s *SessionRecord) Start(start time.Time) {
	if start.IsZero() {
		start = time.Now()
	}
	s.StartTime = start
	// Initialize the first segment
	s.AddSegment(start)
}

func (s *SessionRecord) End(end time.Time) {
	if end.IsZero() {
		end = time.Now()
	}
	s.EndTime = end
	// End the last segment
	if len(s.Segments) > 0 {
		lastIndex := len(s.Segments) - 1
		// only set end time if not already set (aka the user was inactive when session was ended)
		if s.Segments[lastIndex].EndTime.IsZero() {
			s.Segments[lastIndex].EndTime = end
		}
	}
}

func (s *SessionRecord) IsActive() bool {
	return s.EndTime.IsZero()
}

func (s *SessionRecord) IsIdle() bool {
	if len(s.Segments) == 0 {
		return false
	}
	lastSegment := s.Segments[len(s.Segments)-1]
	return lastSegment.IsActive() == false
}

func (s *SessionRecord) AddSegment(start time.Time) error {
	if len(s.Segments) != 0 && !s.IsIdle() {
		return fmt.Errorf("cannot add new segment to active session")
	}

	if start.IsZero() {
		start = time.Now()
	}
	segment := SegmentRecord{
		StartTime: start,
	}
	s.Segments = append(s.Segments, segment)
	return nil
}

func (s *SessionRecord) EndSegment(end time.Time, reason string) {
	if end.IsZero() {
		end = time.Now()
	}
	if len(s.Segments) == 0 {
		return
	}
	lastIndex := len(s.Segments) - 1
	s.Segments[lastIndex].EndTime = end
	s.Segments[lastIndex].Reason = reason
}

func (s *SessionRecord) Duration() int64 {
	// Sum up durations of all segments
	var totalDuration int64
	for _, segment := range s.Segments {
		totalDuration += segment.Duration()
	}
	return totalDuration
}
