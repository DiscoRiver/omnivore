package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// theme holds the styles, adapted to the terminal's background.
type theme struct {
	border      lipgloss.Style
	borderFocus lipgloss.Style
	title       lipgloss.Style
	dim         lipgloss.Style
	ok          lipgloss.Style
	fail        lipgloss.Style
	slow        lipgloss.Style
	running     lipgloss.Style
	accent      lipgloss.Style
	selected    lipgloss.Style
	added       lipgloss.Style
	removed     lipgloss.Style
	barFull     lipgloss.Style
	barEmpty    lipgloss.Style
}

func newTheme(dark bool) theme {
	ld := lipgloss.LightDark(dark)
	c := func(light, dark string) color.Color { return ld(lipgloss.Color(light), lipgloss.Color(dark)) }
	fg := func(light, dark string) lipgloss.Style { return lipgloss.NewStyle().Foreground(c(light, dark)) }
	accent := fg("#8250df", "#bc8cff")
	return theme{
		border:      fg("#afb8c1", "#484f58"),
		borderFocus: accent,
		title:       lipgloss.NewStyle().Bold(true),
		dim:         fg("#6e7781", "#8b949e"),
		ok:          fg("#1a7f37", "#3fb950"),
		fail:        fg("#cf222e", "#f85149"),
		slow:        fg("#9a6700", "#d29922"),
		running:     fg("#0969da", "#58a6ff"),
		accent:      accent,
		selected:    accent.Bold(true),
		added:       fg("#1a7f37", "#3fb950"),
		removed:     fg("#cf222e", "#f85149"),
		barFull:     fg("#1a7f37", "#3fb950"),
		barEmpty:    fg("#d0d7de", "#30363d"),
	}
}

// box draws body in a w×h rounded box with title on the top border, and right aligned at the
// right of it. Lines of body are truncated or padded to fit, so the result is exactly w×h.
func (t theme) box(title, right, body string, w, h int, focused bool) string {
	if w < 4 || h < 2 {
		return ""
	}
	bs := t.border
	if focused {
		bs = t.borderFocus
	}
	inner := w - 2

	var top strings.Builder
	top.WriteString(bs.Render("╭"))
	used := 0
	if title != "" {
		title = ansi.Truncate(title, max(inner-4, 1), "…")
		top.WriteString(bs.Render("─ ") + t.title.Render(title) + " ")
		used = 3 + ansi.StringWidth(title)
	}
	rightW := ansi.StringWidth(right)
	if right != "" && used+rightW+3 > inner {
		right, rightW = "", 0
	}
	fill := inner - used
	if right != "" {
		fill -= rightW + 3
	}
	top.WriteString(bs.Render(strings.Repeat("─", max(fill, 0))))
	if right != "" {
		top.WriteString(" " + right + bs.Render(" ─"))
	}
	top.WriteString(bs.Render("╮"))

	lines := strings.Split(body, "\n")
	out := make([]string, 0, h)
	out = append(out, top.String())
	side := bs.Render("│")
	for i := range h - 2 {
		line := ""
		if i < len(lines) {
			line = ansi.Truncate(lines[i], inner, "…")
		}
		out = append(out, side+line+strings.Repeat(" ", max(inner-ansi.StringWidth(line), 0))+side)
	}
	out = append(out, bs.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return strings.Join(out, "\n")
}
