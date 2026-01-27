package state

import (
	"log"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/session"
)

func (m *Manager) HandleLogin(user string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//log.Println("User logged in:", user)

	u, err := m.state.GetUser(user)
	if err != nil {
		// User does not exist, create new
		u = &session.User{
			Sessions: []session.SessionRecord{},
		}
	}

	// Check if session already exists (e.g., after wake from sleep)
	if u.IsSessionActive(sessionID) {
		// Session exists and is active
		// If it's idle (no active segment), start a new segment
		s, err := u.GetSessionByID(sessionID)
		if err == nil && s.IsIdle() {
			s.AddSegment(time.Now())
			m.state.Users[user] = *u
			m.save()
		}
		return
	}

	// Create new session (which automatically creates first segment)
	u.AddSession(time.Now(), sessionID)
	m.state.Users[user] = *u
	m.save()
}

func (m *Manager) HandleLogout(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	username, u, err := m.state.GetUserBySession(sessionID)
	if err != nil {
		log.Println("Error finding user for session logout:", err)
		return
	}
	//log.Println("User logged out:", username)

	u.EndSession(time.Now(), sessionID)
	m.state.Users[username] = *u
	m.save()
}

func (m *Manager) HandleSleep() {
	m.mu.Lock()
	defer m.mu.Unlock()

	//log.Println("System going to sleep")

	m.state.EndAllSegments("system sleep")
	m.save()
}

func (m *Manager) HandleWake() {
	log.Println("System woke up")
	// Don't create segments here - wait for actual user interaction (unlock/login)
}

func (m *Manager) HandleLock(user string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//fmt.Println("User locked session:", user)

	// End current segment
	u, err := m.state.GetUser(user)
	if err != nil {
		log.Println("Error finding user for lock:", err)
		return
	}
	s, err := u.GetSessionByID(sessionID)
	if err != nil {
		log.Println("Error finding session for lock:", err)
		return
	}
	s.EndSegment(time.Now(), "user lock")

	// Update user in state
	m.state.Users[user] = *u
	m.save()
}

func (m *Manager) HandleUnlock(user string, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	//fmt.Println("User unlocked session:", user)
	u, err := m.state.GetUser(user)
	if err != nil {
		log.Println("Error finding user for unlock:", err)
		return
	}
	s, err := u.GetSessionByID(sessionID)
	if err != nil {
		log.Println("Error finding session for unlock:", err)
		return
	}
	s.AddSegment(time.Now())

	// Update user in state
	m.state.Users[user] = *u
	m.save()
}
