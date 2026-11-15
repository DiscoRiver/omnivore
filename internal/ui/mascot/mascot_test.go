package mascot

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestDetect(t *testing.T) {
	tests := []struct {
		env  func(string) string
		want Protocol
	}{
		{env(), None},
		{env("TERM", "xterm-256color"), None},
		{env("TERM", "xterm-kitty"), Kitty},
		{env("TERM_PROGRAM", "ghostty"), Kitty},
		{env("KONSOLE_VERSION", "240802"), Kitty},
		{env("TERM_PROGRAM", "iTerm.app"), ITerm2},
		{env("TERM_PROGRAM", "WezTerm"), ITerm2},
		{env("TERM", "xterm-kitty", "TMUX", "/tmp/tmux-1/default"), None},
		{env("TERM_PROGRAM", "iTerm.app", Env, "off"), None},
		{env(Env, "iterm2"), ITerm2},
		{env(Env, "kitty", "TMUX", "x"), Kitty},
	}
	for i, tt := range tests {
		if got := Detect(tt.env); got != tt.want {
			t.Errorf("case %d: Detect = %v, want %v", i, got, tt.want)
		}
	}
}

func TestFramesAreValidPNGs(t *testing.T) {
	m := New(ITerm2, 8, 4)
	for _, n := range frameNames {
		img, err := png.Decode(bytes.NewReader(m.frames[n]))
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if b := img.Bounds(); b.Dx() != b.Dy() || b.Dx() < 64 {
			t.Errorf("%s: %v, want a square of at least 64px", n, b)
		}
	}
	if New(None, 8, 4) != nil {
		t.Error("New(None) returned a mascot")
	}
}

func TestITerm2(t *testing.T) {
	m := New(ITerm2, 8, 4)
	if m.Setup() != "" || m.Clear() != "" {
		t.Error("iTerm2 needs no setup or cleanup")
	}
	seq := m.Show(Roar, 1, 2)
	re := regexp.MustCompile(`^\x1b7\x1b\[1;2H\x1b\]1337;File=inline=1;size=(\d+);width=8;height=4;preserveAspectRatio=1:([A-Za-z0-9+/=]+)\a\x1b8$`)
	sub := re.FindStringSubmatch(seq)
	if sub == nil {
		t.Fatalf("unexpected sequence: %q…", seq[:min(len(seq), 80)])
	}
	data, err := base64.StdEncoding.DecodeString(sub[2])
	if err != nil || !bytes.Equal(data, m.frames[Roar]) {
		t.Error("payload isn't the frame")
	}
	if Show := m.Show("nope", 1, 1); Show != "" {
		t.Error("unknown frame produced output")
	}
}

func TestKitty(t *testing.T) {
	m := New(Kitty, 8, 4)
	setup := m.Setup()
	// Every frame is transmitted, in chunks of at most 4096 bytes, with replies suppressed.
	for _, n := range frameNames {
		if !strings.Contains(setup, "\x1b_Ga=t,f=100,i="+strconv.Itoa(kittyID(n))+",q=2,") {
			t.Errorf("frame %s not transmitted", n)
		}
	}
	for _, part := range strings.Split(setup, "\x1b\\") {
		if _, payload, ok := strings.Cut(part, ";"); ok && len(payload) > 4096 {
			t.Errorf("chunk of %d bytes", len(payload))
		}
	}

	first := m.Show(Chomp1, 1, 1)
	if !strings.Contains(first, "a=p,i="+strconv.Itoa(kittyID(Chomp1))+",p=1,c=8,r=4,C=1,q=2") || strings.Contains(first, "a=d") {
		t.Errorf("first placement: %q", first)
	}
	second := m.Show(Chomp2, 1, 1)
	if !strings.Contains(second, "a=d,d=i,i="+strconv.Itoa(kittyID(Chomp1))) {
		t.Errorf("switching frames didn't remove the old placement: %q", second)
	}
	if c := m.Clear(); strings.Count(c, "a=d,d=I") != len(frameNames) {
		t.Errorf("Clear = %q", c)
	}
}

func TestFrameFor(t *testing.T) {
	now := time.Now()
	tests := []struct {
		mood Mood
		tick int
		want string
	}{
		{Mood{}, 0, Chomp1},
		{Mood{}, 1, Chomp2},
		{Mood{Failed: 1, RoarUntil: now.Add(time.Second)}, 1, Roar},
		{Mood{Failed: 1, RoarUntil: now.Add(-time.Second)}, 1, Chomp2},
		{Mood{Done: true}, 3, Happy},
		{Mood{Done: true, Failed: 2}, 0, Curious1},
		{Mood{Done: true, Failed: 2}, 2, Curious2},
	}
	for i, tt := range tests {
		if got := FrameFor(tt.mood, now, tt.tick); got != tt.want {
			t.Errorf("case %d: FrameFor = %s, want %s", i, got, tt.want)
		}
	}
}
