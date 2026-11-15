// Package ui is omnivore's interactive terminal interface. It displays a state.State as it
// changes, and never modifies it.
package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
	"github.com/discoriver/omnivore/internal/logging"
	"github.com/discoriver/omnivore/internal/state"
	"github.com/fatih/color"
)

// View names.
const (
	statusView   = "status"
	todoView     = "todo"
	completeView = "complete"
	outputView   = "output"
	hostView     = "host"
	failedView   = "failed"
	slowView     = "slow"
	commandView  = "command"
	controlsView = "controls"
	logView      = "log"
	searchView   = "search-host"
)

const (
	controlsOutput = "QUIT (CTRL+C) - SHOW/HIDE LOG (CTRL+L) - NEXT HOST (right) - SEARCH HOST (s)"
	controlsHost   = "QUIT (CTRL+C) - SHOW/HIDE LOG (CTRL+L) - NEXT HOST (right) - PREV HOST (left) - BACK (esc)"
	controlsSearch = "QUIT (CTRL+C) - GO TO HOST (enter) - BACK (esc)"
)

// refreshInterval limits how often the screen is redrawn.
const refreshInterval = 100 * time.Millisecond

var (
	magenta = color.New(color.FgMagenta).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
)

type ui struct {
	g     *gocui.Gui
	state *state.State
	logs  *logging.Buffer

	// hosts is the fixed, sorted list of hosts, for paging through host views.
	hosts []string
	// host is the index in hosts of the host shown in the host view, or -1 for the groups view.
	host     int
	showLog  bool
	controls string
	laidOut  bool
}

// Run shows st until the user quits or ctx is cancelled. It must be called from the main
// goroutine.
func Run(ctx context.Context, st *state.State, logs *logging.Buffer) error {
	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		return fmt.Errorf("start UI: %w", err)
	}
	defer g.Close()

	u := &ui{
		g:        g,
		state:    st,
		logs:     logs,
		hosts:    st.Snapshot().Names(func(state.Host) bool { return true }),
		host:     -1,
		controls: controlsOutput,
	}
	g.SetManagerFunc(u.layout)
	if err := u.bindKeys(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go u.refreshLoop(ctx)

	if err := g.MainLoop(); err != nil && !errors.Is(err, gocui.ErrQuit) {
		return err
	}
	return nil
}

// refreshLoop redraws the screen when the state or the log changes, at most every
// refreshInterval, and stops the UI when ctx is cancelled.
func (u *ui) refreshLoop(ctx context.Context) {
	tick := time.NewTicker(refreshInterval)
	defer tick.Stop()
	dirty, logged := true, -1
	for {
		select {
		case <-ctx.Done():
			u.g.Update(func(*gocui.Gui) error { return gocui.ErrQuit })
			return
		case <-u.state.Changed():
			dirty = true
		case <-tick.C:
			if _, n := u.logs.Lines(); dirty || n != logged {
				dirty, logged = false, n
				u.g.Update(u.render)
			}
		}
	}
}

// render redraws every view from a snapshot of the state.
func (u *ui) render(g *gocui.Gui) error {
	snap := u.state.Snapshot()

	if v, err := g.View(statusView); err == nil {
		v.Clear()
		pending := len(snap.Names(func(h state.Host) bool { return h.Status == state.Pending }))
		status := fmt.Sprintf("Omnivore - %d hosts - %d running - %d groups", len(snap.Hosts), pending, len(snap.Groups))
		if snap.Done {
			status += " - finished"
		}
		fmt.Fprint(v, status)
	}

	list := func(name string, colour func(...any) string, keep func(state.Host) bool) {
		if v, err := g.View(name); err == nil {
			v.Clear()
			for _, h := range snap.Names(keep) {
				fmt.Fprintln(v, colour(h))
			}
		}
	}
	list(todoView, green, func(h state.Host) bool { return h.Status == state.Pending && !h.Slow })
	list(slowView, yellow, func(h state.Host) bool { return h.Status == state.Pending && h.Slow })
	list(completeView, green, func(h state.Host) bool { return h.Status == state.Completed })
	list(failedView, red, func(h state.Host) bool { return h.Status == state.Failed })

	if v, err := g.View(outputView); err == nil {
		v.Clear()
		for _, grp := range snap.Groups {
			fmt.Fprintf(v, "Hosts (%d): %s\n%s\n\n", len(grp.Hosts), yellow(strings.Join(grp.Hosts, ", ")),
				magenta(strings.TrimSuffix(string(grp.Output), "\n")))
		}
	}

	if v, err := g.View(hostView); err == nil && u.host >= 0 {
		v.Clear()
		name := u.hosts[u.host]
		h, _ := snap.Host(name)
		v.Title = fmt.Sprintf("%s (%s)", name, h.Status)
		fmt.Fprint(v, magenta(string(h.Result())))
	}

	if v, err := g.View(commandView); err == nil {
		v.Clear()
		fmt.Fprint(v, red(snap.Command))
	}

	if v, err := g.View(controlsView); err == nil {
		v.Clear()
		fmt.Fprint(v, u.controls)
	}

	if v, err := g.View(logView); err == nil {
		v.Clear()
		lines, _ := u.logs.Lines()
		fmt.Fprint(v, strings.Join(lines, "\n"))
	}
	return nil
}

