package storage

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
	"time"
)

type Config interface {
	// Create a volume and returns an associated lun number
	CreateVolume(label string, capacity resource.Quantity) (error, int)
	// Delete a volume
	DeleteVolume(label string) error
	// Run a defragmentation
	Defrag() error
}

func StartDefragJob(config Config, interval time.Duration) {
	for {
		err := config.Defrag()
		if err != nil {
			glog.Infoln("Cannot execute defrag", err)
		}

		time.Sleep(interval)
	}
}
