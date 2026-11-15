// Package sshconfig reads the settings omnivore honours from OpenSSH client config files:
// HostName, Port, User, IdentityFile and ProxyJump.
package sshconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// Settings are the values configured for one host. Empty fields are not configured.
type Settings struct {
	HostName      string
	Port          string
	User          string
	IdentityFiles []string
	// ProxyJump is the jump host, resolved against the config if it is itself an alias. Its
	// Host is empty if there is none.
	ProxyJump Jump
}

// Jump is a resolved jump host.
type Jump struct {
	// Host is the address to connect to, without the port.
	Host          string
	Port          string
	User          string
	IdentityFiles []string
}

// Config is a set of parsed config files, consulted in order.
type Config struct {
	files []*ssh_config.Config
}

// DefaultFiles are the files read by Load when given none.
func DefaultFiles() []string {
	files := []string{"/etc/ssh/ssh_config"}
	if home, err := os.UserHomeDir(); err == nil {
		files = append([]string{filepath.Join(home, ".ssh", "config")}, files...)
	}
	return files
}

// Load parses the given config files, skipping those that don't exist. With no files, it reads
// DefaultFiles.
func Load(paths ...string) (*Config, error) {
	if len(paths) == 0 {
		paths = DefaultFiles()
	}
	c := &Config{}
	for _, p := range paths {
		f, err := os.Open(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		cfg, err := ssh_config.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		c.files = append(c.files, cfg)
	}
	return c, nil
}

func (c *Config) get(alias, key string) string {
	for _, f := range c.files {
		if v, err := f.Get(alias, key); err == nil && v != "" {
			return v
		}
	}
	return ""
}

func (c *Config) getAll(alias, key string) []string {
	var out []string
	for _, f := range c.files {
		if vs, err := f.GetAll(alias, key); err == nil {
			out = append(out, vs...)
		}
	}
	return out
}

// Lookup returns the settings for alias, a host name without a port.
func (c *Config) Lookup(alias string) (Settings, error) {
	s := Settings{
		HostName: c.get(alias, "HostName"),
		Port:     c.get(alias, "Port"),
		User:     c.get(alias, "User"),
	}
	host := cmpOr(s.HostName, alias)
	for _, f := range c.getAll(alias, "IdentityFile") {
		s.IdentityFiles = append(s.IdentityFiles, expandTokens(f, alias, host, s.User))
	}
	s.HostName = expandTokens(s.HostName, alias, alias, s.User)

	if pc := c.get(alias, "ProxyCommand"); pc != "" && !strings.EqualFold(pc, "none") {
		return Settings{}, fmt.Errorf("ProxyCommand is not supported (host %s); use ProxyJump", alias)
	}
	pj := c.get(alias, "ProxyJump")
	if pj == "" || strings.EqualFold(pj, "none") {
		return s, nil
	}
	if strings.Contains(pj, ",") {
		return Settings{}, fmt.Errorf("ProxyJump with several hops is not supported (host %s)", alias)
	}
	jump, err := c.lookupJump(pj)
	if err != nil {
		return Settings{}, err
	}
	s.ProxyJump = jump
	return s, nil
}

// lookupJump parses "[user@]host[:port]" (or "ssh://..."), resolving host as an alias.
func (c *Config) lookupJump(spec string) (Jump, error) {
	spec = strings.TrimPrefix(spec, "ssh://")
	var j Jump
	if u, rest, ok := strings.Cut(spec, "@"); ok {
		j.User, spec = u, rest
	}
	alias := spec
	if i := strings.LastIndexByte(spec, ':'); i >= 0 && !strings.HasSuffix(spec, "]") {
		alias, j.Port = spec[:i], spec[i+1:]
	}
	alias = strings.Trim(alias, "[]")
	if alias == "" {
		return Jump{}, fmt.Errorf("bad ProxyJump %q", spec)
	}

	// The jump host's own config.
	if c.get(alias, "ProxyJump") != "" {
		if pj := c.get(alias, "ProxyJump"); !strings.EqualFold(pj, "none") {
			return Jump{}, fmt.Errorf("ProxyJump with several hops is not supported (jump host %s)", alias)
		}
	}
	j.Host = cmpOr(c.get(alias, "HostName"), alias)
	j.Port = cmpOr(j.Port, c.get(alias, "Port"))
	j.User = cmpOr(j.User, c.get(alias, "User"))
	for _, f := range c.getAll(alias, "IdentityFile") {
		j.IdentityFiles = append(j.IdentityFiles, expandTokens(f, alias, j.Host, j.User))
	}
	return j, nil
}

// expandTokens expands the OpenSSH tokens %h (host name), %n (alias), %r (remote user),
// %u (local user), %d (home directory) and %%, and a leading "~/".
func expandTokens(s, alias, host, remoteUser string) string {
	if !strings.ContainsAny(s, "%~") {
		return s
	}
	home, _ := os.UserHomeDir()
	local := ""
	if u, err := user.Current(); err == nil {
		local = u.Username
	}
	r := strings.NewReplacer("%%", "%", "%h", host, "%n", alias, "%r", cmpOr(remoteUser, local), "%u", local, "%d", home)
	s = r.Replace(s)
	if rest, ok := strings.CutPrefix(s, "~/"); ok && home != "" {
		s = filepath.Join(home, rest)
	}
	return s
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
