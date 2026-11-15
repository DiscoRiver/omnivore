package config

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/discoriver/massh/v2"
)

// masshOpenSSH returns the massh configuration for running c's command with the system ssh
// client. ssh_config applies in full, but omnivore's host key mode always wins over it.
func (c Config) masshOpenSSH(log *slog.Logger) (*massh.Config, error) {
	o := &massh.OpenSSH{
		Path:           c.SSHPath,
		User:           c.Username,
		ConnectTimeout: c.SSHTimeout,
		Env:            scrubEnv(os.Environ()),
		Password:       c.Password,
		Passphrase:     c.PrivateKeyPassphrase,
	}
	if c.PrivateKey != "" {
		o.IdentityFiles = []string{c.PrivateKey}
	}
	switch c.HostKeys {
	case HostKeysInsecure:
		log.Warn("running in INSECURE mode: host keys are not checked")
		o.HostKeyChecking = "no"
		o.Options = append(o.Options, "UserKnownHostsFile=/dev/null")
	case HostKeysAcceptNew:
		o.HostKeyChecking = "accept-new"
	default:
		o.HostKeyChecking = "yes"
	}
	if c.NoSSHConfig {
		o.ConfigFile = os.DevNull
	}
	if o.Password != "" || o.Passphrase != "" {
		// ssh asks this program for the password; main hands it over (massh.HandleAskpass).
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("find omnivore's own path for ssh's password prompt: %w", err)
		}
		o.AskpassProgram = exe
	}
	log.Info("using the system ssh client", "path", o.Path)
	return &massh.Config{
		Hosts:       c.Hosts,
		Jobs:        []massh.Job{c.Job()},
		OpenSSH:     o,
		BastionHost: c.BastionHost,
		Workers:     c.Workers,
		SlowTimeout: c.CommandTimeout,
		CancelSlow:  c.KillSlow,
	}, nil
}

// secretEnv are environment variables that must not reach ssh, which passes its environment
// to ProxyCommand and can send it to hosts with SendEnv.
var secretEnv = []string{PasswordEnv, SudoPasswordEnv}

func scrubEnv(env []string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.Contains(secretEnv, name)
	})
}
