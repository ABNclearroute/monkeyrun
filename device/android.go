package device

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// AndroidDevice implements Device using ADB.
type AndroidDevice struct {
	serial string
	info   DeviceInfo
}

// NewAndroidDevice creates an Android device adapter for the given serial (empty = first device).
func NewAndroidDevice(serial string) *AndroidDevice {
	d := &AndroidDevice{serial: serial}
	d.info = DeviceInfo{
		Platform: "android",
		ID:       serial,
		Name:     d.probeName(),
	}
	d.probeScreen()
	return d
}

func (d *AndroidDevice) Info() DeviceInfo { return d.info }

func (d *AndroidDevice) probeName() string {
	out, err := d.adb(context.Background(), "shell", "getprop", "ro.product.model")
	if err != nil {
		return d.serial
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return d.serial
	}
	return name
}

func (d *AndroidDevice) probeScreen() {
	out, err := d.adb(context.Background(), "shell", "wm", "size")
	if err != nil {
		return
	}
	// "Physical size: 1080x1920"
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "size") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				wh := strings.Split(parts[len(parts)-1], "x")
				if len(wh) == 2 {
					d.info.ScreenWidth, _ = strconv.Atoi(wh[0])
					d.info.ScreenHeight, _ = strconv.Atoi(wh[1])
				}
			}
		}
	}
}

// --- transport ---

func (d *AndroidDevice) adb(ctx context.Context, args ...string) ([]byte, error) {
	cmdArgs := make([]string, 0, len(args)+2)
	if d.serial != "" {
		cmdArgs = append(cmdArgs, "-s", d.serial)
	}
	cmdArgs = append(cmdArgs, args...)
	return exec.CommandContext(ctx, "adb", cmdArgs...).Output()
}

// --- Gesturer ---

func (d *AndroidDevice) Tap(ctx context.Context, x, y int) error {
	_, err := d.adb(ctx, "shell", "input", "tap", itoa(x), itoa(y))
	return err
}

func (d *AndroidDevice) DoubleTap(ctx context.Context, x, y int) error {
	if err := d.Tap(ctx, x, y); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return d.Tap(ctx, x, y)
}

func (d *AndroidDevice) LongPress(ctx context.Context, x, y int, durationMs int) error {
	if durationMs <= 0 {
		durationMs = 500
	}
	_, err := d.adb(ctx, "shell", "input", "swipe",
		itoa(x), itoa(y), itoa(x), itoa(y), itoa(durationMs))
	return err
}

func (d *AndroidDevice) Swipe(ctx context.Context, x1, y1, x2, y2 int) error {
	_, err := d.adb(ctx, "shell", "input", "swipe",
		itoa(x1), itoa(y1), itoa(x2), itoa(y2), "300")
	return err
}

func (d *AndroidDevice) Type(ctx context.Context, text string) error {
	escaped := strings.ReplaceAll(text, " ", "%s")
	_, err := d.adb(ctx, "shell", "input", "text", escaped)
	return err
}

func (d *AndroidDevice) Back(ctx context.Context) error {
	_, err := d.adb(ctx, "shell", "input", "keyevent", "4")
	return err
}

// --- Inspector ---

func (d *AndroidDevice) GetUIHierarchy(ctx context.Context) ([]UIElement, error) {
	if _, err := d.adb(ctx, "shell", "uiautomator", "dump", "/sdcard/window_dump.xml"); err != nil {
		return nil, fmt.Errorf("uiautomator dump: %w", err)
	}
	xmlOut, err := d.adb(ctx, "exec-out", "cat", "/sdcard/window_dump.xml")
	if err != nil {
		return nil, fmt.Errorf("cat dump: %w", err)
	}
	return parseAndroidUIXML(string(xmlOut))
}

func (d *AndroidDevice) Screenshot(ctx context.Context, path string) error {
	out, err := d.adb(ctx, "exec-out", "screencap", "-p")
	if err != nil {
		return err
	}
	return writeFile(path, out)
}

// --- Logger ---

func (d *AndroidDevice) StartLogStream(ctx context.Context, logCh chan<- string) error {
	args := make([]string, 0, 6)
	if d.serial != "" {
		args = append(args, "-s", d.serial)
	}
	args = append(args, "logcat", "-v", "time")
	cmd := exec.CommandContext(ctx, "adb", args...)
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

// --- SetTouchVisuals ---

func (d *AndroidDevice) SetTouchVisuals(ctx context.Context, enabled bool) error {
	v := "0"
	if enabled {
		v = "1"
	}
	d.adb(ctx, "shell", "settings", "put", "system", "show_touches", v)
	d.adb(ctx, "shell", "settings", "put", "system", "pointer_location", v)
	return nil
}

func itoa(n int) string { return strconv.Itoa(n) }
