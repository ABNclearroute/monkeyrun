package device

import "context"

// UIElement represents an actionable UI element from the device's view hierarchy.
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

// CenterPoint returns the center (cx, cy) of the element's bounding box.
func (e *UIElement) CenterPoint() (cx, cy int) {
	return e.X + e.Width/2, e.Y + e.Height/2
}

// DeviceInfo holds metadata about a connected device.
type DeviceInfo struct {
	Platform     string // "android" or "ios"
	ID           string // ADB serial or simulator UDID
	Name         string // human-readable (e.g. "Pixel_5", "iPhone 15 Pro")
	ScreenWidth  int
	ScreenHeight int
}

// Gesturer performs touch gestures on a device.
type Gesturer interface {
	Tap(ctx context.Context, x, y int) error
	DoubleTap(ctx context.Context, x, y int) error
	LongPress(ctx context.Context, x, y int, durationMs int) error
	Swipe(ctx context.Context, x1, y1, x2, y2 int) error
	Type(ctx context.Context, text string) error
	Back(ctx context.Context) error
}

// Inspector reads state from a device (hierarchy, screenshots).
type Inspector interface {
	GetUIHierarchy(ctx context.Context) ([]UIElement, error)
	Screenshot(ctx context.Context, path string) error
}

// Logger streams device logs for crash detection.
type Logger interface {
	StartLogStream(ctx context.Context, logCh chan<- string) error
}

// Device is the full platform-agnostic interface, composed of focused capabilities.
type Device interface {
	Gesturer
	Inspector
	Logger

	// Info returns metadata about this device.
	Info() DeviceInfo
	// SetTouchVisuals enables/disables on-screen touch indicators (best-effort; no-op on platforms that don't support it).
	SetTouchVisuals(ctx context.Context, enabled bool) error
}

// Platform is a convenience that returns dev.Info().Platform.
func Platform(dev Device) string { return dev.Info().Platform }

// ID is a convenience that returns dev.Info().ID.
func ID(dev Device) string { return dev.Info().ID }
