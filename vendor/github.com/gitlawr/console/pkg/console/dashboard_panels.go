package console

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jroimartin/gocui"
)

const (
	harvesterURL        = "harvesterURL"
	harvesterStatus     = "harvesterStatus"
	colorRed        int = 1
	colorGreen      int = 2
	colorYellow     int = 3
)

var once = sync.Once{}

func layoutDashboard(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("url", maxX/3, maxY/3, maxX/3*2, maxY/3+5); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		ip, err := exec.Command("/bin/sh", "-c", `hostname -I|cut -d " " -f 1`).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get IP: %v", err)
		}
		fmt.Fprintf(v, "Harvester is installed. Access it from:\n\nhttps://%s:8443", strings.TrimSpace(string(ip)))
	}
	if v, err := g.SetView("status", maxX/3, maxY/3+5, maxX/3*2, maxY/3+10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprintln(v, "Current status: \033[33;7mUnknown\033[0m")
		go syncHarvesterStatus(context.Background(), g)
	}
	if v, err := g.SetView("footer", 0, maxY-2, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		fmt.Fprintf(v, "<Use F12 to switch between Harvester console and Shell>")
	}

	return nil
}
func syncHarvesterStatus(ctx context.Context, g *gocui.Gui) {
	status := ""
	syncDuration := 5 * time.Second
	ticker := time.NewTicker(syncDuration)
	go func() {
		<-ctx.Done()
		ticker.Stop()
	}()
	for range ticker.C {
		output, err := exec.Command("/bin/sh", "-c", `kubectl get po -n harvester-system -l "app.kubernetes.io/name=harvester" -o jsonpath='{.items[*].status.phase}'`).CombinedOutput()
		// output, err := exec.Command("/bin/sh", "-c", `echo hehe`).CombinedOutput()
		if err != nil {
			status = wrapColor("Error - "+err.Error(), colorRed)
		} else {
			status = string(output)
		}
		if status == "" {
			status = wrapColor("Unknown", colorYellow)
		} else if status == "Ready" {
			status = wrapColor(status, colorGreen)
		} else {
			status = wrapColor(status, colorYellow)
		}
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("status")
			if err != nil {
				return err
			}
			v.Clear()
			fmt.Fprintln(v, "Current status: "+status)
			return nil
		})
	}

}

func wrapColor(s string, color int) string {
	return fmt.Sprintf("\033[3%d;7m%s\033[0m", color, s)
}
