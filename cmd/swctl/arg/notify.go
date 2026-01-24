package arg

import (
	"fmt"
	"log"

	"github.com/SoarinFerret/SessionWarden/internal/ipc"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(notifyCmd)
}

var notifyCmd = &cobra.Command{
	Use:   "notify <username> <message>",
	Short: "Send a notification to a user",
	Long:  "Send a custom notification message to a user's active session",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		message := args[1]

		// If more than 2 args, join them as the message
		if len(args) > 2 {
			message = ""
			for i := 1; i < len(args); i++ {
				if i > 1 {
					message += " "
				}
				message += args[i]
			}
		}

		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))
		call := obj.Call(ipc.InterfaceName+".SendNotification", 0, username, message)
		if call.Err != nil {
			log.Fatal("Failed to send notification:", call.Err)
		}

		fmt.Printf("Notification sent to %s: %s\n", username, message)
	},
}
