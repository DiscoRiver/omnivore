package ossh

import (
	"fmt"
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/test"
	"sync"
	"testing"
	"time"
)

// TestStreamWithOutput ensures we can initiate and stream the massh stdout channels when initiated via the OmniSSHConfig
// funcs.
func TestStream_WithOutput_IntegrationWorkflow(t *testing.T) {
	test.InitTestLogger()
	conf := OmniSSHConfig{}
	conf.Config = test.Config

	if err := conf.Config.SetPrivateKeyAuth("~/.ssh/id_rsa", ""); err != nil {
		t.Log(err)
		t.FailNow()
	}

	conf.StreamChan = make(chan *massh.Result)

	// This should be the last responsibility from the massh package. Handling the Result channel is up to the user.
	s, err := conf.Stream()
	if err != nil {
		t.Logf("Stream failed to initiate: %s", err)
	}

	// Add all our hosts now, before we start processing output.
	for host, _ := range conf.Config.Hosts {
		s.TodoHosts[host] = struct{}{}
	}

	var wg sync.WaitGroup
	wg.Add(len(conf.Config.Hosts))

	//TODO: There might be some weird behaviour here depending on how a host fails to connect, but it's not urgent in this test.
	for {
		select {
		case k := <-s.HostsResultChan:
			t.Logf("Read from s.HostsResultChan\n")
			go func() {
				if k.Error != nil {
					// Group similar errors (these are package errors, not ssh Stderr)
					t.Logf("result error: %s", k.Error)
					wg.Done()
				} else {
					test.ReadStreamWithTimeout(k, 5*time.Second, &wg, t)
				}
			}()
		default:
			if massh.NumberOfStreamingHostsCompleted == len(conf.Config.Hosts) {
				wg.Wait()
				t.Logf("All hosts finished.\n")
				return
			}
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
