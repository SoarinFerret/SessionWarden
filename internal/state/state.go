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

// GetUserBySession finds the user owning the given session ID. logind reuses
// session IDs across reboots, so prefer a user with an active session of that
// ID over one holding an ended record with the same ID.
func (s *State) GetUserBySession(sessionID string) (string, *session.User, error) {
	var foundName string
	var foundUser *session.User

	for uname, user := range s.Users {
		for _, sess := range user.Sessions {
			if sess.SessionId == sessionID {
				if sess.IsActive() {
					return uname, &user, nil
				}
				if foundUser == nil {
					foundName, foundUser = uname, &user
				}
			}
		}
	}

	if foundUser != nil {
		return foundName, foundUser, nil
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
