package storage

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
	"os/exec"
	"syscall"
)

var (
	_ Config = &SmcliConfig{}
)

type SmcliConfig struct {
	SanAddress   *string
	SanGroupName *string
	SanPassword  *string
	SmcliCommand *string
}

func (c *SmcliConfig) GetSanAddress() string {
	return *c.SanAddress
}

func (c *SmcliConfig) GetSanGroupName() string {
	return *c.SanGroupName
}

func (c *SmcliConfig) GetSanPassword() string {
	return *c.SanPassword
}

func (c *SmcliConfig) GetSmcliCommand() string {
	return *c.SmcliCommand
}

/**
 * Create a volume using smcli command
 * Dell uses its own capacity suffix:
 * capacity-spec integer-literal [KB | MB | GB | TB | Bytes]
 * See page 34: http://ftp.respmech.com/pub/MD3000/CLIA20EN.pdf
 */
func (c *SmcliConfig) CreateVolume(label string, size resource.Quantity) error {
	if size.IsZero() {
		msg := "size cannot be zero"
		glog.Errorln(msg)
		return errors.New(msg)
	}

	glog.Infof("creating disk %s in group %s, capacity=%v\n", label, c.GetSanGroupName(), size.Value())

	cmd := fmt.Sprintf(
		"create virtualDisk diskGroup=\"%s\" userLabel=\"%s\" capacity=\"%d Bytes\"",
		c.GetSanGroupName(),
		label,
		size.Value())

	err, _ := c.ExecuteSmcli(cmd)
	return err
}

/**
 * Delete a volume using smcli command
 */
func (c *SmcliConfig) DeleteVolume(label string) error {
	cmd := fmt.Sprintf("deleting virtualdisk [\"%s\"]", label)
	err, _ := c.ExecuteSmcli(cmd)
	return err
}

// Executes arbitrary command via smcli config tool and returns stdout
// and handles error accordingly
func (c *SmcliConfig) ExecuteSmcli(cmd string) (error, string) {
	glog.Infof("executing command to %s: %s\n", c.GetSanAddress(), cmd)

	passOpt := ""
	if c.GetSanPassword() != "" {
		passOpt = fmt.Sprintf("-p \"%s\"", c.GetSanPassword())
	}

	exitCode, stdout, stderr := executeUnixCommand(
		c.GetSmcliCommand(),
		c.GetSanAddress(),
		"-S",
		passOpt,
		"-c",
		fmt.Sprintf("'%s;'", cmd))

	if exitCode != 0 {
		return errors.New(stderr), stdout
	}

	return nil, stdout
}

// Execute arbitrary unix command
// returns exitCode, stdout, stderr
func executeUnixCommand(c string, arg ...string) (int, string, string) {
	exitCode := 0
	cmd := exec.Command(c, arg...)

	cmdStdout := &bytes.Buffer{}
	cmdStderr := &bytes.Buffer{}

	cmd.Stdout = cmdStdout
	cmd.Stderr = cmdStderr

	if err := cmd.Start(); err != nil {
		panic(err.Error())
	}

	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	return exitCode, string(cmdStdout.Bytes()), string(cmdStderr.Bytes())
}
