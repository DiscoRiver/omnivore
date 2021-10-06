package ossh

import (
	"github.com/discoriver/massh"
	"golang.org/x/crypto/ssh"
	"time"
)

var (
	testHosts = map[string]struct{}{"localhost": {}}

	testBastionHost = "localhost"

	testJob = &massh.Job{
		Command: "echo \"Hello, World\"",
	}

	testSSHConfig = &ssh.ClientConfig{
		User:            "runner",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(2) * time.Second,
	}

	testConfig = &massh.Config{
		Hosts:      testHosts,
		SSHConfig:  testSSHConfig,
		Job:        testJob,
		WorkerPool: 10,
	}
)
