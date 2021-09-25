package ssh

import (
	"github.com/discoriver/massh"
	"golang.org/x/crypto/ssh"
)

type OmniSSHConfig struct {
	Config *massh.Config
	StreamChan chan massh.Result
}

func (c *OmniSSHConfig) Stream() error {
	err := c.Config.Stream(c.StreamChan)
	if err != nil {
		return err
	}

	return nil
}

func (c *OmniSSHConfig) AddHosts(h []string) {
	c.Config.SetHosts(h)
}

func (c *OmniSSHConfig) AddSSHConfig(s *ssh.ClientConfig) {
	c.Config.SetSSHConfig(s)
}

func (c *OmniSSHConfig) AddJob(j *massh.Job) {
	c.Config.SetJob(j)
}

func (c *OmniSSHConfig) AddBastionHost(b string) {
	c.Config.SetBastionHost(b)
}

func (c *OmniSSHConfig) AddBastionHostConfig(s *ssh.ClientConfig) {
	c.Config.SetBastionHostConfig(s)
}

func (c *OmniSSHConfig) AddWorkerPool(w int) {
	c.Config.SetWorkerPool(w)
}

func (c *OmniSSHConfig) AddPasswordAuth(p string) error {
	c.Config.SetPasswordAuth([]byte(p))

	return nil
}

func (c *OmniSSHConfig) AddPublicKeyAuth(k string, p string) (err error) {
	if err = c.Config.SetPublicKeyAuth(k, p); err != nil {
		return err
	}

	return nil
}

func (c *OmniSSHConfig) AddSSHSockAuth() error {
	c.Config.SetSSHAuthSockAuth()

	return nil
}






