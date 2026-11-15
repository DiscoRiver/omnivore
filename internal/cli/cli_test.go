package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/discoriver/massh/v2/sshtest"
	"github.com/discoriver/omnivore/internal/config"
	"github.com/discoriver/omnivore/internal/store"
	"golang.org/x/crypto/ssh/knownhosts"
)

// TestRunNoUI runs omnivore end to end against two in-process SSH servers and an unreachable host.
func TestRunNoUI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")

	srv1, srv2 := sshtest.NewServer(t), sshtest.NewServer(t)

	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	var known strings.Builder
	for _, s := range []*sshtest.Server{srv1, srv2} {
		known.WriteString(knownhosts.Line([]string{knownhosts.Normalize(s.Addr)}, s.HostKey) + "\n")
	}
	os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte(known.String()), 0o600)
	// srv1's key is the default key; srv2 doesn't accept it, so it fails authentication.
	os.WriteFile(filepath.Join(sshDir, "id_ed25519"), srv1.ClientKeyPEM, 0o600)

	cfg := config.Config{
		Hosts:      []string{srv1.Addr, srv2.Addr, "127.0.0.1:1"},
		Command:    `echo "same everywhere"`,
		Username:   sshtest.User,
		SSHTimeout: 5 * time.Second,
		Workers:    2,
		NoUI:       true,
	}
	base := filepath.Join(home, ".omnivore")
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := Run(ctx, cfg, base, &out); err == nil || err.Error() != "2 of 3 hosts failed" {
		t.Fatalf("Run() = %v, want 2 of 3 hosts failed", err)
	}

	got := out.String()
	for _, want := range []string{
		"2 failed hosts saved; retry them with: --hosts-file ",
		"Command: echo \"same everywhere\"",
		"1 completed, 2 failed",
		"== 1 host(s): " + srv1.Addr + "\nsame everywhere\n",
		"[error: ",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary is missing %q:\n%s", want, got)
		}
	}

	// History has the log and one file per host.
	runs, _ := filepath.Glob(filepath.Join(base, "history", "*"))
	if len(runs) != 1 {
		t.Fatalf("history runs = %v", runs)
	}
	s := &store.Store{Dir: runs[0]}
	if data, err := s.ReadHost(srv1.Addr); err != nil || string(data) != "same everywhere\n" {
		t.Errorf("saved output = %q, %v", data, err)
	}
	failed, _ := os.ReadFile(filepath.Join(runs[0], store.FailedFile))
	if want := "127.0.0.1:1\n" + srv2.Addr + "\n"; string(failed) != want && string(failed) != srv2.Addr+"\n127.0.0.1:1\n" {
		t.Errorf("failed.txt = %q", failed)
	}
	logData, _ := os.ReadFile(filepath.Join(runs[0], store.LogFile))
	if !strings.Contains(string(logData), "all hosts finished") {
		t.Errorf("log is missing completion:\n%s", logData)
	}
}

func TestCommandRequiresFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cmd := NewCommand()
	cmd.SetArgs([]string{"--hosts", "a"})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "command") {
		t.Errorf("Execute() = %v, want missing --command", err)
	}
}
