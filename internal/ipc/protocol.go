package ipc

import (
	//"fmt"
	//"log"

	"log"
	"time"

	"github.com/SoarinFerret/SessionWarden/internal/config"
	"github.com/SoarinFerret/SessionWarden/internal/eval"
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
	Config  *config.Config
}

func (s *SessionManager) Ping() (string, *dbus.Error) {
	return "pong", nil
}

func (s *SessionManager) CheckLogin(user string) (bool, *dbus.Error) {
	// Implement login check logic here
	log.Println("CheckLogin called via D-Bus for ", user)
	allowed := eval.PermitLogin(user, *s.Manager.GetState(), *s.Config, time.Now())
	return allowed, nil
}

func (s *SessionManager) GetUserStatus(user string) (string, *dbus.Error) {
	//status, err := s.Manager.GetUserStatus(user)
	return "Not implemented", nil
}
