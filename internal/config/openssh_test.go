package config

import (
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func stubLookPath(t *testing.T, path string, err error) {
	t.Helper()
	old := lookPath
	lookPath = func(string) (string, error) { return path, err }
	t.Cleanup(func() { lookPath = old })
}

func TestResolveSSHClient(t *testing.T) {
	isolate(t)
	found, missing := "/usr/bin/ssh", errors.New("not found")
	tests := []struct {
		flag     string
		path     string
		err      error
		want     string
		wantFail bool
	}{
		{"auto", found, nil, SSHClientOpenSSH, false},
		{"auto", "", missing, SSHClientNative, false},
		{"openssh", found, nil, SSHClientOpenSSH, false},
		{"openssh", "", missing, "", true},
		{"native", found, nil, SSHClientNative, false},
		{"putty", found, nil, "", true},
	}
	for _, tt := range tests {
		stubLookPath(t, tt.path, tt.err)
		cfg, err := Load(parse(t, "--hosts", "a", "-c", "x", "--ssh", tt.flag))
		if (err != nil) != tt.wantFail {
			t.Errorf("--ssh=%s (ssh found: %v): err = %v", tt.flag, tt.err == nil, err)
			continue
		}
		if err == nil && cfg.SSHClient != tt.want {
			t.Errorf("--ssh=%s (ssh found: %v): SSHClient = %q, want %q", tt.flag, tt.err == nil, cfg.SSHClient, tt.want)
		}
		if cfg.SSHClient == SSHClientOpenSSH && cfg.SSHPath != found {
			t.Errorf("SSHPath = %q", cfg.SSHPath)
		}
	}
}

func TestMasshOpenSSH(t *testing.T) {
	isolate(t)
	t.Setenv(SudoPasswordEnv, "sudo-secret")
	t.Setenv(PasswordEnv, "ssh-secret")
	cfg := Config{
		Hosts: []string{"web1"}, Command: "uptime", SSHClient: SSHClientOpenSSH, SSHPath: "/usr/bin/ssh",
		Username: "ops", PrivateKey: "~/.ssh/deploy", Password: "pw", HostKeys: HostKeysAcceptNew,
		SSHTimeout: 7 * time.Second, Workers: 3, BastionHost: "jump", NoSSHConfig: true,
		CommandTimeout: 5 * time.Second, KillSlow: true,
	}
	m, err := cfg.Massh(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	o := m.OpenSSH
	if o == nil || m.SSHConfig != nil || m.Resolve != nil {
		t.Fatalf("not configured for OpenSSH: %+v", m)
	}
	exe, _ := os.Executable()
	if o.Path != "/usr/bin/ssh" || o.User != "ops" || !slices.Equal(o.IdentityFiles, []string{"~/.ssh/deploy"}) ||
		o.HostKeyChecking != "accept-new" || o.ConnectTimeout != 7*time.Second || o.ConfigFile != os.DevNull ||
		o.Password != "pw" || o.AskpassProgram != exe || o.AllowForwarding {
		t.Errorf("OpenSSH = %+v", o)
	}
	if m.BastionHost != "jump" || m.Workers != 3 || m.SlowTimeout != 5*time.Second || !m.CancelSlow {
		t.Errorf("massh config = %+v", m)
	}
	for _, kv := range o.Env {
		if strings.HasPrefix(kv, SudoPasswordEnv+"=") || strings.HasPrefix(kv, PasswordEnv+"=") {
			t.Errorf("secret passed to ssh's environment: %s", kv)
		}
	}
	if !slices.ContainsFunc(o.Env, func(kv string) bool { return strings.HasPrefix(kv, "HOME=") }) {
		t.Error("ordinary environment not passed to ssh")
	}

	// Strict by default, and insecure means no known_hosts at all.
	cfg.HostKeys, cfg.Password, cfg.NoSSHConfig = HostKeysStrict, "", false
	m, _ = cfg.Massh(slog.New(slog.DiscardHandler))
	if m.OpenSSH.HostKeyChecking != "yes" || m.OpenSSH.AskpassProgram != "" || m.OpenSSH.ConfigFile != "" {
		t.Errorf("strict: %+v", m.OpenSSH)
	}
	cfg.HostKeys = HostKeysInsecure
	m, _ = cfg.Massh(slog.New(slog.DiscardHandler))
	if m.OpenSSH.HostKeyChecking != "no" || !slices.Contains(m.OpenSSH.Options, "UserKnownHostsFile=/dev/null") {
		t.Errorf("insecure: %+v", m.OpenSSH)
	}
}
