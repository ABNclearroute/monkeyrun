package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// IOSDevice implements Device using WebDriverAgent (WDA) and simctl.
type IOSDevice struct {
	UDID       string // UDID of booted simulator
	WDABaseURL string // e.g. http://localhost:8100
	client     *http.Client
}

// NewIOSDevice creates an iOS device adapter. deviceID empty = first booted simulator.
func NewIOSDevice(deviceID, wdaBaseURL string) *IOSDevice {
	if wdaBaseURL == "" {
		wdaBaseURL = "http://localhost:8100"
	}
	return &IOSDevice{
		UDID:       deviceID,
		WDABaseURL: strings.TrimSuffix(wdaBaseURL, "/"),
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *IOSDevice) Platform() string { return "ios" }
func (d *IOSDevice) DeviceID() string { return d.UDID }

func (d *IOSDevice) wda(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := d.WDABaseURL + path
	var req *http.Request
	var err error
	if len(body) > 0 {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetUIHierarchy fetches source from WDA and parses to UI elements.
func (d *IOSDevice) GetUIHierarchy(ctx context.Context) ([]UIElement, error) {
	// WDA: GET /source or /session/xxx/source
	body, err := d.wda(ctx, "GET", "/source", nil)
	if err != nil {
		return nil, fmt.Errorf("wda source: %w", err)
	}
	return ParseIOSSource(body)
}

// Tap uses WDA tap at coordinates.
func (d *IOSDevice) Tap(ctx context.Context, x, y int) error {
	payload := fmt.Sprintf(`{"x":%d,"y":%d}`, x, y)
	_, err := d.wda(ctx, "POST", "/wda/tap/0", []byte(payload))
	return err
}

// DoubleTap uses WDA doubleTap.
func (d *IOSDevice) DoubleTap(ctx context.Context, x, y int) error {
	payload := fmt.Sprintf(`{"x":%d,"y":%d}`, x, y)
	_, err := d.wda(ctx, "POST", "/wda/doubleTap/0", []byte(payload))
	return err
}

// LongPress uses WDA touchAndHold with duration (seconds).
func (d *IOSDevice) LongPress(ctx context.Context, x, y int, duration int) error {
	if duration <= 0 {
		duration = 1000
	}
	sec := duration / 1000
	if sec < 1 {
		sec = 1
	}
	payload := fmt.Sprintf(`{"x":%d,"y":%d,"duration":%d}`, x, y, sec)
	_, err := d.wda(ctx, "POST", "/wda/touchAndHold", []byte(payload))
	return err
}

// Swipe uses WDA dragfromtoforduration.
func (d *IOSDevice) Swipe(ctx context.Context, x1, y1, x2, y2 int) error {
	duration := 0.3
	payload := fmt.Sprintf(`{"fromX":%d,"fromY":%d,"toX":%d,"toY":%d,"duration":%f}`, x1, y1, x2, y2, duration)
	_, err := d.wda(ctx, "POST", "/wda/dragfromtoforduration", []byte(payload))
	return err
}

// Type sends keys via WDA.
func (d *IOSDevice) Type(ctx context.Context, text string) error {
	escaped, _ := json.Marshal(text)
	_, err := d.wda(ctx, "POST", "/wda/keys", escaped)
	return err
}

// Back - iOS has no back button; send home or rely on app UI. Send keycode home.
func (d *IOSDevice) Back(ctx context.Context) error {
	// WDA may support keyevent or we use simctl. For simulator: keyevent 4 not applicable. Use home.
	_, err := d.wda(ctx, "POST", "/wda/homescreen", nil)
	return err
}

// Screenshot uses simctl io booted screenshot.
func (d *IOSDevice) Screenshot(ctx context.Context, path string) error {
	args := []string{"simctl", "io", "booted", "screenshot", path}
	if d.UDID != "" {
		args = []string{"simctl", "io", d.UDID, "screenshot", path}
	}
	cmd := exec.CommandContext(ctx, "xcrun", args...)
	return cmd.Run()
}

// StartLogStream uses simctl spawn booted log stream.
func (d *IOSDevice) StartLogStream(ctx context.Context, logCh chan<- string) error {
	args := []string{"simctl", "spawn", "booted", "log", "stream"}
	if d.UDID != "" {
		args = []string{"simctl", "spawn", d.UDID, "log", "stream"}
	}
	cmd := exec.CommandContext(ctx, "xcrun", args...)
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
