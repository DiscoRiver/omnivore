// Package logging sets up omnivore's logger, which writes to the run's log file and keeps recent
// messages in memory for the UI's log panel.
package logging

import (
	"io"
	"log/slog"
	"strings"
	"sync"
)

// New returns a logger that writes to file and to buf. Records for buf omit the time, since they
// are shown live.
func New(file io.Writer, buf *Buffer) *slog.Logger {
	return slog.New(slog.NewMultiHandler(
		slog.NewTextHandler(file, nil),
		slog.NewTextHandler(buf, &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if len(groups) == 0 && a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		}),
	))
}

// Buffer keeps the most recent lines written to it. It is safe for concurrent use.
type Buffer struct {
	mu    sync.Mutex
	lines []string
	max   int
	total int
}

// NewBuffer returns a Buffer that keeps up to n lines.
func NewBuffer(n int) *Buffer {
	return &Buffer{max: n}
}

// Write adds each line in p. A trailing newline is optional.
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for line := range strings.Lines(string(p)) {
		b.lines = append(b.lines, strings.TrimSuffix(line, "\n"))
		b.total++
	}
	if over := len(b.lines) - b.max; over > 0 {
		b.lines = append(b.lines[:0], b.lines[over:]...)
	}
	return len(p), nil
}

// Lines returns a copy of the kept lines, oldest first, and the number of lines ever written.
func (b *Buffer) Lines() ([]string, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.lines...), b.total
}
