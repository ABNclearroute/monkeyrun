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
	udid       string
	wdaBaseURL string
	client     *http.Client
	info       DeviceInfo
}

// NewIOSDevice creates an iOS device adapter. wdaBaseURL defaults to http://localhost:8100.
func NewIOSDevice(udid, wdaBaseURL string) *IOSDevice {
	if wdaBaseURL == "" {
		wdaBaseURL = "http://localhost:8100"
	}
	d := &IOSDevice{
		udid:       udid,
		wdaBaseURL: strings.TrimSuffix(wdaBaseURL, "/"),
		client:     &http.Client{Timeout: 30 * time.Second},
	}
	d.info = DeviceInfo{
		Platform: "ios",
		ID:       udid,
		Name:     d.probeName(),
	}
	return d
}

func (d *IOSDevice) Info() DeviceInfo { return d.info }

func (d *IOSDevice) probeName() string {
	out, err := exec.Command("xcrun", "simctl", "list", "devices", "-j").Output()
	if err != nil {
		return d.udid
	}
	var root map[string]interface{}
	if err := json.Unmarshal(out, &root); err != nil {
		return d.udid
	}
	devices, _ := root["devices"].(map[string]interface{})
	for _, runtimes := range devices {
		list, _ := runtimes.([]interface{})
		for _, dev := range list {
			m, _ := dev.(map[string]interface{})
			if m["udid"] == d.udid {
				if name, ok := m["name"].(string); ok {
					return name
				}
			}
		}
	}
	return d.udid
}

// --- transport ---

func (d *IOSDevice) wda(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := d.wdaBaseURL + path
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

// --- Gesturer ---

func (d *IOSDevice) Tap(ctx context.Context, x, y int) error {
	payload := fmt.Sprintf(`{"x":%d,"y":%d}`, x, y)
	_, err := d.wda(ctx, "POST", "/wda/tap/0", []byte(payload))
	return err
}

func (d *IOSDevice) DoubleTap(ctx context.Context, x, y int) error {
	payload := fmt.Sprintf(`{"x":%d,"y":%d}`, x, y)
	_, err := d.wda(ctx, "POST", "/wda/doubleTap/0", []byte(payload))
	return err
}

func (d *IOSDevice) LongPress(ctx context.Context, x, y int, durationMs int) error {
	if durationMs <= 0 {
		durationMs = 1000
	}
	sec := durationMs / 1000
	if sec < 1 {
		sec = 1
	}
	payload := fmt.Sprintf(`{"x":%d,"y":%d,"duration":%d}`, x, y, sec)
	_, err := d.wda(ctx, "POST", "/wda/touchAndHold", []byte(payload))
	return err
}

func (d *IOSDevice) Swipe(ctx context.Context, x1, y1, x2, y2 int) error {
	payload := fmt.Sprintf(`{"fromX":%d,"fromY":%d,"toX":%d,"toY":%d,"duration":0.3}`, x1, y1, x2, y2)
	_, err := d.wda(ctx, "POST", "/wda/dragfromtoforduration", []byte(payload))
	return err
}

func (d *IOSDevice) Type(ctx context.Context, text string) error {
	escaped, _ := json.Marshal(text)
	_, err := d.wda(ctx, "POST", "/wda/keys", escaped)
	return err
}

func (d *IOSDevice) Back(ctx context.Context) error {
	_, err := d.wda(ctx, "POST", "/wda/homescreen", nil)
	return err
}

// --- Inspector ---

func (d *IOSDevice) GetUIHierarchy(ctx context.Context) ([]UIElement, error) {
	body, err := d.wda(ctx, "GET", "/source", nil)
	if err != nil {
		return nil, fmt.Errorf("wda source: %w", err)
	}
	return parseIOSSource(body)
}

func (d *IOSDevice) Screenshot(ctx context.Context, path string) error {
	target := "booted"
	if d.udid != "" {
		target = d.udid
	}
	return exec.CommandContext(ctx, "xcrun", "simctl", "io", target, "screenshot", path).Run()
}

// --- Logger ---

func (d *IOSDevice) StartLogStream(ctx context.Context, logCh chan<- string) error {
	target := "booted"
	if d.udid != "" {
		target = d.udid
	}
	cmd := exec.CommandContext(ctx, "xcrun", "simctl", "spawn", target, "log", "stream")
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

// --- SetTouchVisuals (no-op on iOS) ---

func (d *IOSDevice) SetTouchVisuals(_ context.Context, _ bool) error {
	return nil
}
