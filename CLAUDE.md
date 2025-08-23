# USBTree Development Guide

This file contains development information and build instructions for the USBTree project.

## Project Overview

USBTree is a cross-platform CLI tool that displays connected USB devices in a hierarchical tree structure. It's built in Go using the Cobra CLI framework and the gousb library for USB device detection.

## Architecture

The project follows a clean architecture pattern with the following structure:

```
usbtree/
├── main.go                 # Entry point
├── cmd/
│   └── root.go            # Cobra CLI commands and flags
├── internal/
│   ├── usb/               # USB device detection
│   │   ├── detector.go    # Interface definition
│   │   ├── detector_linux.go   # Linux implementation
│   │   └── detector_darwin.go  # macOS implementation
│   ├── tree/              # Output formatting
│   │   ├── formatter.go   # Tree structure formatting
│   │   └── printer.go     # Terminal output with colors
│   └── models/            # Data structures
│       └── device.go      # USB device model
└── README.md
```

## Dependencies

- **Core**: Go 1.19+
- **CLI Framework**: `github.com/spf13/cobra`
- **USB Library**: `github.com/google/gousb` (requires libusb-1.0)
- **Colors**: `github.com/fatih/color`

## System Requirements

### macOS
```bash
brew install libusb pkg-config
```

### Linux
```bash
# Ubuntu/Debian
sudo apt-get install libusb-1.0-0-dev pkg-config

# Fedora/RHEL
sudo dnf install libusb-devel pkgconf-pkg-config

# Arch Linux
sudo pacman -S libusb pkgconf
```

## Build Commands

### Development Build
```bash
go build -o usbtree .
```

### Cross-Platform Builds
```bash
# Linux from macOS
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o usbtree-linux

# macOS from Linux
GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o usbtree-macos
```

### Release Build with Static Linking
```bash
CGO_ENABLED=1 go build -ldflags "-s -w" -o usbtree .
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Tests with Verbose Output
```bash
go test ./... -v
```

### Run Specific Package Tests
```bash
go test ./internal/models -v
go test ./internal/tree -v
go test ./internal/usb -v
```

### Test Coverage
```bash
go test ./... -cover
```

## Key Features

### USB Hub Detection
The application creates virtual root hubs for each USB bus to show the complete USB topology:
- USB 2.0 Root Hubs: `1d6b:0002` with High Speed (480 Mbps)
- USB 3.0 Root Hubs: `1d6b:0003` with Super Speed (5 Gbps)

### Cross-Platform Implementation
Uses Go build tags to provide platform-specific USB detection:
- `//go:build darwin` for macOS
- `//go:build linux` for Linux

### Output Formats
- **Tree View**: Hierarchical display with Unicode tree characters
- **JSON**: Structured output for scripting and automation
- **Verbose**: Detailed device information including speed, power, bus details
- **Filtered**: Display only devices matching vendor name

## Development Workflow

### Adding New Features
1. Update the appropriate model in `internal/models/`
2. Implement detection logic in platform-specific detectors
3. Update formatting in `internal/tree/`
4. Add comprehensive unit tests
5. Update documentation

### Testing Strategy
- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test USB detection with mock devices
- **Manual Testing**: Test with real USB devices on different platforms

### Code Style
- Follow Go conventions and `gofmt` formatting
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and single-purpose

## Troubleshooting

### Common Issues

#### No USB Devices Detected
- Ensure libusb-1.0 is installed
- Check permissions (may need sudo on Linux)
- Verify USB devices are properly connected

#### Build Errors
- Ensure CGO is enabled: `CGO_ENABLED=1`
- Install pkg-config: Required for libusb linking
- Check cross-compilation toolchain setup

#### Permission Errors (Linux)
```bash
# Add udev rule for USB access
echo 'SUBSYSTEM=="usb", MODE="0666"' | sudo tee /etc/udev/rules.d/99-usb.rules
sudo udevadm control --reload-rules
```

### Debug Mode
Enable libusb debug output by setting debug level in the detector:
```go
ctx.Debug(4) // Add to detector implementation
```

## Performance Considerations

- USB enumeration can be slow on systems with many devices
- The application caches device information during enumeration
- Root hub creation is performed only when needed
- Color output can be disabled for faster rendering

## Contributing

When contributing to this project:
1. Ensure all tests pass
2. Add tests for new functionality
3. Update documentation as needed
4. Follow the existing code structure and patterns
5. Test on multiple platforms when possible

## Deployment

### Single Binary Distribution
The application compiles to a single binary with no runtime dependencies except libusb-1.0.

### Package Managers
Future releases can be distributed via:
- Homebrew (macOS)
- APT/YUM repositories (Linux)
- Go module proxy

## Security Considerations

- USB device access requires appropriate permissions
- The application only reads device metadata, no data transfer
- No network communication or external dependencies
- Input validation on all user-provided filter strings
- Adhere to the Conventional Commits specification if nothing else is specified 
 https://www.conventionalcommits.org/en/v1.0.0/#specification