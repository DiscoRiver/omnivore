// Package group groups hosts by identical output, so that anomalies stand out.
package group

import (
	"bytes"
	"slices"
	"strings"
)

// Group is a set of hosts that produced the same output.
type Group struct {
	Output []byte
	Hosts  []string
}

// Grouping collects host outputs into groups. The zero value is ready to use. It is not safe for
// concurrent use.
type Grouping struct {
	groups map[string]*Group
}

// Add records that host produced output. Adding the same host twice adds it twice.
func (g *Grouping) Add(host string, output []byte) {
	if g.groups == nil {
		g.groups = make(map[string]*Group)
	}
	// Go hashes string keys itself, so the output is its own key.
	key := string(output)
	if grp, ok := g.groups[key]; ok {
		grp.Hosts = append(grp.Hosts, host)
		return
	}
	g.groups[key] = &Group{Output: []byte(key), Hosts: []string{host}}
}

// Len returns the number of groups.
func (g *Grouping) Len() int { return len(g.groups) }

// Groups returns a copy of the groups, largest first, so that the odd ones out come last. Groups
// of the same size are ordered by output, and hosts within a group are sorted.
func (g *Grouping) Groups() []Group {
	out := make([]Group, 0, len(g.groups))
	for _, grp := range g.groups {
		hosts := slices.Clone(grp.Hosts)
		slices.Sort(hosts)
		out = append(out, Group{Output: grp.Output, Hosts: hosts})
	}
	slices.SortFunc(out, func(a, b Group) int {
		if n := len(b.Hosts) - len(a.Hosts); n != 0 {
			return n
		}
		if c := bytes.Compare(a.Output, b.Output); c != 0 {
			return c
		}
		return strings.Compare(a.Hosts[0], b.Hosts[0])
	})
	return out
}
