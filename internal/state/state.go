// Package state holds the progress and output of an omnivore run, shared between the code that
// runs commands and the code that displays them.
package state

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/discoriver/massh/v2"
	"github.com/discoriver/omnivore/internal/group"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Status is where a host is in its lifecycle.
type Status int

const (
	// Pending hosts have not finished yet.
	Pending Status = iota
	// Completed hosts ran the command, which exited successfully.
	Completed
	// Failed hosts could not run the command, or it exited with an error.
	Failed
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"
	case Completed:
		return "completed"
	case Failed:
		return "failed"
	default:
		return fmt.Sprintf("Status(%d)", int(s))
	}
}

// Host is the state of one host.
type Host struct {
	Name   string
	Status Status
	// Slow is set when the host has been quiet for longer than the command timeout.
	Slow bool
	// Stdout and Stderr are what the host has written so far. They are kept apart because
	// the order in which lines from the two arrive varies from run to run.
	Stdout []byte
	Stderr []byte
	// Err is why the host failed.
	Err error
}

// Result is the host's stdout, then its stderr, then a description of its error, if any. This
// is what hosts are grouped by, so it doesn't depend on how stdout and stderr interleaved.
func (h Host) Result() []byte {
	var out []byte
	add := func(b []byte) {
		if len(b) == 0 {
			return
		}
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, b...)
	}
	add(h.Stdout)
	add(h.Stderr)
	if h.Err != nil {
		add([]byte(describe(h.Err) + "\n"))
	}
	return out
}

func describe(err error) string {
	if exit, ok := errors.AsType[*ssh.ExitError](err); ok {
		return fmt.Sprintf("[exit status %d]", exit.ExitStatus())
	}
	if errors.Is(err, massh.ErrNoAuth) {
		return "[error: no SSH key, agent or password to log in with; use -k, --ask-pass, or IdentityFile in ~/.ssh/config]"
	}
	if keyErr, ok := errors.AsType[*knownhosts.KeyError](err); ok {
		if len(keyErr.Want) == 0 {
			return "[error: host key is not in known_hosts; add it, or use --accept-new]"
		}
		return "[error: HOST KEY HAS CHANGED; possible man-in-the-middle attack, or the host was rebuilt]"
	}
	return fmt.Sprintf("[error: %v]", err)
}

// State is the shared, concurrency-safe state of a run.
type State struct {
	mu      sync.Mutex
	command string
	names   []string
	hosts   map[string]*Host
	groups  group.Grouping
	done    bool

	changed chan struct{}
}

// New returns the state for running command on hosts, which all start Pending.
func New(command string, hosts []string) *State {
	s := &State{
		command: command,
		hosts:   make(map[string]*Host, len(hosts)),
		changed: make(chan struct{}, 1),
	}
	for _, h := range hosts {
		if _, ok := s.hosts[h]; ok {
			continue
		}
		s.hosts[h] = &Host{Name: h}
		s.names = append(s.names, h)
	}
	slices.Sort(s.names)
	return s
}

// Changed returns a channel that receives a value after the state changes. Changes made while
// nobody is receiving are coalesced into one notification.
func (s *State) Changed() <-chan struct{} { return s.changed }

// update runs f with the lock held, and then notifies listeners.
func (s *State) update(f func()) {
	s.mu.Lock()
	f()
	s.mu.Unlock()
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// AppendStdout adds data to host's stdout.
func (s *State) AppendStdout(host string, data []byte) {
	s.update(func() {
		if h, ok := s.hosts[host]; ok {
			h.Stdout = append(h.Stdout, data...)
		}
	})
}

// AppendStderr adds data to host's stderr.
func (s *State) AppendStderr(host string, data []byte) {
	s.update(func() {
		if h, ok := s.hosts[host]; ok {
			h.Stderr = append(h.Stderr, data...)
		}
	})
}

// MarkSlow flags host as slow.
func (s *State) MarkSlow(host string) {
	s.update(func() {
		if h, ok := s.hosts[host]; ok {
			h.Slow = true
		}
	})
}

// Finish marks host as Completed, or Failed if err is not nil, adds it to the output groups, and
// returns its Result. It does nothing and returns false if host is unknown or already finished.
func (s *State) Finish(host string, err error) (result []byte, ok bool) {
	s.update(func() {
		h, known := s.hosts[host]
		if !known || h.Status != Pending {
			return
		}
		ok = true
		h.Err = err
		h.Status = Completed
		if err != nil {
			h.Status = Failed
		}
		result = h.Result()
		s.groups.Add(host, result)
	})
	return result, ok
}

// Close records that the run is over.
func (s *State) Close() {
	s.update(func() { s.done = true })
}

// Snapshot is a consistent copy of a State at one moment.
type Snapshot struct {
	Command string
	// Hosts are sorted by name.
	Hosts []Host
	// Groups are ordered as by group.Grouping.Groups.
	Groups []group.Group
	// Done is true once the run is over.
	Done bool
}

// Snapshot returns a copy of the current state. Its byte slices must not be modified.
func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := Snapshot{
		Command: s.command,
		Hosts:   make([]Host, len(s.names)),
		Groups:  s.groups.Groups(),
		Done:    s.done,
	}
	for i, name := range s.names {
		h := *s.hosts[name]
		// Output is only ever appended to, so sharing what has been written so far is safe.
		h.Stdout, h.Stderr = slices.Clip(h.Stdout), slices.Clip(h.Stderr)
		snap.Hosts[i] = h
	}
	return snap
}

// Names returns the names of the hosts for which keep returns true, in order.
func (s Snapshot) Names(keep func(Host) bool) []string {
	var names []string
	for _, h := range s.Hosts {
		if keep(h) {
			names = append(names, h.Name)
		}
	}
	return names
}

// Host returns the named host.
func (s Snapshot) Host(name string) (Host, bool) {
	i, ok := slices.BinarySearchFunc(s.Hosts, name, func(h Host, n string) int {
		return strings.Compare(h.Name, n)
	})
	if !ok {
		return Host{}, false
	}
	return s.Hosts[i], true
}
