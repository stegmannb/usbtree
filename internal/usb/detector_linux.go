//go:build linux

package usb

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/gousb"
	"github.com/user/usbtree/internal/models"
)

type linuxDetector struct{}

func newPlatformDetector() Detector {
	return &linuxDetector{}
}

func (d *linuxDetector) GetDevices() ([]*models.USBDevice, error) {
	// Try libusb first
	devices, err := d.getDevicesViaLibusb()
	if err == nil {
		return devices, nil
	}

	// If libusb fails with permission error, try lsusb
	if strings.Contains(err.Error(), "permission denied") || strings.Contains(err.Error(), "bad access") {
		return d.getDevicesViaLsusb()
	}

	return nil, err
}

func (d *linuxDetector) getDevicesViaLsusb() ([]*models.USBDevice, error) {
	// First get basic device info from lsusb
	devices, err := d.parseLsusbOutput()
	if err != nil {
		return nil, err
	}

	// Then get hierarchy from lsusb -t
	hierarchy, err := d.parseLsusbTree()
	if err != nil {
		// If tree parsing fails, return flat list
		return devices, nil
	}

	// Merge hierarchy info into devices
	return d.mergeHierarchy(devices, hierarchy), nil
}

func (d *linuxDetector) parseLsusbOutput() ([]*models.USBDevice, error) {
	cmd := exec.Command("lsusb")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsusb: %w", err)
	}

	deviceMap := make(map[string]*models.USBDevice)
	
	// Parse lsusb output
	// Format: Bus XXX Device YYY: ID VVVV:PPPP Manufacturer Product
	re := regexp.MustCompile(`Bus (\d{3}) Device (\d{3}): ID ([0-9a-f]{4}):([0-9a-f]{4})\s*(.*)$`)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 6 {
			continue
		}

		bus, _ := strconv.Atoi(matches[1])
		address, _ := strconv.Atoi(matches[2])
		vendorID, _ := strconv.ParseUint(matches[3], 16, 16)
		productID, _ := strconv.ParseUint(matches[4], 16, 16)
		description := strings.TrimSpace(matches[5])

		// Parse manufacturer and product from description
		var vendorName, productName string
		if description != "" {
			// Handle special cases where vendor name contains spaces
			if strings.HasPrefix(description, "Linux Foundation") {
				vendorName = "Linux Foundation"
				productName = strings.TrimPrefix(description, "Linux Foundation ")
			} else if strings.HasPrefix(description, "VIA Labs, Inc.") {
				vendorName = "VIA Labs, Inc."
				productName = strings.TrimPrefix(description, "VIA Labs, Inc. ")
			} else if strings.HasPrefix(description, "Terminus Technology Inc.") {
				vendorName = "Terminus Technology Inc."
				productName = strings.TrimPrefix(description, "Terminus Technology Inc. ")
			} else if strings.HasPrefix(description, "Anker Innovations Limited.") {
				vendorName = "Anker Innovations Limited."
				productName = strings.TrimPrefix(description, "Anker Innovations Limited. ")
			} else if strings.HasPrefix(description, "Valve Software") {
				vendorName = "Valve Software"
				productName = strings.TrimPrefix(description, "Valve Software ")
			} else if strings.HasPrefix(description, "ASIX Electronics Corp.") {
				vendorName = "ASIX Electronics Corp."
				productName = strings.TrimPrefix(description, "ASIX Electronics Corp. ")
			} else if strings.HasPrefix(description, "Intel Corp.") {
				vendorName = "Intel Corp."
				productName = strings.TrimPrefix(description, "Intel Corp. ")
			} else if strings.HasPrefix(description, "Micro Star International") {
				vendorName = "Micro Star International"
				productName = strings.TrimPrefix(description, "Micro Star International ")
			} else if strings.HasPrefix(description, "Genesys Logic, Inc.") {
				vendorName = "Genesys Logic, Inc."
				productName = strings.TrimPrefix(description, "Genesys Logic, Inc. ")
			} else if strings.HasPrefix(description, "SteelSeries ApS") {
				vendorName = "SteelSeries ApS"
				productName = strings.TrimPrefix(description, "SteelSeries ApS ")
			} else if strings.HasPrefix(description, "KYE Systems Corp.") {
				vendorName = "KYE Systems Corp."
				productName = strings.TrimPrefix(description, "KYE Systems Corp. ")
			} else {
				parts := strings.SplitN(description, " ", 2)
				if len(parts) > 0 {
					vendorName = parts[0]
				}
				if len(parts) > 1 {
					productName = parts[1]
				}
			}
		}

		usbDevice := &models.USBDevice{
			VendorID:    uint16(vendorID),
			ProductID:   uint16(productID),
			Bus:         bus,
			Address:     address,
			Port:        0, // Will be filled from tree
			VendorName:  vendorName,
			ProductName: productName,
			Speed:       "Unknown",
		}

		// Determine class based on known patterns
		productLower := strings.ToLower(productName)
		if strings.Contains(productLower, "hub") {
			usbDevice.Class = "Hub"
		} else if strings.Contains(productLower, "keyboard") || strings.Contains(productLower, "mouse") {
			usbDevice.Class = "HID"
		} else if strings.Contains(productLower, "camera") {
			usbDevice.Class = "Video"
		} else if strings.Contains(productLower, "audio") || strings.Contains(productLower, "headset") || strings.Contains(productLower, "arctis") {
			usbDevice.Class = "Audio"
		} else if strings.Contains(productLower, "ethernet") || strings.Contains(productLower, "ax88179") {
			usbDevice.Class = "Communications"
		} else if strings.Contains(productLower, "bluetooth") || strings.Contains(productLower, "ax200") {
			usbDevice.Class = "Wireless"
		} else if strings.Contains(productLower, "controller") {
			usbDevice.Class = "HID"
		} else if strings.Contains(productLower, "jtag") || strings.Contains(productLower, "serial") {
			usbDevice.Class = "Communications"
		} else {
			usbDevice.Class = "Device"
		}

		// Set speed for root hubs
		if vendorID == 0x1d6b {
			if productID == 0x0002 {
				usbDevice.Speed = "High (480 Mbps)"
			} else if productID == 0x0003 {
				usbDevice.Speed = "Super (5 Gbps)"
			}
		}

		deviceKey := fmt.Sprintf("%d-%d", bus, address)
		deviceMap[deviceKey] = usbDevice
	}

	// Convert map to slice
	var result []*models.USBDevice
	for _, device := range deviceMap {
		result = append(result, device)
	}

	return result, nil
}

