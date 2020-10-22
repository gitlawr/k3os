package console

import (
	"fmt"
	"os/exec"
	"strings"

	cfg "github.com/gitlawr/console/pkg/config"
	"github.com/gitlawr/console/pkg/widgets"
	"github.com/jroimartin/gocui"
	"github.com/rancher/k3os/pkg/config"
)

func (c *Console) layoutInstall(g *gocui.Gui) error {
	setPanels(c)
	initElements := []string{
		titlePanel,
		notePanel,
		diskPanel,
	}
	for _, name := range initElements {
		e, err := c.GetElement(name)
		if err != nil {
			return err
		}
		if err := e.Show(); err != nil {
			return err
		}
	}
	return nil
}

func setPanels(c *Console) error {
	funcs := []func(*Console) error{
		addTitleP,
		addNotePanel,
		addDiskPanel,
		addAskCreatePanel,
		addNodeRolePanel,
		addServerURLPanel,
		addPassword1Panel,
		addPassword2Panel,
		addTokenPanel,
		addCloudInitPanel,
		addConfirmPanel,
		addInstallPanel,
	}
	for _, f := range funcs {
		if err := f(c); err != nil {
			return err
		}
	}
	return nil
}

func addTitleP(c *Console) error {
	maxX, maxY := c.Gui.Size()
	titleV := widgets.NewPanel(c.Gui, titlePanel)
	titleV.SetLocation(maxX/4, maxY/4-3, maxX/4*3, maxY/4)
	titleV.Content = "Choose installation target. Device will be formatted"
	c.AddElement(titlePanel, titleV)
	return nil
}

func addNotePanel(c *Console) error {
	maxX, maxY := c.Gui.Size()
	noteV := widgets.NewPanel(c.Gui, notePanel)
	noteV.SetLocation(maxX/4, maxY/4*3, maxX/4*3, maxY/4*3+2)
	noteV.FgColor = gocui.ColorRed
	c.AddElement(notePanel, noteV)
	return nil
}

func addDiskPanel(c *Console) error {
	diskV, err := widgets.NewSelect(c.Gui, diskPanel, "", getDiskOptions)
	if err != nil {
		return err
	}
	diskV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			device, err := diskV.GetData()
			if err != nil {
				return err
			}
			widgets.Debug(g, "dev is:"+device)
			askCreateV, err := c.GetElement(askCreatePanel)
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			cfg.Config.K3OS.Install = &config.Install{
				Device: device,
			}
			widgets.Debug(g, "aft dev:", cfg.Config)
			diskV.Close()
			if err := askCreateV.Show(); err != nil {
				return err
			}
			titleV.SetContent("Choose installation mode")
			return nil
		},
	}
	c.AddElement(diskPanel, diskV)
	return nil
}

func getDiskOptions() ([]widgets.Option, error) {
	output, err := exec.Command("/bin/sh", "-c", `lsblk -r -o NAME,SIZE,TYPE | grep -w disk|cut -d ' ' -f 1,2`).CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	var options []widgets.Option
	for _, line := range lines {
		splits := strings.SplitN(line, " ", 2)
		if len(splits) == 2 {
			options = append(options, widgets.Option{
				Value: "/dev/" + splits[0],
				Text:  line,
			})
		}
	}

	return options, nil
}

