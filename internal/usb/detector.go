package usb

import "github.com/user/usbtree/internal/models"

type Detector interface {
	GetDevices() ([]*models.USBDevice, error)
}

func NewDetector() Detector {
	return newPlatformDetector()
}