package tree

import (
	"strings"
	"testing"

	"github.com/user/usbtree/internal/models"
)

func TestFormatter_FormatDevice(t *testing.T) {
	formatter := NewFormatter(false)

	device := &models.USBDevice{
		VendorID:    0x05AC,
		ProductID:   0x1234,
		ProductName: "USB Hub",
		Class:       "Hub",
		Bus:         1,
		Port:        0,
		Address:     1,
	}

	child1 := &models.USBDevice{
		VendorID:    0x046D,
		ProductID:   0xC52B,
		ProductName: "USB Receiver",
		Class:       "HID",
		Bus:         1,
		Port:        1,
		Address:     2,
	}

	child2 := &models.USBDevice{
		VendorID:    0x0781,
		ProductID:   0x5591,
		ProductName: "SanDisk Ultra",
		Class:       "Mass Storage",
		Bus:         1,
		Port:        2,
		Address:     3,
	}

	device.AddChild(child1)
	device.AddChild(child2)

	lines := formatter.FormatDevice(device, "", false)

	// Check that we have the expected number of lines
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 lines, got %d", len(lines))
	}

	// Check the parent device line
	if !strings.Contains(lines[0], "USB Hub") {
		t.Errorf("Expected parent device name in first line, got: %s", lines[0])
	}

	if !strings.Contains(lines[0], "[05ac:1234]") {
		t.Errorf("Expected device ID in first line, got: %s", lines[0])
	}

	if !strings.Contains(lines[0], "(Hub)") {
		t.Errorf("Expected device class in first line, got: %s", lines[0])
	}

	// Check for child devices
	foundChild1 := false
	foundChild2 := false
	for _, line := range lines {
		if strings.Contains(line, "USB Receiver") {
			foundChild1 = true
		}
		if strings.Contains(line, "SanDisk Ultra") {
			foundChild2 = true
		}
	}

	if !foundChild1 {
		t.Error("Child 1 (USB Receiver) not found in output")
	}

	if !foundChild2 {
		t.Error("Child 2 (SanDisk Ultra) not found in output")
	}
}

func TestFormatter_FormatDevice_Verbose(t *testing.T) {
	formatter := NewFormatter(true)

	device := &models.USBDevice{
		VendorID:    0x05AC,
		ProductID:   0x1234,
		ProductName: "USB Hub",
		Class:       "Hub",
		Bus:         1,
		Port:        0,
		Address:     1,
		Serial:      "ABC123",
		Speed:       "High (480 Mbps)",
		MaxPower:    "500mA",
	}

	lines := formatter.FormatDevice(device, "", true)

	// Check for verbose details
	foundSerial := false
	foundSpeed := false
	foundMaxPower := false
	foundBusInfo := false

	for _, line := range lines {
		if strings.Contains(line, "Serial: ABC123") {
			foundSerial = true
		}
		if strings.Contains(line, "Speed: High (480 Mbps)") {
			foundSpeed = true
		}
		if strings.Contains(line, "Max Power: 500mA") {
			foundMaxPower = true
		}
		if strings.Contains(line, "Bus 1, Port 0, Address 1") {
			foundBusInfo = true
		}
	}

	if !foundSerial {
		t.Error("Serial number not found in verbose output")
	}

	if !foundSpeed {
		t.Error("Speed not found in verbose output")
	}

	if !foundMaxPower {
		t.Error("Max power not found in verbose output")
	}

	if !foundBusInfo {
		t.Error("Bus info not found in verbose output")
	}
}

func TestFormatter_FormatTree(t *testing.T) {
	formatter := NewFormatter(false)

	// Test with no devices
	result := formatter.FormatTree([]*models.USBDevice{})
	if result != "No USB devices found" {
		t.Errorf("Expected 'No USB devices found', got: %s", result)
	}

	// Test with devices
	device1 := &models.USBDevice{
		VendorID:    0x05AC,
		ProductID:   0x1234,
		ProductName: "USB Hub 1",
	}

	device2 := &models.USBDevice{
		VendorID:    0x05AC,
		ProductID:   0x5678,
		ProductName: "USB Hub 2",
	}

	devices := []*models.USBDevice{device1, device2}
	result = formatter.FormatTree(devices)

	if !strings.Contains(result, "USB Device Tree:") {
		t.Error("Expected header 'USB Device Tree:' in output")
	}

	if !strings.Contains(result, "USB Hub 1") {
		t.Error("Expected 'USB Hub 1' in output")
	}

	if !strings.Contains(result, "USB Hub 2") {
		t.Error("Expected 'USB Hub 2' in output")
	}
}

func TestFormatter_TreeConnectors(t *testing.T) {
	formatter := NewFormatter(false)

	parent := &models.USBDevice{
		VendorID:    0x05AC,
		ProductID:   0x1234,
		ProductName: "Parent",
	}

	child1 := &models.USBDevice{
		VendorID:    0x046D,
		ProductID:   0xC52B,
		ProductName: "Child 1",
	}

	child2 := &models.USBDevice{
		VendorID:    0x0781,
		ProductID:   0x5591,
		ProductName: "Child 2",
	}

	parent.AddChild(child1)
	parent.AddChild(child2)

	// Test middle child uses ├──
	lines := formatter.FormatDevice(parent, "", false)
	
	for i, line := range lines {
		if strings.Contains(line, "Child 1") {
			if !strings.Contains(line, "├──") {
				t.Errorf("Expected ├── connector for middle child, got: %s", line)
			}
		}
		if strings.Contains(line, "Child 2") {
			if !strings.Contains(line, "└──") {
				t.Errorf("Expected └── connector for last child, got: %s", line)
			}
		}
		// Debug output
		t.Logf("Line %d: %s", i, line)
	}
}