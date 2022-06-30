// Package test contains relevant test parameters.
package test

import (
	"github.com/discoriver/massh"
	"golang.org/x/crypto/ssh"
	"os"
	"sync"
	"testing"
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

func ReadStreamWithTimeout(res *massh.Result, timeout time.Duration, wg *sync.WaitGroup, t *testing.T) {
	timer := time.NewTimer(timeout)
	defer func() {
		timer.Stop()
		wg.Done()
	}()

	for {
		select {
		case d := <-res.StdOutStream:
			t.Logf("%s: %s\n", res.Host, d)
			timer.Reset(timeout)
		case e := <-res.StdErrStream:
			t.Logf("%s: %s\n", res.Host, e)
			timer.Reset(timeout)
		case <-res.DoneChannel:
			// Confirm that the host has exited.
			t.Logf("Host %s finished.\n", res.Host)
			timer.Reset(timeout)
			return
		case <-timer.C:
			t.Logf("Activity timeout: %s\n", res.Host)
			return
		}
	}
}
