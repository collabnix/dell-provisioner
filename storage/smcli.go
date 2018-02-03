package storage

import (
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	_ Config = &SmcliConfig{}
)

type SmcliConfig struct {
	SanAddress   *string
	SanGroupName *string
	SanPassword  *string
	Command      *string
}

func (c *SmcliConfig) CreateVolume(label string, capacity resource.Quantity) error {
	glog.Infof("Creating disk %s in group %s, capacity=%s\n", label, c.SanGroupName, capacity)

	cmd := fmt.Sprintf(
		"create virtualDisk diskGroup=\"%s\" userLabel=\"%s\" capacity=%v;",
		c.SanGroupName,
		label,
		capacity)

	return c.execute(&cmd)
}

func (c *SmcliConfig) DeleteVolume(label string) error {
	cmd := "help;"
	return c.execute(&cmd)
}

// Execute arbitrary command via smcli config tool
// and handles error
func (c *SmcliConfig) execute(cmd *string) error {
	passOpt := ""
	if c.SanPassword != nil && *c.SanPassword != "" {
		passOpt = fmt.Sprintf("-p %s", c.SanPassword)
	}

	execCmd := fmt.Sprintf("%s %s -S %s -c '%s'", c.Command, c.SanAddress, passOpt, *cmd)
	glog.Infoln("Executing command: ", execCmd)

	return nil
}
