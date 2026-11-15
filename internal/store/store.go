// Package store keeps the history of omnivore runs on disk.
//
// Each run gets its own directory, ~/.omnivore/history/<unix-nanoseconds>/, holding one file per
// host with its output, and the run's log.
package store

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Files in a run's directory besides the host outputs.
const (
	// LogFile is the run's log.
	LogFile = "log.txt"
	// FailedFile lists the hosts that failed, one per line.
	FailedFile = "failed.txt"
)

// Store is the history directory of one run.
type Store struct {
	// Dir is the run's directory.
	Dir string
}

// DefaultBase returns ~/.omnivore.
func DefaultBase() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".omnivore"), nil
}

// New creates a directory for a new run under base/history.
func New(base string) (*Store, error) {
	dir := filepath.Join(base, "history", strconv.FormatInt(time.Now().UnixNano(), 10))
	// Output can be sensitive, so only the user may read it.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create history directory: %w", err)
	}
	return &Store{Dir: dir}, nil
}

// OpenLog opens the run's log file for appending.
func (s *Store) OpenLog() (*os.File, error) {
	return os.OpenFile(filepath.Join(s.Dir, LogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
}

// WriteHost saves host's output.
func (s *Store) WriteHost(host string, output []byte) error {
	return os.WriteFile(s.hostPath(host), output, 0o600)
}

// WriteHostList writes hosts, one per line, to the named file in the run's directory, and returns
// its path.
func (s *Store) WriteHostList(name string, hosts []string) (string, error) {
	path := filepath.Join(s.Dir, name)
	return path, os.WriteFile(path, []byte(strings.Join(hosts, "\n")+"\n"), 0o600)
}

// ReadHost returns the output saved for host.
func (s *Store) ReadHost(host string) ([]byte, error) {
	return os.ReadFile(s.hostPath(host))
}

// hostPath returns the file for host. Names are escaped so that every host gets its own file
// inside s.Dir, and can't clash with the log.
func (s *Store) hostPath(host string) string {
	name := url.PathEscape(host)
	if name == "" || name == "." || name == ".." || name == LogFile || name == FailedFile {
		name = "%" + name
	}
	return filepath.Join(s.Dir, name)
}
