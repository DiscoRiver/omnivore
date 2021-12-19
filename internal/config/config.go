// Package Config provides functionality to read user-required fields, and build a Massh config from which to generate a StreamCycle
package config

import (
	"os"

	"github.com/discoriver/omnivore/internal/log"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	// Config Name and Location
	defaultConfigDir  = "~/.omnivore"
	defaultConfigName = "config"
	defaultConfigType = "yaml"

	// Config keys.
	HostsCommandConfigKey     = "omni.HostsCommand"
	HostsCommandArgsConfigKey = "omni.HostsCommandArgs"

	BastionHostConfigKey            = "omni.BastionHost"
	BastionHostCommandConfigKey     = "omni.BastionHostCommand"
	BastionHostCommandArgsConfigKey = "omni.BastionHostCommandArgs"

	UsernameConfigKey            = "omni.Username"
	UsernameCommandConfigKey     = "omni.UsernameCommand"
	UsernameCommandArgsConfigKey = "omni.UserCommandArgs"

	SSHTimeoutConfigKey     = "omni.SSHTimeout"
	CommandTimeoutConfigKey = "omni.CommandTimeout"

	PrivateKeyLocConfigKey = "omni.PrivateKeyLoc"
	PrivateKeyPassword     = "omni.PrivateKeyPassword"

	ConcurrentWorkerPoolConfigKey = "omni.WorkerPool"

	// Default values where applicable
	SSHTimeoutDefault           = 10
	PrivateKeyLocDefault        = "~/.ssh/id_rsa"
	ConcurrentWorkerPoolDefault = 30

	// Custom config file location
	ConfigFileLoc = ""
)

// OmnivoreConfig contains user-provided values necessary to run the tool.
type OmnivoreConfig struct {
	Hosts              []string
	BastionHost        string
	Command            string
	Username           string
	Password           string
	SSHTimeout         int
	PrivateKeyLoc      string
	PrivateKeyPassword string
}

func SetConfigDefaults() {
	// Defaults
	viper.SetDefault(SSHTimeoutConfigKey, SSHTimeoutDefault)
	viper.SetDefault(PrivateKeyLocConfigKey, PrivateKeyLocDefault)
	viper.SetDefault(ConcurrentWorkerPoolConfigKey, ConcurrentWorkerPoolDefault)
}

// InitConfig reads in a config file, populating Viper with keys used to access values elsewhere in the tool.
func InitConfig() {
	var configHome string

	if ConfigFileLoc != "" {
		// User config file from the flag.
		viper.SetConfigFile(ConfigFileLoc)
	} else {
		// Find log file in home directory (default location)
		var err error
		configHome, err = homedir.Expand(defaultConfigDir)
		if err != nil {
			log.OmniLog.Fatal("Couldn't find user home directory: %s", err)
		}

		viper.AddConfigPath(configHome)
		viper.SetConfigName(defaultConfigName)
		viper.SetConfigType(defaultConfigType)
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.OmniLog.Warn("Config file not found, using defaults.")
		} else {
			// We don't want to use defaults if user is trying to use a custom config, ideally.
			log.OmniLog.Fatal("Config file found, but errored: %s", err)
		}
	}

	// Check config for correct permissions. Required due to the sensitive nature of it's contents.
	file := viper.ConfigFileUsed()
	f, err := os.Stat(file)
	if err == nil && f.Mode().Perm() != 0600 {
		log.OmniLog.Fatal("Config file %s has invalid permissions. Run \"chmod 0600 %s\" to correct.", file, file)
	}
}
