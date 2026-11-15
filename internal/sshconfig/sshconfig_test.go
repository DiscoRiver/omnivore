package sshconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testConfig = `
Host web*
    User deploy
    IdentityFile ~/.ssh/web_key

Host web1
    HostName 10.0.0.1
    Port 2222
    ProxyJump jump

Host jump
    HostName bastion.example.com
    User ops
    Port 2200

Host direct-jump
    ProxyJump admin@other:2022

Host chained
    ProxyJump a,b

Host legacy
    ProxyCommand nc %h %p

Host *
    IdentityFile ~/.ssh/id_%r
`

func load(t *testing.T) *Config {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(testConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLookup(t *testing.T) {
	c := load(t)
	home, _ := os.UserHomeDir()

	got, err := c.Lookup("web1")
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{
		HostName:      "10.0.0.1",
		Port:          "2222",
		User:          "deploy",
		IdentityFiles: []string{filepath.Join(home, ".ssh/web_key"), filepath.Join(home, ".ssh/id_deploy")},
		ProxyJump: Jump{
			Host: "bastion.example.com", Port: "2200", User: "ops",
			IdentityFiles: []string{filepath.Join(home, ".ssh/id_ops")},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lookup(web1) =\n%+v\nwant\n%+v", got, want)
	}

	got, err = c.Lookup("direct-jump")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProxyJump.Host != "other" || got.ProxyJump.User != "admin" || got.ProxyJump.Port != "2022" {
		t.Errorf("Lookup(direct-jump).ProxyJump = %+v", got.ProxyJump)
	}

	got, err = c.Lookup("unconfigured")
	if err != nil || got.HostName != "" || got.User != "" || got.ProxyJump.Host != "" {
		t.Errorf("Lookup(unconfigured) = %+v, %v", got, err)
	}
}

func TestLookupUnsupported(t *testing.T) {
	c := load(t)
	for _, alias := range []string{"chained", "legacy"} {
		if _, err := c.Lookup(alias); err == nil {
			t.Errorf("Lookup(%s) succeeded", alias)
		}
	}
}

func TestLoadNoFiles(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "none"))
	if err != nil {
		t.Fatal(err)
	}
	if s, err := c.Lookup("x"); err != nil || s.HostName != "" {
		t.Errorf("Lookup = %+v, %v", s, err)
	}
}
