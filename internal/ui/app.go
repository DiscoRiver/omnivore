// Package ui is omnivore's interactive terminal interface, built with Bubble Tea. It displays a
// state.State as it changes, and never modifies it.
package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/discoriver/omnivore/internal/group"
	"github.com/discoriver/omnivore/internal/logging"
	"github.com/discoriver/omnivore/internal/state"
	"github.com/discoriver/omnivore/internal/ui/mascot"
)

type screen int

const (
	groupsScreen screen = iota
	hostScreen
	diffScreen
)

type overlay int

const (
	noOverlay overlay = iota
	searchOverlay
	logOverlay
	quitOverlay
)

const (
	// refreshInterval limits how often a changing state is redrawn.
	refreshInterval = 100 * time.Millisecond
	// tickInterval updates the clock and the log.
	tickInterval = 500 * time.Millisecond
	// headerHeight is the height of the status box.
	headerHeight = 4
	// sidebarWidth is the width of the host list, shown when the terminal is wide enough.
	sidebarWidth       = 28
	minWidthForSidebar = 90
)

// Messages.
type (
	changedMsg      struct{}
	waitMsg         struct{}
	tickMsg         time.Time
	mascotTickMsg   time.Time
	mascotRedrawMsg time.Time
)

// waitForChange returns a command that waits for the state to change.
func waitForChange(st *state.State) tea.Cmd {
	return func() tea.Msg {
		<-st.Changed()
		return changedMsg{}
	}
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func mascotTick() tea.Cmd {
	return tea.Tick(mascot.Tick, func(t time.Time) tea.Msg { return mascotTickMsg(t) })
}

// Mascot layout and timing.
const (
	// mascotCols and mascotRows are the cells the mascot covers, to the left of the header.
	mascotCols = 8
	mascotRows = headerHeight
	// minWidthForMascot is the narrowest terminal that shows it.
	minWidthForMascot = 60
	// mascotRefreshTicks is how often an unchanged frame is redrawn anyway, in case a redraw
	// of the text erased it.
	mascotRefreshTicks = 4
	// mascotRedrawDelay gives the renderer time to paint a new screen before the image goes on it.
	mascotRedrawDelay = 60 * time.Millisecond
)

// model is the root Bubble Tea model.
type model struct {
	state *state.State
	logs  *logging.Buffer

	theme   theme
	keys    keyMap
	help    help.Model
	spinner spinner.Model

	width, height int
	start, now    time.Time
	doneAt        time.Time

	snap    state.Snapshot
	screen  screen
	overlay overlay
	filter  filter

	// selected is the index of the selected group.
	selected    int
	groupStarts []int
	// host is the name of the host shown in the host view.
	host   string
	follow bool

	main     viewport.Model
	logView  viewport.Model
	search   textinput.Model
	matches  []string
	matchSel int
	logLines int
	// status is a short message shown in the footer, such as "copied".
	status string

	// mascot is nil if the terminal can't show images.
	mascot      *mascot.Mascot
	mascotFrame string
	mascotTicks int
	roarUntil   time.Time
	lastFailed  int
}

func newModel(st *state.State, logs *logging.Buffer, bear *mascot.Mascot) *model {
	search := textinput.New()
	search.Placeholder = "host name"
	search.Prompt = "> "
	m := &model{
		state:   st,
		logs:    logs,
		theme:   newTheme(true),
		keys:    newKeyMap(),
		help:    help.New(),
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		start:   time.Now(),
		now:     time.Now(),
		follow:  true,
		main:    viewport.New(),
		logView: viewport.New(),
		search:  search,
		mascot:  bear,
	}
	m.main.MouseWheelEnabled = true
	m.logView.MouseWheelEnabled = true
	m.logView.SoftWrap = true
	m.refresh()
	return m
}

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{waitForChange(m.state), tick(), m.spinner.Tick, tea.RequestBackgroundColor}
	if m.mascot != nil {
		cmds = append(cmds, tea.Raw(m.mascot.Setup()), mascotTick())
	}
	return tea.Batch(cmds...)
}

