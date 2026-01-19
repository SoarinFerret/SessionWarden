package arg

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"

	"github.com/SoarinFerret/SessionWarden/internal/ipc"
)

var (
	extraTime    int
	allowedHours string
	expires      string
	reason       string
)

var overrideCmd = &cobra.Command{
	Use:   "override",
	Short: "Manage temporary policy overrides",
	Long:  `Add, list, or remove temporary policy overrides for users`,
}

var overrideAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Add a temporary policy override for a user",
	Long: `Add a temporary override to grant extra time or modify allowed hours.
Examples:
  swctl override add alice --extra-time 60 --reason "Working on urgent project"
  swctl override add bob --allowed-hours "18:00-22:00" --expires "2026-01-20T23:59:59"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]

		if extraTime == 0 && allowedHours == "" {
			log.Fatal("Must specify either --extra-time or --allowed-hours")
		}

		var expiresAt time.Time
		var err error
		if expires != "" {
			expiresAt, err = time.Parse(time.RFC3339, expires)
			if err != nil {
				log.Fatalf("Invalid expires format (use RFC3339): %v", err)
			}
		} else {
			// Default: expire at end of day
			expiresAt = time.Now().Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
		}

		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		err = obj.Call(ipc.InterfaceName+".AddOverride", 0,
			username, reason, extraTime, allowedHours, expiresAt.Unix()).Store()
		if err != nil {
			log.Fatal("Failed to add override:", err)
		}

		fmt.Printf("Override added for user: %s\n", username)
		if extraTime > 0 {
			fmt.Printf("  Extra time: %d minutes\n", extraTime)
		}
		if allowedHours != "" {
			fmt.Printf("  Allowed hours: %s\n", allowedHours)
		}
		fmt.Printf("  Expires: %s\n", expiresAt.Format(time.RFC3339))
		if reason != "" {
			fmt.Printf("  Reason: %s\n", reason)
		}
	},
}

var overrideListCmd = &cobra.Command{
	Use:   "list [username]",
	Short: "List active overrides",
	Long:  `List all active overrides, or overrides for a specific user if username is provided`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := ""
		if len(args) > 0 {
			username = args[0]
		}

		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		var jsonResult string
		err = obj.Call(ipc.InterfaceName+".ListOverrides", 0, username).Store(&jsonResult)
		if err != nil {
			log.Fatal("Failed to list overrides:", err)
		}

		var overrides map[string][]map[string]interface{}
		err = json.Unmarshal([]byte(jsonResult), &overrides)
		if err != nil {
			log.Fatal("Failed to parse response:", err)
		}

		if len(overrides) == 0 {
			fmt.Println("No active overrides")
			return
		}

		for user, userOverrides := range overrides {
			if len(userOverrides) == 0 {
				continue
			}
			fmt.Printf("\nUser: %s\n", user)
			for idx, override := range userOverrides {
				fmt.Printf("  [%d] ", idx)
				if reason, ok := override["reason"].(string); ok && reason != "" {
					fmt.Printf("Reason: %s, ", reason)
				}
				if extraTime, ok := override["extra_hours"].(float64); ok && extraTime > 0 {
					fmt.Printf("Extra time: %d min, ", int(extraTime))
				}
				if allowedHours, ok := override["allowed_hours"].(string); ok && allowedHours != "" {
					fmt.Printf("Allowed hours: %s, ", allowedHours)
				}
				if expiresAt, ok := override["expires_at"].(string); ok {
					fmt.Printf("Expires: %s", expiresAt)
				}
				fmt.Println()
			}
		}
	},
}

var overrideRemoveCmd = &cobra.Command{
	Use:   "remove <username> <index>",
	Short: "Remove an override by index",
	Long:  `Remove an override for a user by its index (use 'override list' to see indices)`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		index, err := strconv.Atoi(args[1])
		if err != nil {
			log.Fatal("Invalid index (must be a number):", err)
		}

		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		err = obj.Call(ipc.InterfaceName+".RemoveOverride", 0, username, index).Store()
		if err != nil {
			log.Fatal("Failed to remove override:", err)
		}

		fmt.Printf("Override %d removed for user: %s\n", index, username)
	},
}

func init() {
	overrideAddCmd.Flags().IntVarP(&extraTime, "extra-time", "t", 0, "Extra time in minutes")
	overrideAddCmd.Flags().StringVarP(&allowedHours, "allowed-hours", "a", "", "Override allowed hours (format: HH:MM-HH:MM)")
	overrideAddCmd.Flags().StringVarP(&expires, "expires", "e", "", "Expiration time (RFC3339 format)")
	overrideAddCmd.Flags().StringVarP(&reason, "reason", "r", "", "Reason for override")

	overrideCmd.AddCommand(overrideAddCmd)
	overrideCmd.AddCommand(overrideListCmd)
	overrideCmd.AddCommand(overrideRemoveCmd)
	rootCmd.AddCommand(overrideCmd)
}
