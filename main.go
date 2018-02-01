package main

import (
	"github.com/nmaupu/dell-provisioner/cli"
)

const (
	AppName = "dell-provisioner"
	AppDesc = "Kubernetes Dell SAN Provisioner"
)

var (
	AppVersion string
)

func main() {
	if AppVersion == "" {
		AppVersion = "master"
	}

	cli.Process(AppName, AppDesc, AppVersion)
}
