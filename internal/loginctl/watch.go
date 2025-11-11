package loginctl

import (
	"context"
	"fmt"
	"log"

	"github.com/SoarinFerret/SessionWarden/internal/state"
	"github.com/godbus/dbus/v5"
)

func Watch(ctx context.Context, sm *state.Manager) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	// Add a match rule for relevant signals
	signalMatches := []struct {
		member string
	}{
		{"SessionNew"},
		{"SessionRemoved"},
		{"PrepareForSleep"},
	}
	for _, match := range signalMatches {
		if err := conn.AddMatchSignal(
			dbus.WithMatchObjectPath("/org/freedesktop/login1"),
			dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
			dbus.WithMatchMember(match.member),
		); err != nil {
			return fmt.Errorf("add match failed: %w", err)
		}
	}

	// watch for property changes (session locked)
	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
		dbus.WithMatchMember("PropertiesChanged"),
	); err != nil {
		return fmt.Errorf("add match for PropertiesChanged failed: %w", err)
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)

	for {
		select {
		case sig := <-c:
			switch sig.Name {
			case "org.freedesktop.login1.Manager.SessionNew":
				if len(sig.Body) >= 2 {
					sessionPath, ok := sig.Body[1].(dbus.ObjectPath)
					if !ok {
						log.Println("SessionNew: failed to get session object path")
						break
					}

					class, err := getSessionClass(conn, sessionPath)
					if err != nil {
						log.Println("SessionNew: failed to get session class:", err)
						break
					}
					if class != "user" {
						break // Ignore non-user sessions
					}

					username, err := getUsernameFromSession(conn, sessionPath)
					if err != nil {
						log.Println("SessionNew: failed to get username:", err)
						break
					}

					log.Println("SessionNew for user", username, "session", sessionPath)
					sm.HandleLogin(username, string(sessionPath))
				}
			case "org.freedesktop.login1.Manager.SessionRemoved":
				if len(sig.Body) >= 2 {
					sessionPath, ok := sig.Body[1].(dbus.ObjectPath)
					if !ok {
						log.Println("SessionNew: failed to get session object path")
						break
					}
					log.Println("SessionRemoved for", sessionPath)
					sm.HandleLogout(string(sessionPath))
				}
			case "org.freedesktop.login1.Manager.PrepareForSleep":
				if len(sig.Body) > 0 {
					sleeping, _ := sig.Body[0].(bool)
					if sleeping {
						log.Println("System is going to sleep")
						sm.HandleSleep()
					} else {
						log.Println("System has woken up")
						sm.HandleWake()
					}
				}

			case "org.freedesktop.DBus.Properties.PropertiesChanged":
				if len(sig.Body) < 3 {
					break
				}
				iface, ok := sig.Body[0].(string)
				if !ok || iface != "org.freedesktop.login1.Session" {
					break
				}
				changedProps, ok := sig.Body[1].(map[string]dbus.Variant)
				if !ok {
					break
				}
				if val, exists := changedProps["LockedHint"]; exists {
					locked, _ := val.Value().(bool)
					// Get the session path from the signal's Path field
					sessionPath := sig.Path
					username, err := getUsernameFromSession(conn, sessionPath)
					if err != nil {
						log.Println("LockedHint: failed to get username:", err)
						break
					}
					if locked {
						sm.HandleLock(username, string(sessionPath))
					} else {
						sm.HandleUnlock(username, string(sessionPath))
					}
				}

			}
		case <-ctx.Done():
			return nil
		}
	}
}

func getUsernameFromSession(conn *dbus.Conn, sessionPath dbus.ObjectPath) (string, error) {
	sessionObj := conn.Object("org.freedesktop.login1", sessionPath)

	var userInfo []interface{}
	err := sessionObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.login1.Session", "User").Store(&userInfo)
	if err != nil || len(userInfo) < 2 {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	userPath, ok := userInfo[1].(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("failed to get user object path")
	}
	userObj := conn.Object("org.freedesktop.login1", userPath)
	var username dbus.Variant
	err = userObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.login1.User", "Name").Store(&username)
	if err != nil {
		return "", fmt.Errorf("failed to get username: %w", err)
	}
	return username.Value().(string), nil
}

func getSessionClass(conn *dbus.Conn, sessionPath dbus.ObjectPath) (string, error) {
	obj := conn.Object("org.freedesktop.login1", sessionPath)
	var class dbus.Variant
	err := obj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.login1.Session", "Class").Store(&class)
	if err != nil {
		return "", err
	}
	// The value is returned as a dbus.Variant
	if v, ok := class.Value().(string); ok {
		return v, nil
	}
	return "", fmt.Errorf("unexpected type for session class")
}
