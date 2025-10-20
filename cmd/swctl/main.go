package main

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
)

const (
	objectPath    = "/io/github/soarinferret/sessionwarden"
	interfaceName = "io.github.soarinferret.sessionwarden.Manager"
	serviceName   = "io.github.soarinferret.sessionwarden"
)

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal("Failed to connect to session bus:", err)
	}
	defer conn.Close()

	obj := conn.Object(serviceName, dbus.ObjectPath(objectPath))

	var result string
	err = obj.Call(interfaceName+".GetSessionStatus", 0).Store(&result)
	if err != nil {
		log.Fatal("Failed to call method:", err)
	}

	fmt.Println("Session Status:", result)
}
