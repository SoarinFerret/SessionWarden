package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
	"github.com/SoarinFerret/SessionWarden/internal/eval"
	"github.com/SoarinFerret/SessionWarden/internal/state"
	"github.com/godbus/dbus/v5"
)

// NotificationEmitter is an interface for sending notifications
type NotificationEmitter interface {
	EmitNotificationSignal(username, title, message string) error
}

// Engine monitors active sessions and enforces time limits
type Engine struct {
	stateMgr         *state.Manager
	config           *config.Config
	conn             *dbus.Conn
	notificationEmit NotificationEmitter
}

// NewEngine creates a new user engine instance
func NewEngine(stateMgr *state.Manager, cfg *config.Config) (*Engine, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %w", err)
	}

	return &Engine{
		stateMgr: stateMgr,
		config:   cfg,
		conn:     conn,
	}, nil
}

// SetNotificationEmitter sets the notification emitter for sending notifications
func (e *Engine) SetNotificationEmitter(emitter NotificationEmitter) {
	e.notificationEmit = emitter
}

// Run starts the periodic checker (runs every minute)
func (e *Engine) Run(ctx context.Context) error {
	defer e.conn.Close()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("User engine started - monitoring active sessions...")

	// Run immediately on start
	e.checkSessions()

	for {
		select {
		case <-ctx.Done():
			log.Println("User engine shutting down...")
			return nil
		case <-ticker.C:
			e.checkSessions()
		}
	}
}

// checkSessions evaluates all active sessions
func (e *Engine) checkSessions() {
	now := time.Now()
	currentState := e.stateMgr.GetState()

	log.Printf("DEBUG: Checking sessions at %s", now.Format(time.RFC3339))

	for username, user := range currentState.Users {
		// Skip if user is paused or has no active sessions
		if user.Paused {
			continue
		}

		// Get user configuration
		userConfig, exists := e.config.Users[username]
		if !exists {
			continue // No policy for this user
		}

		// Skip if user is not enabled
		if userConfig.Enabled != nil && !*userConfig.Enabled {
			continue
		}

		// Check if user has active sessions
		activeSession := user.GetActiveSession()
		if activeSession == nil {
			continue
		}

		// check if eval.PermitLogin would block login now
		if !eval.PermitLogin(username, *currentState, *e.config, now) {
			log.Printf("User %s is not permitted to log in now - locking session", username)
			if err := e.lockSession(username, activeSession.SessionId, userConfig); err != nil {
				log.Printf("Failed to lock session for %s: %v", username, err)
			}
			continue
		}

		// Calculate time remaining using eval package (handles overrides)
		timeRemainingSeconds := eval.GetTimeRemaining(username, *currentState, *e.config, now)

		// Send notifications based on notify_before configuration
		e.sendNotifications(username, activeSession.SessionId, timeRemainingSeconds, userConfig.NotifyBefore)
	}

	// Update heartbeat
	e.stateMgr.Heartbeat()
}

// sendNotifications sends desktop notifications to users when time is running low
func (e *Engine) sendNotifications(username, sessionPath string, timeRemainingSeconds int64, notifyBefore []config.Duration) {

	log.Printf("DEBUG: Preparing to send notifications to %s with %d seconds remaining", username, timeRemainingSeconds)

	// Check if we should send a notification
	if !eval.CheckSendNotification(timeRemainingSeconds, notifyBefore) {
		return
	}

	// Send notification
	timeRemaining := time.Duration(timeRemainingSeconds) * time.Second
	message := formatTimeRemaining(timeRemaining)
	if err := e.sendDesktopNotification(username, sessionPath, message); err != nil {
		log.Printf("Failed to send notification to %s: %v", username, err)
	} else {
		log.Printf("Sent notification to %s: %s remaining", username, message)
	}
}

// formatTimeRemaining formats duration into human-readable string
func formatTimeRemaining(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d hour(s) %d minute(s)", hours, minutes)
	}
	return fmt.Sprintf("%d minute(s)", minutes)
}

// sendDesktopNotification sends a notification signal via D-Bus
func (e *Engine) sendDesktopNotification(username, sessionPath, message string) error {
	if e.notificationEmit == nil {
		return fmt.Errorf("notification emitter not set")
	}

	title := "Session Time Warning"
	body := fmt.Sprintf("You have %s of session time remaining", message)

	return e.notificationEmit.EmitNotificationSignal(username, title, body)
}

// SendNotification sends a custom notification to a user (public method for IPC)
func (e *Engine) SendNotification(username, sessionPath, message string) error {
	if e.notificationEmit == nil {
		return fmt.Errorf("notification emitter not set")
	}

	title := "SessionWarden"
	return e.notificationEmit.EmitNotificationSignal(username, title, message)
}

// LockUserSession locks the active session for a user (public method for IPC)
func (e *Engine) LockUserSession(username string) error {
	currentState := e.stateMgr.GetState()

	// Get user from state
	user, err := currentState.GetUser(username)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get user configuration
	userConfig, exists := e.config.Users[username]
	if !exists {
		// No policy - create default config with lock enabled
		lockEnabled := true
		userConfig = config.UserConfig{
			LockScreen: &lockEnabled,
		}
	}

	// Get active session
	activeSession := user.GetActiveSession()
	if activeSession == nil {
		return fmt.Errorf("no active session for user %s", username)
	}

	return e.lockSession(username, activeSession.SessionId, userConfig)
}

// lockSession locks a specific user session using loginctl
func (e *Engine) lockSession(username, sessionPath string, userConfig config.UserConfig) error {
	// Check if we should lock or just log
	if userConfig.LockScreen != nil && !*userConfig.LockScreen {
		log.Printf("Lock screen disabled for %s - session would be locked but policy says no", username)
		return nil
	}

	// Get the session object
	sessionObj := e.conn.Object("org.freedesktop.login1", dbus.ObjectPath(sessionPath))

	// Check if session is already locked
	lockedVariant, err := sessionObj.GetProperty("org.freedesktop.login1.Session.LockedHint")
	if err != nil {
		return fmt.Errorf("failed to get LockedHint from path %s: %w", sessionPath, err)
	}

	isLocked := lockedVariant.Value().(bool)
	if isLocked {
		log.Printf("Session for %s is already locked, skipping", username)
		return nil
	}

	// Get the actual session ID from the session object
	idVariant, err := sessionObj.GetProperty("org.freedesktop.login1.Session.Id")
	if err != nil {
		return fmt.Errorf("failed to get session ID from path %s: %w", sessionPath, err)
	}

	sessionID := idVariant.Value().(string)

	// Lock the session using the actual ID
	managerObj := e.conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
	call := managerObj.Call("org.freedesktop.login1.Manager.LockSession", 0, sessionID)

	if call.Err != nil {
		return fmt.Errorf("failed to lock session %s for %s: %w", sessionID, username, call.Err)
	}

	log.Printf("Successfully locked session %s for user %s", sessionID, username)
	return nil
}
