package engine

import (
	"math/rand"
	"monkeyrun/device"
	"testing"
)

func TestWeightedActionDistribution(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(42))}
	counts := map[ActionType]int{}
	n := 10000
	for i := 0; i < n; i++ {
		a := m.weightedAction(m.rand, nil)
		counts[a]++
	}
	expect := map[ActionType]int{
		Tap: 35, DoubleTap: 8, LongPress: 8,
		Swipe: 15, Scroll: 8, Type: 4, Back: 4,
		PinchIn: 4, PinchOut: 4, Home: 3,
		OpenNotifications: 3, ClearText: 2, RotateDevice: 2,
	}
	for action, pct := range expect {
		got := float64(counts[action]) / float64(n) * 100
		if got < float64(pct)-5 || got > float64(pct)+5 {
			t.Errorf("%s: expected ~%d%%, got %.1f%%", action, pct, got)
		}
	}
}

func TestAllActionTypesAppear(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(99))}
	seen := map[ActionType]bool{}
	for i := 0; i < 5000; i++ {
		a := m.weightedAction(m.rand, nil)
		seen[a] = true
	}
	all := []ActionType{Tap, DoubleTap, LongPress, Swipe, Scroll, Type, Back,
		PinchIn, PinchOut, Home, OpenNotifications, ClearText, RotateDevice}
	for _, a := range all {
		if !seen[a] {
			t.Errorf("action %s never appeared in 5000 rolls", a)
		}
	}
}

func TestInputFieldPrefersClearText(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(42))}
	el := &device.UIElement{InputField: true, X: 10, Y: 10, Width: 100, Height: 40}
	clearCount := 0
	n := 1000
	for i := 0; i < n; i++ {
		a := m.weightedAction(m.rand, el)
		if a == ClearText {
			clearCount++
		}
	}
	pct := float64(clearCount) / float64(n) * 100
	if pct < 5 {
		t.Errorf("expected ClearText to appear >5%% for input fields, got %.1f%%", pct)
	}
}

func TestWeightedActionPrefersTypeForInput(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(42))}
	el := &device.UIElement{InputField: true, X: 10, Y: 10, Width: 100, Height: 40}
	typeCount := 0
	n := 1000
	for i := 0; i < n; i++ {
		a := m.weightedAction(m.rand, el)
		if a == Type {
			typeCount++
		}
	}
	pct := float64(typeCount) / float64(n) * 100
	if pct < 30 {
		t.Errorf("expected Type to be selected >30%% for input fields, got %.1f%%", pct)
	}
}

func TestWeightedActionPrefersSwipeForScrollable(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(42))}
	el := &device.UIElement{Scrollable: true, X: 0, Y: 0, Width: 400, Height: 800}
	swipeCount := 0
	n := 1000
	for i := 0; i < n; i++ {
		a := m.weightedAction(m.rand, el)
		if a == Swipe {
			swipeCount++
		}
	}
	pct := float64(swipeCount) / float64(n) * 100
	if pct < 30 {
		t.Errorf("expected Swipe to be selected >30%% for scrollable, got %.1f%%", pct)
	}
}

func TestSelectActionWithNoElements(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(42))}
	a := m.selectAction(nil, 1)
	if a.Element != nil {
		t.Error("expected nil element when no elements available")
	}
	if a.X != 400 || a.Y != 600 {
		t.Errorf("expected default coords (400,600), got (%d,%d)", a.X, a.Y)
	}
}

func TestSelectActionUsesElementCenter(t *testing.T) {
	m := &Monkey{rand: rand.New(rand.NewSource(1))}
	elements := []device.UIElement{
		{Text: "OK", X: 100, Y: 200, Width: 80, Height: 40, Clickable: true},
	}
	a := m.selectAction(elements, 1)
	if a.Element == nil {
		t.Fatal("expected element to be selected")
	}
	cx, cy := 100+80/2, 200+40/2
	if a.X != cx || a.Y != cy {
		t.Errorf("expected center (%d,%d), got (%d,%d)", cx, cy, a.X, a.Y)
	}
}
