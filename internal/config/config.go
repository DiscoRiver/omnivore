// Package config resolves omnivore's settings from command-line flags, the config file and
// defaults, and turns them into a massh configuration.
//
// Settings that can be stored in the config file are read in order of precedence from the flag,
// the file (~/.omnivore/config.yaml unless --config is given), and the default. The file must
// only be readable by its owner, since it can hold secrets.
package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config file keys.
const (
	BastionHostKey        = "omni.BastionHost"
	UsernameKey           = "omni.Username"
	PrivateKeyKey         = "omni.PrivateKeyLoc"
	PrivateKeyPasswordKey = "omni.PrivateKeyPassword"
	SSHTimeoutKey         = "omni.SSHTimeout"
	CommandTimeoutKey     = "omni.CommandTimeout"
	WorkersKey            = "omni.WorkerPool"
)

// Defaults.
const (
	DefaultSSHTimeout = 10 * time.Second
	DefaultWorkers    = 30
)

// PasswordEnv is the environment variable read for the SSH password when --password isn't given.
const PasswordEnv = "OMNIVORE_PASSWORD"

// Config holds omnivore's settings for one run.
type Config struct {
	Hosts       []string
	Command     string
	BastionHost string
	Username    string
	Password    string
	// PrivateKey is the private key file to use. If empty, the usual files in ~/.ssh are tried.
	PrivateKey           string
	PrivateKeyPassphrase string
	SSHTimeout           time.Duration
	// CommandTimeout flags hosts as slow when they produce no output for this long. Zero
	// disables it.
	CommandTimeout time.Duration
	Workers        int
	Insecure       bool
	NoUI           bool

	// File is the config file that was read, or "" if none was found.
	File string
}

// Flag names.
const (
	hostsFlag    = "hosts"
	commandFlag  = "command"
	passwordFlag = "password"
	insecureFlag = "insecure"
	noUIFlag     = "no-ui"
	configFlag   = "config"
)

// fileFlags maps flags to the config file keys they override.
var fileFlags = map[string]string{
	"bastion":         BastionHostKey,
	"user":            UsernameKey,
	"key":             PrivateKeyKey,
	"keypass":         PrivateKeyPasswordKey,
	"ssh-timeout":     SSHTimeoutKey,
	"command-timeout": CommandTimeoutKey,
	"workers":         WorkersKey,
}

// RegisterFlags defines omnivore's flags on fs.
func RegisterFlags(fs *pflag.FlagSet) {
	fs.StringSlice(hostsFlag, nil, "Hosts to run the command on (comma separated, or repeat the flag).")
	fs.StringP(commandFlag, "c", "", "Command to run on every host.")
	fs.StringP("bastion", "b", "", "Bastion (jump) host to connect through.")
	fs.StringP("user", "u", "", "SSH user name (default: the current user).")
	fs.StringP(passwordFlag, "p", "", "SSH password. Prefer the "+PasswordEnv+" environment variable, since flags are visible to other users.")
	fs.StringP("key", "k", "", "Private key file (default: ~/.ssh/id_ed25519, id_ecdsa and id_rsa, if present).")
	fs.StringP("keypass", "s", "", "Private key passphrase.")
	fs.IntP("ssh-timeout", "t", int(DefaultSSHTimeout/time.Second), "SSH connection timeout, in seconds.")
	fs.IntP("command-timeout", "d", 0, "Flag hosts as slow after this many seconds without output (0 to disable).")
	fs.IntP("workers", "w", DefaultWorkers, "Number of hosts to run on at once.")
	fs.BoolP(insecureFlag, "x", false, "Don't check host keys against ~/.ssh/known_hosts. Insecure!")
	fs.Bool(noUIFlag, false, "Print grouped output when every host has finished, instead of starting the interactive UI.")
	fs.String(configFlag, "", "Config file (default: ~/.omnivore/config.yaml).")
}