// Run shows st until the user quits or ctx is cancelled.
func Run(ctx context.Context, st *state.State, logs *logging.Buffer) error {
	bear := mascot.New(mascot.Detect(os.Getenv), mascotCols, mascotRows)
	p := tea.NewProgram(newModel(st, logs, bear), tea.WithContext(ctx))
	_, err := p.Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return err
}

// refresh takes a new snapshot and redraws the content of the views.
func (m *model) refresh() {
	m.snap = m.state.Snapshot()
	if m.snap.Done && m.doneAt.IsZero() {
		m.doneAt = time.Now()
	}
	m.selected = min(m.selected, max(len(m.snap.Groups)-1, 0))
	m.updateContent()
}

func (m *model) elapsed() time.Duration {
	if !m.doneAt.IsZero() {
		return m.doneAt.Sub(m.start)
	}
	return m.now.Sub(m.start)
}

// layout sizes the views to the terminal.
func (m *model) layout() {
	m.help.SetWidth(m.width)
	mainW, bodyH := m.mainWidth(), m.bodyHeight()
	m.main.SetWidth(max(mainW-2, 1))
	m.main.SetHeight(max(bodyH-2, 1))
	// The groups view wraps its own lines, so that it knows where each group starts.
	m.main.SoftWrap = m.screen != groupsScreen
	m.logView.SetWidth(max(m.width*3/4-2, 1))
	m.logView.SetHeight(max(m.height*3/4-2, 1))
	m.search.SetWidth(max(min(m.width-10, 50), 10))
	m.updateContent()
}

func (m *model) sidebarWidth() int {
	if m.width >= minWidthForSidebar {
		return sidebarWidth
	}
	return 0
}

func (m *model) mainWidth() int { return m.width - m.sidebarWidth() }

func (m *model) footer() string {
	keys := m.keys.help(m.screen, m.overlay)
	f := m.help.View(keys)
	// Fall back to the one-line help if the full help would leave no room for the body.
	if m.help.ShowAll && lipgloss.Height(f) > m.height-headerHeight-3 {
		f = m.help.ShortHelpView(keys.ShortHelp())
	}
	if m.status != "" {
		f = m.theme.ok.Render(m.status) + "  " + f
	}
	// The help component doesn't always fit itself to the width, so clip each line.
	lines := strings.Split(f, "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "…")
	}
	return strings.Join(lines, "\n")
}

func (m *model) bodyHeight() int {
	return max(m.height-headerHeight-lipgloss.Height(m.footer()), 3)
}

// updateContent fills the main and log views from the snapshot.
func (m *model) updateContent() {
	switch m.screen {
	case groupsScreen:
		content, starts := m.theme.renderGroups(m.snap.Groups, m.selected, m.main.Width())
		m.main.SetContent(content)
		m.groupStarts = starts
		m.showSelected()
	case diffScreen:
		if len(m.snap.Groups) > 0 {
			m.main.SetContent(m.theme.renderDiff(m.snap.Groups, m.selected))
		}
	case hostScreen:
		h, _ := m.snap.Host(m.host)
		m.main.SetContent(m.theme.renderHost(h))
		if m.follow {
			m.main.GotoBottom()
		}
	}

	lines, total := m.logs.Lines()
	if total != m.logLines {
		m.logLines = total
		atBottom := m.logView.AtBottom()
		m.logView.SetContent(strings.Join(lines, "\n"))
		if atBottom {
			m.logView.GotoBottom()
		}
	}
}

// showSelected scrolls the groups view so that the selected group is visible.
func (m *model) showSelected() {
	if m.selected >= len(m.groupStarts) {
		return
	}
	start := m.groupStarts[m.selected]
	end := m.main.TotalLineCount() - 1
	if m.selected+1 < len(m.groupStarts) {
		end = m.groupStarts[m.selected+1] - 1
	}
	top, h := m.main.YOffset(), m.main.Height()
	switch {
	case start < top:
		m.main.SetYOffset(start)
	case end >= top+h:
		m.main.SetYOffset(min(start, end-h+1))
	}
}

// visibleHosts are the hosts that pass the filter.
func (m *model) visibleHosts() []string {
	return m.snap.Names(m.filter.keep)
}

