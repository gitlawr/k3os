package cliinstall

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/rancher/k3os/pkg/config"
)

const (
	debugView   = "debug"
	diskView    = "disk"
	noteView    = "note"
	confirmView = "confirm"
	installView = "install"
)

var (
	cfg = config.CloudConfig{}
)

func RunCuiInstall() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	if err := setKeyBindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	disks, err := getDisks()
	if err != nil {
		return err
	}
	sy := len(disks) + 1
	if v, err := g.SetView(diskView, maxX/2-20, maxY/2-sy, maxX/2+20, maxY/2+sy); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Installation target"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		for i, disk := range disks {
			fmt.Fprintf(v, "%d. %s\n", i+1, disk)
		}
		fmt.Fprintln(v, "")
	}
	if _, err := g.SetCurrentView(diskView); err != nil {
		return err
	}
	if v, err := g.SetView(noteView, maxX/2-20, maxY/2+sy, maxX/2+20, maxY/2+sy+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprint(v, "Note: Device will be formatted.")
	}

	if v, err := g.SetView(debugView, 0, 0, 10, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprintln(v, "debug:")
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func selectDisk(g *gocui.Gui) error {
	output, err := exec.Command("/bin/sh", "-c", "lsblk -r -o NAME,TYPE | grep -w disk | awk '{print $1}'").CombinedOutput()
	if err != nil {
		return err
	}
	strings.Split(string(output), "\n")
	return nil
}

func setKeyBindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := g.SetKeybinding(diskView, gocui.KeyArrowUp, gocui.ModNone, arrowUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(diskView, gocui.KeyArrowDown, gocui.ModNone, arrowDown); err != nil {
		return err
	}
	if err := g.SetKeybinding(diskView, gocui.MouseLeft, gocui.ModNone, arrowUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(diskView, gocui.KeyEnter, gocui.ModNone, confirmInstall); err != nil {
		return err
	}
	if err := g.SetKeybinding(confirmView, gocui.KeyEnter, gocui.ModNone, doInstall); err != nil {
		return err
	}
	return nil
}

func arrowUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil || isAtTop(v) {
		return nil
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}
	return nil
}

func arrowDown(g *gocui.Gui, v *gocui.View) error {
	if v == nil || isAtEnd(v) {
		return nil
	}
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return nil
}

func isAtTop(v *gocui.View) bool {
	_, cy := v.Cursor()
	if cy == 0 {
		return true
	}
	return false
}

func isAtEnd(v *gocui.View) bool {
	_, cy := v.Cursor()
	lines := len(v.BufferLines())
	if lines < 2 || cy == lines-2 {
		return true
	}
	return false
}

func getDisks() ([]string, error) {
	output, err := exec.Command("/bin/sh", "-c", `lsblk -r -o NAME,SIZE,TYPE | grep -w disk | awk 'NR>1{printf "%s (%s)\n",$1,$2} {printf "%s (%s)",$1,$2}'`).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(string(output), "\n"), nil
}

func confirmInstall(g *gocui.Gui, v *gocui.View) error {
	debug(g, "enter confirmInstall")
	maxX, maxY := g.Size()
	if v, err := g.SetView(confirmView, maxX/2-20, maxY/2-10, maxX/2+20, maxY/2+10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Confirm Installation"

		cfg = config.CloudConfig{
			K3OS: config.K3OS{
				Install: &config.Install{
					Device: "/dev/sda",
				},
			},
		}
		installBytes, err := config.PrintInstall(cfg)
		if err != nil {
			return err
		}

		if !cfg.K3OS.Install.Silent {
			fmt.Fprintln(v, "\nConfiguration\n"+"-------------\n\n"+
				string(installBytes)+
				"\nYour disk will be formatted and k3OS will be installed with the above configuration.\nContinue?")
		}
	}
	if _, err := g.SetCurrentView(confirmView); err != nil {
		return err
	}
	if err := g.DeleteView(diskView); err != nil {
		return err
	}
	debug(g, "currentView is confirmView")
	return nil
}

func doInstall(g *gocui.Gui, v *gocui.View) error {
	debug(g, "enter doInstall")
	maxX, maxY := g.Size()
	if v, err := g.SetView(installView, maxX/2-20, maxY/2-10, maxX/2+20, maxY/2+10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Start Installation"
		fmt.Fprintln(v, "Start installation")
	}
	if _, err := g.SetCurrentView(installView); err != nil {
		return err
	}
	if err := g.DeleteView(confirmView); err != nil {
		return err
	}
	ev, err := config.ToEnv(cfg)
	if err != nil {
		return err
	}
	cmd := exec.Command("/usr/libexec/k3os/install")
	cmd.Env = append(os.Environ(), ev...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func debug(g *gocui.Gui, log string) error {
	debugV, err := g.View(debugView)
	if err != nil {
		return err
	}
	fmt.Fprintln(debugV, log)
	return nil
}
