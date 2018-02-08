package storage

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/deckarep/golang-set"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var (
	_       Config = &SmcliConfig{}
	allLuns        = func() mapset.Set {
		ret := mapset.NewSet()
		for i := 0; i < 255; i++ {
			ret.Add(i)
		}
		return ret
	}()
)

const (
	DISKLABEL_MAX_LENGTH        = 30
	DELETE_DISK_NOT_EXIST_ERROR = "Probable cause = incorrect virtual disk name entered."
	CREATE_DISK_ALREADY_CREATED = "Error 44 - The name you have provided cannot be used. The most likely cause is that the name is already used by another virtual disk. Please provide another name."
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

// Dell supports disk label up to 30 chars
func truncateDiskLabel(label string) string {
	return label[0:DISKLABEL_MAX_LENGTH]
}

/**
 * Create a volume using smcli command
 * Dell uses its own capacity suffix:
 * capacity-spec integer-literal [KB | MB | GB | TB | Bytes]
 * See page 34: http://ftp.respmech.com/pub/MD3000/CLIA20EN.pdf
 */
func (c *SmcliConfig) CreateVolume(label string, size resource.Quantity) (error, int) {
	var (
		err error
		lun int
	)
	diskName := truncateDiskLabel(label)

	if size.IsZero() {
		msg := "size cannot be zero"
		glog.Errorln(msg)
		return errors.New(msg), -1
	}

	// Creating disk
	glog.Infof("Creating disk %s in group %s, capacity=%v bytes\n", diskName, c.GetSanGroupName(), size.Value())
	cmd := fmt.Sprintf(
		"create virtualDisk diskGroup=\"%s\" userLabel=\"%s\" capacity=%dBytes",
		c.GetSanGroupName(),
		diskName,
		size.Value())
	err, _ = c.ExecuteSmcli(cmd)
	// Ignoring error in case the disk is already created
	if err != nil && !strings.Contains(err.Error(), CREATE_DISK_ALREADY_CREATED) {
		return err, -1
	}

	// Getting next available lun
	glog.Infoln("Getting next available lun")
	err, lun = c.GetNextAvailableLun()
	if err != nil {
		glog.Errorln("Cannot get next available lun, deleting freshely created %s disk\n", diskName)
		c.DeleteVolume(label)
		return err, -1
	}

	// Bind disk with this lun number
	glog.Infof("Binding disk %s to lun number %d\n", diskName, lun)
	cmd = fmt.Sprintf(
		"set virtualDisk [\"%s\"] logicalUnitNumber=%d hostGroup=\"%s\"",
		diskName,
		lun,
		c.GetSanGroupName(),
	)
	err, _ = c.ExecuteSmcli(cmd)
	return err, lun
}

/**
 * Delete a volume using smcli command
 */
func (c *SmcliConfig) DeleteVolume(label string) error {
	cmd := fmt.Sprintf("delete virtualdisk [\"%s\"]", truncateDiskLabel(label))
	err, _ := c.ExecuteSmcli(cmd)

	// We get an error because this disk does not exist on the SAN.
	// In that case, we don't return any error
	if err != nil && strings.Contains(err.Error(), DELETE_DISK_NOT_EXIST_ERROR) {
		return nil
	}

	return err
}

func (c *SmcliConfig) Defrag() error {
	err, _ := c.ExecuteSmcli(fmt.Sprintf("start diskGroup [\"%s\"] defragment", c.GetSanGroupName()))
	return err
}

// Executes arbitrary command via smcli config tool and returns stdout
// and handles error accordingly
func (c *SmcliConfig) ExecuteSmcli(cmd string) (error, string) {
	glog.Infof("Executing command to %s: %s\n", c.GetSanAddress(), cmd)

	var exitCode int
	var stdout string

	if c.GetSanPassword() != "" {
		exitCode, stdout, _ = executeUnixCommand(
			c.GetSmcliCommand(),
			c.GetSanAddress(),
			"-S",
			"-p",
			c.GetSanPassword(),
			"-c",
			fmt.Sprintf("%s;", cmd),
		)
	} else {
		exitCode, stdout, _ = executeUnixCommand(
			c.GetSmcliCommand(),
			c.GetSanAddress(),
			"-c",
			fmt.Sprintf("%s;", cmd),
		)
	}

	if exitCode != 0 {
		// Yes, smcli prints error using stdout...
		return errors.New(stdout), stdout
	}

	return nil, stdout
}

func (c *SmcliConfig) GetNextAvailableLun() (error, int) {
	err, stdout := c.ExecuteSmcli(fmt.Sprintf("show storageArray lunMappings hostGroup [\"%s\"]", c.GetSanGroupName()))
	if err != nil {
		return err, 0
	}

	// Matching lines:
	//   beginning with 3 spaces
	//   containing volume names (30chars, letters, digits, -, _)
	//   grouping up to next 3 digits (actual associated lun number)
	// output is as follow:
	//  Performing syntax check...
	//
	//  Syntax check complete.
	//
	//  Executing script...
	//
	//  MAPPINGS (Storage Partitioning - Enabled (7 of 32 used))-------------------
	//
	//
	//     Virtual Disk Name               LUN  RAID Controller Module  Accessible by   Virtual Disk status  Virtual Disk Capacity  Type
	//     Access Virtual Disk             31   0,1                     Host Group k8s  Optimal                                     Access
	//     pvc-9e22c1f7-0beb-11e8-b445-18  0    0                       Host Group k8s  Optimal              1.000 GB               Standard
	//     pvc-9e22c2f7-0beb-11e8-b445-18  1    0                       Host Group k8s  Optimal              1.000 GB               Standard
	//     pvc-9e23c2f7-0beb-21e8-b445-18  3    0                       Host Group k8s  Optimal              1.000 GB               Standard
	//
	//  Script execution complete.
	//
	//  SMcli completed successfully.
	re := regexp.MustCompile("(?m)^[ ]{3}[a-zA-Z0-9\\-_]{30}[ ]{2}(31|[0-9]{1,3}).*$")
	res := re.FindAllStringSubmatch(stdout, -1)

	luns := mapset.NewSet()
	// Reserved lun
	luns.Add(31)
	for _, v := range res {
		i, err := strconv.Atoi(v[1])
		if err != nil {
			fmt.Println("Cannot convert lun number to an int", err)
		}
		luns.Add(i)
	}

	availableLuns := func(s []interface{}) []int {
		var ret []int
		for _, v := range s {
			ret = append(ret, v.(int))
		}
		return ret
	}(allLuns.Difference(luns).ToSlice())
	sort.Ints(availableLuns)
	if len(availableLuns) == 0 {
		return errors.New("cannot get next available lun"), 0
	}
	// First element is also the smallest available
	return nil, availableLuns[0]
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
