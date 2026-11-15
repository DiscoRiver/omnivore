package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/discoriver/massh/v2"
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
	_, err := Load(parse(t, "-u", "ops", "-w", "0", "--kill-slow"))
	for _, want := range []string{"--hosts", "--command", "workers", "--command-timeout"} {
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
		PrivateKey: key, Password: "pw", HostKeys: HostKeysInsecure,
		SSHTimeout: 3 * time.Second, CommandTimeout: 2 * time.Second, Workers: 4, BastionHost: "jump",
	}
	m, err := cfg.Massh(log)
	if err != nil {
		t.Fatal(err)
	}
	// One public key method (all keys), then password and keyboard-interactive.
	if len(m.SSHConfig.Auth) != 3 {
		t.Errorf("got %d auth methods, want 3", len(m.SSHConfig.Auth))
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
	cfg := Config{Hosts: []string{"a"}, Command: "x", Username: "u", HostKeys: HostKeysInsecure, PrivateKey: "/nonexistent", NoSSHConfig: true}
	if _, err := cfg.Massh(slog.New(slog.DiscardHandler)); err == nil {
		t.Error("Massh succeeded with no usable auth")
	}
}

func TestMasshKnownHostsRequired(t *testing.T) {
	isolate(t)
	cfg := Config{Hosts: []string{"a"}, Command: "x", Username: "u", Password: "pw"}
	if _, err := cfg.Massh(slog.New(slog.DiscardHandler)); err == nil || !strings.Contains(err.Error(), "--accept-new") {
		t.Errorf("err = %v, want a hint about --accept-new", err)
	}
}

func TestPasswordChallenge(t *testing.T) {
	got, err := passwordChallenge("pw")("", "", []string{"Password:", "Username:"}, []bool{false, true})
	if err != nil || len(got) != 2 || got[0] != "pw" || got[1] != "" {
		t.Errorf("answers = %q, %v", got, err)
	}
}

func TestLoadCleansHosts(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "hosts")
	os.WriteFile(file, []byte("# fleet\ndb[1-2]\na\n"), 0o600)
	cfg, err := Load(parse(t, "--hosts", "a, b,,web[01-02] ,", "-c", "x", "-u", "ops", "-f", file))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(cfg.Hosts, "|") != "a|b|web01|web02|db1|db2" {
		t.Errorf("Hosts = %q", cfg.Hosts)
	}
	if _, err := Load(parse(t, "-c", "x", "-f", filepath.Join(t.TempDir(), "missing"))); err == nil {
		t.Error("Load succeeded with a missing hosts file")
	}
	if _, err := Load(parse(t, "--hosts", " , ", "-c", "x", "-u", "ops")); err == nil {
		t.Error("Load succeeded with only empty hosts")
	}
}

// TestMasshSSHConfig checks that aliases, users, identity files and ProxyJump come from
// ~/.ssh/config.
func TestMasshSSHConfig(t *testing.T) {
	home := isolate(t)
	bastion, target := sshtest.NewServer(t), sshtest.NewServer(t)
	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	os.WriteFile(filepath.Join(sshDir, "target_key"), target.ClientKeyPEM, 0o600)
	os.WriteFile(filepath.Join(sshDir, "bastion_key"), bastion.ClientKeyPEM, 0o600)
	_, tPort, _ := net.SplitHostPort(target.Addr)
	_, bPort, _ := net.SplitHostPort(bastion.Addr)
	conf := fmt.Sprintf(`
Host app
    HostName 127.0.0.1
    Port %s
    User %s
    IdentityFile ~/.ssh/target_key
    ProxyJump jump

Host jump
    HostName 127.0.0.1
    Port %s
    IdentityFile ~/.ssh/bastion_key
`, tPort, sshtest.User, bPort)
	os.WriteFile(filepath.Join(sshDir, "config"), []byte(conf), 0o600)

	cfg := Config{Hosts: []string{"app"}, Command: "echo via alias", HostKeys: HostKeysInsecure, SSHTimeout: 5 * time.Second, Workers: 1}
	m, err := cfg.Massh(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	res, err := m.Run(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Err != nil || string(res[0].Stdout) != "via alias\n" {
		t.Fatalf("result = %+v", res[0])
	}
	if bastion.Forwards() != 1 {
		t.Errorf("bastion forwards = %d, want 1", bastion.Forwards())
	}

	// --no-ssh-config ignores it.
	cfg.NoSSHConfig = true
	cfg.Password = "unused"
	m, err = cfg.Massh(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if m.Resolve != nil {
		t.Error("Resolve set with NoSSHConfig")
	}
}

func TestSudoJob(t *testing.T) {
	isolate(t)
	srv := sshtest.NewServer(t)
	// A fake sudo that checks the password on stdin and runs the rest of its arguments.
	bin := t.TempDir()
	fake := `#!/bin/sh
while [ "$1" != "--" ]; do
  case "$1" in
    -S) read -r pw; [ "$pw" = "secret" ] || { echo "sudo: wrong password" >&2; exit 1; } ;;
    -p) shift ;;
  esac
  shift
done
shift
exec "$@"
`
	os.WriteFile(filepath.Join(bin, "sudo"), []byte(fake), 0o700)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	run := func(c Config) string {
		t.Helper()
		m := &massh.Config{Hosts: []string{srv.Addr}, Jobs: []massh.Job{c.Job()}, SSHConfig: srv.ClientConfig()}
		res, err := m.Run(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		return string(res[0].Stdout) + string(res[0].Stderr)
	}

	// The password is consumed by sudo and never reaches the command's stdin.
	if got := run(Config{Command: `cat; echo "it's done"`, Sudo: true, SudoPassword: "secret"}); got != "it's done\n" {
		t.Errorf("sudo with password: %q", got)
	}
	if got := run(Config{Command: "echo hi", Sudo: true, SudoPassword: "wrong"}); !strings.Contains(got, "wrong password") {
		t.Errorf("sudo with wrong password: %q", got)
	}
	if job := (Config{Command: "id", Sudo: true}).Job(); !strings.HasPrefix(job.Command, "sudo -n ") || job.Stdin != nil {
		t.Errorf("passwordless sudo job = %+v", job)
	}
	if job := (Config{Command: "id"}).Job(); job.Command != "id" {
		t.Errorf("plain job = %+v", job)
	}
}

func TestMasshAcceptNew(t *testing.T) {
	home := isolate(t)
	srv := sshtest.NewServer(t)
	key := filepath.Join(home, "key")
	os.WriteFile(key, srv.ClientKeyPEM, 0o600)
	cfg := Config{Hosts: []string{srv.Addr}, Command: "true", Username: sshtest.User, PrivateKey: key,
		HostKeys: HostKeysAcceptNew, SSHTimeout: 5 * time.Second, Workers: 1}
	m, err := cfg.Massh(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if res, _ := m.Run(t.Context()); res[0].Err != nil {
		t.Fatal(res[0].Err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil || !strings.Contains(string(data), "ssh-ed25519") {
		t.Errorf("known_hosts = %q, %v", data, err)
	}
	// Now strict mode accepts it too.
	cfg.HostKeys = HostKeysStrict
	if m, err = cfg.Massh(slog.New(slog.DiscardHandler)); err != nil {
		t.Fatal(err)
	}
	if res, _ := m.Run(t.Context()); res[0].Err != nil {
		t.Error(res[0].Err)
	}
}