func (u *ui) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	views := []struct {
		name           string
		x0, y0, x1, y1 int
		title          string
		setup          func(*gocui.View)
	}{
		{statusView, 0, 0, maxX - 1, maxY/20 + 1, "Status", nil},
		{todoView, 0, maxY/20 + 2, maxX/10 - 1, maxY/2 - 1, "Todo", nil},
		{completeView, 0, maxY / 2, maxX/10 - 1, maxY - 5, "Complete", nil},
		{hostView, maxX / 10, maxY/20 + 2, maxX/10*9 - 1, maxY - 5, "Host", nil},
		{outputView, maxX / 10, maxY/20 + 2, maxX/10*9 - 1, maxY - 5, "Output", nil},
		{failedView, maxX / 10 * 9, maxY/20 + 2, maxX - 1, maxY/2 - 1, "Failed", nil},
		{slowView, maxX / 10 * 9, maxY / 2, maxX - 1, maxY - 5, "Slow", nil},
		{commandView, 0, maxY - 4, maxX/2 - 1, maxY - 1, "Command", nil},
		{controlsView, maxX / 2, maxY - 4, maxX - 1, maxY - 1, "Controls", nil},
		{logView, maxX / 8, maxY / 8, maxX - maxX/8, maxY - maxY/8, "Log", func(v *gocui.View) { v.Autoscroll = true }},
		{searchView, maxX/2 - 20, maxY / 3, maxX/2 + 20, maxY/3 + 2, "Go to host", func(v *gocui.View) {
			v.Editable = true
			v.Wrap = false
		}},
	}
	for _, vw := range views {
		v, err := g.SetView(vw.name, vw.x0, vw.y0, vw.x1, vw.y1, 0)
		if err == nil {
			continue
		}
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		v.Title = vw.title
		v.Wrap = true
		if vw.setup != nil {
			vw.setup(v)
		}
	}

	if !u.laidOut {
		u.laidOut = true
		// Views are stacked in creation order; hide the overlays and start on the groups view.
		for _, name := range []string{searchView, logView, hostView} {
			if _, err := g.SetViewOnBottom(name); err != nil {
				return err
			}
		}
		if _, err := g.SetCurrentView(outputView); err != nil {
			return err
		}
	}
	return nil
}

func (u *ui) bindKeys() error {
	type binding struct {
		view    string
		key     any
		handler func(*gocui.Gui, *gocui.View) error
	}
	bindings := []binding{
		{"", gocui.KeyCtrlC, func(*gocui.Gui, *gocui.View) error { return gocui.ErrQuit }},
		{"", gocui.KeyCtrlL, u.toggleLog},
		{outputView, gocui.KeyArrowRight, u.nextHost},
		{hostView, gocui.KeyArrowRight, u.nextHost},
		{hostView, gocui.KeyArrowLeft, u.prevHost},
		{hostView, gocui.KeyEsc, u.showGroups},
		{outputView, 's', u.openSearch},
		{searchView, gocui.KeyEsc, u.closeSearch},
		{searchView, gocui.KeyEnter, u.search},
	}
	for _, b := range bindings {
		if err := u.g.SetKeybinding(b.view, b.key, gocui.ModNone, b.handler); err != nil {
			return err
		}
	}
	return nil
}

func (u *ui) toggleLog(g *gocui.Gui, _ *gocui.View) error {
	u.showLog = !u.showLog
	var err error
	if u.showLog {
		_, err = g.SetViewOnTop(logView)
	} else {
		_, err = g.SetViewOnBottom(logView)
	}
	return err
}

func (u *ui) showHost(g *gocui.Gui, i int) error {
	if len(u.hosts) == 0 {
		return nil
	}
	u.host = min(max(i, 0), len(u.hosts)-1)
	u.controls = controlsHost
	if _, err := g.SetViewOnTop(hostView); err != nil {
		return err
	}
	if _, err := g.SetCurrentView(hostView); err != nil {
		return err
	}
	return u.render(g)
}

func (u *ui) nextHost(g *gocui.Gui, _ *gocui.View) error { return u.showHost(g, u.host+1) }
func (u *ui) prevHost(g *gocui.Gui, _ *gocui.View) error { return u.showHost(g, u.host-1) }

func (u *ui) showGroups(g *gocui.Gui, _ *gocui.View) error {
	u.host = -1
	u.controls = controlsOutput
	if _, err := g.SetViewOnTop(outputView); err != nil {
		return err
	}
	if _, err := g.SetCurrentView(outputView); err != nil {
		return err
	}
	return u.render(g)
}

func (u *ui) openSearch(g *gocui.Gui, _ *gocui.View) error {
	u.controls = controlsSearch
	g.Cursor = true
	if _, err := g.SetViewOnTop(searchView); err != nil {
		return err
	}
	_, err := g.SetCurrentView(searchView)
	return err
}

func (u *ui) closeSearch(g *gocui.Gui, v *gocui.View) error {
	g.Cursor = false
	v.Clear()
	v.SetCursor(0, 0)
	if _, err := g.SetViewOnBottom(searchView); err != nil {
		return err
	}
	return u.showGroups(g, v)
}

func (u *ui) search(g *gocui.Gui, v *gocui.View) error {
	name := strings.TrimSpace(v.Buffer())
	if err := u.closeSearch(g, v); err != nil {
		return err
	}
	if i, ok := slices.BinarySearch(u.hosts, name); ok {
		return u.showHost(g, i)
	}
	return nil
}
