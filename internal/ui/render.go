package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/aymanbagabas/go-udiff"
	"github.com/charmbracelet/x/ansi"
	"github.com/discoriver/omnivore/internal/group"
	"github.com/discoriver/omnivore/internal/state"
)

// The functions in this file are pure: they turn a snapshot into text, so they can be tested
// without a terminal.

// sanitize makes remote output safe to draw: it removes terminal escape sequences (which a host
// could use to change the window title, write the clipboard or redraw the screen), expands tabs,
// and drops other control characters.
func sanitize(b []byte) string {
	s := ansi.Strip(string(b))
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			out.WriteRune(r)
		case r == '\t':
			out.WriteString("    ")
		case unicode.IsControl(r):
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// counts are how many hosts are in each state.
type counts struct {
	total, done, completed, failed, slow, running int
}

func countHosts(snap state.Snapshot) counts {
	c := counts{total: len(snap.Hosts)}
	for _, h := range snap.Hosts {
		switch {
		case h.Status == state.Completed:
			c.completed++
		case h.Status == state.Failed:
			c.failed++
		case h.Slow:
			c.slow++
		default:
			c.running++
		}
	}
	c.done = c.completed + c.failed
	return c
}

func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	return fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

// progressBar draws a bar of width cells, filled in proportion to done/total.
func (t theme) progressBar(done, total, width int) string {
	if width <= 0 {
		return ""
	}
	filled := 0
	if total > 0 {
		filled = done * width / total
	}
	return t.barFull.Render(strings.Repeat("█", filled)) + t.barEmpty.Render(strings.Repeat("░", width-filled))
}

// renderHeader draws the status box at the top of the screen.
func (t theme) renderHeader(snap state.Snapshot, width int, spin string, elapsed time.Duration) string {
	c := countHosts(snap)
	status := t.running.Render(spin + " running")
	if snap.Done {
		status = t.ok.Render("✓ finished")
	}
	stats := fmt.Sprintf("%d/%d   %s  %s  %s   %s",
		c.done, c.total,
		t.ok.Render(fmt.Sprintf("✓ %d", c.completed)),
		t.fail.Render(fmt.Sprintf("✗ %d", c.failed)),
		t.slow.Render(fmt.Sprintf("◷ %d slow", c.slow)),
		t.dim.Render(formatElapsed(elapsed)))
	barW := min(max(width-2-ansi.StringWidth(status)-ansi.StringWidth(stats)-4, 0), 40)
	line1 := status + "  " + t.progressBar(c.done, c.total, barW) + "  " + stats
	line2 := t.dim.Render("$ ") + sanitize([]byte(snap.Command))
	return t.box("omnivore", "", line1+"\n"+line2, width, headerHeight, false)
}

// joinHosts lists hosts in at most width cells, ending with "… +N" if they don't all fit.
func joinHosts(hosts []string, width int) string {
	var b strings.Builder
	used := 0
	for i, h := range hosts {
		sep := ""
		if i > 0 {
			sep = ", "
		}
		more := ""
		if rest := len(hosts) - i - 1; rest > 0 {
			more = fmt.Sprintf(", … +%d", rest)
		}
		w := ansi.StringWidth(sep + h)
		if i > 0 && used+w+ansi.StringWidth(more) > width {
			fmt.Fprintf(&b, " … +%d", len(hosts)-i)
			return b.String()
		}
		b.WriteString(sep + h)
		used += w
	}
	return b.String()
}

// previewLines is how many lines of each group's output the groups list shows.
const previewLines = 8

// renderGroups draws the list of groups, with the selected one highlighted, in lines of at most
// width cells. It returns the text and the line on which each group starts.
func (t theme) renderGroups(groups []group.Group, selected, width int) (string, []int) {
	if len(groups) == 0 {
		return t.dim.Render("Waiting for the first host to finish…"), nil
	}
	var lines []string
	starts := make([]int, len(groups))
	for i, g := range groups {
		starts[i] = len(lines)
		marker, countStyle := "  ", t.title
		if i == selected {
			marker, countStyle = t.selected.Render("▸ "), t.selected
		}
		tag := ""
		// Only call it different from the majority if the first group really is bigger.
		if i > 0 && len(g.Hosts) < len(groups[0].Hosts) {
			tag = t.slow.Render("≠ majority")
		}
		count := fmt.Sprintf("%d host", len(g.Hosts))
		if len(g.Hosts) != 1 {
			count += "s"
		}
		hostsW := width - 2 - len(count) - 2 - ansi.StringWidth(tag) - 2
		hosts := joinHosts(g.Hosts, max(hostsW, 10))
		head := marker + countStyle.Render(count) + "  " + t.accent.Render(hosts)
		if tag != "" {
			gap := width - ansi.StringWidth(head) - ansi.StringWidth(tag)
			head += strings.Repeat(" ", max(gap, 2)) + tag
		}
		lines = append(lines, head)

		// Wrap long lines rather than cutting them off.
		text := ansi.Hardwrap(strings.TrimSuffix(sanitize(g.Output), "\n"), max(width-2, 10), true)
		out := strings.Split(text, "\n")
		if len(g.Output) == 0 {
			out = []string{t.dim.Render("(no output)")}
		}
		shown := out
		if len(out) > previewLines {
			shown = out[:previewLines]
		}
		for _, l := range shown {
			lines = append(lines, "  "+l)
		}
		if len(out) > previewLines {
			lines = append(lines, t.dim.Render(fmt.Sprintf("  … %d more lines (enter to open)", len(out)-previewLines)))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n"), starts
}

// renderDiff shows how the selected group's output differs from the first (largest) group's.
func (t theme) renderDiff(groups []group.Group, selected int) string {
	if selected == 0 {
		return t.dim.Render("This is the majority group, so there is nothing to compare it with.\nSelect another group and press d.")
	}
	before, after := sanitize(groups[0].Output), sanitize(groups[selected].Output)
	unified := udiff.Unified("majority", "this group", before, after)
	if unified == "" {
		return t.dim.Render("The outputs are identical.")
	}
	var lines []string
	for l := range strings.Lines(unified) {
		l = strings.TrimSuffix(l, "\n")
		switch {
		case strings.HasPrefix(l, "---"), strings.HasPrefix(l, "+++"):
			continue
		case strings.HasPrefix(l, "@@"):
			lines = append(lines, t.dim.Render(l))
		case strings.HasPrefix(l, "+"):
			lines = append(lines, t.added.Render(l))
		case strings.HasPrefix(l, "-"):
			lines = append(lines, t.removed.Render(l))
		default:
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}

// filter selects which hosts the sidebar shows and the host view pages through.
type filter int

const (
	filterAll filter = iota
	filterRunning
	filterSlow
	filterFailed
	filterCompleted
	numFilters
)

var filterNames = [numFilters]string{"All", "Run", "Slow", "Fail", "Ok"}

func (f filter) keep(h state.Host) bool {
	switch f {
	case filterRunning:
		return h.Status == state.Pending && !h.Slow
	case filterSlow:
		return h.Status == state.Pending && h.Slow
	case filterFailed:
		return h.Status == state.Failed
	case filterCompleted:
		return h.Status == state.Completed
	}
	return true
}

// renderSidebar lists hosts by status, marking current.
func (t theme) renderSidebar(snap state.Snapshot, f filter, current string) string {
	var tabs []string
	for i, name := range filterNames {
		if filter(i) == f {
			tabs = append(tabs, t.selected.Render("["+name+"]"))
		} else {
			tabs = append(tabs, t.dim.Render(name))
		}
	}
	lines := []string{strings.Join(tabs, " "), ""}

	sections := []struct {
		name  string
		icon  string
		style func(...string) string
		f     filter
	}{
		{"Running", "●", t.running.Render, filterRunning},
		{"Slow", "◷", t.slow.Render, filterSlow},
		{"Failed", "✗", t.fail.Render, filterFailed},
		{"Completed", "✓", t.ok.Render, filterCompleted},
	}
	for _, sec := range sections {
		if f != filterAll && f != sec.f {
			continue
		}
		names := snap.Names(sec.f.keep)
		lines = append(lines, sec.style(fmt.Sprintf("%s %-10s %d", sec.icon, sec.name, len(names))))
		for _, n := range names {
			marker := "  "
			if n == current {
				marker = t.selected.Render("▸ ")
			}
			lines = append(lines, marker+n)
		}
	}
	return strings.Join(lines, "\n")
}

// renderHost shows a host's output, followed by its error if it failed.
func (t theme) renderHost(h state.Host) string {
	out := sanitize(h.Result())
	if out == "" {
		if h.Status == state.Pending {
			return t.dim.Render("(no output yet)")
		}
		return t.dim.Render("(no output)")
	}
	return out
}

// hostTitle describes a host for the host view's title.
func (t theme) hostTitle(h state.Host) string {
	parts := []string{h.Name}
	switch h.Status {
	case state.Completed:
		parts = append(parts, t.ok.Render("completed"))
	case state.Failed:
		parts = append(parts, t.fail.Render("failed"))
	default:
		parts = append(parts, t.running.Render("running"))
	}
	if h.Slow {
		parts = append(parts, t.slow.Render("slow"))
	}
	return strings.Join(parts, " · ")
}

// statusLabel is a short, coloured description of a host's status.
func (t theme) statusLabel(h state.Host) string {
	switch {
	case h.Status == state.Completed:
		return t.ok.Render("✓ completed")
	case h.Status == state.Failed:
		return t.fail.Render("✗ failed")
	case h.Slow:
		return t.slow.Render("◷ slow")
	}
	return t.running.Render("● running")
}

// maxMatches is how many hosts the search box lists.
const maxMatches = 8

// matchHosts returns up to maxMatches hosts matching query, best first: names starting with it,
// then containing it, then containing its characters in order. Matching ignores case.
func matchHosts(hosts []string, query string) []string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return hosts[:min(len(hosts), maxMatches)]
	}
	type match struct {
		host  string
		score int
	}
	var matches []match
	for _, h := range hosts {
		l := strings.ToLower(h)
		switch {
		case strings.HasPrefix(l, q):
			matches = append(matches, match{h, 0})
		case strings.Contains(l, q):
			matches = append(matches, match{h, 1})
		case subsequence(l, q):
			matches = append(matches, match{h, 2})
		}
	}
	slices.SortStableFunc(matches, func(a, b match) int { return a.score - b.score })
	out := make([]string, 0, min(len(matches), maxMatches))
	for _, m := range matches[:min(len(matches), maxMatches)] {
		out = append(out, m.host)
	}
	return out
}

func subsequence(s, sub string) bool {
	for _, r := range sub {
		i := strings.IndexRune(s, r)
		if i < 0 {
			return false
		}
		s = s[i+utf8.RuneLen(r):]
	}
	return true
}
