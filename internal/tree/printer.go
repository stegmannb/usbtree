package tree

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/user/usbtree/internal/models"
)

type Printer struct {
	formatter *Formatter
	useColor  bool
}

func NewPrinter(verbose bool) *Printer {
	return &Printer{
		formatter: NewFormatter(verbose),
		useColor:  !color.NoColor,
	}
}

func (p *Printer) Print(devices []*models.USBDevice) {
	if len(devices) == 0 {
		p.printNoDevices()
		return
	}
	
	p.printHeader()
	
	for i, device := range devices {
		isLast := i == len(devices)-1
		p.printDevice(device, "", isLast)
	}
}

func (p *Printer) printHeader() {
	header := color.New(color.FgCyan, color.Bold)
	header.Println("USB Device Tree:")
	fmt.Println()
}

func (p *Printer) printNoDevices() {
	warning := color.New(color.FgYellow)
	warning.Println("No USB devices found")
	fmt.Println("\nNote: This tool requires libusb-1.0 to be installed.")
	fmt.Println("On macOS: brew install libusb")
	fmt.Println("On Linux: sudo apt-get install libusb-1.0-0 (or equivalent)")
}

func (p *Printer) printDevice(device *models.USBDevice, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	
	treeColor := color.New(color.FgHiBlack)
	nameColor := color.New(color.FgWhite, color.Bold)
	idColor := color.New(color.FgGreen)
	classColor := color.New(color.FgMagenta)
	
	fmt.Print(prefix)
	treeColor.Print(connector)
	
	name := device.GetDisplayName()
	nameColor.Print(name)
	
	fmt.Print(" ")
	idColor.Printf("[%s]", device.GetIDString())
	
	if device.Class != "" && device.Class != "Device" {
		fmt.Print(" ")
		classColor.Printf("(%s)", device.Class)
	}
	
	fmt.Println()
	
	if p.formatter.verbose {
		p.printDetails(device, prefix, isLast)
	}
	
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}
	
	for i, child := range device.Children {
		isLastChild := i == len(device.Children)-1
		p.printDevice(child, childPrefix, isLastChild)
	}
}

func (p *Printer) printDetails(device *models.USBDevice, prefix string, isLast bool) {
	detailPrefix := prefix
	if isLast {
		detailPrefix += "    "
	} else {
		detailPrefix += "│   "
	}
	
	detailColor := color.New(color.FgHiBlack)
	valueColor := color.New(color.FgCyan)
	
	if device.Serial != "" {
		fmt.Print(detailPrefix)
		detailColor.Print("├─ Serial: ")
		valueColor.Println(device.Serial)
	}
	
	if device.Speed != "" && device.Speed != "Unknown" {
		fmt.Print(detailPrefix)
		detailColor.Print("├─ Speed: ")
		valueColor.Println(device.Speed)
	}
	
	if device.MaxPower != "" {
		fmt.Print(detailPrefix)
		detailColor.Print("├─ Max Power: ")
		valueColor.Println(device.MaxPower)
	}
	
	fmt.Print(detailPrefix)
	detailColor.Print("└─ ")
	valueColor.Printf("Bus %d, Port %d, Address %d\n", 
		device.Bus, device.Port, device.Address)
}