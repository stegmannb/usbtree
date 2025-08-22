package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/usbtree/internal/models"
	"github.com/user/usbtree/internal/tree"
	"github.com/user/usbtree/internal/usb"
)

var (
	jsonOutput bool
	verbose    bool
	filter     string
)

var rootCmd = &cobra.Command{
	Use:   "usbtree",
	Short: "Display USB devices in a tree view",
	Long: `USBTree is a cross-platform CLI tool that displays connected USB devices
in a hierarchical tree structure. It works on both macOS and Linux systems.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		detector := usb.NewDetector()
		
		devices, err := detector.GetDevices()
		if err != nil {
			return fmt.Errorf("failed to get USB devices: %w", err)
		}

		if filter != "" {
			devices = filterDevices(devices, filter)
		}

		if jsonOutput {
			return outputJSON(devices)
		}

		printer := tree.NewPrinter(verbose)
		printer.Print(devices)
		
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output in JSON format")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed device information")
	rootCmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter devices by vendor name")
}

func filterDevices(devices []*models.USBDevice, filter string) []*models.USBDevice {
	var filtered []*models.USBDevice
	for _, device := range devices {
		if containsFilter(device, filter) {
			filtered = append(filtered, device)
		}
	}
	return filtered
}

func containsFilter(device *models.USBDevice, filter string) bool {
	if device.VendorName == filter || device.ProductName == filter {
		return true
	}
	
	for _, child := range device.Children {
		if containsFilter(child, filter) {
			return true
		}
	}
	
	return false
}

func outputJSON(devices []*models.USBDevice) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(devices)
}