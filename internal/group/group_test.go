package group

import (
	"reflect"
	"testing"
)

func TestGroups(t *testing.T) {
	var g Grouping
	g.Add("web3", []byte("ok\n"))
	g.Add("db1", []byte("disk full\n"))
	g.Add("web1", []byte("ok\n"))
	g.Add("web2", []byte("ok\n"))
	g.Add("cache1", []byte("timeout\n"))
	g.Add("", nil)

	if g.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", g.Len())
	}
	got := g.Groups()
	want := []Group{
		{Output: []byte("ok\n"), Hosts: []string{"web1", "web2", "web3"}},
		{Output: []byte(""), Hosts: []string{""}},
		{Output: []byte("disk full\n"), Hosts: []string{"db1"}},
		{Output: []byte("timeout\n"), Hosts: []string{"cache1"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Groups() =\n%q\nwant\n%q", got, want)
	}
}

func TestGroupsIsACopy(t *testing.T) {
	var g Grouping
	g.Add("b", []byte("x"))
	g.Add("a", []byte("x"))
	snap := g.Groups()
	g.Add("c", []byte("x"))
	if len(snap[0].Hosts) != 2 {
		t.Errorf("snapshot changed: %v", snap[0].Hosts)
	}
}

func BenchmarkAdd(b *testing.B) {
	out := make([]byte, 1<<20)
	var g Grouping
	for b.Loop() {
		g.Add("host", out)
	}
}
