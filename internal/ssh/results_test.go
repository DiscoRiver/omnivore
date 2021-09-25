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
	mockResult []massh.Result

	sampleResult1 = massh.Result{
		Host: "host1",
	}

	sampleResult2 = massh.Result{
		Host: "host2",
	}
)

func TestPopulateResultsMap(t *testing.T) {
	mockResult = append(mockResult, sampleResult1, sampleResult2)
	s := StreamCycle{}
	s.Initialise()
	ch := make(chan massh.Result)

	for i := range mockResult {
		i := i
		go func() {ch <- mockResult[i]}()
	}

	err := s.populateResultsMap(ch, len(mockResult))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	if len(s.HostsResultMap) == len(mockResult) {
		if _, ok := s.HostsResultMap["host1"]; !ok {
			t.FailNow()
		}
	} else {
		t.Errorf("number of hosts expected %v, got %v", len(mockResult), len(s.HostsResultMap))
	}
}

func TestReadStdoutFromStreamCycle(t *testing.T) {
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

	cfg := &massh.Config{
		// In this example I was testing with two working hosts, and two non-existent IPs.
		Hosts:      []string{"192.168.1.130", "192.168.1.125", "192.168.1.129", "192.168.1.212"},
		SSHConfig:  sshc,
		Job:        j,
		WorkerPool: 10,
	}

	s := StreamCycle{}
	s.Initialise()

	resChan := make(chan massh.Result)

	// This should be the last responsibility from the massh package. Handling the Result channel is up to the user.
	err := cfg.Stream(resChan)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	err = s.populateResultsMap(resChan, len(cfg.Hosts))
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	wg.Add(len(cfg.Hosts))

		if len(s.HostsResultMap) == len(cfg.Hosts) {
			for k, _ := range s.HostsResultMap {
				k := k
				go func() {
					if s.HostsResultMap[k].Error != nil {
						fmt.Printf("%s: %s\n", s.HostsResultMap[k].Host, s.HostsResultMap[k].Error)
						wg.Done()
					} else {
						readStream(s.HostsResultMap[k], &wg, t)
					}
				}()
			}
		} else {
			t.Errorf("number of hosts expected %v, got %v", len(cfg.Hosts), len(s.HostsResultMap))
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
func readStream(res massh.Result, wg *sync.WaitGroup, t *testing.T) {
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