func (m *model) openHost(name string) {
	if !slices.Contains(m.visibleHosts(), name) {
		m.filter = filterAll
	}
	m.host = name
	m.screen = hostScreen
	m.follow = true
	m.layout()
}

// stepHost moves to the next (+1) or previous (-1) visible host.
func (m *model) stepHost(delta int) {
	hosts := m.visibleHosts()
	if len(hosts) == 0 {
		return
	}
	i := slices.Index(hosts, m.host)
	switch {
	case i < 0 && delta > 0:
		i = 0
	case i < 0:
		i = len(hosts) - 1
	default:
		i = min(max(i+delta, 0), len(hosts)-1)
	}
	m.openHost(hosts[i])
}

func (m *model) setScreen(s screen) {
	m.screen = s
	m.main.SetYOffset(0)
	m.layout()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		// A resize redraws the whole screen, which erases the image: draw it again once the new
		// screen has been painted (this is also how it first appears).
		return m, tea.Tick(mascotRedrawDelay, func(t time.Time) tea.Msg { return mascotRedrawMsg(t) })

	case mascotRedrawMsg:
		m.mascotFrame = ""
		return m, m.animateMascot(time.Time(msg))

	case mascotTickMsg:
		m.mascotTicks++
		return m, tea.Batch(m.animateMascot(time.Time(msg)), mascotTick())

	case tea.BackgroundColorMsg:
		m.theme = newTheme(msg.IsDark())
		m.updateContent()
		return m, nil

	case changedMsg:
		m.refresh()
		// Coalesce bursts of changes into one redraw per interval.
		return m, tea.Tick(refreshInterval, func(time.Time) tea.Msg { return waitMsg{} })

	case waitMsg:
		return m, waitForChange(m.state)

	case tickMsg:
		m.now = time.Time(msg)
		m.updateContent()
		return m, tick()

	case spinner.TickMsg:
		if m.snap.Done {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		m.status = ""
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		var cmd tea.Cmd
		if m.overlay == logOverlay {
			m.logView, cmd = m.logView.Update(msg)
		} else {
			m.main, cmd = m.main.Update(msg)
			m.follow = m.follow && m.main.AtBottom()
		}
		return m, cmd
	}

	if m.overlay == searchOverlay {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := m.keys
	switch m.overlay {
	case quitOverlay:
		switch {
		case key.Matches(msg, k.Confirm):
			return m, m.exit()
		case key.Matches(msg, k.Cancel):
			m.overlay = noOverlay
		}
		return m, nil

	case searchOverlay:
		switch {
		case key.Matches(msg, k.Back):
			m.overlay = noOverlay
			return m, nil
		case key.Matches(msg, k.Open):
			m.overlay = noOverlay
			if m.matchSel < len(m.matches) {
				m.openHost(m.matches[m.matchSel])
			}
			return m, nil
		case key.Matches(msg, k.MatchUp):
			m.matchSel = max(m.matchSel-1, 0)
			return m, nil
		case key.Matches(msg, k.MatchDown):
			m.matchSel = min(m.matchSel+1, max(len(m.matches)-1, 0))
			return m, nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.matches = matchHosts(m.snap.Names(nil), m.search.Value())
		m.matchSel = 0
		return m, cmd

	case logOverlay:
		switch {
		case key.Matches(msg, k.Log, k.Back):
			m.overlay = noOverlay
		case key.Matches(msg, k.Quit):
			return m.quit()
		default:
			var cmd tea.Cmd
			m.logView, cmd = m.logView.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Keys that work on every screen.
	switch {
	case key.Matches(msg, k.Quit):
		return m.quit()
	case key.Matches(msg, k.Help):
		m.help.ShowAll = !m.help.ShowAll
		m.layout()
		return m, nil
	case key.Matches(msg, k.Search):
		m.overlay = searchOverlay
		m.search.Reset()
		m.matches = matchHosts(m.snap.Names(nil), "")
		m.matchSel = 0
		return m, m.search.Focus()
	case key.Matches(msg, k.Log):
		m.overlay = logOverlay
		m.logView.GotoBottom()
		return m, nil
	case key.Matches(msg, k.Filter):
		m.filter = (m.filter + 1) % numFilters
		return m, nil
	case key.Matches(msg, k.FilterBack):
		m.filter = (m.filter + numFilters - 1) % numFilters
		return m, nil
	}

	switch m.screen {
	case groupsScreen:
		switch {
		case key.Matches(msg, k.Up):
			m.selected = max(m.selected-1, 0)
			m.updateContent()
			return m, nil
		case key.Matches(msg, k.Down):
			m.selected = min(m.selected+1, max(len(m.snap.Groups)-1, 0))
			m.updateContent()
			return m, nil
		case key.Matches(msg, k.Open):
			if g, ok := m.selectedGroup(); ok {
				m.openHost(g.Hosts[0])
			}
			return m, nil
		case key.Matches(msg, k.Diff):
			if len(m.snap.Groups) > 0 {
				m.setScreen(diffScreen)
			}
			return m, nil
		case key.Matches(msg, k.Next):
			m.stepHost(+1)
			return m, nil
		case key.Matches(msg, k.Copy):
			return m, m.copyHosts()
		}

	case diffScreen:
		switch {
		case key.Matches(msg, k.Back):
			m.setScreen(groupsScreen)
			return m, nil
		case key.Matches(msg, k.Open):
			if g, ok := m.selectedGroup(); ok {
				m.openHost(g.Hosts[0])
			}
			return m, nil
		case key.Matches(msg, k.Copy):
			return m, m.copyHosts()
		}

	case hostScreen:
		switch {
		case key.Matches(msg, k.Back):
			m.setScreen(groupsScreen)
			return m, nil
		case key.Matches(msg, k.Next):
			m.stepHost(+1)
			return m, nil
		case key.Matches(msg, k.Prev):
			m.stepHost(-1)
			return m, nil
		case key.Matches(msg, k.Follow):
			m.follow = !m.follow
			if m.follow {
				m.main.GotoBottom()
			}
			return m, nil
		case key.Matches(msg, k.Copy):
			m.status = "copied " + m.host
			return m, tea.SetClipboard(m.host)
		}
	}

	// Scrolling, for the host and diff views (the groups view moves by selection).
	var cmd tea.Cmd
	m.main, cmd = m.main.Update(msg)
	if m.screen == hostScreen && !m.main.AtBottom() {
		m.follow = false
	}
	return m, cmd
}

func (m *model) selectedGroup() (group.Group, bool) {
	if m.selected >= len(m.snap.Groups) {
		return group.Group{}, false
	}
	return m.snap.Groups[m.selected], true
}

// copyHosts copies the selected group's hosts to the clipboard, one per line.
func (m *model) copyHosts() tea.Cmd {
	g, ok := m.selectedGroup()
	if !ok {
		return nil
	}
	m.status = "copied " + plural(len(g.Hosts), "host")
	return tea.SetClipboard(strings.Join(g.Hosts, "\n"))
}

// quit stops the UI, asking first if hosts are still running.
func (m *model) quit() (tea.Model, tea.Cmd) {
	if !m.snap.Done {
		m.overlay = quitOverlay
		return m, nil
	}
	return m, m.exit()
}

// exit removes the mascot's images and stops the program.
func (m *model) exit() tea.Cmd {
	if m.mascot == nil {
		return tea.Quit
	}
	return tea.Sequence(tea.Raw(m.mascot.Clear()), tea.Quit)
}

// showMascot reports whether the terminal is wide enough for the mascot.
func (m *model) showMascot() bool {
	return m.mascot != nil && m.width >= minWidthForMascot
}

// animateMascot picks the mascot's frame for the run's state and draws it if it changed. The bear
// chomps while hosts run, roars briefly when one fails, and is happy or puzzled at the end.
func (m *model) animateMascot(now time.Time) tea.Cmd {
	if !m.showMascot() {
		return nil
	}
	c := countHosts(m.snap)
	if c.failed > m.lastFailed && !m.snap.Done {
		m.roarUntil = now.Add(mascot.RoarFor)
	}
	m.lastFailed = c.failed
	frame := mascot.FrameFor(mascot.Mood{Done: m.snap.Done, Failed: c.failed, RoarUntil: m.roarUntil}, now, m.mascotTicks)
	// Redraw now and then even if unchanged, in case a redraw erased it.
	if frame == m.mascotFrame && m.mascotTicks%mascotRefreshTicks != 0 {
		return nil
	}
	m.mascotFrame = frame
	return tea.Raw(m.mascot.Show(frame, 1, 1))
}

func (m *model) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("")
	}
	t := m.theme
	var header string
	if m.showMascot() {
		// Leave the top-left corner blank: the mascot image is drawn over it.
		w := mascotCols + 1
		blank := strings.Repeat(strings.Repeat(" ", w)+"\n", headerHeight-1) + strings.Repeat(" ", w)
		header = lipgloss.JoinHorizontal(lipgloss.Top, blank, t.renderHeader(m.snap, m.width-w, m.spinner.View(), m.elapsed()))
	} else {
		header = t.renderHeader(m.snap, m.width, m.spinner.View(), m.elapsed())
	}

	var title, right string
	switch m.screen {
	case groupsScreen:
		title = fmt.Sprintf("Groups (%d)", len(m.snap.Groups))
	case diffScreen:
		if g, ok := m.selectedGroup(); ok {
			first := m.snap.Groups[0]
			against := "majority"
			if len(first.Hosts) == len(g.Hosts) {
				against = "largest group"
			}
			title = fmt.Sprintf("Diff: %s vs %s (%s)", joinHosts(g.Hosts, 30), against, plural(len(first.Hosts), "host"))
		}
	case hostScreen:
		h, _ := m.snap.Host(m.host)
		title = t.hostTitle(h)
		if m.follow {
			right = t.ok.Render("following ●")
		}
	}
	bodyH := m.bodyHeight()
	body := t.box(title, right, m.main.View(), m.mainWidth(), bodyH, true)
	if w := m.sidebarWidth(); w > 0 {
		side := t.box("Hosts", "", t.renderSidebar(m.snap, m.filter, m.currentHost()), w, bodyH, false)
		body = lipgloss.JoinHorizontal(lipgloss.Top, side, body)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, m.footer())

	if m.overlay != noOverlay {
		modal := m.renderOverlay()
		mw, mh := lipgloss.Size(modal)
		content = lipgloss.NewCompositor(
			lipgloss.NewLayer(content),
			lipgloss.NewLayer(modal).X(max((m.width-mw)/2, 0)).Y(max((m.height-mh)/3, 0)).Z(1),
		).Render()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "omnivore"
	// The terminal's own progress indicator (tab or taskbar), where supported.
	switch c := countHosts(m.snap); {
	case m.snap.Done || c.total == 0:
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarNone, 0)
	case c.failed > 0:
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarWarning, c.done*100/c.total)
	default:
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarDefault, c.done*100/c.total)
	}
	return v
}

