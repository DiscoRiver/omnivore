package state

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/discoriver/massh/v2"
)

func TestLifecycle(t *testing.T) {
	s := New("uptime", []string{"b", "a", "c", "a"})

	s.AppendStdout("a", []byte("up 1 day\n"))
	s.AppendStdout("b", []byte("up 1 day\n"))
	s.AppendStdout("c", []byte("partial"))
	s.MarkSlow("c")
	s.AppendStdout("unknown", []byte("ignored"))

	snap := s.Snapshot()
	if got := snap.Names(func(h Host) bool { return h.Status == Pending }); len(got) != 3 || got[0] != "a" {
		t.Fatalf("pending = %v, want sorted a, b, c", got)
	}

	if got, ok := s.Finish("a", nil); !ok || string(got) != "up 1 day\n" {
		t.Errorf("Finish(a) = %q, %v", got, ok)
	}
	s.Finish("b", nil)
	if got, _ := s.Finish("c", errors.New("boom")); string(got) != "partial\n[error: boom]\n" {
		t.Errorf("Finish(c) = %q", got)
	}
	if _, ok := s.Finish("a", errors.New("again")); ok {
		t.Error("finishing a host twice changed it")
	}
	if _, ok := s.Finish("unknown", nil); ok {
		t.Error("finished an unknown host")
	}
	s.Close()

	snap = s.Snapshot()
	if !snap.Done {
		t.Error("not done")
	}
	c, ok := snap.Host("c")
	if !ok || c.Status != Failed || !c.Slow || string(c.Stdout) != "partial" {
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
		s.AppendStdout("a", []byte("."))
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
				s.AppendStdout(h, []byte("line\n"))
			}
			s.Finish(h, nil)
		})
	}
	wg.Go(func() {
		for range 100 {
			snap := s.Snapshot()
			for _, h := range snap.Hosts {
				_ = len(h.Stdout)
			}
		}
	})
	wg.Wait()
	if g := s.Snapshot().Groups; len(g) != 1 || len(g[0].Hosts) != 2 {
		t.Errorf("groups = %v", g)
	}
}

// TestResultIgnoresInterleaving checks that hosts whose stdout and stderr arrived in a different
// order still group together.
func TestResultIgnoresInterleaving(t *testing.T) {
	s := New("x", []string{"a", "b", "c"})
	s.AppendStdout("a", []byte("active\n"))
	s.AppendStderr("a", []byte("nginx/1.27.3\n"))
	s.AppendStderr("b", []byte("nginx/1.27.3\n"))
	s.AppendStdout("b", []byte("active\n"))
	s.AppendStdout("c", []byte("no newline"))
	s.AppendStderr("c", []byte("warning"))
	ra, _ := s.Finish("a", nil)
	rb, _ := s.Finish("b", nil)
	rc, _ := s.Finish("c", errors.New("boom"))
	if string(ra) != "active\nnginx/1.27.3\n" || string(ra) != string(rb) {
		t.Errorf("results differ: %q vs %q", ra, rb)
	}
	if string(rc) != "no newline\nwarning\n[error: boom]\n" {
		t.Errorf("result c = %q", rc)
	}
	if g := s.Snapshot().Groups; len(g) != 2 || len(g[0].Hosts) != 2 {
		t.Errorf("groups = %+v", g)
	}
}

func TestDescribeNoAuth(t *testing.T) {
	h := Host{Err: fmt.Errorf("connect: %w", massh.ErrNoAuth)}
	if got := string(h.Result()); !strings.Contains(got, "no SSH key, agent or password") {
		t.Errorf("Result() = %q", got)
	}
}
