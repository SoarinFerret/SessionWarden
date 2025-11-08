package state

import (
	"fmt"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/session"
)

// State is the top-level structure stored in the state.json file.
type State struct {
	Users     map[string]session.User `json:"users"`
	Version   int                     `json:"version"`
	HeartBeat time.Time               `json:"-"` // not stored in JSON
}

func (s *State) GetUser(username string) (*session.User, error) {
	user, exists := s.Users[username]
	if !exists {
		return nil, fmt.Errorf("user %s not found", username)
	}
	return &user, nil
}

func (s *State) GetUserBySession(sessionID string) (string, *session.User, error) {

	for uname, user := range s.Users {
		for _, session := range user.Sessions {
			if session.SessionId == sessionID {
				return uname, &user, nil
			}
		}
	}

	return "", nil, fmt.Errorf("session ID %s not found", sessionID)
}

func (s *State) EndAllSegments(reason string) {
	for uname, user := range s.Users {
		user.EndAllSegments(reason)
		s.Users[uname] = user
	}
}

func (s *State) StartNewSegments() {
	for uname, user := range s.Users {
		user.StartNewSegments()
		s.Users[uname] = user
	}
}
