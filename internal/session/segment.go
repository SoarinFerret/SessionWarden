package session

import "time"

func (s *SegmentRecord) Duration() int64 {
	if s.EndTime.IsZero() {
		return time.Now().Unix() - s.StartTime.Unix()
	}
	return s.EndTime.Unix() - s.StartTime.Unix()
}

func (s *SegmentRecord) IsActive() bool {
	return s.EndTime.IsZero()
}
