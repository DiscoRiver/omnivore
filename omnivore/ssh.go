package omnivore

import (
	"fmt"
	"sync"
	"time"

	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
)

func OmniRun(cmd *OmniCommandFlags) {
	// This is our OSSH conig only for doing the work, and doesn't include any UI config. This is all background conf.
	conf := getOSSHConfig(cmd)

	// This should be the last responsibility from the massh package.
	s, err := conf.Stream()
	if err != nil {
		log.OmniLog.Fatal(err.Error())
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
		log.OmniLog.Error("number of hosts expected %v, got %v", len(conf.Config.Hosts), len(s.HostsResultMap))
	}

	for {
		if massh.NumberOfStreamingHostsCompleted == len(s.HostsResultMap) {
			wg.Wait()
			log.OmniLog.Info("All hosts finished.")
			break
		}
	}
}

/*
Some notes for design here going forward. From here, we want to update the stream cycle to move hosts
around between complete, slow, failed groups in the StreamCycle. Movement shouldn't conflict with the
concurrency, but there are mutexes in place anyway.

For now, I will test the grouping package here only when the command as completed. Real-time grouping
is more tricky as it requires us to keep creating a new hash for the output if there are multiple lines.
*/
func readStreamWithTimeout(res massh.Result, t time.Duration, wg *sync.WaitGroup) {
	timeout := time.Second * t
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case d := <-res.StdOutStream:
			fmt.Printf("%s: %s", res.Host, d)
			timer.Reset(timeout)
		case e := <-res.StdErrStream:
			fmt.Printf("%s: %s", res.Host, e)
			timer.Reset(timeout)
		case <-res.DoneChannel:
			// Confirm that the host has exited.
			fmt.Printf("%s: Finished\n", res.Host)
			timer.Reset(timeout)
			wg.Done()
		case <-timer.C:
			fmt.Printf("%s: Timeout due to inactivity\n", res.Host)
			wg.Done()
		}
	}
}

// Read Stdout stream
func readStream(res massh.Result, wg *sync.WaitGroup) {
	for {
		select {
		case d := <-res.StdOutStream:
			fmt.Printf("%s: %s", res.Host, d)
		case e := <-res.StdErrStream:
			fmt.Printf("%s: %s", res.Host, e)
		case <-res.DoneChannel:
			// Confirm that the remote command has finished.
			fmt.Printf("%s: Finished\n", res.Host)
			wg.Done()
		}
	}
}
