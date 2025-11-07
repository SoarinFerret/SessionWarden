package state

import (
	"fmt"
	"log"
	"time"
)

func (m *Manager) HandleLogin(user string, sessionID string) {
	log.Println("User logged in:", user)

	m.AddSession(user, sessionID, time.Now())
	m.Save()
}

func (m *Manager) HandleLogout(sessionID string) {
	username, u, err := m.GetUserBySession(sessionID)
	if err != nil {
		log.Println("Error finding user for session logout:", err)
		return
	}
	log.Println("User logged out:", username)

	u.EndSession(sessionID)
	m.Save()
}

func (m *Manager) HandleSleep() {
	log.Println("System going to sleep")

	m.EndAllSegments()
	m.Save()
}

func (m *Manager) HandleWake() {
	log.Println("System woke up")

	m.StartNewSegments()
	m.Save()
}

func (m *Manager) HandleLock(user string, sessionID string) {
	fmt.Println("User locked session:", user)

	err := m.StopSegmentForSession(user, sessionID, "lockscreen")
	if err != nil {
		log.Println("Error stopping segment on lock:", err)
	}
	m.Save()
}

func (m *Manager) HandleUnlock(user string, sessionID string) {
	fmt.Println("User unlocked session:", user)

	err := m.StartSegment(user, sessionID)
	if err != nil {
		log.Println("Error starting segment on unlock:", err)
	}
	m.Save()
}
