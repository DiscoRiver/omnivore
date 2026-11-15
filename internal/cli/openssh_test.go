package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/discoriver/massh/v2"
	"github.com/discoriver/omnivore/internal/config"
)

// TestMain lets the test binary act as ssh's askpass helper.
func TestMain(m *testing.M) {
	massh.HandleAskpass()
	os.Exit(m.Run())
}

// TestRunOpenSSH runs omnivore end to end through a stand-in ssh client that runs commands
// locally, and fails to connect to the host "unreachable".
func TestRunOpenSSH(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fake := filepath.Join(t.TempDir(), "ssh")
	script := `#!/bin/sh
while [ $# -gt 0 ] && [ "$1" != "--" ]; do shift; done
shift
if [ "$1" = unreachable ]; then echo "ssh: connect to host unreachable port 22: Connection refused" >&2; exit 255; fi
if [ "$1" = stranger ]; then echo "Host key verification failed." >&2; exit 255; fi
exec sh -c "$2"
`
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Hosts:     []string{"web1", "web2", "unreachable", "stranger"},
		Command:   "echo same",
		SSHClient: config.SSHClientOpenSSH,
		SSHPath:   fake,
		Workers:   2,
		NoUI:      true,
	}
	var out bytes.Buffer
	err := Run(t.Context(), cfg, filepath.Join(home, ".omnivore"), &out)
	if err == nil || err.Error() != "2 of 4 hosts failed" {
		t.Fatalf("Run() = %v\n%s", err, out.String())
	}
	got := out.String()
	for _, want := range []string{
		"== 2 host(s): web1, web2\nsame\n",
		"Connection refused",
		"host key verification failed: the host is not in known_hosts (use --accept-new)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary is missing %q:\n%s", want, got)
		}
	}
}
