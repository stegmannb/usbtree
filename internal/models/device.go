package models

import "fmt"

type USBDevice struct {
	VendorID    uint16       `json:"vendor_id"`
	ProductID   uint16       `json:"product_id"`
	VendorName  string       `json:"vendor_name"`
	ProductName string       `json:"product_name"`
	Bus         int          `json:"bus"`
	Port        int          `json:"port"`
	Address     int          `json:"address"`
	Serial      string       `json:"serial,omitempty"`
	Speed       string       `json:"speed"`
	Class       string       `json:"class,omitempty"`
	SubClass    string       `json:"subclass,omitempty"`
	Protocol    string       `json:"protocol,omitempty"`
	MaxPower    string       `json:"max_power,omitempty"`
	Children    []*USBDevice `json:"children,omitempty"`
}

func (d *USBDevice) AddChild(child *USBDevice) {
	d.Children = append(d.Children, child)
}

func (d *USBDevice) HasChildren() bool {
	return len(d.Children) > 0
}

func (d *USBDevice) GetDisplayName() string {
	if d.ProductName != "" {
		return d.ProductName
	}
	if d.VendorName != "" {
		return d.VendorName
	}
	return "Unknown Device"
}

func (d *USBDevice) GetIDString() string {
	return fmt.Sprintf("%04x:%04x", d.VendorID, d.ProductID)
}