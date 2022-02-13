package omnivore

import (
	"github.com/discoriver/omnivore/internal/ui"
	"github.com/discoriver/omnivore/pkg/group"
	"sync"
	"time"

	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
)

func OmniRun(cmd *OmniCommandFlags) {
	// This is our OSSH conig only for doing the work, and doesn't include any UI config. This is all background conf.
	conf := getOSSHConfig(cmd)

	// This should be the last responsibility from the massh package.
	s, err := conf.Stream() // <-- Slow to return if host doesn't connect
	if err != nil {
		log.OmniLog.Fatal(err.Error())
	}
	ui.DP.StreamCycle = s

	go func() {
		for {
			select {
			case <-ui.DP.Group.Update:
			default:
				ui.DP.Refresh()
				time.Sleep(1 * time.Second)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(conf.Config.Hosts))
	if len(s.HostsResultMap) == len(conf.Config.Hosts) {
		for k, _ := range s.HostsResultMap {
			k := k

			go func() {
				if s.HostsResultMap[k].Error != nil {
					// Group similar errors (these are package errors, not ssh Stderr)
					ui.DP.Group.AddToGroup(group.NewIdentifyingPair(s.HostsResultMap[k].Host, []byte(s.HostsResultMap[k].Error.Error())))
					ui.DP.StreamCycle.AddFailedHost(s.HostsResultMap[k].Host)

					wg.Done()
				} else {
					readStreamWithTimeout(s.HostsResultMap[k], time.Duration(cmd.CommandTimeout), ui.DP.Group, &wg)
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
	defer func() {
		timer.Stop()
		wg.Done()
	}()

	var bes []byte
	for {
		select {
		case d := <-res.StdOutStream:
			bes = append(bes, d...)
			timer.Reset(timeout)
		case e := <-res.StdErrStream:
			bes = append(bes, e...)
			timer.Reset(timeout)
		case <-res.DoneChannel:
			// Confirm that the host has exited.
			log.OmniLog.Info("Host %s finished.", res.Host)
			timer.Reset(timeout)
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, bes))
			ui.DP.StreamCycle.AddCompletedHost(res.Host)
			return
		case <-timer.C:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, []byte("Activity timeout.")))
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
			ui.DP.StreamCycle.AddCompletedHost(res.Host)
			wg.Done()
			return
		}
	}
}
