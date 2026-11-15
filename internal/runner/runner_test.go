package runner

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/discoriver/massh/v2"
	"github.com/discoriver/omnivore/internal/state"
)

type history map[string]string

func (h history) WriteHost(host string, output []byte) error {
	h[host] = string(output)
	return nil
}

func TestRun(t *testing.T) {
	st := state.New("cmd", []string{"a", "b"})
	hist := history{}
	events := make(chan massh.Event)
	go func() {
		defer close(events)
		events <- massh.Event{Kind: massh.EventStdout, Host: "a", Data: []byte("out\n")}
		events <- massh.Event{Kind: massh.EventSlow, Host: "b"}
		events <- massh.Event{Kind: massh.EventStderr, Host: "b", Data: []byte("err\n")}
		events <- massh.Event{Kind: massh.EventDone, Host: "a"}
		events <- massh.Event{Kind: massh.EventDone, Host: "b", Err: errors.New("boom")}
		// A duplicate result must not overwrite the saved output.
		events <- massh.Event{Kind: massh.EventDone, Host: "a", Err: errors.New("again")}
	}()
	Run(events, st, hist, slog.New(slog.DiscardHandler))

	snap := st.Snapshot()
	if !snap.Done {
		t.Error("state not closed")
	}
	a, _ := snap.Host("a")
	b, _ := snap.Host("b")
	if a.Status != state.Completed || b.Status != state.Failed || !b.Slow {
		t.Errorf("a = %+v, b = %+v", a, b)
	}
	if hist["a"] != "out\n" || hist["b"] != "err\n[error: boom]\n" {
		t.Errorf("history = %q", hist)
	}
}
