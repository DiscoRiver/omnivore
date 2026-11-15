package hosts

import (
	"reflect"
	"strings"
	"testing"
)

func TestExpand(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{[]string{"web1", "web2"}, []string{"web1", "web2"}},
		{[]string{"web[01-03]"}, []string{"web01", "web02", "web03"}},
		{[]string{"db[1,4-5].example.com"}, []string{"db1.example.com", "db4.example.com", "db5.example.com"}},
		{[]string{"r[1-2]n[1-2]"}, []string{"r1n1", "r1n2", "r2n1", "r2n2"}},
		{[]string{"web[8-10]"}, []string{"web8", "web9", "web10"}},
		{[]string{"web[1-2]:2222"}, []string{"web1:2222", "web2:2222"}},
		{[]string{"[::1]:22", "[fe80::1]"}, []string{"[::1]:22", "[fe80::1]"}},
		{[]string{"a", " ", "a", "b"}, []string{"a", "b"}},
	}
	for _, tt := range tests {
		got, err := Expand(tt.in...)
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Expand(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}

func TestExpandErrors(t *testing.T) {
	for _, in := range []string{"web[1-", "web[a-b]", "web[5-1]", "web[1-9999999]", "a[0-999]b[0-999]"} {
		if _, err := Expand(in); err == nil {
			t.Errorf("Expand(%q) succeeded", in)
		}
	}
}

func TestRead(t *testing.T) {
	got, err := Read(strings.NewReader("# fleet\nweb[1-2]  db1,db2\n\n  cache1 # primary\n"))
	want := []string{"web[1-2]", "db1", "db2", "cache1"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Read = %q, %v; want %q", got, err, want)
	}
}
