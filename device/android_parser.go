package device

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type androidNode struct {
	XMLName xml.Name      `xml:"node"`
	Text    string        `xml:"text,attr"`
	ResID   string        `xml:"resource-id,attr"`
	Bounds  string        `xml:"bounds,attr"`
	Click   string        `xml:"clickable,attr"`
	Input   string        `xml:"focusable,attr"`
	Nodes   []androidNode `xml:"node"`
}

func parseAndroidUIXML(xmlContent string) ([]UIElement, error) {
	xmlContent = strings.TrimSpace(xmlContent)
	if idx := strings.Index(xmlContent, "<"); idx > 0 {
		xmlContent = xmlContent[idx:]
	}
	var root struct {
		Nodes []androidNode `xml:"node"`
	}
	dec := xml.NewDecoder(strings.NewReader(xmlContent))
	dec.CharsetReader = nil
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode UI XML: %w", err)
	}
	var out []UIElement
	collectAndroidElements(&root.Nodes, &out)
	return out, nil
}

func collectAndroidElements(nodes *[]androidNode, out *[]UIElement) {
	for i := range *nodes {
		n := &(*nodes)[i]
		x, y, w, h := parseBounds(n.Bounds)
		if w <= 0 || h <= 0 {
			collectAndroidElements(&n.Nodes, out)
			continue
		}
		clickable := strings.EqualFold(n.Click, "true")
		input := strings.EqualFold(n.Input, "true") || strings.Contains(strings.ToLower(n.ResID), "edit")
		hasText := strings.TrimSpace(n.Text) != ""
		hasID := strings.TrimSpace(n.ResID) != ""
		if clickable || input || (hasText && hasID) {
			*out = append(*out, UIElement{
				Text: n.Text, ResourceID: n.ResID,
				X: x, Y: y, Width: w, Height: h,
				Clickable: clickable, InputField: input,
			})
		}
		collectAndroidElements(&n.Nodes, out)
	}
}

func parseBounds(bounds string) (x, y, w, h int) {
	parts := strings.Split(bounds, "][")
	if len(parts) != 2 {
		return 0, 0, 0, 0
	}
	lt := strings.Split(strings.Trim(parts[0], "[]"), ",")
	rb := strings.Split(strings.Trim(parts[1], "[]"), ",")
	if len(lt) != 2 || len(rb) != 2 {
		return 0, 0, 0, 0
	}
	x, _ = strconv.Atoi(strings.TrimSpace(lt[0]))
	y, _ = strconv.Atoi(strings.TrimSpace(lt[1]))
	r, _ := strconv.Atoi(strings.TrimSpace(rb[0]))
	b, _ := strconv.Atoi(strings.TrimSpace(rb[1]))
	return x, y, r - x, b - y
}
