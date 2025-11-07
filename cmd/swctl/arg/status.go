package arg

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"

	"github.com/SoarinFerret/SessionWarden/internal/ipc"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if SessionWarden is running and get session status",
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to session bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		var result string
		err = obj.Call(ipc.InterfaceName+".GetStatus", 0).Store(&result)
		if err != nil {
			log.Fatal("Failed to call method:", err)
		}

		fmt.Println("SessionWarden Status:", result)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
