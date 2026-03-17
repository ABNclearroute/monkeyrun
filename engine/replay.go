package engine

import (
	"context"
	"monkeyrun/device"
	"math/rand"
)

// Replay runs a sequence of logged events on the device (element matched by text when possible).
func Replay(ctx context.Context, dev device.Device, events []EventLog, onEvent func(EventLog)) error {
	for i, ev := range events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		elements, _ := dev.GetUIHierarchy(ctx)
		action := eventToAction(ev, elements)
		_ = ExecuteAction(ctx, dev, action, 0, 0)
		if onEvent != nil {
			onEvent(ev)
		}
		_ = i
	}
	return nil
}

func eventToAction(ev EventLog, elements []device.UIElement) Action {
	at := actionTypeFromString(ev.Action)
	a := Action{Type: at, X: 400, Y: 600}
	var match *device.UIElement
	for i := range elements {
		if elements[i].Text == ev.Element || elements[i].ResourceID == ev.Element {
			match = &elements[i]
			break
		}
	}
	if match != nil {
		a.Element = match
		a.X, a.Y = match.X+match.Width/2, match.Y+match.Height/2
	} else if len(elements) > 0 {
		idx := rand.Intn(len(elements))
		a.Element = &elements[idx]
		a.X, a.Y = a.Element.X+a.Element.Width/2, a.Element.Y+a.Element.Height/2
	}
	if at == Type {
		a.Text = ev.Element
		if a.Text == "" {
			a.Text = randomTypingSample()
		}
	}
	return a
}

func actionTypeFromString(s string) ActionType {
	switch s {
	case "tap":
		return Tap
	case "doubleTap":
		return DoubleTap
	case "longPress":
		return LongPress
	case "swipe", "scroll":
		return Swipe
	case "type":
		return Type
	case "back":
		return Back
	default:
		return Tap
	}
}
