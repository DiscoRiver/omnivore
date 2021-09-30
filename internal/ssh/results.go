package ssh

import (
	"errors"
	"fmt"
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/log"
)

var (
	todoHostMapLoc      = "todo"
	completedHostMapLoc = "completed"
	failedHostMapLoc    = "failed"
	slowHostMapLoc      = "slow"

	ErrHostAlreadyMoved          = errors.New("host cannot be moved, is already in a termination map")
	ErrHostIsNotInStreamCycle    = errors.New("host does not exist in StreamCycle")
	ErrStreamCycleNotInitialised = errors.New("StreamCycle is not initialised")
)

// StreamCycle contains values for the lifecycle of a stream job. Hosts should begin their life in the TodoHosts map, and
// must be moved to one of the termination maps when the massh.Result.DoneChannel is written to. Hosts must not be moved
// back into the TodoHosts map once moved.
type StreamCycle struct {
	HostsResultMap map[string]massh.Result

	// Lifecycle begin
	TodoHosts map[string]struct{}

	// Termination
	CompletedHosts map[string]struct{}
	FailedHosts    map[string]struct{}
	SlowHosts      map[string]struct{}

	initialised bool
	cyclePtrMap map[string]map[string]struct{}
}

func newStreamCycle(rc chan massh.Result, numHosts int) *StreamCycle {
	ss := &StreamCycle{}
	ss.Initialise()
	ss.populateResultsMap(rc, numHosts)
	return ss
}

// Initialise sets adds pending, completed, failed, and slow host pointers to a relevant map var for a specific StreamCycle.
func (s *StreamCycle) Initialise() {
	// Initialise map in struct
	s.cyclePtrMap = map[string]map[string]struct{}{}

	// Initialise HostResultMap
	s.HostsResultMap = make(map[string]massh.Result)

	// Assign pointers to map for specific states for a host
	s.cyclePtrMap[todoHostMapLoc] = s.TodoHosts
	s.cyclePtrMap[completedHostMapLoc] = s.CompletedHosts
	s.cyclePtrMap[failedHostMapLoc] = s.FailedHosts
	s.cyclePtrMap[slowHostMapLoc] = s.SlowHosts

	s.initialised = true

	log.OmniLog.Info("StreamCycle was initialised.")
}

func (s *StreamCycle) isInitialised() bool {
	return s.initialised
}

func (s *StreamCycle) populateResultsMap(ch chan massh.Result, numHosts int) error {
	if !s.isInitialised() {
		return ErrStreamCycleNotInitialised
	}

	for {
		select {
		case result := <-ch:
			// TODO: Update this to handle duplicate hostnames
			s.HostsResultMap[result.Host] = result
		default:
			if len(s.HostsResultMap) == numHosts {
				log.OmniLog.Info(fmt.Sprintf("StreamCycle HostsResultMap populated with %d hosts.", len(s.HostsResultMap)))

				return nil
			}
		}
	}
}

func (s *StreamCycle) AddTodoHost(host string) error {
	if !s.isInitialised() {
		return ErrStreamCycleNotInitialised
	}

	// Check to ensure the host hasn't already been processed. This handles duplicate host names gracefully.
	if err := s.hostIsAlreadyMoved(host); err != nil {
		return err
	}

	s.moveHost(host, todoHostMapLoc)

	return nil
}

func (s *StreamCycle) AddCompletedHost(host string) error {
	if !s.isInitialised() {
		return ErrStreamCycleNotInitialised
	}

	if err := s.hostIsAlreadyMoved(host); err != nil {
		return err
	}

	s.moveHost(host, completedHostMapLoc)

	return nil
}

func (s *StreamCycle) AddFailedHost(host string) error {
	if !s.isInitialised() {
		return ErrStreamCycleNotInitialised
	}

	if err := s.hostIsAlreadyMoved(host); err != nil {
		return err
	}

	s.moveHost(host, failedHostMapLoc)

	return nil
}

func (s *StreamCycle) AddSlowHost(host string) error {
	if !s.isInitialised() {
		return ErrStreamCycleNotInitialised
	}

	if err := s.hostIsAlreadyMoved(host); err != nil {
		return err
	}

	s.moveHost(host, slowHostMapLoc)

	return nil
}

func (s *StreamCycle) moveHost(host string, loc string) {
	(*s).cyclePtrMap[loc][host] = struct{}{}

	// Delete host from TodoHosts
	s.deleteTodoHost(host)
}

func (s *StreamCycle) deleteTodoHost(host string) {
	delete((*s).cyclePtrMap[todoHostMapLoc], host)
}

func (s *StreamCycle) hostIsAlreadyMoved(host string) error {
	if _, ok := s.TodoHosts[host]; ok {
		return nil
	}

	if _, ok := s.CompletedHosts[host]; ok {
		return ErrHostAlreadyMoved
	}

	if _, ok := s.FailedHosts[host]; ok {
		return ErrHostAlreadyMoved
	}

	if _, ok := s.SlowHosts[host]; ok {
		return ErrHostAlreadyMoved
	}

	return ErrHostIsNotInStreamCycle
}
