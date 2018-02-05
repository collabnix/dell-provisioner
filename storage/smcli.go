package storage

import (
	"errors"
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

/**
 * Create a volume using smcli command
 * Dell uses its own capacity suffix:
 * capacity-spec integer-literal [KB | MB | GB | TB | Bytes]
 * See page 34: http://ftp.respmech.com/pub/MD3000/CLIA20EN.pdf
 */
func (c *SmcliConfig) CreateVolume(label string, size resource.Quantity) error {
	if size.IsZero() {
		msg := "Size cannot be zero"
		glog.Errorln(msg)
		return errors.New(msg)
	}

	glog.Infof("Creating disk %s in group %s, capacity=%v\n", label, c.SanGroupName, size.Value())

	cmd := fmt.Sprintf(
		"create virtualDisk diskGroup=\"%s\" userLabel=\"%s\" capacity=\"%d Bytes\"",
		c.SanGroupName,
		label,
		size.Value())

	return c.execute(&cmd)
}

/**
 * Delete a volume using smcli command
 */
func (c *SmcliConfig) DeleteVolume(label string) error {
	cmd := fmt.Sprintf("delete virtualdisk [\"%s\"]", label)
	return c.execute(&cmd)
}

// Executes arbitrary command via smcli config tool
// and handles error accordingly
func (c *SmcliConfig) execute(cmd *string) error {
	glog.Infof("Executing command to c.SanAddress: %s\n", cmd)

	passOpt := ""
	if c.SanPassword != nil && *c.SanPassword != "" {
		passOpt = fmt.Sprintf("-p \"%s\"", c.SanPassword)
	}

	execCmd := fmt.Sprintf("%s %s -S %s -c '%s;'", c.Command, c.SanAddress, passOpt, *cmd)
	glog.Infof("Global command to execute: %s\n", execCmd)

	return nil
}
