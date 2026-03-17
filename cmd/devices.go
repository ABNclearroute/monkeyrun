package cmd

import (
	"context"
	"fmt"

	"monkeyrun/device"

	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List connected Android devices and booted iOS simulators",
	RunE:  runDevices,
}

func init() {
	rootCmd.AddCommand(devicesCmd)
}

func runDevices(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	fmt.Println("Android devices:")
	androidIDs, err := device.DetectAndroidDevices(ctx)
	if err != nil {
		fmt.Println("  (adb not found or error:", err.Error()+")")
	} else if len(androidIDs) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, id := range androidIDs {
			fmt.Println("  ", id)
		}
	}
	fmt.Println("iOS booted simulator:")
	iosUDID, err := device.DetectIOSBootedSimulator(ctx)
	if err != nil {
		fmt.Println("  (xcrun/simctl error:", err.Error()+")")
	} else if iosUDID == "" {
		fmt.Println("  (none)")
	} else {
		fmt.Println("  ", iosUDID)
	}
	return nil
}
