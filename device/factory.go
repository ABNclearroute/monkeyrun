package device

import (
	"context"
	"fmt"
)

// Options configures device creation.
type Options struct {
	DeviceID   string // override auto-detection
	WDABaseURL string // iOS only: WDA endpoint
}

// New auto-detects and creates a Device for the given platform.
// Pass opts.DeviceID to skip auto-detection.
func New(ctx context.Context, platform string, opts Options) (Device, error) {
	switch platform {
	case "android":
		id := opts.DeviceID
		if id == "" {
			ids, err := DetectAndroidDevices(ctx)
			if err != nil {
				return nil, fmt.Errorf("detect android devices: %w", err)
			}
			if len(ids) == 0 {
				return nil, fmt.Errorf("no android device found (run 'adb devices')")
			}
			id = ids[0]
		}
		return NewAndroidDevice(id), nil
	case "ios":
		id := opts.DeviceID
		if id == "" {
			var err error
			id, err = DetectIOSBootedSimulator(ctx)
			if err != nil {
				return nil, fmt.Errorf("detect ios simulator: %w", err)
			}
			if id == "" {
				return nil, fmt.Errorf("no booted ios simulator found (run 'xcrun simctl list devices')")
			}
		}
		return NewIOSDevice(id, opts.WDABaseURL), nil
	default:
		return nil, fmt.Errorf("unknown platform %q (use android or ios)", platform)
	}
}
