package ssh

import (
	"fmt"
	"github.com/discoriver/massh"
	"golang.org/x/crypto/ssh"
	"sync"
	"testing"
	"time"
)

var (
	testHosts = map[string]struct{}{
		"192.168.1.130": struct{}{},
		"192.168.1.125": struct{}{},
		"192.168.1.129": struct{}{},
		"192.168.1.212": struct{}{},
	}
)

// TestStreamWithOutput ensures we can initiate and stream the massh stdout channels when initiated via the OmniSSHConfig
// funcs.
func TestStreamWithOutput(t *testing.T) {
	conf := OmniSSHConfig{}

	j := &massh.Job{
		Command: "echo \"Hello, World\"",
	}

	sshc := &ssh.ClientConfig{
		// Fake credentials
		User:            "u01",
		Auth:            []ssh.AuthMethod{ssh.Password("password")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(2) * time.Second,
	}

	conf.Config = &massh.Config{
		// In this example I was testing with two working hosts, and two non-existent IPs.
		Hosts:      testHosts,
		SSHConfig:  sshc,
		Job:        j,
		WorkerPool: 10,
	}

	conf.StreamChan = make(chan massh.Result)

	// This should be the last responsibility from the massh package. Handling the Result channel is up to the user.
	s, err := conf.Stream()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(conf.Config.Hosts))

	if len(s.HostsResultMap) == len(conf.Config.Hosts) {
		for k, _ := range s.HostsResultMap {
			k := k
			go func() {
				if s.HostsResultMap[k].Error != nil {
					fmt.Printf("%s: %s\n", s.HostsResultMap[k].Host, s.HostsResultMap[k].Error)
					wg.Done()
				} else {
					readStream(s.HostsResultMap[k], &wg)
				}
			}()
		}
	} else {
		t.Errorf("number of hosts expected %v, got %v", len(conf.Config.Hosts), len(s.HostsResultMap))
	}

	for {
		if massh.NumberOfStreamingHostsCompleted == len(s.HostsResultMap) {
			wg.Wait()
			t.Log("All hosts finished.")
			break
		}
	}
}

// Read Stdout stream
func readStream(res massh.Result, wg *sync.WaitGroup) {
	for {
		select {
		case d := <-res.StdOutStream:
			fmt.Printf("%s: %s", res.Host, d)
		case <-res.DoneChannel:
			// Confirm that the host has exited.
			fmt.Printf("%s: Finished\n", res.Host)
			wg.Done()
		}
	}
}
