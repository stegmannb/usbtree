//go:build linux

package usb

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/user/usbtree/internal/models"
)

type linuxDetector struct{}

func newPlatformDetector() Detector {
	return &linuxDetector{}
}

func (d *linuxDetector) GetDevices() ([]*models.USBDevice, error) {
	// Use lsusb for USB device detection on Linux
	return d.getDevicesViaLsusb()
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

