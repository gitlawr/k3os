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

func customizeConfig() {
	if installMode == modeJoin && nodeRole == nodeRoleCompute {
		cfg.Config.Runcmd = []string{
			"mkdir -p /var/lib/rancher/k3s/agent/images",
			"cp -n /usr/var/lib/rancher/k3s/agent/images/* /var/lib/rancher/k3s/agent/images/",
		}
		return
	}
	cfg.Config.Runcmd = []string{
		//"mkdir -p /var/lib/rancher/k3s/server/manifests",
		"mkdir -p /var/lib/rancher/k3s/server/static/charts",
		"mkdir -p /var/lib/rancher/k3s/agent/images",
		"cp -n /usr/var/lib/rancher/k3s/server/static/charts/* /var/lib/rancher/k3s/server/static/charts/",
		//"cp -n /usr/var/lib/rancher/k3s/server/manifests/* /var/lib/rancher/k3s/server/manifests/",
		"cp -n /usr/var/lib/rancher/k3s/agent/images/* /var/lib/rancher/k3s/agent/images/",
	}
	harvesterChartValues["minio.persistence.size"] = "20Gi"
	harvesterChartValues["containers.apiserver.image.imagePullPolicy"] = "IfNotPresent"
	harvesterChartValues["containers.apiserver.image.tag"] = "v0.0.1"

	cfg.Config.WriteFiles = []config.File{
		{
			Owner:              "root",
			Path:               "/var/lib/rancher/k3s/server/manifests/harvester.yaml",
			RawFilePermissions: "0600",
			Content:            getHarvesterManifestContent(harvesterChartValues),
		},
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
