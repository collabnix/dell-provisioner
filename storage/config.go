package storage

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

type Config interface {
	CreateVolume(label string, capacity resource.Quantity) error
	DeleteVolume(label string) error
}
