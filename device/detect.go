package device

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
)

// DetectAndroidDevices returns device IDs from adb devices (one per line, skip "List" and empty).
func DetectAndroidDevices(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, "adb", "devices").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var ids []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == "device" {
			ids = append(ids, parts[0])
		}
	}
	return ids, nil
}

// DetectIOSBootedSimulator returns UDID of first booted simulator from xcrun simctl list devices.
func DetectIOSBootedSimulator(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "xcrun", "simctl", "list", "devices", "-j").Output()
	if err != nil {
		return "", err
	}
	// Parse JSON: devices by runtime, then look for "state": "Booted"
	var root map[string]interface{}
	if err := json.Unmarshal(out, &root); err != nil {
		return "", err
	}
	devices, _ := root["devices"].(map[string]interface{})
	for _, runtimes := range devices {
		list, _ := runtimes.([]interface{})
		for _, d := range list {
			m, _ := d.(map[string]interface{})
			if m["state"] == "Booted" {
				if udid, ok := m["udid"].(string); ok {
					return udid, nil
				}
			}
		}
	}
	return "", nil
}

