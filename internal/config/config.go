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
	"os/exec"
	"path/filepath"
	"time"

	"github.com/discoriver/omnivore/internal/hosts"
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

// Environment variables for secrets, which are safer than flags (visible to other users).
const (
	// PasswordEnv holds the SSH password when --password isn't given.
	PasswordEnv = "OMNIVORE_PASSWORD"
	// SudoPasswordEnv holds the sudo password for --sudo.
	SudoPasswordEnv = "OMNIVORE_SUDO_PASSWORD"
)

// SSH clients.
const (
	SSHClientNative  = "native"
	SSHClientOpenSSH = "openssh"
	sshClientAuto    = "auto"
)

// lookPath finds the ssh client; tests replace it.
var lookPath = exec.LookPath

// HostKeyMode says how host keys are checked.
type HostKeyMode int

const (
	// HostKeysStrict accepts only hosts in ~/.ssh/known_hosts.
	HostKeysStrict HostKeyMode = iota
	// HostKeysAcceptNew also accepts, and records, hosts not seen before.
	HostKeysAcceptNew
	// HostKeysInsecure doesn't check host keys.
	HostKeysInsecure
)

// Config holds omnivore's settings for one run.
type Config struct {
	// Hosts are the expanded hosts, from --hosts and --hosts-file.
	Hosts       []string
	Command     string
	BastionHost string
	// Username overrides the user from ~/.ssh/config. If empty, that user or else the current
	// user is used.
	Username string
	Password string
	// AskPass prompts for the SSH password before connecting.
	AskPass bool
	// Sudo runs the command with sudo, using SudoPassword (or sudo -n if it is empty).
	Sudo         bool
	SudoPassword string
	// PrivateKey is the private key file to use. If empty, the usual files in ~/.ssh are tried.
	PrivateKey           string
	PrivateKeyPassphrase string
	SSHTimeout           time.Duration
	// CommandTimeout flags hosts as slow when they produce no output for this long. Zero
	// disables it.
	CommandTimeout time.Duration
	// SSHClient is SSHClientOpenSSH to run the system ssh client, or SSHClientNative (the zero
	// value) to use the Go SSH library. Load resolves --ssh=auto to one of them.
	SSHClient string
	// SSHPath is the ssh client found for SSHClientOpenSSH.
	SSHPath string
	// KillSlow stops hosts once they are flagged slow.
	KillSlow    bool
	Workers     int
	HostKeys    HostKeyMode
	NoUI        bool
	NoSSHConfig bool

	// File is the config file that was read, or "" if none was found.
	File string
}

