package omnivore

import (
	"github.com/discoriver/omnivore/pkg/group"
	"sync"
	"time"

	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
)

func OmniRun(cmd *OmniCommandFlags, grp *group.ValueGrouping) {
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
					// Group similar errors (these are package errors, not ssh Stderr)
					grp.AddToGroup(group.NewIdentifyingPair(s.HostsResultMap[k].Host, []byte(s.HostsResultMap[k].Error.Error())))
					wg.Done()
				} else {
					readStream(s.HostsResultMap[k], grp, &wg)
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
func readStreamWithTimeout(res massh.Result, t time.Duration, grp *group.ValueGrouping, wg *sync.WaitGroup) {
	timeout := time.Second * t
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case d := <-res.StdOutStream:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, d))
			timer.Reset(timeout)
		case e := <-res.StdErrStream:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, e))
			timer.Reset(timeout)
		case <-res.DoneChannel:
			// Confirm that the host has exited.
			log.OmniLog.Info("Host %s finished.", res.Host)
			timer.Reset(timeout)
			wg.Done()
			return
		case t := <-timer.C:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, []byte(t.String())))
			wg.Done()
			return
		}
	}
}

// Read Stdout stream
func readStream(res massh.Result, grp *group.ValueGrouping, wg *sync.WaitGroup) {
	for {
		select {
		case d := <-res.StdOutStream:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, d))
		case e := <-res.StdErrStream:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, e))
		case <-res.DoneChannel:
			// Confirm that the remote command has finished.
			wg.Done()
			return
		}
	}
}