func addAskCreatePanel(c *Console) error {
	askOptionsFunc := func() ([]widgets.Option, error) {
		return []widgets.Option{
			{
				Value: modecreate,
				Text:  "Create a new Harvester cluster",
			}, {
				Value: modeJoin,
				Text:  "Join an existing harvester cluster",
			},
		}, nil
	}
	// new cluster or join existing cluster
	askCreateV, err := widgets.NewSelect(c.Gui, askCreatePanel, "", askOptionsFunc)
	if err != nil {
		return err
	}
	askCreateV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			selected, err := askCreateV.GetData()
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			if selected == modecreate {
				g.Cursor = true
				password1V, err := c.GetElement(password1Panel)
				if err != nil {
					return err
				}
				password2V, err := c.GetElement(password2Panel)
				if err != nil {
					return err
				}
				if err := password2V.Show(); err != nil {
					return err
				}
				if err := password1V.Show(); err != nil {
					return err
				}
				titleV.SetContent("Set admin password")
			} else {
				// joining an existing cluster
				nodeRoleV, err := c.GetElement(nodeRolePanel)
				if err != nil {
					return err
				}
				if err := nodeRoleV.Show(); err != nil {
					return err
				}
				titleV.SetContent("Choose role for the node")
			}
			askCreateV.Close()
			return nil
		},
	}
	c.AddElement(askCreatePanel, askCreateV)
	return nil
}

func addNodeRolePanel(c *Console) error {
	askOptionsFunc := func() ([]widgets.Option, error) {
		return []widgets.Option{
			{
				Value: nodeRoleCompute,
				Text:  "Join as a compute node",
			}, {
				Value: nodeRoleManagement,
				Text:  "Join as a management node",
			},
		}, nil
	}
	// ask node role on join
	nodeRoleV, err := widgets.NewSelect(c.Gui, nodeRolePanel, "", askOptionsFunc)
	if err != nil {
		return err
	}
	nodeRoleV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			selected, err := nodeRoleV.GetData()
			if err != nil {
				return err
			}
			widgets.Debug(g, selected)
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			serverURLV, err := c.GetElement(serverURLPanel)
			if err != nil {
				return err
			}
			g.Cursor = true
			nodeRoleV.Close()
			titleV.SetContent("Specify exisiting server URL")
			return serverURLV.Show()
		},
	}
	c.AddElement(nodeRolePanel, nodeRoleV)
	return nil
}

func addServerURLPanel(c *Console) error {
	serverURLV, err := widgets.NewInput(c.Gui, serverURLPanel, "server URL", false)
	if err != nil {
		return err
	}
	serverURLV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			serverURL, err := serverURLV.GetData()
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			tokenV, err := c.GetElement(tokenPanel)
			if err != nil {
				return err
			}
			serverURLV.Close()
			cfg.Config.K3OS.ServerURL = serverURL
			titleV.SetContent("Specify cluster token")
			return tokenV.Show()
		},
	}
	c.AddElement(serverURLPanel, serverURLV)
	return nil
}

func addPassword1Panel(c *Console) error {
	maxX, maxY := c.Gui.Size()
	password1V, err := widgets.NewInput(c.Gui, password1Panel, "Password", true)
	if err != nil {
		return err
	}
	password1V.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			password2V, err := c.GetElement(password2Panel)
			if err != nil {
				return err
			}
			return password2V.Show()
		},
		gocui.KeyArrowDown: func(g *gocui.Gui, v *gocui.View) error {
			password2V, err := c.GetElement(password2Panel)
			if err != nil {
				return err
			}
			return password2V.Show()
		},
	}
	password1V.SetLocation(maxX/4, maxY/4, maxX/4*3, maxY/4+2)
	c.AddElement(password1Panel, password1V)
	return nil
}

func addPassword2Panel(c *Console) error {
	maxX, maxY := c.Gui.Size()
	password2V, err := widgets.NewInput(c.Gui, password2Panel, "Confirm password", true)
	if err != nil {
		return err
	}
	password2V.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyArrowUp: func(g *gocui.Gui, v *gocui.View) error {
			password1V, err := c.GetElement(password1Panel)
			if err != nil {
				return err
			}
			return password1V.Show()
		},
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			password1V, err := c.GetElement(password1Panel)
			if err != nil {
				return err
			}
			noteV, err := c.GetElement(notePanel)
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			tokenV, err := c.GetElement(tokenPanel)
			if err != nil {
				return err
			}
			password1, err := password1V.GetData()
			if err != nil {
				return err
			}
			password2, err := password2V.GetData()
			if err != nil {
				return err
			}
			if password1 != password2 {
				noteV.SetContent("password mismatching")
				return nil
			}
			noteV.Close()
			password1V.Close()
			password2V.Close()
			cfg.Config.K3OS.Password = password1
			if err := updatePasswd(password1); err != nil {
				return err
			}
			titleV.SetContent("Specify cluster token")
			return tokenV.Show()
		},
	}
	password2V.SetLocation(maxX/4, maxY/4+3, maxX/4*3, maxY/4+5)
	c.AddElement(password2Panel, password2V)
	return nil
}

