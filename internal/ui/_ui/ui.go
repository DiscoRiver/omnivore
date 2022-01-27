package main

import (
	"fmt"
	"github.com/discoriver/omnivore/pkg/group"
	"github.com/jroimartin/gocui"
	"math/rand"
	"time"
)

var (
	dp *Data
	hostInc int
    letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

// Data needed for UI to process.
type Data struct {
	Grp *group.ValueGrouping
	ui *gocui.Gui
}

func main() {
	rand.Seed(time.Now().UnixNano())

	dp = &Data{}
	makeData()

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		panic(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		panic(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, update); err != nil {
		panic(err)
	}

	dp.ui = g

	go func(){
		for {
			select {
			case <-dp.Grp.Update:
			default:
				dp.RefreshValue()
				time.Sleep(1 * time.Second)
				addNewGroup()
			}
		}
	}()

	dp.StartGUI()
	//wg.Wait()

	// Change value after StartUI is called and keybind is defined.
	//value.i++
	// Should print 1
	//fmt.Printf("Actual Value of value.i = %d\n", *d.v)
}

func MakeDP() {
	dp = &Data{
		Grp: group.NewValueGrouping(),
	}
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func makeData() {
	MakeDP()

	go func(){
		for {
			select {
			case <-dp.Grp.Update:
			}
		}
	}()

	addNewGroup()
}

func addNewGroup() {
	dp.Grp.AddToGroup(group.NewIdentifyingPair(fmt.Sprintf("host%d", hostInc), []byte(randSeq(10))))
	hostInc++
}

func (data *Data) StartGUI() {
	if err := data.ui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		panic(err)
	}
	if err := data.ui.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, update); err != nil {
		panic(err)
	}

	if err := data.ui.MainLoop(); err != nil && err != gocui.ErrQuit {
		panic(err)
	}
}

func (data *Data) RefreshValue() error {
	data.ui.Update(func(g *gocui.Gui) error {
		vw, err := g.View("output")
		if err != nil {
		}

		vw.Clear()

		for k, v := range dp.Grp.EncodedValueGroup {
			fmt.Fprintf(vw, "Key: %v\n\t Value: %s\n\n", v, dp.Grp.EncodedValueToOriginal[k])
		}

		return nil
	})
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
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
	dp.RefreshValue()
	return nil
}
