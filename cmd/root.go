package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "omnivore",
		Short: "Omniore devours all SSH output, and provides intelligent grouping.",
		Long:  `An intelligent distributed SSH tool, providing advanced grouping to identify anomalies and unexpected output.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP()
	// Flags
}
