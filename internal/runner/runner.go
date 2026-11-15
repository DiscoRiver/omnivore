// Package runner feeds the events of a massh stream into omnivore's state and history.
package runner

import (
	"log/slog"

	"github.com/discoriver/massh/v2"
	"github.com/discoriver/omnivore/internal/state"
)

// History saves each host's result.
type History interface {
	WriteHost(host string, output []byte) error
}

// Run applies events to st until the channel is closed, saving each host's result to hist as it
// finishes, and then closes st.
func Run(events <-chan massh.Event, st *state.State, hist History, log *slog.Logger) {
	defer st.Close()
	for ev := range events {
		switch ev.Kind {
		case massh.EventStdout:
			st.AppendStdout(ev.Host, ev.Data)
		case massh.EventStderr:
			st.AppendStderr(ev.Host, ev.Data)
		case massh.EventSlow:
			log.Warn("host is slow", "host", ev.Host)
			st.MarkSlow(ev.Host)
		case massh.EventDone:
			if ev.Err != nil {
				log.Warn("host failed", "host", ev.Host, "err", ev.Err)
			} else {
				log.Info("host finished", "host", ev.Host)
			}
			result, ok := st.Finish(ev.Host, ev.Err)
			if !ok {
				log.Warn("unexpected result for host", "host", ev.Host)
				continue
			}
			if err := hist.WriteHost(ev.Host, result); err != nil {
				log.Warn("can't save host output", "host", ev.Host, "err", err)
			}
		}
	}
	log.Info("all hosts finished")
}
