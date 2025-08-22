package tree

import (
	"fmt"
	"strings"

	"github.com/user/usbtree/internal/models"
)

type Formatter struct {
	verbose bool
}

func NewFormatter(verbose bool) *Formatter {
	return &Formatter{verbose: verbose}
}

func (f *Formatter) FormatDevice(device *models.USBDevice, prefix string, isLast bool) []string {
	var lines []string
	
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	
	deviceLine := fmt.Sprintf("%s%s%s", prefix, connector, f.getDeviceString(device))
	lines = append(lines, deviceLine)
	
	if f.verbose {
		detailPrefix := prefix
		if isLast {
			detailPrefix += "    "
		} else {
			detailPrefix += "│   "
		}
		lines = append(lines, f.getDetailLines(device, detailPrefix)...)
	}
	
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}
	
	for i, child := range device.Children {
		isLastChild := i == len(device.Children)-1
		childLines := f.FormatDevice(child, childPrefix, isLastChild)
		lines = append(lines, childLines...)
	}
	
	return lines
}

func (f *Formatter) getDeviceString(device *models.USBDevice) string {
	name := device.GetDisplayName()
	idString := device.GetIDString()
	
	if device.Class != "" && device.Class != "Device" {
		return fmt.Sprintf("%s [%s] (%s)", name, idString, device.Class)
	}
	
	return fmt.Sprintf("%s [%s]", name, idString)
}

func (f *Formatter) getDetailLines(device *models.USBDevice, prefix string) []string {
	var lines []string
	
	if device.Serial != "" {
		lines = append(lines, fmt.Sprintf("%s├─ Serial: %s", prefix, device.Serial))
	}
	
	if device.Speed != "" && device.Speed != "Unknown" {
		lines = append(lines, fmt.Sprintf("%s├─ Speed: %s", prefix, device.Speed))
	}
	
	if device.MaxPower != "" {
		lines = append(lines, fmt.Sprintf("%s├─ Max Power: %s", prefix, device.MaxPower))
	}
	
	lines = append(lines, fmt.Sprintf("%s└─ Bus %d, Port %d, Address %d", 
		prefix, device.Bus, device.Port, device.Address))
	
	return lines
}

func (f *Formatter) FormatTree(devices []*models.USBDevice) string {
	if len(devices) == 0 {
		return "No USB devices found"
	}
	
	var allLines []string
	allLines = append(allLines, "USB Device Tree:")
	allLines = append(allLines, "")
	
	for i, device := range devices {
		isLast := i == len(devices)-1
		lines := f.FormatDevice(device, "", isLast)
		allLines = append(allLines, lines...)
	}
	
	return strings.Join(allLines, "\n")
}