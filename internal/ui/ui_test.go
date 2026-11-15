package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/discoriver/omnivore/internal/group"
	"github.com/discoriver/omnivore/internal/logging"
	"github.com/discoriver/omnivore/internal/state"
)

// fixture is a run of 12 hosts: 9 agree, 2 differ, 1 failed and 1 still running (slow).
func fixture(t *testing.T) *state.State {
	t.Helper()
	var hosts []string
	for i := 1; i <= 12; i++ {
		hosts = append(hosts, fmt.Sprintf("web%02d", i))
	}
	st := state.New("systemctl is-active nginx", hosts)
	for _, h := range hosts[:9] {
		st.AppendStdout(h, []byte("active\n"))
		st.Finish(h, nil)
	}
	st.AppendStdout("web10", []byte("inactive\n"))
	st.Finish("web10", errors.New("exit 3"))
	st.AppendStdout("web11", []byte("activating\n"))
	st.Finish("web11", nil)
	st.AppendStdout("web12", []byte("still going\n"))
	st.MarkSlow("web12")
	return st
}

func newTestModel(t *testing.T, w, h int) *model {
	t.Helper()
	m := newModel(fixture(t), logging.NewBuffer(10))
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

func press(m *model, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEsc}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "up":
			msg = tea.KeyPressMsg{Code: tea.KeyUp}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		case "left":
			msg = tea.KeyPressMsg{Code: tea.KeyLeft}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		default:
			r := []rune(k)[0]
			msg = tea.KeyPressMsg{Code: r, Text: k}
		}
		m.Update(msg)
	}
}

func screenText(m *model) string { return ansi.Strip(m.View().Content) }

// TestViewFits checks that every screen fills the terminal exactly, at many sizes.
func TestViewFits(t *testing.T) {
	for _, size := range [][2]int{{140, 40}, {100, 30}, {80, 24}, {60, 15}, {40, 12}} {
		w, h := size[0], size[1]
		for _, keys := range [][]string{nil, {"d"}, {"enter"}, {"/"}, {"l"}, {"q"}, {"?"}} {
			m := newTestModel(t, w, h)
			press(m, keys...)
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != h {
				t.Errorf("%dx%d after %v: %d lines, want %d", w, h, keys, len(lines), h)
			}
			for i, l := range lines {
				if lw := ansi.StringWidth(l); lw > w {
					t.Errorf("%dx%d after %v: line %d is %d wide", w, h, keys, i, lw)
				}
			}
		}
	}
}

func TestGroupsScreen(t *testing.T) {
	m := newTestModel(t, 140, 40)
	s := screenText(m)
	for _, want := range []string{
		"9 hosts  web01, web02", "active", "≠ majority", "inactive", "[error: exit 3]", "Groups (3)",
		"11/12", "◷ 1 slow", "$ systemctl is-active nginx",
		"[All]", "Slow", "web12",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("groups screen is missing %q:\n%s", want, s)
		}
	}
	// Narrow terminals drop the sidebar.
	if s := screenText(newTestModel(t, 70, 30)); strings.Contains(s, "[All]") {
		t.Error("sidebar shown on a narrow terminal")
	}
}

func TestDiffScreen(t *testing.T) {
	m := newTestModel(t, 140, 40)
	press(m, "d")
	if s := screenText(m); !strings.Contains(s, "majority group") {
		t.Errorf("diff of the majority group:\n%s", s)
	}
	press(m, "esc", "down", "d")
	s := screenText(m)
	if !strings.Contains(s, "-active") || !strings.Contains(s, "+activating") {
		t.Errorf("diff screen:\n%s", s)
	}
	press(m, "enter")
	if m.screen != hostScreen || m.host != "web11" {
		t.Errorf("enter on diff opened %v %q", m.screen, m.host)
	}
}

