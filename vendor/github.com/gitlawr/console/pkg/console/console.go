package console

import (
	"context"
	"fmt"
	"os"

	"github.com/gitlawr/console/pkg/widgets"
	"github.com/jroimartin/gocui"
)

type Console struct {
	context context.Context
	*gocui.Gui
	elements map[string]widgets.Element
}

func RunConsole() error {
	c, err := NewConsole()
	if err != nil {
		return err
	}
	return c.doRun()
}

func NewConsole() (*Console, error) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		return nil, err
	}
	return &Console{
		context:  context.Background(),
		Gui:      g,
		elements: make(map[string]widgets.Element),
	}, nil
}

func (c *Console) GetElement(name string) (widgets.Element, error) {
	e, ok := c.elements[name]
	if ok {
		return e, nil
	}
	return nil, fmt.Errorf("element %q is not found", name)
}

func (c *Console) AddElement(name string, element widgets.Element) {
	c.elements[name] = element
}

func (c *Console) doRun() error {
	defer c.Close()

	if hd, _ := os.LookupEnv("HARVESTER_DASHBOARD"); hd == "true" {
		c.SetManagerFunc(layoutDashboard)
	} else {
		c.SetManagerFunc(c.layoutInstall)
	}

	if err := setGlobalKeyBindings(c.Gui); err != nil {
		return err
	}

	if err := c.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

func setGlobalKeyBindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyF12, gocui.ModNone, quit); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
