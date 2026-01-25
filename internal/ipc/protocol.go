package ipc

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
	"github.com/SoarinFerret/SessionWarden/internal/eval"
	"github.com/SoarinFerret/SessionWarden/internal/session"
	"github.com/SoarinFerret/SessionWarden/internal/state"
	"github.com/godbus/dbus/v5"
)

const (
	ObjectPath    = "/io/github/soarinferret/sessionwarden"
	InterfaceName = "io.github.soarinferret.sessionwarden.Manager"
	ServiceName   = "io.github.soarinferret.sessionwarden"
)

// Engine interface to avoid circular dependency
type Engine interface {
	LockUserSession(username string) error
	SendNotification(username, sessionPath, message string) error
}

type SessionManager struct {
	Manager *state.Manager
	Config  *config.Config
	Engine  Engine
	conn    *dbus.Conn // D-Bus connection for emitting signals
}

// SetConnection sets the D-Bus connection for signal emission
func (s *SessionManager) SetConnection(conn *dbus.Conn) {
	s.conn = conn
}

// EmitNotificationSignal sends a notification signal on the system bus
// This signal will be picked up by user-mode sessionwardend instances
// The username parameter allows user-mode instances to filter notifications
func (s *SessionManager) EmitNotificationSignal(username, title, message string) error {
	if s.conn == nil {
		return fmt.Errorf("D-Bus connection not set")
	}

	if err := s.conn.Emit(dbus.ObjectPath(ObjectPath), InterfaceName+".NotificationSignal", username, title, message); err != nil {
		return fmt.Errorf("failed to emit notification signal: %w", err)
	}

	log.Printf("Emitted notification signal for user %s: %s - %s", username, title, message)
	return nil
}

func (s *SessionManager) Ping() (string, *dbus.Error) {
	return "pong", nil
}

func (s *SessionManager) CheckLogin(user string) (bool, *dbus.Error) {
	log.Println("CheckLogin called via D-Bus for", user)
	allowed := eval.PermitLogin(user, *s.Manager.GetState(), *s.Config, time.Now())
	return allowed, nil
}

func (s *SessionManager) GetUserStatus(user string) (string, *dbus.Error) {
	log.Println("GetUserStatus called via D-Bus for", user)

	st := s.Manager.GetState()
	u, err := st.GetUser(user)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}

	// Create a response with user data
	type Response struct {
		Paused          bool                   `json:"paused"`
		TimeUsedSeconds int64                  `json:"time_used_seconds"`
		Sessions        []session.SessionRecord `json:"sessions"`
		Overrides       []session.Override      `json:"exceptions"`
	}

	resp := Response{
		Paused:          u.Paused,
		TimeUsedSeconds: u.GetTimeUsed(),
		Sessions:        u.Sessions,
		Overrides:       u.Overrides,
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}

	return string(jsonData), nil
}

func (s *SessionManager) PauseUser(user string) *dbus.Error {
	log.Println("PauseUser called via D-Bus for", user)

	st := s.Manager.GetState()
	u, err := st.GetUser(user)
	if err != nil {
		return dbus.MakeFailedError(err)
	}

	u.Pause()
	st.Users[user] = *u

	// Persist the change
	if err := s.Manager.Save(); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to save state: %w", err))
	}

	// Lock the user's session if they have an active session
	if s.Engine != nil {
		if err := s.Engine.LockUserSession(user); err != nil {
			log.Printf("Warning: Failed to lock session for %s: %v", user, err)
			// Don't return error - pause was successful even if lock failed
		} else {
			log.Printf("Successfully locked session for paused user %s", user)
		}
	}

	return nil
}

func (s *SessionManager) ResumeUser(user string) *dbus.Error {
	log.Println("ResumeUser called via D-Bus for", user)

	st := s.Manager.GetState()
	u, err := st.GetUser(user)
	if err != nil {
		return dbus.MakeFailedError(err)
	}

	u.Resume()
	st.Users[user] = *u

	// Persist the change
	if err := s.Manager.Save(); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to save state: %w", err))
	}

	return nil
}

