package engine

import (
	"context"
	"math"
	"math/rand"
	"monkeyrun/device"
	"time"
)

// ExecuteAction runs a single action on the device with human-like behavior.
func ExecuteAction(ctx context.Context, dev device.Device, action Action) error {
	// Human-like delay 200-800ms
	delay := 200 + rand.Intn(600)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(delay) * time.Millisecond):
	}
	return execute(ctx, dev, action)
}

func execute(ctx context.Context, dev device.Device, action Action) error {
	switch action.Type {
	case Tap:
		x, y := action.X, action.Y
		if action.Element != nil {
			x, y = randomPointInElement(action.Element)
		}
		return dev.Tap(ctx, x, y)
	case DoubleTap:
		x, y := action.X, action.Y
		if action.Element != nil {
			x, y = randomPointInElement(action.Element)
		}
		return dev.DoubleTap(ctx, x, y)
	case LongPress:
		x, y := action.X, action.Y
		if action.Element != nil {
			x, y = randomPointInElement(action.Element)
		}
		dur := action.Duration
		if dur <= 0 {
			dur = 500 + rand.Intn(500)
		}
		return dev.LongPress(ctx, x, y, dur)
	case Swipe, Scroll:
		x1, y1, x2, y2 := action.X, action.Y, action.X2, action.Y2
		if action.Element != nil && (x1 == 0 && y1 == 0 && x2 == 0 && y2 == 0) {
			x1, y1 = randomPointInElement(action.Element)
			dx, dy := randomSwipeDelta()
			x2, y2 = x1+dx, y1+dy
		}
		return dev.Swipe(ctx, x1, y1, x2, y2)
	case Type:
		text := action.Text
		if text == "" {
			text = randomTypingSample()
		}
		return dev.Type(ctx, text)
	case Back:
		return dev.Back(ctx)
	default:
		return dev.Tap(ctx, action.X, action.Y)
	}
}

func randomPointInElement(el *device.UIElement) (x, y int) {
	if el.Width <= 0 || el.Height <= 0 {
		return el.X, el.Y
	}
	x = el.X + 4 + rand.Intn(el.Width-8)
	y = el.Y + 4 + rand.Intn(el.Height-8)
	if x < el.X {
		x = el.X
	}
	if y < el.Y {
		y = el.Y
	}
	return x, y
}

func randomSwipeDelta() (dx, dy int) {
	angle := rand.Float64() * 2 * math.Pi
	dist := 150 + rand.Intn(200)
	dx = int(float64(dist) * math.Cos(angle))
	dy = int(float64(dist) * math.Sin(angle))
	return dx, dy
}

var typingSamples = []string{"hello", "test", "foo", "bar", "a", "1", "ok"}

func randomTypingSample() string {
	return typingSamples[rand.Intn(len(typingSamples))]
}
