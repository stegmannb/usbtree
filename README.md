# USBTree

A cross-platform CLI tool that displays connected USB devices in a hierarchical tree structure. Works on both macOS and Linux systems.

## Features

- Tree view visualization of USB device hierarchy
- **USB hub detection and display** - Shows root hubs and USB bus structure
- Colored output for better readability
- Detailed device information (vendor/product IDs, speed, power consumption)
- JSON output format for scripting
- Device filtering by vendor name
- Cross-platform support (macOS and Linux)

## Prerequisites

This tool requires `libusb-1.0` to be installed on your system:

### macOS
```bash
brew install libusb
```

### Linux
```bash
# Ubuntu/Debian
sudo apt-get install libusb-1.0-0-dev

# Fedora/RHEL
sudo dnf install libusb-devel

# Arch Linux
sudo pacman -S libusb
```

## Installation

```bash
go install github.com/user/usbtree@latest
```

Or build from source:

```bash
git clone https://github.com/user/usbtree
cd usbtree
go build -o usbtree
```

## Usage

### Basic Usage
Display all USB devices in tree format:
```bash
usbtree
```

### Verbose Mode
Show detailed information including serial numbers, speed, and power consumption:
```bash
usbtree --verbose
# or
usbtree -v
```

### JSON Output
Output device information in JSON format:
```bash
usbtree --json
# or
usbtree -j
```

### Filter Devices
Filter devices by vendor name:
```bash
usbtree --filter "Apple"
# or
usbtree -f "Apple"
```

### Help
Display help information:
```bash
usbtree --help
```

## Example Output

### Basic Tree View
```
USB Device Tree:

├── USB 2.0 Root Hub [1d6b:0002] (Hub)
├── USB 2.0 Root Hub [1d6b:0002] (Hub)
├── USB 3.0 Root Hub [1d6b:0003] (Hub)
└── USB 3.0 Root Hub [1d6b:0003] (Hub)
```

### With Connected Devices
```
USB Device Tree:

├── USB 2.0 Root Hub [1d6b:0002] (Hub)
│   ├── Apple Internal Keyboard [05ac:027e] (HID)
│   └── USB Receiver [046d:c52b] (HID)
│       ├── Logitech Mouse [046d:1001]
│       └── Logitech Keyboard [046d:2001]
└── USB 3.0 Root Hub [1d6b:0003] (Hub)
    └── USB 3.0 Hub [0451:8142] (Hub)
        └── SanDisk Ultra [0781:5591] (Mass Storage)
```

### Verbose Output
```
USB Device Tree:

├── USB 2.0 Root Hub [1d6b:0002] (Hub)
│   ├─ Speed: High (480 Mbps)
│   └─ Bus 1, Port 0, Address 1
└── USB 3.0 Root Hub [1d6b:0003] (Hub)
    ├─ Speed: Super (5 Gbps)
    └─ Bus 2, Port 0, Address 1
```

## Permissions

### Linux
On Linux, you may need to run the tool with `sudo` or configure udev rules to access USB devices without root privileges:

```bash
# Run with sudo
sudo usbtree

# Or create a udev rule (recommended)
echo 'SUBSYSTEM=="usb", MODE="0666"' | sudo tee /etc/udev/rules.d/99-usb.rules
sudo udevadm control --reload-rules
```

### macOS
On macOS, no special permissions are typically required.

## Building for Different Platforms

### Build for current platform
```bash
go build -o usbtree
```

### Cross-compile for Linux (from macOS)
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o usbtree-linux
```

### Cross-compile for macOS (from Linux)
```bash
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o usbtree-macos
```

Note: Cross-compilation with CGO requires appropriate cross-compilation toolchain.

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.