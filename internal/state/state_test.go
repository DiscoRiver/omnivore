package state

import (
	"errors"
	"sync"
	"testing"
)

func TestLifecycle(t *testing.T) {
	s := New("uptime", []string{"b", "a", "c", "a"})

	s.AppendOutput("a", []byte("up 1 day\n"))
	s.AppendOutput("b", []byte("up 1 day\n"))
	s.AppendOutput("c", []byte("partial"))
	s.MarkSlow("c")
	s.AppendOutput("unknown", []byte("ignored"))

	snap := s.Snapshot()
	if got := snap.Names(func(h Host) bool { return h.Status == Pending }); len(got) != 3 || got[0] != "a" {
		t.Fatalf("pending = %v, want sorted a, b, c", got)
	}

	if got := string(s.Finish("a", nil)); got != "up 1 day\n" {
		t.Errorf("Finish(a) = %q", got)
	}
	s.Finish("b", nil)
	if got := string(s.Finish("c", errors.New("boom"))); got != "partial\n[error: boom]\n" {
		t.Errorf("Finish(c) = %q", got)
	}
	if s.Finish("a", errors.New("again")) != nil {
		t.Error("finishing a host twice changed it")
	}
	s.Close()

	snap = s.Snapshot()
	if !snap.Done {
		t.Error("not done")
	}
	c, ok := snap.Host("c")
	if !ok || c.Status != Failed || !c.Slow || string(c.Output) != "partial" {
		t.Errorf("host c = %+v", c)
	}
	if _, ok := snap.Host("zzz"); ok {
		t.Error("found a host that doesn't exist")
	}
	if len(snap.Groups) != 2 || len(snap.Groups[0].Hosts) != 2 {
		t.Errorf("groups = %+v", snap.Groups)
	}
}

func TestChangedCoalesces(t *testing.T) {
	s := New("x", []string{"a"})
	for range 10 {
		s.AppendOutput("a", []byte("."))
	}
	<-s.Changed()
	select {
	case <-s.Changed():
		t.Error("got a second notification")
	default:
	}
}

func TestConcurrentUse(t *testing.T) {
	s := New("x", []string{"a", "b"})
	var wg sync.WaitGroup
	for _, h := range []string{"a", "b"} {
		wg.Go(func() {
			for range 1000 {
				s.AppendOutput(h, []byte("line\n"))
			}
			s.Finish(h, nil)
		})
	}
	wg.Go(func() {
		for range 100 {
			snap := s.Snapshot()
			for _, h := range snap.Hosts {
				_ = len(h.Output)
			}
		}
	})
	wg.Wait()
	if g := s.Snapshot().Groups; len(g) != 1 || len(g[0].Hosts) != 2 {
		t.Errorf("groups = %v", g)
	}
}
