package usb

import (
	"strings"
	"testing"

	"github.com/stegmannb/usbtree/internal/models"
)

// MockDetector implements the Detector interface for testing
type MockDetector struct {
	devices []*models.USBDevice
	err     error
}

func (m *MockDetector) GetDevices() ([]*models.USBDevice, error) {
	return m.devices, m.err
}

func TestNewDetector(t *testing.T) {
	detector := NewDetector()
	if detector == nil {
		t.Error("NewDetector() returned nil")
	}

	// Test that it implements the Detector interface
	_, ok := detector.(Detector)
	if !ok {
		t.Error("NewDetector() does not implement Detector interface")
	}
}

func TestMockDetector(t *testing.T) {
	// Test successful case
	expectedDevices := []*models.USBDevice{
		{
			VendorID:    0x05AC,
			ProductID:   0x1234,
			ProductName: "Test Device",
		},
	}

	mockDetector := &MockDetector{
		devices: expectedDevices,
		err:     nil,
	}

	devices, err := mockDetector.GetDevices()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(devices))
	}

	if devices[0].ProductName != "Test Device" {
		t.Errorf("Expected 'Test Device', got %s", devices[0].ProductName)
	}

	// Test error case
	mockDetector.err = ErrNoDevices
	devices, err = mockDetector.GetDevices()
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestHubDetection(t *testing.T) {
	// Test that hub devices are properly detected
	hubDevices := []*models.USBDevice{
		{
			VendorID:    0x1d6b, // Linux Foundation
			ProductID:   0x0002, // USB 2.0 Root Hub
			VendorName:  "Linux Foundation",
			ProductName: "USB 2.0 Root Hub",
			Bus:         1,
			Address:     1,
			Port:        0,
			Speed:       "High (480 Mbps)",
			Class:       "Hub",
			SubClass:    "00",
			Protocol:    "00",
		},
		{
			VendorID:    0x1d6b, // Linux Foundation
			ProductID:   0x0003, // USB 3.0 Root Hub
			VendorName:  "Linux Foundation",
			ProductName: "USB 3.0 Root Hub",
			Bus:         2,
			Address:     1,
			Port:        0,
			Speed:       "Super (5 Gbps)",
			Class:       "Hub",
			SubClass:    "00",
			Protocol:    "00",
		},
	}

	mockDetector := &MockDetector{
		devices: hubDevices,
		err:     nil,
	}

	devices, err := mockDetector.GetDevices()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("Expected 2 hub devices, got %d", len(devices))
	}

	// Verify both are hub devices
	for i, device := range devices {
		if device.Class != "Hub" {
			t.Errorf("Device %d should be a Hub, got class: %s", i, device.Class)
		}

		if device.VendorName != "Linux Foundation" {
			t.Errorf("Device %d should be from Linux Foundation, got: %s", i, device.VendorName)
		}

		if !strings.Contains(device.ProductName, "Root Hub") {
			t.Errorf("Device %d should be a Root Hub, got: %s", i, device.ProductName)
		}
	}
}

var ErrNoDevices = &DetectorError{"no devices found"}

type DetectorError struct {
	message string
}

func (e *DetectorError) Error() string {
	return e.message
}