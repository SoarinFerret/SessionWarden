package arg

import (
	"fmt"
	"log"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"

	"github.com/SoarinFerret/SessionWarden/internal/ipc"
)

var pauseCmd = &cobra.Command{
	Use:     "pause <username>",
	Aliases: []string{"p", "lock"},
	Short:   "Pause / lock user session until manually resumed",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		err = obj.Call(ipc.InterfaceName+".PauseUser", 0, args[0]).Store()
		if err != nil {
			log.Fatal("Failed to call method:", err)
		}

		fmt.Printf("Session paused for user: %s\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)

}
