package device

import (
	"testing"
)

func TestParseIOSXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<XCUIElementTypeApplication type="XCUIElementTypeApplication" name="TestApp" enabled="true" visible="true" x="0" y="0" width="390" height="844">
  <XCUIElementTypeButton type="XCUIElementTypeButton" name="Login" label="Login" enabled="true" visible="true" x="100" y="200" width="190" height="44">
  </XCUIElementTypeButton>
  <XCUIElementTypeTextField type="XCUIElementTypeTextField" name="Email" enabled="true" visible="true" x="50" y="300" width="290" height="40">
  </XCUIElementTypeTextField>
  <XCUIElementTypeStaticText type="XCUIElementTypeStaticText" name="Hidden" visible="false" x="0" y="0" width="100" height="20">
  </XCUIElementTypeStaticText>
</XCUIElementTypeApplication>`

	elements, err := parseIOSXML(xml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Hidden element should be skipped
	buttonFound := false
	inputFound := false
	for _, el := range elements {
		if el.Text == "Login" && el.Clickable {
			buttonFound = true
			if el.X != 100 || el.Y != 200 || el.Width != 190 || el.Height != 44 {
				t.Errorf("button bounds wrong: x=%d y=%d w=%d h=%d", el.X, el.Y, el.Width, el.Height)
			}
		}
		if el.Text == "Email" && el.InputField {
			inputFound = true
		}
		if el.Text == "Hidden" {
			t.Error("hidden element should have been skipped")
		}
	}
	if !buttonFound {
		t.Error("expected Login button to be found as clickable")
	}
	if !inputFound {
		t.Error("expected Email text field to be found as input")
	}
}

func TestParseIOSFrame(t *testing.T) {
	tests := []struct {
		frame   string
		x, y, w, h int
	}{
		{"{{0, 0}, {390, 844}}", 0, 0, 390, 844},
		{"{{100.5, 200.7}, {50.3, 30.9}}", 101, 201, 50, 31},
		{"0,0,100,200", 0, 0, 100, 200},
		{"", 0, 0, 0, 0},
	}
	for _, tc := range tests {
		x, y, w, h := parseIOSFrame(tc.frame)
		if x != tc.x || y != tc.y || w != tc.w || h != tc.h {
			t.Errorf("parseIOSFrame(%q) = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
				tc.frame, x, y, w, h, tc.x, tc.y, tc.w, tc.h)
		}
	}
}

func TestParseIOSJSON(t *testing.T) {
	body := []byte(`{
		"value": "<XCUIElementTypeApplication type=\"XCUIElementTypeApplication\" name=\"App\" visible=\"true\" enabled=\"true\" x=\"0\" y=\"0\" width=\"390\" height=\"844\"><XCUIElementTypeButton type=\"XCUIElementTypeButton\" name=\"OK\" visible=\"true\" enabled=\"true\" x=\"10\" y=\"20\" width=\"60\" height=\"30\"></XCUIElementTypeButton></XCUIElementTypeApplication>"
	}`)
	elements, err := parseIOSJSON(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, el := range elements {
		if el.Text == "OK" && el.Clickable {
			found = true
		}
	}
	if !found {
		t.Error("expected OK button from JSON-wrapped XML")
	}
}

func TestIOSClickableTypes(t *testing.T) {
	clickable := []string{
		"XCUIElementTypeButton", "XCUIElementTypeCell",
		"XCUIElementTypeLink", "XCUIElementTypeSwitch",
	}
	for _, typ := range clickable {
		if !iosClickableTypes[typ] {
			t.Errorf("expected %s to be clickable", typ)
		}
	}
	if iosClickableTypes["XCUIElementTypeWindow"] {
		t.Error("XCUIElementTypeWindow should not be clickable")
	}
}

func TestIOSInputTypes(t *testing.T) {
	inputs := []string{
		"XCUIElementTypeTextField", "XCUIElementTypeSecureTextField",
		"XCUIElementTypeTextView", "XCUIElementTypeSearchField",
	}
	for _, typ := range inputs {
		if !iosInputTypes[typ] {
			t.Errorf("expected %s to be input type", typ)
		}
	}
}
