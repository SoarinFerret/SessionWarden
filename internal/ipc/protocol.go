package ipc

import (
	//"fmt"
	//"log"

	"github.com/SoarinFerret/SessionWarden/internal/state"
	"github.com/godbus/dbus/v5"
)

const (
	ObjectPath    = "/io/github/soarinferret/sessionwarden"
	InterfaceName = "io.github.soarinferret.sessionwarden.Manager"
	ServiceName   = "io.github.soarinferret.sessionwarden"
)

type SessionManager struct {
	Manager *state.Manager
}

func (s *SessionManager) GetStatus() (string, *dbus.Error) {
	return "Service is running", nil
}

func (s *SessionManager) GetUserStatus(user string) (string, *dbus.Error) {
	//status, err := s.Manager.GetUserStatus(user)
	return "Not implemented", nil
}
