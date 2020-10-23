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

var (
	installMode string
	nodeRole    string
)

func (c *Console) layoutInstall(g *gocui.Gui) error {
	setPanels(c)
	initElements := []string{
		titlePanel,
		notePanel,
		askCreatePanel,
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
		addOsPasswordPanels,
		addAdminPasswordPanels,
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
	titleV.Content = "Choose installation mode"
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
			cfg.Config.K3OS.Install = &config.Install{
				Device: device,
			}
			diskV.Close()
			g.Cursor = true
			return showNext(c, "Specify the password to access the operating system(user rancher)", osPasswordConfirmPanel, osPasswordPanel)
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
				Value: modeCreate,
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
			if selected == modeCreate {
				installMode = modeCreate
				// if err := showNext(c, "Set Harvester admin password", osPasswordConfirmPanel, osPasswordPanel); err != nil {
				// 	return err
				// }
				if err := showNext(c, "Choose installation target. Device will be formatted", diskPanel); err != nil {
					return err
				}
			} else {
				// joining an existing cluster
				installMode = modeJoin
				if err := showNext(c, "Choose role for the node", nodeRolePanel); err != nil {
					return err
				}
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
			nodeRole = selected
			nodeRoleV.Close()
			return showNext(c, "Choose installation target. Device will be formatted", diskPanel)
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
			serverURLV.Close()
			cfg.Config.K3OS.ServerURL = serverURL
			return showNext(c, "Specify cluster token", tokenPanel)
		},
	}
	c.AddElement(serverURLPanel, serverURLV)
	return nil
}

func addOsPasswordPanels(c *Console) error {
	maxX, maxY := c.Gui.Size()
	osPasswordV, err := widgets.NewInput(c.Gui, osPasswordPanel, "Password", true)
	if err != nil {
		return err
	}
	osPasswordV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			return showNext(c, "", osPasswordConfirmPanel)
		},
		gocui.KeyArrowDown: func(g *gocui.Gui, v *gocui.View) error {
			return showNext(c, "", osPasswordConfirmPanel)
		},
	}
	osPasswordV.SetLocation(maxX/4, maxY/4, maxX/4*3, maxY/4+2)
	c.AddElement(osPasswordPanel, osPasswordV)

	osPasswordConfirmV, err := widgets.NewInput(c.Gui, osPasswordConfirmPanel, "Confirm password", true)
	if err != nil {
		return err
	}
	osPasswordConfirmV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyArrowUp: func(g *gocui.Gui, v *gocui.View) error {
			return showNext(c, "", osPasswordPanel)
		},
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			password1V, err := c.GetElement(osPasswordPanel)
			if err != nil {
				return err
			}
			noteV, err := c.GetElement(notePanel)
			if err != nil {
				return err
			}
			password1, err := password1V.GetData()
			if err != nil {
				return err
			}
			password2, err := osPasswordConfirmV.GetData()
			if err != nil {
				return err
			}
			if password1 != password2 {
				noteV.SetContent("password mismatching")
				return nil
			}
			noteV.Close()
			password1V.Close()
			osPasswordConfirmV.Close()
			encrpyted, err := getEncrptedPasswd(password1)
			if err != nil {
				return err
			}
			cfg.Config.K3OS.Password = encrpyted
			if installMode == modeCreate {
				return showNext(c, "Specify Harvester admin password", adminPasswordConfirmPanel, adminPasswordPanel)
			} else {
				return showNext(c, "Specify exisiting server URL", serverURLPanel)
			}
		},
	}
	osPasswordConfirmV.SetLocation(maxX/4, maxY/4+3, maxX/4*3, maxY/4+5)
	c.AddElement(osPasswordConfirmPanel, osPasswordConfirmV)

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
			cfg.Config.K3OS.Token = token
			tokenV.Close()
			return showNext(c, "Specify cloud-init(Optional)", cloudInitPanel)
		},
	}
	c.AddElement(tokenPanel, tokenV)
	return nil
}

func addAdminPasswordPanels(c *Console) error {
	maxX, maxY := c.Gui.Size()
	adminPasswordV, err := widgets.NewInput(c.Gui, adminPasswordPanel, "Password", true)
	if err != nil {
		return err
	}
	adminPasswordV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			return showNext(c, "", adminPasswordConfirmPanel)
		},
		gocui.KeyArrowDown: func(g *gocui.Gui, v *gocui.View) error {
			return showNext(c, "", adminPasswordConfirmPanel)
		},
	}
	adminPasswordV.SetLocation(maxX/4, maxY/4, maxX/4*3, maxY/4+2)
	c.AddElement(adminPasswordPanel, adminPasswordV)

	adminPasswordConfirmV, err := widgets.NewInput(c.Gui, adminPasswordConfirmPanel, "Confirm password", true)
	if err != nil {
		return err
	}
	adminPasswordConfirmV.KeyBindings = map[gocui.Key]func(*gocui.Gui, *gocui.View) error{
		gocui.KeyArrowUp: func(g *gocui.Gui, v *gocui.View) error {
			return showNext(c, "", adminPasswordPanel)
		},
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			password1V, err := c.GetElement(adminPasswordPanel)
			if err != nil {
				return err
			}
			noteV, err := c.GetElement(notePanel)
			if err != nil {
				return err
			}
			password1, err := password1V.GetData()
			if err != nil {
				return err
			}
			password2, err := adminPasswordConfirmV.GetData()
			if err != nil {
				return err
			}
			if password1 != password2 {
				noteV.SetContent("password mismatching")
				return nil
			}
			noteV.Close()
			password1V.Close()
			adminPasswordConfirmV.Close()
			return showNext(c, "Specify cluster token", tokenPanel)
		},
	}
	adminPasswordConfirmV.SetLocation(maxX/4, maxY/4+3, maxX/4*3, maxY/4+5)
	c.AddElement(adminPasswordConfirmPanel, adminPasswordConfirmV)

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
			options := fmt.Sprintf("install mode: %v\n", installMode)
			if installMode == modeJoin {
				options += fmt.Sprintf("node role: %v\n", nodeRole)
			}
			options += string(installBytes)
			widgets.Debug(g, "cfm cfg: ", fmt.Sprintf("%+v", cfg.Config.K3OS.Install))
			if cfg.Config.K3OS.Install != nil && !cfg.Config.K3OS.Install.Silent {
				confirmV.SetContent(options +
					"\nYour disk will be formatted and Harvester will be installed with \nthe above configuration. Continue?\n")
			}
			g.Cursor = false
			return showNext(c, "Confirm installation options", confirmPanel)
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
			widgets.Debug(g, curLine)
			confirmV.Close()
			go widgets.DoInstall(g)
			return showNext(c, "Start Installation", installPanel)
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
	installV.Wrap = false
	return nil
}
