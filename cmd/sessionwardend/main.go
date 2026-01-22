package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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
	// initialize the state manager
	stateMgr, err := state.NewManager("state.json")
	if err != nil {
		log.Fatal("Failed to initialize state manager:", err)
	}

	// check for argument to determine config location
	argPath := "/etc/sessionwarden/config.toml"
	if len(os.Args) > 1 {
		argPath = os.Args[1]
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
		if err := serveSessionWarden(ctx, stateMgr, config, true); err != nil {
			log.Println("sessionwarden service error:", err)
		}
	}()

	// Start the user engine (periodic session checker)
	wg.Add(1)
	go func() {
		defer wg.Done()
		userEngine, err := engine.NewEngine(stateMgr, config)
		if err != nil {
			log.Println("Failed to create user engine:", err)
			return
		}
		if err := userEngine.Run(ctx); err != nil {
			log.Println("user engine error:", err)
		}
	}()

	wg.Wait()
	fmt.Println("Shutdown complete")

}

func serveSessionWarden(ctx context.Context, stateMgr *state.Manager, config *config.Config, system bool) error {
	conn := &dbus.Conn{}
	if system {
		var err error
		conn, err = dbus.ConnectSystemBus()
		if err != nil {
			return fmt.Errorf("failed to connect to system bus: %w", err)
		}
	} else {
		var err error
		conn, err = dbus.ConnectSessionBus()
		if err != nil {
			return fmt.Errorf("failed to connect to system bus: %w", err)
		}
	}
	defer conn.Close()

	reply, err := conn.RequestName(ipc.ServiceName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return fmt.Errorf("failed to request name: %w", err)
	}

	sm := &ipc.SessionManager{Manager: stateMgr, Config: config}
	err = conn.Export(sm, dbus.ObjectPath(ipc.ObjectPath), ipc.InterfaceName)
	if err != nil {
		return fmt.Errorf("failed to export interface: %w", err)
	}

	<-ctx.Done()
	return nil
}
