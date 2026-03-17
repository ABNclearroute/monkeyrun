package device

import (
	"testing"
)

func TestParseAndroidUIXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<hierarchy>
  <node text="Login" resource-id="com.app:id/login_btn" class="android.widget.Button"
        bounds="[100,200][300,260]" clickable="true" enabled="true" focusable="true"
        long-clickable="false" scrollable="false" checkable="false">
  </node>
  <node text="" resource-id="com.app:id/username" class="android.widget.EditText"
        bounds="[50,300][400,360]" clickable="true" enabled="true" focusable="true"
        long-clickable="false" scrollable="false" checkable="false">
  </node>
  <node text="Disabled" resource-id="com.app:id/disabled" class="android.widget.Button"
        bounds="[100,400][300,460]" clickable="true" enabled="false" focusable="true"
        long-clickable="false" scrollable="false" checkable="false">
  </node>
</hierarchy>`

	elements, err := parseAndroidUIXML(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(elements) != 2 {
		t.Fatalf("expected 2 elements (disabled skipped), got %d", len(elements))
	}

	btn := elements[0]
	if btn.Text != "Login" {
		t.Errorf("expected text 'Login', got %q", btn.Text)
	}
	if btn.ResourceID != "com.app:id/login_btn" {
		t.Errorf("expected resource-id 'com.app:id/login_btn', got %q", btn.ResourceID)
	}
	if !btn.Clickable {
		t.Error("expected button to be clickable")
	}
	if btn.X != 100 || btn.Y != 200 || btn.Width != 200 || btn.Height != 60 {
		t.Errorf("unexpected bounds: x=%d y=%d w=%d h=%d", btn.X, btn.Y, btn.Width, btn.Height)
	}

	input := elements[1]
	if !input.InputField {
		t.Error("expected EditText to be detected as input field")
	}
}

func TestParseBounds(t *testing.T) {
	tests := []struct {
		input string
		x, y, w, h int
	}{
		{"[0,0][1080,1920]", 0, 0, 1080, 1920},
		{"[100,200][300,400]", 100, 200, 200, 200},
		{"invalid", 0, 0, 0, 0},
		{"", 0, 0, 0, 0},
	}
	for _, tc := range tests {
		x, y, w, h := parseBounds(tc.input)
		if x != tc.x || y != tc.y || w != tc.w || h != tc.h {
			t.Errorf("parseBounds(%q) = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
				tc.input, x, y, w, h, tc.x, tc.y, tc.w, tc.h)
		}
	}
}

func TestAttrBool(t *testing.T) {
	if !attrBool("true", false) {
		t.Error("expected true")
	}
	if attrBool("false", true) {
		t.Error("expected false")
	}
	if !attrBool("", true) {
		t.Error("expected default true")
	}
	if attrBool("", false) {
		t.Error("expected default false")
	}
}

func TestIsAndroidInputField(t *testing.T) {
	if !isAndroidInputField("android.widget.EditText", "", false) {
		t.Error("EditText should be input")
	}
	if !isAndroidInputField("", "com.app:id/search_input", false) {
		t.Error("resource-id with 'input' should be input")
	}
	if isAndroidInputField("android.widget.Button", "com.app:id/btn", false) {
		t.Error("Button should not be input")
	}
}
