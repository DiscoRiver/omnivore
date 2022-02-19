package cmd

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/config"
	"github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/store"
	"github.com/discoriver/omnivore/omnivore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	omniFlags = omnivore.OmniCommandFlags{}

	rootCmd = &cobra.Command{
		Use:   "omnivore",
		Short: "Omniore devours all SSH output, and provides intelligent grouping.",
		Long:  `An intelligent distributed SSH tool, providing advanced grouping to identify anomalies and unexpected output.`,
		Run: func(cmd *cobra.Command, args []string) {
			omnivore.Run(&omniFlags)
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	store.NewStorageSession()
	logFile, err := os.OpenFile(fmt.Sprintf("%s/%s", store.Session.SessionDir, "log.txt"), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	log.OmniLog = &log.OmniLogger{FileOutput: logFile}

	cobra.OnInitialize(log.OmniLog.Init, config.InitConfig)

	// Flags
	rootCmd.Flags().StringSliceVar(&omniFlags.Hosts, "hosts", nil, "Sets hosts to target in Omniore command.")
	rootCmd.Flags().StringVarP(&omniFlags.BastionHost, "bastion", "b", "", "Set bastion host.")
	rootCmd.Flags().StringVarP(&omniFlags.Username, "user", "u", "", "Set username for SSH.")
	rootCmd.Flags().StringVarP(&omniFlags.Password, "password", "p", "", "Set password for SSH.")
	rootCmd.Flags().StringVarP(&omniFlags.PrivateKeyLocation, "key", "k", "", "Private key location.")
	rootCmd.Flags().StringVarP(&omniFlags.PrivateKeyPassword, "keypass", "s", "", "Private key password.")
	rootCmd.Flags().StringVarP(&omniFlags.Command, "command", "c", "", "SSH command to execute.")
	rootCmd.Flags().IntVarP(&omniFlags.SSHTimeout, "ssh-timeout", "t", 0, "SSH connection timeout.")
	rootCmd.Flags().IntVarP(&omniFlags.CommandTimeout, "command-timeout", "d", 0, "Remote command inactivity timeout.")
	rootCmd.Flags().BoolVarP(&omniFlags.Insecure, "insecure", "x", false, "Ignore host key callback and run in insecure mode.")

	// Persistent Flags
	rootCmd.PersistentFlags().StringVar(&config.ConfigFileLoc, "config", "", "Config file to use with Omnivore.")

	// Config file mapping
	viper.BindPFlag(config.BastionHostConfigKey, rootCmd.Flags().Lookup("bastion"))
	viper.BindPFlag(config.UsernameConfigKey, rootCmd.Flags().Lookup("user"))
	viper.BindPFlag(config.PrivateKeyLocConfigKey, rootCmd.Flags().Lookup("key"))
	viper.BindPFlag(config.PrivateKeyPassword, rootCmd.Flags().Lookup("keypass"))
	viper.BindPFlag(config.SSHTimeoutConfigKey, rootCmd.Flags().Lookup("ssh-timeout"))
	viper.BindPFlag(config.CommandTimeoutConfigKey, rootCmd.Flags().Lookup("command-timeout"))

	// Set defaults in viper
	config.SetConfigDefaults()

	// Required flags
	rootCmd.MarkFlagRequired("hosts")
	rootCmd.MarkFlagRequired("command")
}
