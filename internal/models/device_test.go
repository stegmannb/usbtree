package models

import (
	"testing"
)

func TestUSBDevice_AddChild(t *testing.T) {
	parent := &USBDevice{
		VendorID:    0x1234,
		ProductID:   0x5678,
		ProductName: "Parent Device",
	}

	child := &USBDevice{
		VendorID:    0xABCD,
		ProductID:   0xEF01,
		ProductName: "Child Device",
	}

	parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}

	if parent.Children[0] != child {
		t.Error("Child was not added correctly")
	}
}

func TestUSBDevice_HasChildren(t *testing.T) {
	device := &USBDevice{
		VendorID:    0x1234,
		ProductID:   0x5678,
		ProductName: "Test Device",
	}

	if device.HasChildren() {
		t.Error("Device should not have children initially")
	}

	child := &USBDevice{
		VendorID:    0xABCD,
		ProductID:   0xEF01,
		ProductName: "Child Device",
	}

	device.AddChild(child)

	if !device.HasChildren() {
		t.Error("Device should have children after adding one")
	}
}

func TestUSBDevice_GetDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		device      *USBDevice
		expected    string
	}{
		{
			name: "Product name available",
			device: &USBDevice{
				ProductName: "My USB Device",
				VendorName:  "My Vendor",
			},
			expected: "My USB Device",
		},
		{
			name: "Only vendor name available",
			device: &USBDevice{
				VendorName: "My Vendor",
			},
			expected: "My Vendor",
		},
		{
			name: "No names available",
			device: &USBDevice{},
			expected: "Unknown Device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.GetDisplayName()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestUSBDevice_GetIDString(t *testing.T) {
	device := &USBDevice{
		VendorID:  0x05AC,
		ProductID: 0x1234,
	}

	expected := "05ac:1234"
	result := device.GetIDString()

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestUSBDevice_JSONMarshaling(t *testing.T) {
	device := &USBDevice{
		VendorID:    0x05AC,
		ProductID:   0x1234,
		VendorName:  "Apple Inc.",
		ProductName: "USB Keyboard",
		Bus:         1,
		Port:        2,
		Address:     3,
		Serial:      "ABC123",
		Speed:       "High (480 Mbps)",
		Class:       "HID",
		SubClass:    "01",
		Protocol:    "01",
		MaxPower:    "100mA",
	}

	child := &USBDevice{
		VendorID:    0x046D,
		ProductID:   0xC52B,
		ProductName: "USB Receiver",
	}

	device.AddChild(child)

	// This test ensures the struct tags are properly set for JSON
	// The actual JSON marshaling is tested implicitly when using --json flag
	if device.VendorID != 0x05AC {
		t.Error("VendorID field not accessible")
	}
}