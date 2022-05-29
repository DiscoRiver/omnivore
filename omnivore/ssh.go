package omnivore

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/store"
	"github.com/discoriver/omnivore/internal/ui"
	"github.com/discoriver/omnivore/pkg/group"
	"sync"
	"time"

	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
)

func OmniRun(cmd *OmniCommandFlags, safe chan struct{}, uiStarted chan struct{}) {
	// This is our OSSH conig only for doing the work, and doesn't include any UI config. This is all background conf.
	conf := getOSSHConfig(cmd)

	// This should be the last responsibility from the massh package.
	s, err := conf.Stream() // <-- Slow to return if host doesn't connect
	if err != nil {
		log.OmniLog.Fatal(err.Error())
	}
	ui.Collective.StreamCycle = s

	// Add all our hosts now, before we start processing output.
	for host, _ := range conf.Config.Hosts {
		s.TodoHosts[host] = struct{}{}
	}

	/*
		Tell the UI it's safe to start, and wait for it to report that it's safe to start interacting with it.

		Avoids a race condition where the UI isn't initialised before we try to update it.
	*/
	safe <- struct{}{}
	waitForUI := func() {
		t := time.NewTimer(5 * time.Second)
		for {
			select {
			case <-uiStarted:
				return
			case <-t.C: // This should never happen, but just in case.
				fmt.Println("UI is not stating in a timely manner. Please wait, or press CTRL+C to quit.")
				t.Stop()
			}
		}
	}
	waitForUI()

	// Begin updating the UI panels.
	go func() {
		for {
			select {
			case <-ui.Collective.Group.Update:
				ui.Collective.Refresh()
			default:
				ui.Collective.Refresh()
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Start processing output.
	var wg sync.WaitGroup
	wg.Add(len(conf.Config.Hosts))
	processResults := func() {
		for {
			select {
			case k := <-s.HostsResultChan:
				go func() {
					if k.Error != nil {
						// Group similar errors (these are package errors, not ssh Stderr)
						hostErrOutput := group.NewIdentifyingPair(k.Host, []byte(k.Error.Error()))
						ui.Collective.Group.AddToGroup(hostErrOutput)
						ui.Collective.StreamCycle.AddFailedHost(k.Host)

						// This will print package errors, which includes dial errors to the host, so we include it here
						// as it's "technically" output, and is displayed on the UI. Although not strictly stderr,
						// it is still a response from interacting with the SSH protocol.
						store.Session.WriteOutputFileForHost(hostErrOutput)

						wg.Done()
					} else {
						readStreamWithTimeout(k, time.Duration(cmd.CommandTimeout), ui.Collective.Group, &wg)
					}
				}()
			default:
				if massh.NumberOfStreamingHostsCompleted == len(conf.Config.Hosts) {
					wg.Wait()
					log.OmniLog.Info("All hosts finished.")
					return
				}
			}
		}
	}
	processResults()
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
			hostOutputPair := group.NewIdentifyingPair(res.Host, bes)
			grp.AddToGroup(hostOutputPair)
			ui.Collective.StreamCycle.AddCompletedHost(res.Host)
			store.Session.WriteOutputFileForHost(hostOutputPair)
			return
		case <-timer.C:
			grp.AddToGroup(group.NewIdentifyingPair(res.Host, []byte("Activity timeout.")))
			ui.Collective.StreamCycle.AddSlowHost(res.Host)
			return
		}
	}
}
