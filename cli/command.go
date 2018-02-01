package cli

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/jawher/mow.cli"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	dellProvisioner "github.com/nmaupu/dell-provisioner/provisioner"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"syscall"
	"time"
)

const (
	provisionerName           = "nmaupu.org/dell-provisioner"
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
)

var (
	// cli parameters
	identifier                  *string
	sanAddress                  *string
	sanPassword                 *string
	sanGroupName, sanVolumeName *string
	smcliCommand                *string
)

func Process(appName, appDesc, appVersion string) {
	syscall.Umask(0)
	flag.Set("logtostderr", "true")

	app := cli.App(appName, appDesc)
	app.Version("v version", fmt.Sprintf("%s version %s", appName, appVersion))

	identifier = app.String(cli.StringOpt{
		Name:   "i identifier",
		Desc:   "Provisioner identifier (e.g. if ensure, set it to current node's name)",
		EnvVar: "IDENTIFIER",
	})

	sanAddress = app.String(cli.StringOpt{
		Name:   "a sanAddress",
		Desc:   "SAN address to connect to",
		EnvVar: "SAN_ADDRESS",
	})

	sanPassword = app.String(cli.StringOpt{
		Name:   "p sanPassword",
		Desc:   "SAN password",
		EnvVar: "SAN_PASSWORD",
	})

	sanGroupName = app.String(cli.StringOpt{
		Name:   "g sanGroupName",
		Desc:   "SAN Group name",
		EnvVar: "SAN_GROUP_NAME",
	})

	smcliCommand = app.String(cli.StringOpt{
		Name:   "s smcliCommand",
		Desc:   "Path to the smcli command",
		EnvVar: "SMCLI_COMMAND",
	})

	app.Action = execute
	app.Run(os.Args)
}

func execute() {
	var err error

	/* Params checking */
	var msgs []string
	if *identifier == "" {
		msgs = append(msgs, "Identifier must be specified")
	}
	if *sanAddress == "" {
		msgs = append(msgs, "San address must be specified")
	}
	if *sanPassword == "" {
		msgs = append(msgs, "San password must be specified")
	}
	if *sanGroupName == "" {
		msgs = append(msgs, "San group name must be specified")
	}
	if *smcliCommand == "" {
		msgs = append(msgs, "Path to the smcli command")
	} else {
		info, _ := os.Stat(*smcliCommand)
		mode := info.Mode()
		if mode&0111 != 0 {
			msgs = append(msgs, "The binary for smcli is not executable")
		}
	}

	// Print all errors and exit if need be
	if len(msgs) > 0 {
		fmt.Fprintf(os.Stderr, "The following error(s) occured:\n")
		for _, m := range msgs {
			fmt.Fprintf(os.Stderr, "  - %s\n", m)
		}
		os.Exit(1)
	}
	/* End params checking */

	/* Everything's good so far, ready to start */
	glog.Infoln("Starting dell-provisioner with the following parameters")
	glog.Infof("  Identifier: %s", *identifier)
	glog.Infof("  San address: %s", *sanAddress)
	glog.Infof("  San group name: %s", *sanGroupName)

	/* Creating Kubernetes cluster configuration */
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// Verifying Kubernetes version (out-of-tree provisioners aren't officially supported before v1.5)
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	clientDellProvisioner := dellProvisioner.New(
		*identifier,
		*sanAddress,
		*sanPassword,
		*sanGroupName,
		*smcliCommand,
	)

	pc := controller.NewProvisionController(
		clientset,
		15*time.Second,
		provisionerName,
		clientDellProvisioner,
		serverVersion.GitVersion,
		exponentialBackOffOnError,
		failedRetryThreshold,
		leasePeriod,
		renewDeadline,
		retryPeriod,
		termLimit,
	)
	pc.Run(wait.NeverStop)
}
