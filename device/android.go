package device

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// AndroidDevice implements Device using ADB.
type AndroidDevice struct {
	Serial string // ADB device serial (empty = first device)
}

// NewAndroidDevice creates an Android device adapter for the given device ID (empty = first device).
func NewAndroidDevice(deviceID string) *AndroidDevice {
	return &AndroidDevice{Serial: deviceID}
}

func (d *AndroidDevice) Platform() string { return "android" }
func (d *AndroidDevice) DeviceID() string { return d.Serial }

func (d *AndroidDevice) adb(ctx context.Context, args ...string) ([]byte, error) {
	cmdArgs := []string{}
	if d.Serial != "" {
		cmdArgs = append(cmdArgs, "-s", d.Serial)
	}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.CommandContext(ctx, "adb", cmdArgs...)
	return cmd.Output()
}

func (d *AndroidDevice) adbCombined(ctx context.Context, args ...string) (string, error) {
	out, err := d.adb(ctx, args...)
	return strings.TrimSpace(string(out)), err
}

// GetUIHierarchy dumps UI via uiautomator and parses XML.
func (d *AndroidDevice) GetUIHierarchy(ctx context.Context) ([]UIElement, error) {
	_, err := d.adb(ctx, "shell", "uiautomator", "dump", "/sdcard/window_dump.xml")
	if err != nil {
		return nil, fmt.Errorf("uiautomator dump: %w", err)
	}
	xmlOut, err := d.adb(ctx, "exec-out", "cat", "/sdcard/window_dump.xml")
	if err != nil {
		return nil, fmt.Errorf("cat dump: %w", err)
	}
	return ParseAndroidUIXML(string(xmlOut))
}

// Tap runs input tap x y.
func (d *AndroidDevice) Tap(ctx context.Context, x, y int) error {
	_, err := d.adb(ctx, "shell", "input", "tap", fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
	return err
}

// DoubleTap performs two taps with short delay.
func (d *AndroidDevice) DoubleTap(ctx context.Context, x, y int) error {
	if err := d.Tap(ctx, x, y); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return d.Tap(ctx, x, y)
}

// LongPress uses swipe from point to same point with duration.
func (d *AndroidDevice) LongPress(ctx context.Context, x, y int, duration int) error {
	if duration <= 0 {
		duration = 500
	}
	_, err := d.adb(ctx, "shell", "input", "swipe",
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
		fmt.Sprintf("%d", duration))
	return err
}

// Swipe from (x1,y1) to (x2,y2) with default 300ms.
func (d *AndroidDevice) Swipe(ctx context.Context, x1, y1, x2, y2 int) error {
	duration := 300
	_, err := d.adb(ctx, "shell", "input", "swipe",
		fmt.Sprintf("%d", x1), fmt.Sprintf("%d", y1),
		fmt.Sprintf("%d", x2), fmt.Sprintf("%d", y2),
		fmt.Sprintf("%d", duration))
	return err
}

// Type sends input text (ADB escapes spaces as %s).
func (d *AndroidDevice) Type(ctx context.Context, text string) error {
	escaped := strings.ReplaceAll(text, " ", "%s")
	_, err := d.adb(ctx, "shell", "input", "text", escaped)
	return err
}

// Back sends keyevent 4.
func (d *AndroidDevice) Back(ctx context.Context) error {
	_, err := d.adb(ctx, "shell", "input", "keyevent", "4")
	return err
}

// Screenshot captures screen via exec-out screencap -p.
func (d *AndroidDevice) Screenshot(ctx context.Context, path string) error {
	out, err := d.adb(ctx, "exec-out", "screencap", "-p")
	if err != nil {
		return err
	}
	return writeFile(path, out)
}

// StartLogStream runs adb logcat and sends lines to logCh.
func (d *AndroidDevice) StartLogStream(ctx context.Context, logCh chan<- string) error {
	args := []string{}
	if d.Serial != "" {
		args = append(args, "-s", d.Serial)
	}
	args = append(args, "logcat", "-v", "time")
	cmd := exec.CommandContext(ctx, "adb", args...)
	cmd.Stderr = nil
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go streamLines(ctx, stdout, logCh)
	go cmd.Wait()
	return nil
}

// SetAndroidTouchVisuals toggles Android's built-in touch visualizations (best-effort).
// Requires developer options enabled on the device/emulator.
func SetAndroidTouchVisuals(ctx context.Context, d *AndroidDevice, enabled bool) error {
	v := "0"
	if enabled {
		v = "1"
	}
	// show_touches (visual touch dots) and pointer_location (crosshair + trail).
	_, _ = d.adb(ctx, "shell", "settings", "put", "system", "show_touches", v)
	_, _ = d.adb(ctx, "shell", "settings", "put", "system", "pointer_location", v)
	return nil
}
