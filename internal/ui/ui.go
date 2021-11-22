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
	return nil
}

func layout(g *gocui.Gui) error {
	if todoView, err := g.SetView("output", 0, 0, 50, 25); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		todoView.Title = "Output"
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
