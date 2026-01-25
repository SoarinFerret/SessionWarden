package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"sync"
	"syscall"

	"github.com/SoarinFerret/SessionWarden/internal/config"
	"github.com/SoarinFerret/SessionWarden/internal/engine"
	"github.com/SoarinFerret/SessionWarden/internal/ipc"
	"github.com/SoarinFerret/SessionWarden/internal/loginctl"
	"github.com/SoarinFerret/SessionWarden/internal/state"
	"github.com/godbus/dbus/v5"
)

func main() {
	// Parse command line flags
	userMode := flag.Bool("user", false, "Run in user mode (listen for notifications from system daemon)")
	flag.Parse()

	// If user mode, run notification listener only
	if *userMode {
		if err := runUserMode(); err != nil {
			log.Fatal("User mode error:", err)
		}
		return
	}

	// Otherwise, run as system daemon
	runSystemDaemon()
}

func runSystemDaemon() {
	// initialize the state manager
	stateMgr, err := state.NewManager("state.json")
	if err != nil {
		log.Fatal("Failed to initialize state manager:", err)
	}

	// check for argument to determine config location
	argPath := "/etc/sessionwarden/config.toml"
	if flag.NArg() > 0 {
		argPath = flag.Arg(0)
	}
	log.Println("Using config file at:", argPath)
	// load config
	config, err := config.LoadConfigFromFile(argPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	// Create the user engine (needs to be accessible by IPC)
	userEngine, err := engine.NewEngine(stateMgr, config)
	if err != nil {
		log.Fatal("Failed to create user engine:", err)
	}

	// Create SessionManager for IPC and signal emission
	sm := &ipc.SessionManager{Manager: stateMgr, Config: config, Engine: userEngine}

	// Set the notification emitter on the engine
	userEngine.SetNotificationEmitter(sm)

	var wg sync.WaitGroup

	// Start the loginctl listener (system D-Bus)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Monitoring dbus for session changes...")
		if err := loginctl.Watch(ctx, stateMgr); err != nil {
			log.Println("logind watcher error:", err)
		}
	}()

	// Start your own DBus service (sessionwarden)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Opening system D-Bus service...")
		if err := serveSessionWarden(ctx, sm); err != nil {
			log.Println("sessionwarden service error:", err)
		}
	}()

	// Start the user engine (periodic session checker)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := userEngine.Run(ctx); err != nil {
			log.Println("user engine error:", err)
		}
	}()

	wg.Wait()
	fmt.Println("Shutdown complete")
}

// runUserMode runs the notification listener for user sessions
func runUserMode() error {
	log.Println("Starting sessionwardend in user mode (notification listener)...")

	// Get current username for filtering
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	username := currentUser.Username
	log.Printf("Running in user mode for user: %s", username)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	// Connect to system bus to receive signals from root daemon
	systemConn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer systemConn.Close()

	// Subscribe to notification signals from system daemon
	if err := systemConn.AddMatchSignal(
		dbus.WithMatchObjectPath(ipc.ObjectPath),
		dbus.WithMatchInterface(ipc.InterfaceName),
		dbus.WithMatchMember("NotificationSignal"),
	); err != nil {
		return fmt.Errorf("failed to add match signal: %w", err)
	}

	// Also connect to session bus for sending desktop notifications
	sessionConn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %w", err)
	}
	defer sessionConn.Close()

	signalChan := make(chan *dbus.Signal, 10)
	systemConn.Signal(signalChan)

	log.Println("Listening for notification signals from system daemon...")

	for {
		select {
		case <-ctx.Done():
			log.Println("User mode shutting down...")
			return nil
		case sig := <-signalChan:
			if sig.Name == ipc.InterfaceName+".NotificationSignal" {
				handleNotificationSignal(sessionConn, sig, username)
			}
		}
	}
}

// handleNotificationSignal processes a notification signal and sends desktop notification
// It filters notifications to only show those meant for the current user
func handleNotificationSignal(conn *dbus.Conn, sig *dbus.Signal, currentUsername string) {
	if len(sig.Body) < 3 {
		log.Printf("Invalid notification signal: expected 3 arguments, got %d", len(sig.Body))
		return
	}

	targetUsername, ok := sig.Body[0].(string)
	if !ok {
		log.Printf("Invalid notification signal: username is not a string")
		return
	}

	// Filter: only process notifications for the current user
	if targetUsername != currentUsername {
		log.Printf("Ignoring notification for user %s (current user: %s)", targetUsername, currentUsername)
		return
	}

	title, ok := sig.Body[1].(string)
	if !ok {
		log.Printf("Invalid notification signal: title is not a string")
		return
	}

	message, ok := sig.Body[2].(string)
	if !ok {
		log.Printf("Invalid notification signal: message is not a string")
		return
	}

	log.Printf("Received notification signal for %s: %s - %s", currentUsername, title, message)

	// Send desktop notification via org.freedesktop.Notifications
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		"SessionWarden",        // app_name
		uint32(0),              // replaces_id
		"dialog-warning",       // app_icon
		title,                  // summary
		message,                // body
		[]string{},             // actions
		map[string]dbus.Variant{ // hints
			"urgency": dbus.MakeVariant(byte(1)), // normal urgency
		},
		int32(10000), // expire_timeout (10 seconds)
	)

	if call.Err != nil {
		log.Printf("Failed to send desktop notification: %v", call.Err)
	} else {
		log.Printf("Successfully sent desktop notification: %s", title)
	}
}

func serveSessionWarden(ctx context.Context, sm *ipc.SessionManager) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %w", err)
	}
	defer conn.Close()

	reply, err := conn.RequestName(ipc.ServiceName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("failed to request name: %w", err)
	}

	sm.SetConnection(conn)

	err = conn.Export(sm, dbus.ObjectPath(ipc.ObjectPath), ipc.InterfaceName)
	if err != nil {
		return fmt.Errorf("failed to export interface: %w", err)
	}

	<-ctx.Done()
	return nil
}
