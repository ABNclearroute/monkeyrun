package device

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
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

type iosNode struct {
	XMLName  xml.Name  `xml:",any"`
	Type     string    `xml:"type,attr"`
	Name     string    `xml:"name,attr"`
	Label    string    `xml:"label,attr"`
	Value    string    `xml:"value,attr"`
	Enabled  string    `xml:"enabled,attr"`
	Frame    string    `xml:"frame,attr"`
	Children []iosNode `xml:",any"`
}

func parseIOSXML(s string) ([]UIElement, error) {
	var root iosNode
	if err := xml.NewDecoder(strings.NewReader(s)).Decode(&root); err != nil {
		return nil, fmt.Errorf("ios xml decode: %w", err)
	}
	return collectIOSElements([]iosNode{root}), nil
}

func collectIOSElements(nodes []iosNode) []UIElement {
	var out []UIElement
	for _, n := range nodes {
		x, y, w, h := parseIOSFrame(n.Frame)
		if w <= 0 || h <= 0 {
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
		clickable := strings.EqualFold(n.Enabled, "true") &&
			(n.Type == "XCUIElementTypeButton" || n.Type == "XCUIElementTypeCell" ||
				n.Type == "XCUIElementTypeStaticText" || text != "")
		input := strings.Contains(n.Type, "Field") || strings.Contains(n.Type, "Text")
		out = append(out, UIElement{
			Text: text, ResourceID: n.Name,
			X: x, Y: y, Width: w, Height: h,
			Clickable: clickable, InputField: input,
		})
		out = append(out, collectIOSElements(n.Children)...)
	}
	return out
}

func parseIOSFrame(frame string) (x, y, w, h int) {
	parts := strings.Split(frame, ",")
	if len(parts) >= 4 {
		x, _ = strconv.Atoi(strings.Trim(parts[0], " {}\t"))
		y, _ = strconv.Atoi(strings.Trim(parts[1], " {}\t"))
		w, _ = strconv.Atoi(strings.Trim(parts[2], " {}\t"))
		h, _ = strconv.Atoi(strings.Trim(parts[3], " {}\t"))
	}
	return x, y, w, h
}

func parseIOSJSON(body []byte) ([]UIElement, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if v, ok := raw["value"].(string); ok && strings.HasPrefix(strings.TrimSpace(v), "<") {
		return parseIOSXML(v)
	}
	return nil, nil
}
