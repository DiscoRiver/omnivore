package config

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/discoriver/massh/v2"
	"github.com/discoriver/omnivore/internal/sshconfig"
	"golang.org/x/crypto/ssh"
)

// defaultKeys are tried, if they exist, when no private key is configured.
var defaultKeys = []string{"~/.ssh/id_ed25519", "~/.ssh/id_ecdsa", "~/.ssh/id_rsa"}

// knownHostsFile is checked, and added to with --accept-new.
const knownHostsFile = "~/.ssh/known_hosts"

// Massh returns the massh configuration for running c's command on c.Hosts. Unless
// c.NoSSHConfig is set, each host's address, user, identity files and ProxyJump are taken from
// ~/.ssh/config. Problems with optional auth sources are logged.
func (c Config) Massh(log *slog.Logger) (*massh.Config, error) {
	defaultUser := c.Username
	if defaultUser == "" {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("find current user (use --user): %w", err)
		}
		defaultUser = u.Username
	}

	a := &auth{
		password: c.Password,
		timeout:  c.SSHTimeout,
		keys:     &keyring{passphrase: c.PrivateKeyPassphrase, log: log, signers: map[string]ssh.Signer{}},
		log:      log,
	}
	if c.PrivateKey != "" {
		a.base = a.keys.load(true, c.PrivateKey)
	} else {
		a.base = a.keys.load(false, defaultKeys...)
	}
	var err error
	if a.agent, err = massh.AgentSigners(); err != nil {
		log.Info("not using SSH agent", "err", err)
	}

	switch c.HostKeys {
	case HostKeysInsecure:
		log.Warn("running in INSECURE mode: host keys are not checked")
		a.hostKeys = ssh.InsecureIgnoreHostKey()
	case HostKeysAcceptNew:
		if a.hostKeys, err = massh.KnownHostsAcceptNew(knownHostsFile); err != nil {
			return nil, err
		}
	default:
		if a.hostKeys, err = massh.KnownHosts(knownHostsFile); err != nil {
			return nil, fmt.Errorf("%w (use --accept-new to trust new hosts)", err)
		}
	}

	m := &massh.Config{
		Hosts:       c.Hosts,
		Jobs:        []massh.Job{c.Job()},
		SSHConfig:   a.clientConfig(defaultUser),
		BastionHost: c.BastionHost,
		Workers:     c.Workers,
		SlowTimeout: c.CommandTimeout,
		CancelSlow:  c.KillSlow,
	}

	if c.NoSSHConfig {
		if len(m.SSHConfig.Auth) == 0 {
			return nil, errors.New("no SSH auth available: no usable private key, SSH agent or password")
		}
		return m, nil
	}
	sc, err := sshconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("read ssh config: %w", err)
	}
	m.Resolve = func(host string) (massh.HostOptions, error) {
		name, port := splitHost(host)
		s, err := sc.Lookup(name)
		if err != nil {
			return massh.HostOptions{}, err
		}
		opts := massh.HostOptions{
			Addr:      net.JoinHostPort(cmp.Or(s.HostName, name), cmp.Or(port, s.Port, "22")),
			SSHConfig: a.clientConfig(cmp.Or(c.Username, s.User, defaultUser), s.IdentityFiles...),
		}
		// --bastion overrides ProxyJump, like ssh -J.
		if j := s.ProxyJump; c.BastionHost == "" && j.Host != "" {
			opts.Bastion = net.JoinHostPort(j.Host, cmp.Or(j.Port, "22"))
			opts.BastionSSHConfig = a.clientConfig(cmp.Or(j.User, c.Username, defaultUser), j.IdentityFiles...)
		}
		return opts, nil
	}
	return m, nil
}

// splitHost splits an optional port from host.
func splitHost(host string) (name, port string) {
	if h, p, err := net.SplitHostPort(host); err == nil {
		return h, p
	}
	return strings.Trim(host, "[]"), ""
}

// Job returns the job to run: the command, wrapped in sudo if c.Sudo is set.
//
// With a sudo password, sudo reads it from stdin (-S) with no prompt, and -k makes it always
// read it, so it never reaches the command; the command's own stdin is /dev/null. Without one,
// sudo -n fails at once instead of waiting for a password.
func (c Config) Job() massh.Job {
	if !c.Sudo {
		return massh.Job{Command: c.Command}
	}
	inner := shellQuote("exec </dev/null; " + c.Command)
	if c.SudoPassword == "" {
		return massh.Job{Command: "sudo -n -- sh -c " + inner}
	}
	return massh.Job{
		Command: "sudo -k -S -p '' -- sh -c " + inner,
		Stdin:   []byte(c.SudoPassword + "\n"),
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// auth builds SSH client configs that share keys, the agent and the password.
type auth struct {
	base     []ssh.Signer
	agent    func() ([]ssh.Signer, error)
	password string
	hostKeys ssh.HostKeyCallback
	timeout  time.Duration
	keys     *keyring
	log      *slog.Logger
}

// clientConfig returns a client config for user that also offers the keys in identityFiles.
func (a *auth) clientConfig(user string, identityFiles ...string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User:            user,
		Auth:            a.methods(a.keys.load(false, identityFiles...)),
		HostKeyCallback: a.hostKeys,
		Timeout:         a.timeout,
	}
}

// methods returns public key auth (agent, then extra and base keys) and then password auth.
// All keys are offered by a single method, because the SSH client tries each kind of method only
// once.
func (a *auth) methods(extra []ssh.Signer) []ssh.AuthMethod {
	var methods []ssh.AuthMethod
	files := append(extra[:len(extra):len(extra)], a.base...)
	if len(files) > 0 || a.agent != nil {
		methods = append(methods, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
			if a.agent == nil {
				return files, nil
			}
			fromAgent, err := a.agent()
			if err != nil {
				a.log.Warn("can't list SSH agent keys", "err", err)
			}
			return append(fromAgent, files...), nil
		}))
	}
	if a.password != "" {
		// Many servers (e.g. OpenSSH with PAM) only offer password login as keyboard-interactive.
		methods = append(methods, ssh.Password(a.password), ssh.KeyboardInteractive(passwordChallenge(a.password)))
	}
	return methods
}

// keyring loads private keys once each, and is safe for concurrent use.
type keyring struct {
	passphrase string
	log        *slog.Logger

	mu      sync.Mutex
	signers map[string]ssh.Signer // nil value: tried and unusable
}

// load returns the usable keys among files. Missing files are skipped silently unless required,
// as ssh does for IdentityFile; other problems are logged once per file.
func (k *keyring) load(required bool, files ...string) []ssh.Signer {
	k.mu.Lock()
	defer k.mu.Unlock()
	var out []ssh.Signer
	for _, f := range files {
		s, seen := k.signers[f]
		if !seen {
			var err error
			s, err = massh.PrivateKeySigner(f, k.passphrase)
			switch {
			case err == nil:
				k.log.Info("using private key", "file", f)
			case errors.Is(err, os.ErrNotExist) && !required:
			default:
				k.log.Warn("can't use private key", "file", f, "err", err)
			}
			k.signers[f] = s
		}
		if s != nil {
			out = append(out, s)
		}
	}
	return out
}

// passwordChallenge answers keyboard-interactive prompts that hide their input (password
// prompts) with password, and leaves other prompts blank.
func passwordChallenge(password string) ssh.KeyboardInteractiveChallenge {
	return func(_, _ string, questions []string, echos []bool) ([]string, error) {
		answers := make([]string, len(questions))
		for i := range questions {
			if !echos[i] {
				answers[i] = password
			}
		}
		return answers, nil
	}
}