// currentHost is the host to mark in the sidebar.
func (m *model) currentHost() string {
	if m.screen == hostScreen {
		return m.host
	}
	return ""
}

func (m *model) renderOverlay() string {
	t := m.theme
	switch m.overlay {
	case searchOverlay:
		w := m.search.Width() + 6
		lines := []string{m.search.View()}
		for i, h := range m.matches {
			marker := "  "
			if i == m.matchSel {
				marker = t.selected.Render("▸ ")
			}
			status := ""
			if host, ok := m.snap.Host(h); ok {
				status = t.statusLabel(host)
			}
			lines = append(lines, marker+h+"  "+status)
		}
		if len(m.matches) == 0 {
			lines = append(lines, t.dim.Render("  no matching hosts"))
		}
		return t.box("Go to host", "", strings.Join(lines, "\n"), w, len(lines)+2, true)
	case logOverlay:
		return t.box("Log", "", m.logView.View(), m.logView.Width()+2, m.logView.Height()+2, true)
	case quitOverlay:
		c := countHosts(m.snap)
		n := c.running + c.slow
		verb := "are"
		if n == 1 {
			verb = "is"
		}
		text := fmt.Sprintf("%s %s still running.\nQuit and stop them?\n\n%s  %s",
			plural(n, "host"), verb, t.fail.Render("y  quit"), t.dim.Render("n  stay"))
		return t.box("Quit?", "", text, 34, 7, true)
	}
	return ""
}

// plural returns "1 host", "2 hosts" and so on.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
