package ui

import (
	"fmt"
	"github.com/awesome-gocui/gocui"
	"github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/ossh"
	"github.com/discoriver/omnivore/internal/store"
	"github.com/discoriver/omnivore/pkg/group"
	"github.com/fatih/color"
	"strings"
)

var (
	Collective *InterfaceCollective

	magenta = color.New(color.FgMagenta).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()

	logShowing    = false
	isStarted     = false
	hostViewIndex = 0

	// Keybinds
	exitStandard                 = gocui.KeyCtrlC
	controlsString               = ""
	controlsStringOutputView     = "QUIT (CTRL+C) - SHOW/HIDE LOG (CTRL+L) - Next Host (right)"
	controlsStringHostSearchView = "QUIT (CTRL+C) - SEARCH (enter) - BACK (esc)"
	controlsStringHostView       = "QUIT (CTRL+C) - SHOW/HIDE LOG (CTRL+L) - Next Host (right) - Prev Host (left) - BACK (esc)"
)

// InterfaceCollective are the values required for UI rendering and updates.
type InterfaceCollective struct {
	Group       *group.ValueGrouping
	StreamCycle *ossh.StreamCycle

	UI *gocui.Gui
}

// MakeCollective initialised a new InterfaceCollective to ui.Collective
func MakeCollective() {
	Collective = &InterfaceCollective{}
	Collective.Group = group.NewValueGrouping()
}

func (data *InterfaceCollective) StartUI(started chan struct{}) {
	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	data.UI = g

	data.UI.SetManagerFunc(data.layout)

	controlsString = controlsStringOutputView

	if err := data.setKeybinds(); err != nil {
		panic(err)
	}

	// Report that it's safe to refresh.
	started <- struct{}{}

	if err := data.UI.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.OmniLog.Fatal("%s", err)
	}
}

func (data *InterfaceCollective) setKeybinds() error {
	var err error

	// Normal exit of CTRL+C
	if err = data.UI.SetKeybinding("", exitStandard, gocui.ModNone, quit); err != nil {
		return err
	}

	err = data.UI.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// This is to account for no updates once hosts have returned, since update doesn't get called from omnivore.
		data.Refresh()

		if hostViewIndex != 0 {
			hostViewIndex--
		}
		controlsString = controlsStringHostView
		g.SetViewOnTop(data.StreamCycle.AllHosts[hostViewIndex])
		g.SetCurrentView(data.StreamCycle.AllHosts[hostViewIndex])
		return err
	})

	err = data.UI.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// This is to account for no updates once hosts have returned, since update doesn't get called from omnivore.
		data.Refresh()

		if g.CurrentView().Name() != "output" && hostViewIndex < len(data.StreamCycle.AllHosts)-1 {
			hostViewIndex++
		}
		controlsString = controlsStringHostView
		g.SetViewOnTop(data.StreamCycle.AllHosts[hostViewIndex])
		g.SetCurrentView(data.StreamCycle.AllHosts[hostViewIndex])
		return err
	})

	// Toggle log window to front
	err = data.UI.SetKeybinding("", gocui.KeyCtrlL, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if !logShowing {
			_, err := g.SetViewOnTop("log")
			logShowing = true
			return err
		} else {
			_, err := g.SetViewOnBottom("log")
			logShowing = false
			return err
		}
	})

	err = data.UI.SetKeybinding("output", 's', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		_, err := g.SetViewOnTop("search-host")
		if err != nil {
			panic(err)
		}
		_, err = g.SetCurrentView("search-host")
		if err != nil {
			panic(err)
		}
		controlsString = controlsStringHostSearchView
		return err
	})

	err = data.UI.SetKeybinding("search-host", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		_, err := g.SetViewOnBottom("search-host")
		if err != nil {
			panic(err)
		}
		_, err = g.SetCurrentView("output")
		if err != nil {
			panic(err)
		}
		controlsString = controlsStringOutputView
		return err
	})

	err = data.UI.SetKeybinding("search-host", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		var err error
		_, err = g.SetViewOnBottom("search-host")
		if err != nil {
			panic(err)
		}

		searchedHost := v.Buffer()
		_, err = g.SetViewOnTop(searchedHost)
		if err != nil && err == gocui.ErrUnknownView {
			log.OmniLog.Warn("%s: %s", "Tried to set view for host in user search, but host does not exist", searchedHost)
			g.SetViewOnTop("output")
			_, _ = g.SetCurrentView("output")
			hostViewIndex = 0
			return nil
		} else if err != nil {
			return err
		}
		_, err = g.SetCurrentView(searchedHost)
		if err != nil {
			panic(err)
		}
		log.OmniLog.Info("%s: %s", "Set current view to host in user-search", searchedHost)
		controlsString = controlsStringOutputView
		return nil
	})

	return nil
}

func (data *InterfaceCollective) Close() {
	data.UI.Close()
}

