package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "omnivore",
	Short: "Omniore consumes all SSH output, and provides intelligent grouping.",
	Long: `An intelligent distributed SSH tool, providing advanced grouping to identify anomalies and unexpected output.`,
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
