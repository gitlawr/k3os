package console

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	cfg "github.com/gitlawr/console/pkg/config"
	"github.com/rancher/k3os/pkg/config"
)

func getEncrptedPasswd(pass string) (string, error) {
	oldShadow, err := ioutil.ReadFile("/etc/shadow")
	if err != nil {
		return "", err
	}
	defer func() {
		ioutil.WriteFile("/etc/shadow", oldShadow, 0640)
	}()

	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("rancher:%s", pass))
	errBuffer := &bytes.Buffer{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = errBuffer

	if err := cmd.Run(); err != nil {
		os.Stderr.Write(errBuffer.Bytes())
		return "", err
	}
	f, err := os.Open("/etc/shadow")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) > 1 && fields[0] == "rancher" {
			return fields[1], nil
		}
	}

	return "", scanner.Err()
}

func showNext(c *Console, title string, names ...string) error {
	if title != "" {
		titleV, err := c.GetElement(titlePanel)
		if err != nil {
			return err
		}
		titleV.SetContent(title)
	}
	for _, name := range names {
		v, err := c.GetElement(name)
		if err != nil {
			return err
		}
		if err := v.Show(); err != nil {
			return err
		}
	}
	return nil
}

func setNote(c *Console, note string) error {
	noteV, err := c.GetElement(notePanel)
	if err != nil {
		return err
	}
	noteV.SetContent(note)
	return noteV.Show()
}

func customizeConfig() {
	//common configs for both server and agent
	cfg.Config.K3OS.DNSNameservers = []string{"8.8.8.8"}
	cfg.Config.K3OS.NTPServers = []string{"ntp.ubuntu.com"}
	cfg.Config.K3OS.Modules = []string{"kvm"}
	cfg.Config.Bootcmd = []string{
		"mkdir -p /opt/cni/bin",
		"cp /var/lib/cni/bin/* /opt/cni/bin",
		//"for i in bridge flannel host-local loopback portmap;do ln -fs /var/lib/rancher/k3s/data/*/bin/cni /opt/cni/bin/$i;done",
	}

	if installMode == modeJoin && nodeRole == nodeRoleCompute {
		return
	}

	harvesterChartValues["minio.persistence.size"] = "20Gi"
	harvesterChartValues["containers.apiserver.image.imagePullPolicy"] = "IfNotPresent"
	harvesterChartValues["containers.apiserver.image.tag"] = "master-head"
	harvesterChartValues["service.harvester.type"] = "LoadBalancer"
	harvesterChartValues["containers.apiserver.authMode"] = "localUser"
	harvesterChartValues["minio.mode"] = "distributed"

	cfg.Config.WriteFiles = []config.File{
		{
			Owner:              "root",
			Path:               "/var/lib/rancher/k3s/server/manifests/harvester.yaml",
			RawFilePermissions: "0600",
			Content:            getHarvesterManifestContent(harvesterChartValues),
		},
	}
	cfg.Config.K3OS.K3sArgs = []string{
		"server",
		"--disable",
		"local-storage",
		"--flannel-backend",
		"none",
	}
}

func getHarvesterManifestContent(values map[string]string) string {
	base := `apiVersion: v1
kind: Namespace
metadata:
  name: harvester-system
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: harvester
  namespace: kube-system
spec:
  chart: https://%{KUBERNETES_API}%/static/charts/harvester-0.1.0.tgz
  targetNamespace: harvester-system
  set:
`
	var buffer = bytes.Buffer{}
	buffer.WriteString(base)
	for k, v := range values {
		buffer.WriteString(fmt.Sprintf("    %s: %q\n", k, v))
	}
	return buffer.String()
}