func (data *InterfaceCollective) Refresh() error {
	data.UI.Update(func(g *gocui.Gui) error {
		for i := range data.StreamCycle.AllHosts {
			vw, err := g.View(data.StreamCycle.AllHosts[i])
			if err != nil {
				return err
			}

			vw.Clear()

			content, _ := store.Session.Read(data.StreamCycle.AllHosts[i])

			_, err = fmt.Fprintf(vw, magenta("%s"), content)
			if err != nil {
				panic(err)
			}
		}
		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("log")
		if err != nil {
			return err
		}

		vw.Clear()

		fmt.Fprintf(vw, strings.Join(log.OmniLog.Messages, "\n"))
		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("status")
		if err != nil {
			return err
		}

		vw.Clear()

		maxX, _ := g.Size()
		dashString := ""
		ovTitle := "Omnivore v0"
		for i := 0; i < maxX/2-7; i++ {
			dashString += "-"
		}
		fmt.Fprintf(vw, "%s%s%s", dashString, ovTitle, dashString)
		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("output")
		if err != nil {
			return err
		}

		vw.Clear()

		if data.Group.EncodedValueGroup != nil {
			keys := group.GetSortedGroupMapKeys(data.Group.EncodedValueGroup)

			for _, n := range keys {
				fmt.Fprintf(vw, "Hosts: %s\n\t Output: %s\n\n", yellow(strings.Join(data.Group.EncodedValueGroup[n], ", ")), magenta(fmt.Sprintf("%s", data.Group.EncodedValueToOriginal[n])))
			}
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("todo")
		if err != nil {
			return err
		}

		vw.Clear()

		if data.StreamCycle.TodoHosts != nil {
			for _, v := range ossh.GetSortedHostMapKeys(data.StreamCycle.TodoHosts) {
				fmt.Fprintf(vw, "%s\n", green(v))
			}
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("complete")
		if err != nil {
			return err
		}

		vw.Clear()

		if data.StreamCycle.CompletedHosts != nil {
			for _, v := range ossh.GetSortedHostMapKeys(data.StreamCycle.CompletedHosts) {
				fmt.Fprintf(vw, "%s\n", green(v))
			}
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("failed")
		if err != nil {
			return err
		}

		vw.Clear()

		if data.StreamCycle.FailedHosts != nil {
			for _, v := range ossh.GetSortedHostMapKeys(data.StreamCycle.FailedHosts) {
				fmt.Fprintf(vw, "%s\n", red(v))
			}
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("slow")
		if err != nil {
			return err
		}

		vw.Clear()

		if data.StreamCycle.SlowHosts != nil {
			for _, v := range ossh.GetSortedHostMapKeys(data.StreamCycle.SlowHosts) {
				fmt.Fprintf(vw, "%s\n", yellow(v))
			}
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("command")
		if err != nil {
			return err
		}

		vw.Clear()

		fmt.Fprintf(vw, "%s", red(data.StreamCycle.Command))

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("controls")
		if err != nil {
			return err
		}

		vw.Clear()

		fmt.Fprintf(vw, "%s", controlsString)

		return nil
	})

	return nil
}

func (data *InterfaceCollective) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if logView, err := g.SetView("log", maxX/4, maxY/4, maxX-10, maxY-10, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		logView.Title = "Log"
		logView.Wrap = true
		logView.Autoscroll = true
	}

	if hostSearchView, err := g.SetView("search-host", maxX/4, maxY/4, maxX/4+30, maxY/4+2, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		hostSearchView.Title = "Enter Host"
		hostSearchView.Editable = true
		hostSearchView.Wrap = false
		g.Cursor = true
	}

	if statusView, err := g.SetView("status", 0, 0, maxX-1, maxY/20, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		statusView.Title = "Status"
		statusView.Wrap = true
	}

	// Hosts to do.
	if todoView, err := g.SetView("todo", 0, maxY/20+1, maxX/10, maxY/2-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		todoView.Title = "Todo"
		todoView.Wrap = true
	}

	// Hosts completed successfully.
	if completeView, err := g.SetView("complete", 0, maxY/2, maxX/10, maxY-5, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		completeView.Title = "Complete"
		completeView.Wrap = true
	}

	for i := range data.StreamCycle.AllHosts {
		// Output grouping.
		if outputView, err := g.SetView(data.StreamCycle.AllHosts[i], maxX/10+1, maxY/20+1, maxX/10*9-1, maxY-5, 0); err != nil {
			_ = data.UI.SetKeybinding(data.StreamCycle.AllHosts[i], gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
				_, err := g.SetViewOnTop("output")
				_, err = g.SetCurrentView("output")
				hostViewIndex = 0
				if err != nil {
					panic(err)
				}
				controlsString = controlsStringOutputView

				return err
			})

			if err != gocui.ErrUnknownView {
				return err
			}
			outputView.Title = data.StreamCycle.AllHosts[i]
			outputView.Wrap = true
		}
	}

	// Output grouping.
	if outputView, err := g.SetView("output", maxX/10+1, maxY/20+1, maxX/10*9-1, maxY-5, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		outputView.Title = "Output"
		outputView.Wrap = true
	}

	// I don't know of a better place to make the output view current, until layout is first called.
	if !isStarted {
		// Make sure output view is set as current, as it's used for the main view for keybinds.
		_, err := g.SetCurrentView("output")
		if err != nil {
			panic(err)
		}
		isStarted = true
	}

	// Hosts failed.
	if failedView, err := g.SetView("failed", maxX/10*9, maxY/20+1, maxX-1, maxY/2-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		failedView.Title = "Failed"
		failedView.Wrap = true
	}

	// Hosts that are slow
	if slowView, err := g.SetView("slow", maxX/10*9, maxY/2, maxX-1, maxY-5, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		slowView.Title = "Slow"
		slowView.Wrap = true
	}

	if commandView, err := g.SetView("command", 0, maxY-4, maxX/2-1, maxY-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		commandView.Title = "Command"
		commandView.Wrap = true
	}

	if commandView, err := g.SetView("controls", maxX/2, maxY-4, maxX-1, maxY-1, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		commandView.Title = "controls"
		commandView.Wrap = true
	}

	return nil
}

func update(g *gocui.Gui, v *gocui.View) error {
	err := Collective.Refresh()
	if err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
