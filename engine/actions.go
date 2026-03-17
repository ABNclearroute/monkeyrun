package engine

import (
	"fmt"
	"monkeyrun/device"
	"strings"
)

// ActionType represents the kind of gesture/action to perform.
type ActionType string

const (
	Tap          ActionType = "tap"
	DoubleTap    ActionType = "doubleTap"
	LongPress    ActionType = "longPress"
	Swipe        ActionType = "swipe"
	Scroll       ActionType = "scroll"
	Type         ActionType = "type"
	Back         ActionType = "back"
	PinchIn      ActionType = "pinchIn"
	PinchOut     ActionType = "pinchOut"
	Home         ActionType = "home"
	ClearText    ActionType = "clearText"
	RotateDevice ActionType = "rotateDevice"
)

// AllActions maps canonical names to ActionType for lookup and validation.
var AllActions = map[string]ActionType{
	"tap":          Tap,
	"doubletap":    DoubleTap,
	"longpress":    LongPress,
	"swipe":        Swipe,
	"scroll":       Scroll,
	"type":         Type,
	"back":         Back,
	"pinchin":      PinchIn,
	"pinchout":     PinchOut,
	"home":         Home,
	"cleartext":    ClearText,
	"rotatedevice": RotateDevice,
}

// AllActionTypes returns every supported ActionType in a stable order.
func AllActionTypes() []ActionType {
	return []ActionType{
		Tap, DoubleTap, LongPress, Swipe, Scroll, Type,
		Back, PinchIn, PinchOut, Home, ClearText, RotateDevice,
	}
}

// ActionNames returns a sorted, comma-separated list of valid action names.
func ActionNames() string {
	names := make([]string, 0, len(AllActions))
	for _, a := range AllActionTypes() {
		names = append(names, string(a))
	}
	return strings.Join(names, ", ")
}

// ParseActions splits a comma-separated string and validates each name.
// Returns the matching ActionTypes or an error listing invalid names.
func ParseActions(raw string) ([]ActionType, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("--actions cannot be empty; valid actions: %s", ActionNames())
	}
	parts := strings.Split(raw, ",")
	seen := map[ActionType]bool{}
	var result []ActionType
	var invalid []string

	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		at, ok := AllActions[strings.ToLower(name)]
		if !ok {
			invalid = append(invalid, name)
			continue
		}
		if !seen[at] {
			seen[at] = true
			result = append(result, at)
		}
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("unknown action(s): %s\nValid actions: %s",
			strings.Join(invalid, ", "), ActionNames())
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("--actions resolved to an empty set; valid actions: %s", ActionNames())
	}
	return result, nil
}

// Action represents a single executable action with optional element and params.
type Action struct {
	Type     ActionType
	Element  *device.UIElement
	Text     string
	Duration int
	Scale    float64 // for pinch gestures
	// For swipe/scroll: from (X,Y) to (X2,Y2)
	X, Y, X2, Y2 int
}
