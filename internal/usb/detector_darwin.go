//go:build darwin

package usb

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stegmannb/usbtree/internal/models"
)

type darwinDetector struct{}

func newPlatformDetector() Detector {
	return &darwinDetector{}
}

func (d *darwinDetector) GetDevices() ([]*models.USBDevice, error) {
	// Use system_profiler for USB device detection on macOS
	return d.getDevicesViaSystemProfiler()
}

func (d *darwinDetector) getDevicesViaSystemProfiler() ([]*models.USBDevice, error) {
	cmd := exec.Command("system_profiler", "SPUSBDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run system_profiler: %w", err)
	}

	var spData struct {
		SPUSBDataType []spUSBController `json:"SPUSBDataType"`
	}

	if err := json.Unmarshal(output, &spData); err != nil {
		return nil, fmt.Errorf("failed to parse system_profiler output: %w", err)
	}

	var result []*models.USBDevice
	busNumber := 1

	for _, controller := range spData.SPUSBDataType {
		rootHub := d.createRootHubFromController(controller, busNumber)
		if controller.Items != nil {
			d.processSystemProfilerItems(controller.Items, rootHub)
		}
		result = append(result, rootHub)
		busNumber++
	}

	return result, nil
}

type spUSBController struct {
	Name             string       `json:"_name"`
	HostController   string       `json:"host_controller,omitempty"`
	VendorID         string       `json:"vendor_id,omitempty"`
	ProductID        string       `json:"product_id,omitempty"`
	Manufacturer     string       `json:"manufacturer,omitempty"`
	SerialNum        string       `json:"serial_num,omitempty"`
	Speed            string       `json:"device_speed,omitempty"`
	CurrentAvailable string       `json:"current_available,omitempty"`
	CurrentRequired  string       `json:"current_required,omitempty"`
	Items            []spUSBDevice `json:"_items,omitempty"`
}

type spUSBDevice struct {
	Name             string       `json:"_name"`
	VendorID         string       `json:"vendor_id,omitempty"`
	ProductID        string       `json:"product_id,omitempty"`
	Manufacturer     string       `json:"manufacturer,omitempty"`
	SerialNum        string       `json:"serial_num,omitempty"`
	Speed            string       `json:"device_speed,omitempty"`
	CurrentAvailable string       `json:"current_available,omitempty"`
	CurrentRequired  string       `json:"current_required,omitempty"`
	Items            []spUSBDevice `json:"_items,omitempty"`
}

func (d *darwinDetector) createRootHubFromController(controller spUSBController, busNumber int) *models.USBDevice {
	vendorID := d.parseHexID(controller.VendorID)
	productID := d.parseHexID(controller.ProductID)
	
	// Determine USB version from controller name and host controller
	isUSB3 := strings.Contains(controller.Name, "31") || strings.Contains(controller.HostController, "XHCI")
	
	// Default to standard root hub IDs if not provided
	if vendorID == 0 {
		vendorID = 0x05ac // Apple Inc.
	}
	if productID == 0 {
		if isUSB3 {
			productID = 0x0003
		} else {
			productID = 0x0002
		}
	}
	
	// Create a more descriptive name for the root hub
	hubName := controller.Name
	if controller.HostController != "" {
		if isUSB3 {
			hubName = "USB 3.1 Root Hub"
		} else {
			hubName = "USB 2.0 Root Hub"
		}
	}

	rootHub := &models.USBDevice{
		VendorID:    vendorID,
		ProductID:   productID,
		VendorName:  controller.Manufacturer,
		ProductName: hubName,
		Bus:         busNumber,
		Address:     1,
		Port:        0,
		Speed:       d.convertSystemProfilerSpeed(controller.Speed),
		Class:       "Hub",
		SubClass:    "00",
		Protocol:    "00",
	}

	if rootHub.VendorName == "" {
		rootHub.VendorName = "Apple Inc."
	}
	
	// Set appropriate speed for the root hub
	if rootHub.Speed == "Unknown" || rootHub.Speed == "" {
		if isUSB3 {
			rootHub.Speed = "Super (5 Gbps)"
		} else {
			rootHub.Speed = "High (480 Mbps)"
		}
	}

	if controller.CurrentAvailable != "" {
		rootHub.MaxPower = controller.CurrentAvailable
	}

	return rootHub
}

func (d *darwinDetector) processSystemProfilerItems(items []spUSBDevice, parent *models.USBDevice) {
	for _, item := range items {
		device := d.createDeviceFromSystemProfiler(item, parent.Bus)
		parent.AddChild(device)
		
		// Recursively process child items
		if item.Items != nil {
			d.processSystemProfilerItems(item.Items, device)
		}
	}
}

func (d *darwinDetector) createDeviceFromSystemProfiler(item spUSBDevice, busNumber int) *models.USBDevice {
	device := &models.USBDevice{
		VendorID:    d.parseHexID(item.VendorID),
		ProductID:   d.parseHexID(item.ProductID),
		VendorName:  item.Manufacturer,
		ProductName: item.Name,
		Bus:         busNumber,
		Address:     0, // system_profiler doesn't provide address
		Port:        0, // system_profiler doesn't provide port
		Speed:       d.convertSystemProfilerSpeed(item.Speed),
		Serial:      item.SerialNum,
	}

	// Determine class based on product name
	productLower := strings.ToLower(item.Name)
	if strings.Contains(productLower, "hub") {
		device.Class = "Hub"
	} else if strings.Contains(productLower, "keyboard") || strings.Contains(productLower, "mouse") || strings.Contains(productLower, "trackpad") {
		device.Class = "HID"
	} else if strings.Contains(productLower, "camera") || strings.Contains(productLower, "facetime") {
		device.Class = "Video"
	} else if strings.Contains(productLower, "audio") || strings.Contains(productLower, "headphone") || strings.Contains(productLower, "speaker") {
		device.Class = "Audio"
	} else if strings.Contains(productLower, "ethernet") || strings.Contains(productLower, "network") {
		device.Class = "Communications"
	} else if strings.Contains(productLower, "bluetooth") {
		device.Class = "Wireless"
	} else if strings.Contains(productLower, "storage") || strings.Contains(productLower, "disk") {
		device.Class = "Mass Storage"
	} else {
		device.Class = "Device"
	}

	if item.CurrentRequired != "" {
		device.MaxPower = item.CurrentRequired
	}

	return device
}

func (d *darwinDetector) parseHexID(hexStr string) uint16 {
	if hexStr == "" {
		return 0
	}
	// Remove "0x" prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	// Parse as hexadecimal
	val, err := strconv.ParseUint(hexStr, 16, 16)
	if err != nil {
		return 0
	}
	return uint16(val)
}

func (d *darwinDetector) convertSystemProfilerSpeed(speed string) string {
	switch {
	case strings.Contains(speed, "low_speed"):
		return "Low (1.5 Mbps)"
	case strings.Contains(speed, "full_speed"):
		return "Full (12 Mbps)"
	case strings.Contains(speed, "high_speed"):
		return "High (480 Mbps)"
	case strings.Contains(speed, "super_speed_5gbps"):
		return "Super (5 Gbps)"
	case strings.Contains(speed, "super_speed_10gbps"):
		return "Super+ (10 Gbps)"
	case strings.Contains(speed, "super_speed"):
		return "Super (5 Gbps)"
	default:
		if speed != "" {
			return speed
		}
		return "Unknown"
	}
}

