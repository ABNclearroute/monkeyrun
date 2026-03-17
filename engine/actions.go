package engine

import "monkeyrun/device"

// ActionType represents the kind of gesture/action to perform.
type ActionType string

const (
	Tap       ActionType = "tap"
	DoubleTap ActionType = "doubleTap"
	LongPress ActionType = "longPress"
	Swipe     ActionType = "swipe"
	Scroll    ActionType = "scroll"
	Type      ActionType = "type"
	Back      ActionType = "back"
)

// Action represents a single executable action with optional element and params.
type Action struct {
	Type     ActionType
	Element  *device.UIElement
	Text     string
	Duration int
	// For swipe/scroll: from (X,Y) to (X2,Y2)
	X, Y, X2, Y2 int
}
