package ossh

import (
	"errors"
	"fmt"
	"github.com/discoriver/omnivore/internal/path"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	KnownHostsFile string

	HostNotKnownErr    = errors.New("host is not known")
	HostKeyMismatchErr = errors.New("host is known, but has a mismatched key")
)

func GetKnownHostsPath() (string, error) {
	home, err := path.GetUserHome()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/.ssh/known_hosts", home), err
}

// KnownHosts returns host key callback from a custom known hosts path.
func GetKnownHosts() (ssh.HostKeyCallback, error) {
	defaultKnownHosts, err := GetKnownHostsPath()
	if err != nil {
		return nil, err
	}
	return knownhosts.New(defaultKnownHosts)
}
