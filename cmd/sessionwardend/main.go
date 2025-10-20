package main

import (
	"log"

	"github.com/godbus/dbus/v5"
)

const (
	objectPath    = "/io/github/soarinferret/sessionwarden"
	interfaceName = "io.github.soarinferret.sessionwarden.Manager"
	serviceName   = "io.github.soarinferret.sessionwarden"
)

type SessionManager struct{}

func (s *SessionManager) GetSessionStatus() (string, *dbus.Error) {
	return "Session is active", nil
}

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal("Failed to connect to session bus:", err)
	}
	defer conn.Close()

	reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		log.Fatal("Name already taken or failed to request:", err)
	}

	sm := &SessionManager{}
	err = conn.Export(sm, dbus.ObjectPath(objectPath), interfaceName)
	if err != nil {
		log.Fatal("Failed to export object:", err)
	}

	log.Println("Service running. Press Ctrl+C to exit.")
	select {} // block forever
}
