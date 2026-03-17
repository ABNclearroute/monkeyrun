package engine

import (
	"monkeyrun/device"
	"testing"
)

func TestHashElementsDeterministic(t *testing.T) {
	elements := []device.UIElement{
		{Text: "Login", X: 10, Y: 20, Width: 100, Height: 40, Clickable: true},
		{Text: "Password", X: 10, Y: 80, Width: 100, Height: 40, InputField: true},
	}
	h1 := hashElements(elements)
	h2 := hashElements(elements)
	if h1 != h2 {
		t.Error("hash should be deterministic for same input")
	}
}

func TestHashElementsChangesOnDiff(t *testing.T) {
	a := []device.UIElement{{Text: "A", X: 0, Y: 0, Width: 50, Height: 50}}
	b := []device.UIElement{{Text: "B", X: 0, Y: 0, Width: 50, Height: 50}}
	if hashElements(a) == hashElements(b) {
		t.Error("different elements should produce different hashes")
	}
}

func TestShouldCaptureMinimal(t *testing.T) {
	s := &Screenshotter{config: ScreenshotConfig{Mode: ScreenshotMinimal, Interval: 10}}
	if s.ShouldCapture(10, nil) {
		t.Error("minimal mode should never capture for regular events")
	}
	if s.ShouldCapture(1, nil) {
		t.Error("minimal mode should never capture for regular events")
	}
}

func TestShouldCaptureFull(t *testing.T) {
	s := &Screenshotter{config: ScreenshotConfig{Mode: ScreenshotFull}}
	for i := 1; i <= 5; i++ {
		if !s.ShouldCapture(i, nil) {
			t.Errorf("full mode should capture every event, failed at %d", i)
		}
	}
}

func TestShouldCaptureBalancedInterval(t *testing.T) {
	s := &Screenshotter{config: ScreenshotConfig{Mode: ScreenshotBalanced, Interval: 10}}
	if !s.ShouldCapture(10, nil) {
		t.Error("balanced mode should capture at interval multiples")
	}
	if !s.ShouldCapture(20, nil) {
		t.Error("balanced mode should capture at interval multiples")
	}
}

func TestShouldCaptureBalancedUIChange(t *testing.T) {
	s := &Screenshotter{config: ScreenshotConfig{Mode: ScreenshotBalanced, Interval: 100}}
	elA := []device.UIElement{{Text: "A", X: 0, Y: 0, Width: 50, Height: 50}}
	elB := []device.UIElement{{Text: "B", X: 0, Y: 0, Width: 50, Height: 50}}

	// First call always sees a change (empty → A)
	if !s.ShouldCapture(1, elA) {
		t.Error("should capture on first UI state")
	}
	// Same state — no capture (and not on interval)
	if s.ShouldCapture(2, elA) {
		t.Error("should not capture when UI unchanged")
	}
	// Different state — capture
	if !s.ShouldCapture(3, elB) {
		t.Error("should capture when UI changes")
	}
}
