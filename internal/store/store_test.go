package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	base := t.TempDir()
	s, err := New(base)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(s.Dir)) != base {
		t.Errorf("Dir = %s, want it under %s/history", s.Dir, base)
	}

	for _, host := range []string{"web1", "10.0.0.1:2222", "../escape", "log.txt", "..", "a/b", "a_b", "a%2Fb"} {
		if err := s.WriteHost(host, []byte(host)); err != nil {
			t.Fatalf("WriteHost(%q): %v", host, err)
		}
		got, err := s.ReadHost(host)
		if err != nil || string(got) != host {
			t.Errorf("ReadHost(%q) = %q, %v", host, got, err)
		}
	}

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 8 {
		t.Errorf("got %d files, want 8 (nothing written outside the run directory)", len(entries))
	}
	info, err := os.Stat(filepath.Join(s.Dir, "web1"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("host file mode = %v, %v; want 0600", info.Mode(), err)
	}

	f, err := s.OpenLog()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}
