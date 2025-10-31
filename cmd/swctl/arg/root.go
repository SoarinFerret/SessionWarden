package arg

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "swctl",
	Short: "swctl is the command line tool for SessionWarden",
	Long: `swctl allows you to interact with the SessionWarden service via D-Bus.
			You can use it to query session status, manage sessions, and more.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
