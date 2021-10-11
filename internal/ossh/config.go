package ossh

import (
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
	"golang.org/x/crypto/ssh"
)

type OmniSSHConfig struct {
	Config     *massh.Config
	StreamChan chan massh.Result
}

// NewConfig initialises a new OmniSSHConfig
func NewConfig() *OmniSSHConfig {
	c := &OmniSSHConfig{
		Config:     massh.NewConfig(),
		StreamChan: make(chan massh.Result),
	}
	return c
}

// Stream executes work contained in the massh.Config, and returns a StreamCycle for monitoring output and status.
func (c *OmniSSHConfig) Stream() (*StreamCycle, error) {
	err := c.Config.Stream(c.StreamChan)
	if err != nil {
		return nil, err
	}

	log.OmniLog.Info("Massh Streaming started successfully.")

	ss := newStreamCycle(c.StreamChan, len(c.Config.Hosts))
	return ss, nil
}

// AddHosts populated OmniSSHConfig with target hosts.
func (c *OmniSSHConfig) AddHosts(h []string) {
	c.Config.SetHosts(h)
}

// AddSSHConfig adds an ssh.ClientConfig to the OmniSSHConfig
func (c *OmniSSHConfig) AddSSHConfig(s *ssh.ClientConfig) {
	c.Config.SetSSHConfig(s)
}

// AddJob adds a massh.Job to the OmniSSHConfig
func (c *OmniSSHConfig) AddJob(j *massh.Job) {
	c.Config.SetJob(j)
}

// AddBastionHosts adds bastion host to massh.Config
func (c *OmniSSHConfig) AddBastionHost(b string) {
	c.Config.SetBastionHost(b)
}

// AddBastionHostConfig adds a custom ssh.ClientConfig for the bastion host.
func (c *OmniSSHConfig) AddBastionHostConfig(s *ssh.ClientConfig) {
	c.Config.SetBastionHostConfig(s)
}

// AddWorkerPool adds number of concurrent workers to config.
func (c *OmniSSHConfig) AddWorkerPool(w int) {
	c.Config.SetWorkerPool(w)
}

// AddPasswordAuth sets the user and password auth in massh.Config.
func (c *OmniSSHConfig) AddPasswordAuth(user string, password string) {
	c.Config.SetPasswordAuth(user, password)
}

// AddPrivateKeyAuth configures the private key for auth in massh.Config. Returns error on failure.
func (c *OmniSSHConfig) AddPrivateKeyAuth(keyPath string, password string) (err error) {
	if err = c.Config.SetPrivateKeyAuth(keyPath, password); err != nil {
		return err
	}

	return nil
}

// AddSSHSockAuth sets the SSH_AUTH_SOCK variable for auth in massh.Config.
func (c *OmniSSHConfig) AddSSHSockAuth() error {
	c.Config.SetSSHAuthSock()

	return nil
}