type treeNode struct {
	bus      int
	port     int
	dev      int
	speed    string
	parent   *treeNode
	children []*treeNode
}

func (d *linuxDetector) parseLsusbTree() (map[string]*treeNode, error) {
	cmd := exec.Command("lsusb", "-t")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsusb -t: %w", err)
	}

	nodes := make(map[string]*treeNode)
	var currentBusRoot *treeNode
	var parentStack []*treeNode

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Count indentation level
		indent := 0
		for i := 0; i < len(line); i++ {
			if line[i] == ' ' {
				indent++
			} else {
				break
			}
		}
		level := indent / 4

		// Parse root hub line
		if strings.HasPrefix(strings.TrimSpace(line), "/:") {
			// Format: /:  Bus 001.Port 001: Dev 001, Class=root_hub, Driver=xhci_hcd/6p, 480M
			busRe := regexp.MustCompile(`Bus (\d+)\.Port (\d+): Dev (\d+).*?(\d+M)?$`)
			matches := busRe.FindStringSubmatch(line)
			if len(matches) >= 4 {
				bus, _ := strconv.Atoi(matches[1])
				port, _ := strconv.Atoi(matches[2])
				dev, _ := strconv.Atoi(matches[3])
				speed := ""
				if len(matches) > 4 {
					speed = matches[4]
				}

				node := &treeNode{
					bus:   bus,
					port:  port,
					dev:   dev,
					speed: speed,
				}
				currentBusRoot = node
				parentStack = []*treeNode{node}
				nodeKey := fmt.Sprintf("%d-%d", bus, dev)
				nodes[nodeKey] = node
			}
		} else if strings.Contains(line, "Port") {
			// Format: |__ Port 004: Dev 003, If 0, Class=Wireless, Driver=btusb, 12M
			portRe := regexp.MustCompile(`Port (\d+): Dev (\d+).*?(\d+M)?$`)
			matches := portRe.FindStringSubmatch(line)
			if len(matches) >= 3 && currentBusRoot != nil {
				port, _ := strconv.Atoi(matches[1])
				dev, _ := strconv.Atoi(matches[2])
				speed := ""
				if len(matches) > 3 {
					speed = matches[3]
				}

				// Adjust parent stack based on indentation
				for len(parentStack) > level {
					parentStack = parentStack[:len(parentStack)-1]
				}

				var parent *treeNode
				if len(parentStack) > 0 {
					parent = parentStack[len(parentStack)-1]
				}

				node := &treeNode{
					bus:    currentBusRoot.bus,
					port:   port,
					dev:    dev,
					speed:  speed,
					parent: parent,
				}

				if parent != nil {
					parent.children = append(parent.children, node)
				}

				nodeKey := fmt.Sprintf("%d-%d", currentBusRoot.bus, dev)
				if _, exists := nodes[nodeKey]; !exists {
					nodes[nodeKey] = node
					if level >= len(parentStack) {
						parentStack = append(parentStack, node)
					} else {
						parentStack[level] = node
					}
				}
			}
		}
	}

	return nodes, nil
}

