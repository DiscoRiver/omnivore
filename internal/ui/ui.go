package ui

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/ossh"
	"github.com/discoriver/omnivore/pkg/group"
	"github.com/jroimartin/gocui"
)

var (
	DP *Data
)
// Data needed for UI to process.
type Data struct {
	Group *group.ValueGrouping
	StreamCycle *ossh.StreamCycle

	UI *gocui.Gui
}

func MakeDP() {
	DP = &Data{}
	DP.Group = group.NewValueGrouping()
}

func (data *Data) StartUI() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	data.UI = g

	data.UI.SetManagerFunc(layout)

	if err := data.UI.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		panic(err)
	}

	if err := data.UI.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, update); err != nil {
		panic(err)
	}

	if err := data.UI.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.OmniLog.Fatal("Error")
	}
}

func (data *Data) Refresh() error {
	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("output")
		if err != nil {
		}

		vw.Clear()

		for k, v := range data.Group.EncodedValueGroup {
			fmt.Fprintf(vw, "Hosts: %v\n\t Output: %s\n\n", v, data.Group.EncodedValueToOriginal[k])
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("todo")
		if err != nil {
		}

		vw.Clear()

		for h := range data.StreamCycle.TodoHosts {
			fmt.Fprintf(vw, "%s\n", h)
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("complete")
		if err != nil {
		}

		vw.Clear()

		for h := range data.StreamCycle.CompletedHosts {
			fmt.Fprintf(vw, "%s\n", h)
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("failed")
		if err != nil {
		}

		vw.Clear()

		for h := range data.StreamCycle.FailedHosts {
			fmt.Fprintf(vw, "%s\n", h)
		}

		return nil
	})

	data.UI.Update(func(g *gocui.Gui) error {
		vw, err := g.View("slow")
		if err != nil {
		}

		vw.Clear()

		for h := range data.StreamCycle.SlowHosts {
			fmt.Fprintf(vw, "%s\n", h)
		}

		return nil
	})

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if todoView, err := g.SetView("todo", 0, 0, maxX/10, maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		todoView.Title = "Todo"
		todoView.Wrap = true
	}

	if completeView, err := g.SetView("complete", 0, maxY/2, maxX/10, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		completeView.Title = "Complete"
		completeView.Wrap = true
	}

	if outputView, err := g.SetView("output", maxX/10+1, 0, maxX/10*9-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		outputView.Title = "Output"
		outputView.Wrap = true
	}

	if failedView, err := g.SetView("failed", maxX/10*9, 0, maxX-1, maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		failedView.Title = "Failed"
		failedView.Wrap = true
	}

	if slowView, err := g.SetView("slow", maxX/10*9, maxY/2, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		slowView.Title = "Slow"
		slowView.Wrap = true
	}

	return nil
}

func update(g *gocui.Gui, v *gocui.View) error {
	DP.Refresh()
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
