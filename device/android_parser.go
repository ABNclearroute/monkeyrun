package device

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

type androidNode struct {
	XMLName      xml.Name      `xml:"node"`
	Text         string        `xml:"text,attr"`
	ResID        string        `xml:"resource-id,attr"`
	Class        string        `xml:"class,attr"`
	ContentDesc  string        `xml:"content-desc,attr"`
	Bounds       string        `xml:"bounds,attr"`
	Clickable    string        `xml:"clickable,attr"`
	LongClick    string        `xml:"long-clickable,attr"`
	Scrollable   string        `xml:"scrollable,attr"`
	Focusable    string        `xml:"focusable,attr"`
	Enabled      string        `xml:"enabled,attr"`
	Checkable    string        `xml:"checkable,attr"`
	Nodes        []androidNode `xml:"node"`
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

		enabled := attrBool(n.Enabled, true)
		if !enabled {
			collectAndroidElements(&n.Nodes, out)
			continue
		}

		clickable := attrBool(n.Clickable, false)
		longClickable := attrBool(n.LongClick, false)
		scrollable := attrBool(n.Scrollable, false)
		focusable := attrBool(n.Focusable, false)
		checkable := attrBool(n.Checkable, false)

		input := isAndroidInputField(n.Class, n.ResID, focusable)

		hasText := strings.TrimSpace(n.Text) != ""
		hasDesc := strings.TrimSpace(n.ContentDesc) != ""
		hasID := strings.TrimSpace(n.ResID) != ""

		actionable := clickable || longClickable || scrollable || input || checkable
		identifiable := hasText || hasDesc || hasID

		if !actionable && !identifiable {
			collectAndroidElements(&n.Nodes, out)
			continue
		}

		text := n.Text
		if text == "" {
			text = n.ContentDesc
		}

		*out = append(*out, UIElement{
			Text: text, ResourceID: n.ResID,
			X: x, Y: y, Width: w, Height: h,
			Clickable:  clickable || longClickable || checkable,
			InputField: input,
			Scrollable: scrollable,
		})
		collectAndroidElements(&n.Nodes, out)
	}
}

var androidInputClasses = []string{
	"edittext", "autocompletextview", "searchview",
	"textinputedittext", "appcompatedittext",
}

func isAndroidInputField(class, resID string, focusable bool) bool {
	lc := strings.ToLower(class)
	for _, c := range androidInputClasses {
		if strings.Contains(lc, c) {
			return true
		}
	}
	lr := strings.ToLower(resID)
	if strings.Contains(lr, "edit") || strings.Contains(lr, "input") || strings.Contains(lr, "search") {
		return true
	}
	if focusable && (strings.Contains(lc, "text") && !strings.Contains(lc, "textview")) {
		return true
	}
	return false
}

func attrBool(val string, defaultVal bool) bool {
	if val == "" {
		return defaultVal
	}
	return strings.EqualFold(val, "true")
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
