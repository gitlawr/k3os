package cliinstall

import (
	"bufio"
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
	cfg   = config.CloudConfig{}
	index = 0
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
	if v, err := g.SetView(diskView, maxX/2-40, maxY/2-sy, maxX/2+40, maxY/2+sy); err != nil {
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
	if v, err := g.SetView(noteView, maxX/2-40, maxY/2+sy, maxX/2+40, maxY/2+sy+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprint(v, "Note: Device will be formatted.")
	}

	// if v, err := g.SetView(debugView, 0, 0, 30, maxY); err != nil {
	// 	if err != gocui.ErrUnknownView {
	// 		return err
	// 	}
	// 	v.Frame = false
	// 	fmt.Fprintln(v, "debug:")
	// }
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
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, nextDialog); err != nil {
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
	output, err := exec.Command("/bin/sh", "-c", `lsblk -r -o NAME,SIZE,TYPE | grep -w disk|cut -d ' ' -f 1,2`).CombinedOutput()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSuffix(string(output), "\n"), "\n"), nil
}

func nextDialog(g *gocui.Gui, v *gocui.View) error {
	debug(g, "next dialog,"+v.Name())
	index++
	switch index {
	case 1:
		return confirmInstall(g, v)
	case 2:
		return installF(g, v)
	}
	return nil
}
func confirmInstall(g *gocui.Gui, v *gocui.View) error {
	debug(g, "enter confirmInstall")
	maxX, maxY := g.Size()
	if v, err := g.SetView(confirmView, maxX/2-40, maxY/2-10, maxX/2+40, maxY/2+10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Confirm Configuration"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		cfg = config.CloudConfig{
			K3OS: config.K3OS{
				Install: &config.Install{
					Device: "/dev/vda",
				},
			},
		}
		installBytes, err := config.PrintInstall(cfg)
		if err != nil {
			return err
		}

		if !cfg.K3OS.Install.Silent {
			fmt.Fprintln(v, string(installBytes)+
				"\nYour disk will be formatted and k3OS will be installed with \nthe above configuration. Continue?\n")

			fmt.Fprintln(v, "Yes")
			fmt.Fprintln(v, "No")
			cx, cy := v.Cursor()
			debug(g, fmt.Sprint(cx)+" "+fmt.Sprint(cy))
			if err := v.SetCursor(cx, cy+5); err != nil {
				debug(g, err.Error())
			}
		}
	}
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View(diskView)
		if err != nil {
			debug(g, err.Error())
		}
		v.Clear()
		return nil
	})
	if _, err := g.SetCurrentView(confirmView); err != nil {
		return err
	}
	debug(g, "currentView is "+g.CurrentView().Name())
	return nil
}

func installF(g *gocui.Gui, v *gocui.View) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView(installView, maxX/2-40, maxY/2-10, maxX/2+40, maxY/2+10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Start Installation"
	}
	if _, err := g.SetCurrentView(installView); err != nil {
		return err
	}
	g.Update(
		func(g *gocui.Gui) error {
			v, err := g.View(confirmView)
			if err != nil {
				debug(g, err.Error())
			}
			v.Clear()
			return nil
		})
	go doInstall(g)
	return nil
}

func doInstall(g *gocui.Gui) error {
	debug(g, "enter doInstall")
	ev, err := config.ToEnv(cfg)
	if err != nil {
		return err
	}
	cmd := exec.Command("/usr/libexec/k3os/install")
	// cmd := exec.Command("sh", "-c", "sleep 2;echo hello;sleep 2; echo world;sleep 2;echo good")
	cmd.Env = append(os.Environ(), ev...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View(installView)
			if err != nil {
				return err
			}
			fmt.Fprintln(v, m)

			lines := len(v.BufferLines())
			_, sy := v.Size()
			if lines > sy {
				ox, oy := v.Origin()
				v.SetOrigin(ox, oy+1)
			}
			return nil
		})
	}
	scanner = bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View(installView)
			if err != nil {
				return err
			}
			fmt.Fprintln(v, m)

			lines := len(v.BufferLines())
			_, sy := v.Size()
			if lines > sy {
				ox, oy := v.Origin()
				v.SetOrigin(ox, oy+1)
			}
			return nil
		})
	}
	return nil
}
func debug(g *gocui.Gui, log string) error {
	logfile := "/var/log/console.log"
	f, err := os.OpenFile(logfile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.WriteString(log); err != nil {
		return err
	}
	// debugV, err := g.View(debugView)
	// if err != nil {
	// 	return err
	// }
	// fmt.Fprintln(debugV, log)
	return nil
}
