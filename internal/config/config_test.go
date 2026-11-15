package config

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/discoriver/massh/v2/sshtest"
	"github.com/spf13/pflag"
)

func parse(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	return fs
}

// isolate points the home directory at an empty temporary directory.
func isolate(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv(PasswordEnv, "")
	return home
}

func TestLoadDefaults(t *testing.T) {
	isolate(t)
	cfg, err := Load(parse(t, "--hosts", "a,b", "--hosts", "c", "-c", "uptime", "-u", "ops"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(cfg.Hosts, " ") != "a b c" || cfg.Command != "uptime" || cfg.Username != "ops" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.SSHTimeout != DefaultSSHTimeout || cfg.Workers != DefaultWorkers || cfg.CommandTimeout != 0 {
		t.Errorf("defaults not applied: %+v", cfg)
	}
	if cfg.File != "" {
		t.Errorf("File = %q, want none", cfg.File)
	}
}

func TestLoadConfigFile(t *testing.T) {
	home := isolate(t)
	dir := filepath.Join(home, ".omnivore")
	os.MkdirAll(dir, 0o700)
	file := filepath.Join(dir, "config.yaml")
	yaml := "omni:\n  Username: fromfile\n  BastionHost: jump\n  SSHTimeout: 3\n  CommandTimeout: 7\n  WorkerPool: 5\n"
	if err := os.WriteFile(file, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(parse(t, "--hosts", "a", "-c", "x", "-t", "4"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.File != file || cfg.Username != "fromfile" || cfg.BastionHost != "jump" || cfg.Workers != 5 {
		t.Errorf("file not applied: %+v", cfg)
	}
	if cfg.SSHTimeout != 4*time.Second {
		t.Errorf("flag didn't override file: SSHTimeout = %v", cfg.SSHTimeout)
	}
	if cfg.CommandTimeout != 7*time.Second {
		t.Errorf("CommandTimeout = %v", cfg.CommandTimeout)
	}

	// The file may hold secrets, so it must be private.
	os.Chmod(file, 0o644)
	if _, err := Load(parse(t, "--hosts", "a", "-c", "x")); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Errorf("Load with a world-readable config: err = %v", err)
	}
}

func TestLoadExplicitConfigFileMissing(t *testing.T) {
	isolate(t)
	if _, err := Load(parse(t, "--hosts", "a", "-c", "x", "--config", "/nonexistent.yaml")); err == nil {
		t.Error("Load succeeded with a missing --config file")
	}
}

func TestLoadValidation(t *testing.T) {
	isolate(t)
	_, err := Load(parse(t, "-u", "ops", "-w", "0"))
	for _, want := range []string{"--hosts", "--command", "workers"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want mention of %s", err, want)
		}
	}
}

func TestLoadPasswordFromEnv(t *testing.T) {
	isolate(t)
	t.Setenv(PasswordEnv, "secret")
	cfg, err := Load(parse(t, "--hosts", "a", "-c", "x", "-u", "ops"))
	if err != nil || cfg.Password != "secret" {
		t.Errorf("Password = %q, %v", cfg.Password, err)
	}
}

func TestMassh(t *testing.T) {
	home := isolate(t)
	srv := sshtest.NewServer(t)
	key := filepath.Join(home, "key")
	os.WriteFile(key, srv.ClientKeyPEM, 0o600)

	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))

	cfg := Config{
		Hosts: []string{srv.Addr}, Command: "echo hi", Username: sshtest.User,
		PrivateKey: key, Password: "pw", Insecure: true,
		SSHTimeout: 3 * time.Second, CommandTimeout: 2 * time.Second, Workers: 4, BastionHost: "jump",
	}
	m, err := cfg.Massh(log)
	if err != nil {
		t.Fatal(err)
	}
	// One public key method (all keys) and one password method.
	if len(m.SSHConfig.Auth) != 2 {
		t.Errorf("got %d auth methods, want 2", len(m.SSHConfig.Auth))
	}
	if m.SSHConfig.Timeout != 3*time.Second || m.SlowTimeout != 2*time.Second || m.Workers != 4 ||
		m.BastionHost != "jump" || m.Jobs[0].Command != "echo hi" || m.SSHConfig.User != sshtest.User {
		t.Errorf("massh config = %+v", m)
	}
	if !strings.Contains(logs.String(), "INSECURE") {
		t.Error("insecure mode was not logged")
	}

	// The key works.
	m.BastionHost = ""
	res, err := m.Run(t.Context())
	if err != nil || res[0].Err != nil || string(res[0].Stdout) != "hi\n" {
		t.Fatalf("Run: %v %+v", err, res)
	}
}

func TestMasshNoAuth(t *testing.T) {
	isolate(t)
	cfg := Config{Hosts: []string{"a"}, Command: "x", Username: "u", Insecure: true, PrivateKey: "/nonexistent"}
	if _, err := cfg.Massh(slog.New(slog.DiscardHandler)); err == nil {
		t.Error("Massh succeeded with no usable auth")
	}
}

func TestMasshKnownHostsRequired(t *testing.T) {
	isolate(t)
	cfg := Config{Hosts: []string{"a"}, Command: "x", Username: "u", Password: "pw"}
	if _, err := cfg.Massh(slog.New(slog.DiscardHandler)); err == nil || !strings.Contains(err.Error(), "--insecure") {
		t.Errorf("err = %v, want a hint about --insecure", err)
	}
}
