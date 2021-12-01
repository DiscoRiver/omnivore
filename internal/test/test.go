// Package test contains relevant test parameters.
package test

import (
	"github.com/discoriver/massh"
	"golang.org/x/crypto/ssh"
	"os"
	"time"

	"github.com/discoriver/omnivore/internal/log"
)

var (
	Hosts = map[string]struct{}{"localhost": {}}

	Job = &massh.Job{
		Command: "echo \"Hello, World\"",
	}

	SSHConfig = &ssh.ClientConfig{
		User:            "runner",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(2) * time.Second,
	}

	Config = &massh.Config{
		Hosts:      Hosts,
		SSHConfig:  SSHConfig,
		Job:        Job,
		WorkerPool: 10,
	}
)

func InitTestLogger() {
	log.OmniLog = &log.OmniLogger{FileOutput: os.Stdout}
	log.OmniLog.Init()
}
