package ossh

import (
	"fmt"
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/test"
	"sync"
	"testing"
)

// TestStreamWithOutput ensures we can initiate and stream the massh stdout channels when initiated via the OmniSSHConfig
// funcs.
func TestStreamWithOutput(t *testing.T) {
	test.InitTestLogger()
	conf := OmniSSHConfig{}
	conf.Config = testConfig

	if err := testConfig.SetPrivateKeyAuth("~/.ssh/id_rsa", ""); err != nil {
		t.Log(err)
		t.FailNow()
	}

	conf.StreamChan = make(chan massh.Result)

	// This should be the last responsibility from the massh package. Handling the Result channel is up to the user.
	s, err := conf.Stream()
	if err != nil {
		t.Logf("Stream failed to initiate: %s", err)
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