// Load resolves the settings from the flags in fs, which must have been defined by
// RegisterFlags and parsed.
func Load(fs *pflag.FlagSet) (Config, error) {
	v := viper.New()
	v.SetDefault(SSHTimeoutKey, int(DefaultSSHTimeout/time.Second))
	v.SetDefault(WorkersKey, DefaultWorkers)
	for flag, key := range fileFlags {
		if err := v.BindPFlag(key, fs.Lookup(flag)); err != nil {
			return Config{}, err
		}
	}

	file, err := readConfigFile(v, mustString(fs, configFlag))
	if err != nil {
		return Config{}, err
	}

	c := Config{
		Hosts:                cleanHosts(mustStringSlice(fs, hostsFlag)),
		Command:              mustString(fs, commandFlag),
		Password:             mustString(fs, passwordFlag),
		Insecure:             mustBool(fs, insecureFlag),
		NoUI:                 mustBool(fs, noUIFlag),
		BastionHost:          v.GetString(BastionHostKey),
		Username:             v.GetString(UsernameKey),
		PrivateKey:           v.GetString(PrivateKeyKey),
		PrivateKeyPassphrase: v.GetString(PrivateKeyPasswordKey),
		SSHTimeout:           time.Duration(v.GetInt(SSHTimeoutKey)) * time.Second,
		CommandTimeout:       time.Duration(v.GetInt(CommandTimeoutKey)) * time.Second,
		Workers:              v.GetInt(WorkersKey),
		File:                 file,
	}
	if c.Password == "" {
		c.Password = os.Getenv(PasswordEnv)
	}
	if c.Username == "" {
		if u, err := user.Current(); err == nil {
			c.Username = u.Username
		}
	}
	return c, c.validate()
}

func (c Config) validate() error {
	var errs []error
	if len(c.Hosts) == 0 {
		errs = append(errs, errors.New("no hosts given (use --hosts)"))
	}
	if c.Command == "" {
		errs = append(errs, errors.New("no command given (use --command)"))
	}
	if c.Username == "" {
		errs = append(errs, errors.New("no user name given (use --user)"))
	}
	if c.Workers < 1 {
		errs = append(errs, fmt.Errorf("workers must be at least 1, got %d", c.Workers))
	}
	if c.SSHTimeout < 0 || c.CommandTimeout < 0 {
		errs = append(errs, errors.New("timeouts can't be negative"))
	}
	return errors.Join(errs...)
}

// cleanHosts trims spaces and drops empty entries, so that "a, b,,c" means a, b and c.
func cleanHosts(hosts []string) []string {
	var out []string
	for _, h := range hosts {
		if h = strings.TrimSpace(h); h != "" {
			out = append(out, h)
		}
	}
	return out
}

// readConfigFile reads the config file into v. If path is empty, the default file is read if it
// exists. It returns the file used.
func readConfigFile(v *viper.Viper, path string) (string, error) {
	if path != "" {
		v.SetConfigFile(path)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory: %w", err)
		}
		v.AddConfigPath(filepath.Join(home, ".omnivore"))
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return "", nil
		}
		return "", fmt.Errorf("read config file: %w", err)
	}

	file := v.ConfigFileUsed()
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return "", fmt.Errorf("config file %s can be read by other users (mode %04o); run \"chmod 600 %s\"", file, perm, file)
	}
	return file, nil
}

// The flags are defined by RegisterFlags, so lookups can only fail on a programming error.

func mustString(fs *pflag.FlagSet, name string) string {
	s, err := fs.GetString(name)
	if err != nil {
		panic(err)
	}
	return s
}

func mustStringSlice(fs *pflag.FlagSet, name string) []string {
	s, err := fs.GetStringSlice(name)
	if err != nil {
		panic(err)
	}
	return s
}

func mustBool(fs *pflag.FlagSet, name string) bool {
	b, err := fs.GetBool(name)
	if err != nil {
		panic(err)
	}
	return b
}
