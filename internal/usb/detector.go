package usb

import "github.com/stegmannb/usbtree/internal/models"

type Detector interface {
	GetDevices() ([]*models.USBDevice, error)
}

func NewDetector() Detector {
	return newPlatformDetector()
}