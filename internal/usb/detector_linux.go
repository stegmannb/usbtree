//go:build linux

package usb

import (
	"fmt"

	"github.com/google/gousb"
	"github.com/user/usbtree/internal/models"
)

type linuxDetector struct{}

func newPlatformDetector() Detector {
	return &linuxDetector{}
}

func (d *linuxDetector) GetDevices() ([]*models.USBDevice, error) {
	usbCtx := gousb.NewContext()
	defer usbCtx.Close()

	devices, err := usbCtx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate devices: %w", err)
	}
	defer func() {
		for _, dev := range devices {
			dev.Close()
		}
	}()

	rootDevices := make(map[string]*models.USBDevice)
	deviceMap := make(map[string]*models.USBDevice)
	busNumbers := make(map[int]bool)

	// Track bus numbers for creating root hubs
	for _, dev := range devices {
		desc := dev.Desc
		busNumbers[int(desc.Bus)] = true

		usbDevice := &models.USBDevice{
			VendorID:    uint16(desc.Vendor),
			ProductID:   uint16(desc.Product),
			Bus:         int(desc.Bus),
			Address:     int(desc.Address),
			Port:        int(desc.Port),
			Speed:       getSpeedString(desc.Speed),
		}

		if manufacturer, err := dev.Manufacturer(); err == nil {
			usbDevice.VendorName = manufacturer
		}

		if product, err := dev.Product(); err == nil {
			usbDevice.ProductName = product
		}

		if serial, err := dev.SerialNumber(); err == nil {
			usbDevice.Serial = serial
		}

		usbDevice.Class = getClassString(desc.Class)
		usbDevice.SubClass = fmt.Sprintf("%02x", desc.SubClass)
		usbDevice.Protocol = fmt.Sprintf("%02x", desc.Protocol)
		
		// Get MaxPower from the active configuration if available
		if len(desc.Configs) > 0 {
			for _, cfg := range desc.Configs {
				if cfg.MaxPower > 0 {
					usbDevice.MaxPower = fmt.Sprintf("%dmA", cfg.MaxPower)
					break
				}
			}
		}

		deviceKey := fmt.Sprintf("%d-%d", desc.Bus, desc.Address)
		deviceMap[deviceKey] = usbDevice
	}

	// Create root hub entries for each bus, even if no devices are found
	for busNum := range busNumbers {
		rootKey := fmt.Sprintf("bus-%d", busNum)
		if _, exists := rootDevices[rootKey]; !exists {
			rootHub := &models.USBDevice{
				VendorID:    0x1d6b, // Linux Foundation
				ProductID:   getProductIDForUSBVersion(busNum),
				VendorName:  "Linux Foundation",
				ProductName: fmt.Sprintf("USB %d.0 Root Hub", getUSBVersion(busNum)),
				Bus:         busNum,
				Address:     1,
				Port:        0,
				Speed:       getDefaultSpeedForBus(busNum),
				Class:       "Hub",
				SubClass:    "00",
				Protocol:    "00",
			}
			rootDevices[rootKey] = rootHub
			deviceMap[fmt.Sprintf("%d-1", busNum)] = rootHub
		}
	}

	// If no devices found, create virtual root hubs for known buses
	if len(busNumbers) == 0 {
		// Create at least one root hub to show the bus structure
		for i := 1; i <= 2; i++ {
			rootKey := fmt.Sprintf("bus-%d", i)
			rootHub := &models.USBDevice{
				VendorID:    0x1d6b, // Linux Foundation
				ProductID:   getProductIDForUSBVersion(i),
				VendorName:  "Linux Foundation",
				ProductName: fmt.Sprintf("USB %d.0 Root Hub", getUSBVersion(i)),
				Bus:         i,
				Address:     1,
				Port:        0,
				Speed:       getDefaultSpeedForBus(i),
				Class:       "Hub",
				SubClass:    "00",
				Protocol:    "00",
			}
			rootDevices[rootKey] = rootHub
			deviceMap[fmt.Sprintf("%d-1", i)] = rootHub
		}
	}

	organizeHierarchy(deviceMap, rootDevices)

	var result []*models.USBDevice
	for _, device := range rootDevices {
		result = append(result, device)
	}

	return result, nil
}

func getUSBVersion(busNum int) int {
	// Simple heuristic: assume higher bus numbers are newer USB versions
	if busNum <= 2 {
		return 2 // USB 2.0
	}
	return 3 // USB 3.0+
}

func getDefaultSpeedForBus(busNum int) string {
	if busNum <= 2 {
		return "High (480 Mbps)" // USB 2.0
	}
	return "Super (5 Gbps)" // USB 3.0+
}

func getProductIDForUSBVersion(busNum int) uint16 {
	if busNum <= 2 {
		return 0x0002 // USB 2.0 Root Hub
	}
	return 0x0003 // USB 3.0 Root Hub
}

func organizeHierarchy(deviceMap map[string]*models.USBDevice, rootDevices map[string]*models.USBDevice) {
	for _, device := range deviceMap {
		if device.Port > 0 {
			parentKey := fmt.Sprintf("%d-%d", device.Bus, device.Port)
			if parent, exists := deviceMap[parentKey]; exists {
				parent.AddChild(device)
			} else {
				busKey := fmt.Sprintf("bus-%d", device.Bus)
				if root, exists := rootDevices[busKey]; exists {
					root.AddChild(device)
				}
			}
		}
	}
}

func getSpeedString(speed gousb.Speed) string {
	switch speed {
	case gousb.SpeedLow:
		return "Low (1.5 Mbps)"
	case gousb.SpeedFull:
		return "Full (12 Mbps)"
	case gousb.SpeedHigh:
		return "High (480 Mbps)"
	case gousb.SpeedSuper:
		return "Super (5 Gbps)"
	default:
		return "Unknown"
	}
}

func getClassString(class gousb.Class) string {
	classNames := map[gousb.Class]string{
		0x00: "Device",
		0x01: "Audio",
		0x02: "Communications",
		0x03: "HID",
		0x05: "Physical",
		0x06: "Image",
		0x07: "Printer",
		0x08: "Mass Storage",
		0x09: "Hub",
		0x0a: "CDC Data",
		0x0b: "Smart Card",
		0x0d: "Content Security",
		0x0e: "Video",
		0x0f: "Personal Healthcare",
		0x10: "Audio/Video",
		0xdc: "Diagnostic",
		0xe0: "Wireless",
		0xef: "Miscellaneous",
		0xfe: "Application Specific",
		0xff: "Vendor Specific",
	}

	if name, ok := classNames[class]; ok {
		return name
	}
	return fmt.Sprintf("Class %02x", class)
}