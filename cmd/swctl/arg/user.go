package arg

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"

	"github.com/SoarinFerret/SessionWarden/internal/ipc"
)

var userCmd = &cobra.Command{
	Use:   "user <username>",
	Short: "Show detailed status for a user",
	Long:  `Display detailed session information, time usage, and active overrides for a user`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]

		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			log.Fatal("Failed to connect to system bus:", err)
		}
		defer conn.Close()

		obj := conn.Object(ipc.ServiceName, dbus.ObjectPath(ipc.ObjectPath))

		var jsonResult string
		err = obj.Call(ipc.InterfaceName+".GetUserStatus", 0, username).Store(&jsonResult)
		if err != nil {
			log.Fatal("Failed to get user status:", err)
		}

		var userData map[string]interface{}
		err = json.Unmarshal([]byte(jsonResult), &userData)
		if err != nil {
			log.Fatal("Failed to parse response:", err)
		}

		fmt.Printf("User: %s\n", username)
		fmt.Println("=" + repeat("=", len(username)+5))

		// Paused status
		if paused, ok := userData["paused"].(bool); ok && paused {
			fmt.Println("Status: PAUSED")
		} else {
			fmt.Println("Status: Active")
		}

		// Time usage
		if timeUsed, ok := userData["time_used_seconds"].(float64); ok {
			duration := time.Duration(int64(timeUsed)) * time.Second
			fmt.Printf("Time used today: %s\n", formatDuration(duration))
		}

		// Active sessions
		if sessions, ok := userData["sessions"].([]interface{}); ok && len(sessions) > 0 {
			fmt.Printf("\nActive Sessions (%d):\n", countActiveSessions(sessions))
			for _, s := range sessions {
				sessionMap, ok := s.(map[string]interface{})
				if !ok {
					continue
				}

				// Check if session is active (no end time)
				if endTimeStr, ok := sessionMap["end"].(string); ok && endTimeStr != "" {
					if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil && !endTime.IsZero() {
						continue // Skip ended sessions
					}
				}

				if sessionID, ok := sessionMap["session_id"].(string); ok {
					fmt.Printf("  Session: %s\n", sessionID)
				}
				if startStr, ok := sessionMap["start"].(string); ok {
					if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
						fmt.Printf("    Started: %s (%s ago)\n",
							startTime.Format("15:04:05"),
							time.Since(startTime).Round(time.Second))
					}
				}

				// Show segment info
				if segments, ok := sessionMap["segments"].([]interface{}); ok {
					activeSegment := false
					var totalDuration time.Duration
					for _, seg := range segments {
						segMap, ok := seg.(map[string]interface{})
						if !ok {
							continue
						}
						startStr, _ := segMap["start"].(string)
						endStr, _ := segMap["stop"].(string)

						if startStr != "" {
							start, err := time.Parse(time.RFC3339, startStr)
							if err != nil {
								continue
							}

							var end time.Time
							if endStr != "" {
								end, err = time.Parse(time.RFC3339, endStr)
								if err != nil {
									continue
								}
							} else {
								// Active segment
								end = time.Now()
								activeSegment = true
							}

							totalDuration += end.Sub(start)
						}
					}
					fmt.Printf("    Active time: %s\n", formatDuration(totalDuration))
					if activeSegment {
						fmt.Printf("    Status: Currently active\n")
					} else {
						fmt.Printf("    Status: Idle\n")
					}
				}
			}
		} else {
			fmt.Println("\nNo active sessions")
		}

		// Overrides
		if overrides, ok := userData["exceptions"].([]interface{}); ok && len(overrides) > 0 {
			fmt.Printf("\nActive Overrides (%d):\n", len(overrides))
			for idx, o := range overrides {
				overrideMap, ok := o.(map[string]interface{})
				if !ok {
					continue
				}

				fmt.Printf("  [%d] ", idx)
				if reason, ok := overrideMap["reason"].(string); ok && reason != "" {
					fmt.Printf("Reason: %s\n      ", reason)
				}
				if extraTime, ok := overrideMap["extra_hours"].(float64); ok && extraTime > 0 {
					fmt.Printf("Extra time: %d min, ", int(extraTime))
				}
				if allowedHours, ok := overrideMap["allowed_hours"].(string); ok && allowedHours != "" {
					fmt.Printf("Allowed hours: %s, ", allowedHours)
				}
				if expiresAt, ok := overrideMap["expires_at"].(string); ok {
					if exp, err := time.Parse(time.RFC3339, expiresAt); err == nil {
						fmt.Printf("Expires: %s", exp.Format("2006-01-02 15:04"))
					}
				}
				fmt.Println()
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(userCmd)
}

func repeat(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func countActiveSessions(sessions []interface{}) int {
	count := 0
	for _, s := range sessions {
		sessionMap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if session is active (no end time or zero end time)
		if endTimeStr, ok := sessionMap["end"].(string); ok && endTimeStr != "" {
			if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil && !endTime.IsZero() {
				continue // Skip ended sessions
			}
		}
		count++
	}
	return count
}
