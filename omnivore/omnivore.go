package omnivore

import (
	"fmt"
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/ossh"
	"golang.org/x/crypto/ssh"
	"sync"
	"time"
)

type OmniCommandFlags struct {
	Hosts              []string
	BastionHost        string
	Username           string
	Password           string
	PrivateKeyLocation string
	PrivateKeyPassword string
	Command            string
}

func OmniRun(cmd *OmniCommandFlags) {
	conf := ossh.NewConfig()

	conf.AddJob(&massh.Job{Command: cmd.Command})
	conf.AddHosts(cmd.Hosts)
	conf.AddPasswordAuth(cmd.Username, cmd.Password)
	conf.Config.SSHConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	conf.Config.SSHConfig.Timeout = time.Duration(2) * time.Second
	conf.AddWorkerPool(10)

	conf.StreamChan = make(chan massh.Result)

	// This should be the last responsibility from the massh package. Handling the Result channel is up to the user.
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
