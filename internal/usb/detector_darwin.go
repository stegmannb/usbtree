//go:build darwin

package usb

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/gousb"
	"github.com/user/usbtree/internal/models"
)

type darwinDetector struct{}

func newPlatformDetector() Detector {
	return &darwinDetector{}
}

func (d *darwinDetector) GetDevices() ([]*models.USBDevice, error) {
	// Try libusb first
	devices, err := d.getDevicesViaLibusb()
	if err == nil {
		return devices, nil
	}

	// If libusb fails, try system_profiler
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
	
	// Default to standard root hub IDs if not provided
	if vendorID == 0 {
		vendorID = 0x05ac // Apple Inc.
	}
	if productID == 0 {
		if strings.Contains(controller.Name, "3.") || strings.Contains(controller.Name, "USB 3") {
			productID = 0x0003
		} else {
			productID = 0x0002
		}
	}

	rootHub := &models.USBDevice{
		VendorID:    vendorID,
		ProductID:   productID,
		VendorName:  controller.Manufacturer,
		ProductName: controller.Name,
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

func (d *darwinDetector) getDevicesViaLibusb() ([]*models.USBDevice, error) {
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
				VendorID:    0x05ac, // Apple Inc.
				ProductID:   getProductIDForUSBVersion(busNum),
				VendorName:  "Apple Inc.",
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
		// Check system_profiler for USB buses
		for i := 1; i <= 4; i++ {
			rootKey := fmt.Sprintf("bus-%d", i)
			rootHub := &models.USBDevice{
				VendorID:    0x05ac, // Apple Inc.
				ProductID:   getProductIDForUSBVersion(i),
				VendorName:  "Apple Inc.",
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