func (d *linuxDetector) mergeHierarchy(devices []*models.USBDevice, hierarchy map[string]*treeNode) []*models.USBDevice {
	deviceMap := make(map[string]*models.USBDevice)
	rootDevices := make(map[string]*models.USBDevice)

	// Create a map of devices by bus-address
	for _, device := range devices {
		key := fmt.Sprintf("%d-%d", device.Bus, device.Address)
		deviceMap[key] = device

		// Update port and speed from hierarchy if available
		if node, exists := hierarchy[key]; exists {
			device.Port = node.port
			if node.speed != "" {
				device.Speed = d.convertSpeed(node.speed)
			}
		}

		// Identify root hubs
		if device.Address == 1 {
			rootKey := fmt.Sprintf("bus-%d", device.Bus)
			rootDevices[rootKey] = device
		}
	}

	// Build device hierarchy based on tree structure
	for key, node := range hierarchy {
		device, exists := deviceMap[key]
		if !exists {
			continue
		}

		if node.parent != nil {
			parentKey := fmt.Sprintf("%d-%d", node.parent.bus, node.parent.dev)
			if parent, parentExists := deviceMap[parentKey]; parentExists {
				parent.AddChild(device)
			}
		}
	}

	// Return only root devices (they contain the full tree)
	var result []*models.USBDevice
	for _, device := range rootDevices {
		result = append(result, device)
	}

	// If no hierarchy was built, return all devices attached to root hubs
	if len(result) == 0 {
		for _, device := range devices {
			if device.Address == 1 {
				rootKey := fmt.Sprintf("bus-%d", device.Bus)
				rootDevices[rootKey] = device
			}
		}

		// Attach non-root devices to their bus root
		for _, device := range devices {
			if device.Address != 1 {
				busKey := fmt.Sprintf("bus-%d", device.Bus)
				if root, exists := rootDevices[busKey]; exists {
					root.AddChild(device)
				}
			}
		}

		for _, device := range rootDevices {
			result = append(result, device)
		}
	}

	return result
}

func (d *linuxDetector) convertSpeed(speed string) string {
	switch speed {
	case "1.5M":
		return "Low (1.5 Mbps)"
	case "12M":
		return "Full (12 Mbps)"
	case "480M":
		return "High (480 Mbps)"
	case "5000M":
		return "Super (5 Gbps)"
	case "10000M":
		return "Super+ (10 Gbps)"
	default:
		return speed
	}
}

func (d *linuxDetector) getDevicesViaLibusb() ([]*models.USBDevice, error) {
	usbCtx := gousb.NewContext()
	defer usbCtx.Close()

	devices, err := usbCtx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return true
	})
	if err != nil {
		// Check if it's a permission error
		if strings.Contains(err.Error(), "bad access") {
			return nil, fmt.Errorf("permission denied")
		}
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