func addTokenPanel(c *Console) error {
	tokenV, err := widgets.NewInput(c.Gui, tokenPanel, "Cluster token", false)
	if err != nil {
		return err
	}
	tokenV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			token, err := tokenV.GetData()
			if err != nil {
				return err
			}
			cloudInitV, err := c.GetElement(cloudInitPanel)
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			cfg.Config.K3OS.Token = token
			tokenV.Close()
			if err := cloudInitV.Show(); err != nil {
				return err
			}
			titleV.SetContent("Specify cloud-init(Optional)")
			return nil
		},
	}
	c.AddElement(tokenPanel, tokenV)
	return nil
}

func addCloudInitPanel(c *Console) error {
	cloudInitV, err := widgets.NewInput(c.Gui, cloudInitPanel, "File location(http URL)", false)
	if err != nil {
		return err
	}
	cloudInitV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			configURL, err := cloudInitV.GetData()
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			confirmV, err := c.GetElement(confirmPanel)
			if err != nil {
				return err
			}
			cfg.Config.K3OS.Install.ConfigURL = configURL
			cloudInitV.Close()
			installBytes, err := config.PrintInstall(cfg.Config)
			if err != nil {
				return err
			}
			widgets.Debug(g, "cfm cfg: ", fmt.Sprintf("%+v", cfg.Config.K3OS.Install))
			if cfg.Config.K3OS.Install != nil && !cfg.Config.K3OS.Install.Silent {
				confirmV.SetContent(string(installBytes) +
					"\nYour disk will be formatted and Harvester will be installed with \nthe above configuration. Continue?\n")
			}
			g.Cursor = false
			if err := confirmV.Show(); err != nil {
				return err
			}
			titleV.SetContent("Confirm installation options")
			return nil
		},
	}
	c.AddElement(cloudInitPanel, cloudInitV)
	return nil
}

func addConfirmPanel(c *Console) error {
	askOptionsFunc := func() ([]widgets.Option, error) {
		return []widgets.Option{
			{
				Value: "yes",
				Text:  "Yes",
			}, {
				Value: "No",
				Text:  "No",
			},
		}, nil
	}
	// ask node role on join
	confirmV, err := widgets.NewSelect(c.Gui, confirmPanel, "", askOptionsFunc)
	if err != nil {
		return err
	}
	confirmV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			curLine, err := confirmV.GetData()
			if err != nil {
				return err
			}
			titleV, err := c.GetElement(titlePanel)
			if err != nil {
				return err
			}
			installV, err := c.GetElement(installPanel)
			if err != nil {
				return err
			}
			widgets.Debug(g, curLine)
			if err := installV.Show(); err != nil {
				return err
			}
			confirmV.Close()
			go widgets.DoInstall(g)
			titleV.SetContent("Start Installation")
			return nil
		},
	}
	c.AddElement(confirmPanel, confirmV)
	return nil
}

func addInstallPanel(c *Console) error {
	maxX, maxY := c.Gui.Size()
	installV := widgets.NewPanel(c.Gui, installPanel)
	installV.SetLocation(maxX/8, maxY/8, maxX/8*7, maxY/8*7)
	c.AddElement(installPanel, installV)
	installV.Frame = true
	return nil
}