func (s *SessionManager) AddOverride(user string, reason string, extraTime int, allowedHours string, expiresAtUnix int64) *dbus.Error {
	log.Println("AddOverride called via D-Bus for", user)

	st := s.Manager.GetState()
	u, err := st.GetUser(user)
	if err != nil {
		// User doesn't exist yet, create them
		u = &session.User{
			Sessions:  []session.SessionRecord{},
			Overrides: []session.Override{},
			Paused:    false,
		}
	}

	expiresAt := time.Unix(expiresAtUnix, 0)

	var override session.Override
	if extraTime > 0 && allowedHours != "" {
		return dbus.MakeFailedError(fmt.Errorf("cannot specify both extra time and allowed hours"))
	} else if extraTime > 0 {
		override = session.NewExtraTimeOverride(reason, extraTime, expiresAt)
	} else if allowedHours != "" {
		timeRange, err := config.ParseTimeRange(allowedHours)
		if err != nil {
			return dbus.MakeFailedError(fmt.Errorf("invalid time range: %w", err))
		}
		override = session.NewAllowedHoursOverride(reason, timeRange, expiresAt)
	} else {
		return dbus.MakeFailedError(fmt.Errorf("must specify either extra time or allowed hours"))
	}

	u.AddOverride(override)
	st.Users[user] = *u

	// Persist the change
	if err := s.Manager.Save(); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to save state: %w", err))
	}

	return nil
}

func (s *SessionManager) ListOverrides(user string) (string, *dbus.Error) {
	log.Println("ListOverrides called via D-Bus for", user)

	st := s.Manager.GetState()

	result := make(map[string][]session.Override)

	if user != "" {
		// List overrides for specific user
		u, err := st.GetUser(user)
		if err != nil {
			return "", dbus.MakeFailedError(err)
		}
		result[user] = u.Overrides
	} else {
		// List all overrides for all users
		for username, userData := range st.Users {
			if len(userData.Overrides) > 0 {
				result[username] = userData.Overrides
			}
		}
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", dbus.MakeFailedError(err)
	}

	return string(jsonData), nil
}

func (s *SessionManager) RemoveOverride(user string, index int) *dbus.Error {
	log.Println("RemoveOverride called via D-Bus for", user, "index", index)

	st := s.Manager.GetState()
	u, err := st.GetUser(user)
	if err != nil {
		return dbus.MakeFailedError(err)
	}

	if index < 0 || index >= len(u.Overrides) {
		return dbus.MakeFailedError(fmt.Errorf("invalid index: %d (user has %d overrides)", index, len(u.Overrides)))
	}

	// Remove the override at the specified index
	u.Overrides = append(u.Overrides[:index], u.Overrides[index+1:]...)
	st.Users[user] = *u

	// Persist the change
	if err := s.Manager.Save(); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to save state: %w", err))
	}

	return nil
}

func (s *SessionManager) SendNotification(user string, message string) *dbus.Error {
	log.Println("SendNotification called via D-Bus for", user, "message:", message)

	st := s.Manager.GetState()
	u, err := st.GetUser(user)
	if err != nil {
		return dbus.MakeFailedError(fmt.Errorf("user not found: %w", err))
	}

	// Get the user's active session
	activeSession := u.GetActiveSession()
	if activeSession == nil {
		return dbus.MakeFailedError(fmt.Errorf("no active session for user %s", user))
	}

	// Use the Engine to send the notification
	if s.Engine == nil {
		return dbus.MakeFailedError(fmt.Errorf("engine not available"))
	}

	if err := s.Engine.SendNotification(user, activeSession.SessionId, message); err != nil {
		return dbus.MakeFailedError(fmt.Errorf("failed to send notification: %w", err))
	}

	return nil
}