func TestHostNavigation(t *testing.T) {
	m := newTestModel(t, 140, 40)
	press(m, "enter")
	if m.screen != hostScreen || m.host != "web01" {
		t.Fatalf("enter opened %v %q", m.screen, m.host)
	}
	if s := screenText(m); !strings.Contains(s, "web01 · completed") || !strings.Contains(s, "active") {
		t.Errorf("host screen:\n%s", s)
	}
	press(m, "right", "right", "left")
	if m.host != "web02" {
		t.Errorf("after right, right, left: host %q", m.host)
	}
	// With the Fail filter, paging only visits failed hosts.
	press(m, "tab", "tab", "tab", "right")
	if m.filter != filterFailed || m.host != "web10" {
		t.Errorf("filter %v, host %q; want failed hosts only", m.filter, m.host)
	}
	press(m, "esc")
	if m.screen != groupsScreen {
		t.Error("esc didn't go back")
	}
}

func TestSlowHostFollowsOutput(t *testing.T) {
	m := newTestModel(t, 140, 40)
	press(m, "/")
	for _, r := range "w12" {
		press(m, string(r))
	}
	if len(m.matches) == 0 || m.matches[0] != "web12" {
		t.Fatalf("matches for w12 = %v", m.matches)
	}
	press(m, "enter")
	if m.host != "web12" || !m.follow {
		t.Fatalf("host %q follow %v", m.host, m.follow)
	}
	if s := screenText(m); !strings.Contains(s, "running · slow") || !strings.Contains(s, "following") {
		t.Errorf("slow host screen:\n%s", s)
	}
	press(m, "f")
	if m.follow {
		t.Error("f didn't stop following")
	}
}

func TestQuitConfirmation(t *testing.T) {
	m := newTestModel(t, 100, 30)
	press(m, "q")
	if m.overlay != quitOverlay || !strings.Contains(screenText(m), "still running") {
		t.Fatal("quit with running hosts didn't ask")
	}
	press(m, "n")
	if m.overlay != noOverlay {
		t.Fatal("n didn't cancel")
	}
	press(m, "q")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("y didn't quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("y didn't return tea.Quit")
	}
}

func TestSanitize(t *testing.T) {
	in := "ok\x1b]0;pwned\x07 \x1b[31mred\x1b[0m\tx\r\nnext\x00\n"
	if got, want := sanitize([]byte(in)), "ok red    x\nnext\n"; got != want {
		t.Errorf("sanitize = %q, want %q", got, want)
	}
}

func TestMatchHosts(t *testing.T) {
	hosts := []string{"db1", "web01", "web02", "xweb", "w-e-b"}
	got := matchHosts(hosts, "web")
	want := []string{"web01", "web02", "xweb", "w-e-b"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("matchHosts(web) = %v, want %v", got, want)
	}
	if got := matchHosts(hosts, "zzz"); len(got) != 0 {
		t.Errorf("matchHosts(zzz) = %v", got)
	}
}

func TestJoinHosts(t *testing.T) {
	hosts := []string{"web01", "web02", "web03", "web04", "web05"}
	if got := joinHosts(hosts, 100); got != "web01, web02, web03, web04, web05" {
		t.Errorf("wide: %q", got)
	}
	if got := joinHosts(hosts, 22); got != "web01, web02 … +3" {
		t.Errorf("narrow: %q", got)
	}
}

func TestRenderGroupsPreview(t *testing.T) {
	out := strings.Repeat("line\n", 20)
	content, starts := newTheme(true).renderGroups([]group.Group{{Output: []byte(out), Hosts: []string{"a"}}}, 0, 80)
	if !strings.Contains(content, "12 more lines") || len(starts) != 1 {
		t.Errorf("preview:\n%s", content)
	}
}

func TestMajorityTagAndWrapping(t *testing.T) {
	th := newTheme(true)
	long := strings.Repeat("x", 150) + "END\n"
	tied := []group.Group{{Output: []byte("a\n"), Hosts: []string{"h1"}}, {Output: []byte(long), Hosts: []string{"h2"}}}
	content, _ := th.renderGroups(tied, 0, 60)
	plain := ansi.Strip(content)
	if strings.Contains(plain, "majority") {
		t.Errorf("tied groups tagged as differing from the majority:\n%s", plain)
	}
	if !strings.Contains(plain, "END") {
		t.Errorf("long line was cut off instead of wrapped:\n%s", plain)
	}
	for _, l := range strings.Split(plain, "\n") {
		if ansi.StringWidth(l) > 60 {
			t.Errorf("line wider than 60: %q", l)
		}
	}
}
