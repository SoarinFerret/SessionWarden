package arg

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"

	"github.com/SoarinFerret/SessionWarden/internal/ipc"
)

var resumeCmd = &cobra.Command{
	Use:     "resume <username>",
	Aliases: []string{"r"},
	Short:   "Resume session tracking for a user",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		err = obj.Call(ipc.InterfaceName+".ResumeUser", 0, args[0]).Store()
		if err != nil {
			log.Fatal("Failed to call method:", err)
		}

		fmt.Printf("Session tracking resumed for user: %s\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