// Flag names.
const (
	hostsFlag       = "hosts"
	hostsFileFlag   = "hosts-file"
	commandFlag     = "command"
	passwordFlag    = "password"
	askPassFlag     = "ask-pass"
	sudoFlag        = "sudo"
	insecureFlag    = "insecure"
	acceptNewFlag   = "accept-new"
	killSlowFlag    = "kill-slow"
	noUIFlag        = "no-ui"
	noSSHConfigFlag = "no-ssh-config"
	sshFlag         = "ssh"
	configFlag      = "config"
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
	fs.StringSlice(hostsFlag, nil, "Hosts to run the command on (comma separated, or repeat the flag). Ranges like web[01-20] are expanded.")
	fs.StringP(hostsFileFlag, "f", "", "File of hosts to run the command on: one or more per line, # starts a comment.")
	fs.StringP(commandFlag, "c", "", "Command to run on every host.")
	fs.StringP("bastion", "b", "", "Bastion (jump) host to connect through.")
	fs.StringP("user", "u", "", "SSH user name (default: the current user).")
	fs.StringP(passwordFlag, "p", "", "SSH password. Prefer --ask-pass or the "+PasswordEnv+" environment variable, since flags are visible to other users.")
	fs.BoolP(askPassFlag, "P", false, "Prompt for the SSH password.")
	fs.Bool(sudoFlag, false, "Run the command with sudo. Prompts once for the sudo password (or set "+SudoPasswordEnv+"); leave it empty for passwordless sudo.")
	fs.StringP("key", "k", "", "Private key file (default: ~/.ssh/id_ed25519, id_ecdsa and id_rsa, if present).")
	fs.StringP("keypass", "s", "", "Private key passphrase.")
	fs.IntP("ssh-timeout", "t", int(DefaultSSHTimeout/time.Second), "SSH connection timeout, in seconds.")
	fs.IntP("command-timeout", "d", 0, "Flag hosts as slow after this many seconds without output (0 to disable).")
	fs.IntP("workers", "w", DefaultWorkers, "Number of hosts to run on at once.")
	fs.Bool(killSlowFlag, false, "Stop hosts that are flagged slow (needs --command-timeout).")
	fs.Bool(acceptNewFlag, false, "Accept and remember host keys of hosts not in ~/.ssh/known_hosts yet. Changed keys are still rejected.")
	fs.BoolP(insecureFlag, "x", false, "Don't check host keys at all. Insecure!")
	fs.Bool(noSSHConfigFlag, false, "Don't read ~/.ssh/config.")
	fs.String(sshFlag, sshClientAuto, "SSH client: \"openssh\" runs your ssh (all of ~/.ssh/config, Kerberos, security keys), \"native\" uses the built-in client, \"auto\" uses ssh if it is installed.")
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

	patterns := mustStringSlice(fs, hostsFlag)
	if path := mustString(fs, hostsFileFlag); path != "" {
		fromFile, err := readHostsFile(path)
		if err != nil {
			return Config{}, err
		}
		patterns = append(patterns, fromFile...)
	}
	hostList, err := hosts.Expand(patterns...)
	if err != nil {
		return Config{}, err
	}

	hostKeys := HostKeysStrict
	switch {
	case mustBool(fs, insecureFlag):
		hostKeys = HostKeysInsecure
	case mustBool(fs, acceptNewFlag):
		hostKeys = HostKeysAcceptNew
	}

	c := Config{
		Hosts:                hostList,
		Command:              mustString(fs, commandFlag),
		Password:             mustString(fs, passwordFlag),
		AskPass:              mustBool(fs, askPassFlag),
		Sudo:                 mustBool(fs, sudoFlag),
		SudoPassword:         os.Getenv(SudoPasswordEnv),
		KillSlow:             mustBool(fs, killSlowFlag),
		HostKeys:             hostKeys,
		NoUI:                 mustBool(fs, noUIFlag),
		NoSSHConfig:          mustBool(fs, noSSHConfigFlag),
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
	if err := c.resolveSSHClient(mustString(fs, sshFlag)); err != nil {
		return Config{}, err
	}
	return c, c.validate()
}

// resolveSSHClient turns the --ssh flag into SSHClient and SSHPath.
func (c *Config) resolveSSHClient(flag string) error {
	switch flag {
	case SSHClientNative:
		c.SSHClient = SSHClientNative
	case SSHClientOpenSSH, sshClientAuto:
		path, err := lookPath("ssh")
		switch {
		case err == nil:
			c.SSHClient, c.SSHPath = SSHClientOpenSSH, path
		case flag == SSHClientOpenSSH:
			return fmt.Errorf("--ssh=openssh: %w", err)
		default:
			c.SSHClient = SSHClientNative
		}
	default:
		return fmt.Errorf("--ssh must be auto, openssh or native, not %q", flag)
	}
	return nil
}

func readHostsFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("hosts file: %w", err)
	}
	defer f.Close()
	list, err := hosts.Read(f)
	if err != nil {
		return nil, fmt.Errorf("hosts file %s: %w", path, err)
	}
	return list, nil
}

func (c Config) validate() error {
	var errs []error
	if len(c.Hosts) == 0 {
		errs = append(errs, errors.New("no hosts given (use --hosts or --hosts-file)"))
	}
	if c.Command == "" {
		errs = append(errs, errors.New("no command given (use --command)"))
	}
	if c.KillSlow && c.CommandTimeout == 0 {
		errs = append(errs, errors.New("--kill-slow needs --command-timeout"))
	}
	if c.Workers < 1 {
		errs = append(errs, fmt.Errorf("workers must be at least 1, got %d", c.Workers))
	}
	if c.SSHTimeout < 0 || c.CommandTimeout < 0 {
		errs = append(errs, errors.New("timeouts can't be negative"))
	}
	return errors.Join(errs...)
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
