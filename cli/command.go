package cli

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/jawher/mow.cli"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	dellProvisioner "github.com/nmaupu/dell-provisioner/provisioner"
	"github.com/nmaupu/dell-provisioner/storage"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"syscall"
	"time"
)

const (
	provisionerName           = "maupu.org/dell-provisioner"
	exponentialBackOffOnError = false
	failedRetryThreshold      = 5
	leasePeriod               = controller.DefaultLeaseDuration
	retryPeriod               = controller.DefaultRetryPeriod
	renewDeadline             = controller.DefaultRenewDeadline
	termLimit                 = controller.DefaultTermLimit
)

func Process(appName, appDesc, appVersion string) {
	syscall.Umask(0)
	flag.Set("logtostderr", "true")

	app := cli.App(appName, appDesc)
	app.Version("v version", fmt.Sprintf("%s version %s", appName, appVersion))

	app.Command("smcli", "Provision volumes using smcli command", commandSmcli)

	app.Run(os.Args)
}

// When using smcli command
func commandSmcli(cmd *cli.Cmd) {
	var (
		msgs        []string
		identifier  *string
		smcliConfig storage.SmcliConfig
	)

	identifier = cmd.String(cli.StringOpt{
		Name:   "i identifier",
		Desc:   "Provisioner identifier (e.g. if ensure, set it to current node's name)",
		EnvVar: "IDENTIFIER",
	})

	smcliConfig.SanAddress = cmd.String(cli.StringOpt{
		Name:   "a sanAddress",
		Desc:   "SAN address to connect to",
		EnvVar: "SAN_ADDRESS",
	})

	smcliConfig.SanPassword = cmd.String(cli.StringOpt{
		Name:   "p sanPassword",
		Desc:   "SAN password",
		EnvVar: "SAN_PASSWORD",
	})

	smcliConfig.SanGroupName = cmd.String(cli.StringOpt{
		Name:   "g sanGroupName",
		Desc:   "SAN Group name",
		EnvVar: "SAN_GROUP_NAME",
	})

	smcliConfig.Command = cmd.String(cli.StringOpt{
		Name:   "s smcliCommand",
		Value:  "/opt/dell/mdstoragemanager/client/SMcli",
		Desc:   "Path to the smcli command",
		EnvVar: "SMCLI_COMMAND",
	})

	cmd.Action = func() {
		/* Params checking */
		if *identifier == "" {
			msgs = append(msgs, "Identifier must be specified")
		}
		if *smcliConfig.SanAddress == "" {
			msgs = append(msgs, "San address must be specified")
		}
		if *smcliConfig.SanGroupName == "" {
			msgs = append(msgs, "San group name must be specified")
		}
		if *smcliConfig.Command == "" {
			msgs = append(msgs, "Path to the smcli command")
		}
		processErrors(msgs)

		/* Everything's good so far, ready to start */
		glog.Infoln("Starting dell-provisioner with the following parameters")
		glog.Infof("  Identifier: %s", *identifier)
		glog.Infof("  San address: %s", *smcliConfig.SanAddress)
		glog.Infof("  San group name: %s", *smcliConfig.SanGroupName)
		glog.Infof("  Smcli command: %s", *smcliConfig.Command)

		// Execute with this particular config implem
		execute(*identifier, &smcliConfig)
	}
}

func execute(identifier string, provisionerConfig storage.Config) {
	/* Creating Kubernetes cluster configuration */
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to create config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	// Verifying Kubernetes version (out-of-tree provisioners aren't officially supported before v1.5)
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		glog.Fatalf("Error getting server version: %v", err)
	}

	clientDellProvisioner := dellProvisioner.New(
		identifier,
		provisionerConfig,
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

	// Starting the main thread
	pc.Run(wait.NeverStop)
}

// Print all errors and exit
func processErrors(msgs []string) {
	if len(msgs) > 0 {
		fmt.Fprintf(os.Stderr, "The following error(s) occured:\n")
		for _, m := range msgs {
			fmt.Fprintf(os.Stderr, "  - %s\n", m)
		}
		os.Exit(1)
	}
}
