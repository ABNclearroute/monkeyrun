package device

import "context"

// UIElement represents an actionable UI element from the hierarchy.
type UIElement struct {
	Text       string
	ResourceID string
	X          int
	Y          int
	Width      int
	Height     int
	Clickable  bool
	InputField bool
	Scrollable bool
}

// Bounds returns center point (cx, cy) for the element.
func (e *UIElement) Bounds() (cx, cy int) {
	cx = e.X + e.Width/2
	cy = e.Y + e.Height/2
	return cx, cy
}

// Device is the platform-agnostic interface for mobile devices.
type Device interface {
	// GetUIHierarchy returns actionable UI elements from the current screen.
	GetUIHierarchy(ctx context.Context) ([]UIElement, error)
	// Tap performs a tap at (x, y).
	Tap(ctx context.Context, x, y int) error
	// DoubleTap performs a double tap at (x, y).
	DoubleTap(ctx context.Context, x, y int) error
	// LongPress holds at (x, y) for duration milliseconds.
	LongPress(ctx context.Context, x, y int, duration int) error
	// Swipe from (x1,y1) to (x2,y2).
	Swipe(ctx context.Context, x1, y1, x2, y2 int) error
	// Type sends text input (for focused field or element).
	Type(ctx context.Context, text string) error
	// Back sends back button (Android) or equivalent.
	Back(ctx context.Context) error
	// Screenshot saves a screenshot to path.
	Screenshot(ctx context.Context, path string) error
	// StartLogStream sends log lines to the channel (for crash detection).
	StartLogStream(ctx context.Context, logCh chan<- string) error
	// Platform returns "android" or "ios".
	Platform() string
	// DeviceID returns the device/simulator identifier.
	DeviceID() string
}
