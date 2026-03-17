package device

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseIOSSource(body []byte) ([]UIElement, error) {
	s := strings.TrimSpace(string(body))
	if s == "" {
		return nil, nil
	}
	if strings.HasPrefix(s, "<") {
		return parseIOSXML(s)
	}
	if strings.HasPrefix(s, "{") {
		return parseIOSJSON(body)
	}
	return nil, fmt.Errorf("unknown iOS source format")
}

// --- XML format (WDA /source) ---

type iosNode struct {
	XMLName    xml.Name
	Type       string    `xml:"type,attr"`
	Name       string    `xml:"name,attr"`
	Label      string    `xml:"label,attr"`
	Value      string    `xml:"value,attr"`
	Enabled    string    `xml:"enabled,attr"`
	Visible    string    `xml:"visible,attr"`
	Accessible string    `xml:"accessible,attr"`
	X          string    `xml:"x,attr"`
	Y          string    `xml:"y,attr"`
	Width      string    `xml:"width,attr"`
	Height     string    `xml:"height,attr"`
	Frame      string    `xml:"frame,attr"`
	Children   []iosNode `xml:",any"`
}

func parseIOSXML(s string) ([]UIElement, error) {
	var root iosNode
	if err := xml.NewDecoder(strings.NewReader(s)).Decode(&root); err != nil {
		return nil, fmt.Errorf("ios xml decode: %w", err)
	}
	return collectIOSElements([]iosNode{root}), nil
}

var iosClickableTypes = map[string]bool{
	"XCUIElementTypeButton":        true,
	"XCUIElementTypeCell":          true,
	"XCUIElementTypeLink":          true,
	"XCUIElementTypeMenuItem":      true,
	"XCUIElementTypeTab":           true,
	"XCUIElementTypeSwitch":        true,
	"XCUIElementTypeToggle":        true,
	"XCUIElementTypeSegmentedControl": true,
	"XCUIElementTypeStepper":       true,
	"XCUIElementTypeSlider":        true,
	"XCUIElementTypePickerWheel":   true,
	"XCUIElementTypePageIndicator": true,
	"XCUIElementTypeImage":         true,
	"XCUIElementTypeIcon":          true,
	"XCUIElementTypeStaticText":    true,
}

var iosInputTypes = map[string]bool{
	"XCUIElementTypeTextField":       true,
	"XCUIElementTypeSecureTextField": true,
	"XCUIElementTypeTextView":        true,
	"XCUIElementTypeSearchField":     true,
}

var iosScrollableTypes = map[string]bool{
	"XCUIElementTypeScrollView": true,
	"XCUIElementTypeTable":      true,
	"XCUIElementTypeCollectionView": true,
	"XCUIElementTypeWebView":    true,
}

func collectIOSElements(nodes []iosNode) []UIElement {
	var out []UIElement
	for _, n := range nodes {
		x, y, w, h := iosNodeRect(n)
		if w <= 0 || h <= 0 {
			out = append(out, collectIOSElements(n.Children)...)
			continue
		}

		visible := attrBool(n.Visible, true)
		enabled := attrBool(n.Enabled, true)
		if !visible {
			out = append(out, collectIOSElements(n.Children)...)
			continue
		}

		text := n.Name
		if text == "" {
			text = n.Label
		}
		if text == "" {
			text = n.Value
		}

		clickable := enabled && iosClickableTypes[n.Type]
		input := iosInputTypes[n.Type]
		scrollable := iosScrollableTypes[n.Type]
		accessible := attrBool(n.Accessible, false)

		actionable := clickable || input || scrollable
		identifiable := text != "" || accessible

		if !actionable && !identifiable {
			out = append(out, collectIOSElements(n.Children)...)
			continue
		}

		out = append(out, UIElement{
			Text: text, ResourceID: n.Name,
			X: x, Y: y, Width: w, Height: h,
			Clickable: clickable, InputField: input, Scrollable: scrollable,
		})
		out = append(out, collectIOSElements(n.Children)...)
	}
	return out
}

func iosNodeRect(n iosNode) (x, y, w, h int) {
	// Try individual attributes first (WDA sometimes provides these).
	if n.X != "" && n.Y != "" && n.Width != "" && n.Height != "" {
		x, _ = strconv.Atoi(n.X)
		y, _ = strconv.Atoi(n.Y)
		w, _ = strconv.Atoi(n.Width)
		h, _ = strconv.Atoi(n.Height)
		if w > 0 && h > 0 {
			return x, y, w, h
		}
	}
	return parseIOSFrame(n.Frame)
}

func parseIOSFrame(frame string) (x, y, w, h int) {
	// "{{x, y}, {w, h}}" or "x,y,w,h"
	clean := strings.NewReplacer("{", "", "}", "", " ", "").Replace(frame)
	parts := strings.Split(clean, ",")
	if len(parts) >= 4 {
		xf, _ := strconv.ParseFloat(parts[0], 64)
		yf, _ := strconv.ParseFloat(parts[1], 64)
		wf, _ := strconv.ParseFloat(parts[2], 64)
		hf, _ := strconv.ParseFloat(parts[3], 64)
		return int(math.Round(xf)), int(math.Round(yf)), int(math.Round(wf)), int(math.Round(hf))
	}
	return 0, 0, 0, 0
}

// --- JSON format (WDA sometimes returns JSON tree) ---

func parseIOSJSON(body []byte) ([]UIElement, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	// "value" might be an XML string
	if v, ok := raw["value"].(string); ok && strings.HasPrefix(strings.TrimSpace(v), "<") {
		return parseIOSXML(v)
	}
	// "value" might be a JSON tree
	if v, ok := raw["value"].(map[string]interface{}); ok {
		var out []UIElement
		collectIOSJSONTree(v, &out)
		return out, nil
	}
	return nil, nil
}

func collectIOSJSONTree(node map[string]interface{}, out *[]UIElement) {
	typ, _ := node["type"].(string)
	label, _ := node["label"].(string)
	name, _ := node["name"].(string)
	value, _ := node["value"].(string)
	enabled, _ := node["enabled"].(bool)
	visible, _ := node["visible"].(bool)

	if !visible {
		goto children
	}

	{
		rect := jsonRect(node)
		if rect[2] <= 0 || rect[3] <= 0 {
			goto children
		}

		text := name
		if text == "" {
			text = label
		}
		if text == "" {
			text = value
		}

		clickable := enabled && iosClickableTypes[typ]
		input := iosInputTypes[typ]
		scrollable := iosScrollableTypes[typ]

		if clickable || input || scrollable || text != "" {
			*out = append(*out, UIElement{
				Text: text, ResourceID: name,
				X: rect[0], Y: rect[1], Width: rect[2], Height: rect[3],
				Clickable: clickable, InputField: input, Scrollable: scrollable,
			})
		}
	}

children:
	if kids, ok := node["children"].([]interface{}); ok {
		for _, k := range kids {
			if km, ok := k.(map[string]interface{}); ok {
				collectIOSJSONTree(km, out)
			}
		}
	}
}

func jsonRect(node map[string]interface{}) [4]int {
	// Try "rect" or "frame" sub-object
	for _, key := range []string{"rect", "frame"} {
		if r, ok := node[key].(map[string]interface{}); ok {
			x := jsonInt(r, "x")
			y := jsonInt(r, "y")
			w := jsonInt(r, "width")
			h := jsonInt(r, "height")
			if w > 0 && h > 0 {
				return [4]int{x, y, w, h}
			}
		}
	}
	return [4]int{}
}

func jsonInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(math.Round(v))
	case int:
		return v
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}
