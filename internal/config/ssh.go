package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/discoriver/massh/v2"
	"golang.org/x/crypto/ssh"
)

// defaultKeys are tried, if they exist, when no private key is configured.
var defaultKeys = []string{"~/.ssh/id_ed25519", "~/.ssh/id_ecdsa", "~/.ssh/id_rsa"}

// Massh returns the massh configuration for running c.Command on c.Hosts. Problems with optional
// auth sources are logged; it fails if no auth method is usable or host keys can't be checked.
func (c Config) Massh(log *slog.Logger) (*massh.Config, error) {
	auth, err := c.authMethods(log)
	if err != nil {
		return nil, err
	}

	var hostKeys ssh.HostKeyCallback
	if c.Insecure {
		log.Warn("running in INSECURE mode: host keys are not checked")
		hostKeys = ssh.InsecureIgnoreHostKey()
	} else if hostKeys, err = massh.KnownHosts(); err != nil {
		return nil, fmt.Errorf("%w (use --insecure to skip host key checks)", err)
	}

	return &massh.Config{
		Hosts: c.Hosts,
		Jobs:  []massh.Job{{Command: c.Command}},
		SSHConfig: &ssh.ClientConfig{
			User:            c.Username,
			Auth:            auth,
			HostKeyCallback: hostKeys,
			Timeout:         c.SSHTimeout,
		},
		BastionHost: c.BastionHost,
		Workers:     c.Workers,
		SlowTimeout: c.CommandTimeout,
	}, nil
}

// authMethods returns public key auth (agent and key files) and then password auth. All keys are
// offered by a single method, because the SSH client tries each kind of method only once.
func (c Config) authMethods(log *slog.Logger) ([]ssh.AuthMethod, error) {
	var signers []ssh.Signer
	if c.PrivateKey != "" {
		s, err := massh.PrivateKeySigner(c.PrivateKey, c.PrivateKeyPassphrase)
		if err != nil {
			log.Warn("can't use private key", "file", c.PrivateKey, "err", err)
		} else {
			log.Info("using private key", "file", c.PrivateKey)
			signers = append(signers, s)
		}
	} else {
		for _, f := range defaultKeys {
			s, err := massh.PrivateKeySigner(f, c.PrivateKeyPassphrase)
			switch {
			case errors.Is(err, os.ErrNotExist):
			case err != nil:
				log.Warn("can't use private key", "file", f, "err", err)
			default:
				log.Info("using private key", "file", f)
				signers = append(signers, s)
			}
		}
	}

	agentSigners, err := massh.AgentSigners()
	if err != nil {
		log.Info("not using SSH agent", "err", err)
	}

	var auth []ssh.AuthMethod
	if len(signers) > 0 || agentSigners != nil {
		auth = append(auth, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
			if agentSigners == nil {
				return signers, nil
			}
			fromAgent, err := agentSigners()
			if err != nil {
				log.Warn("can't list SSH agent keys", "err", err)
			}
			return append(fromAgent, signers...), nil
		}))
	}
	if c.Password != "" {
		// Many servers (e.g. OpenSSH with PAM) only offer password login as keyboard-interactive.
		auth = append(auth, ssh.Password(c.Password), ssh.KeyboardInteractive(passwordChallenge(c.Password)))
	}
	if len(auth) == 0 {
		return nil, errors.New("no SSH auth available: no usable private key, SSH agent or password")
	}
	return auth, nil
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
