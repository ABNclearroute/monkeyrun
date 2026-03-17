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
		Tap: 40, DoubleTap: 10, LongPress: 10,
		Swipe: 20, Scroll: 10, Type: 5, Back: 5,
	}
	for action, pct := range expect {
		got := float64(counts[action]) / float64(n) * 100
		if got < float64(pct)-5 || got > float64(pct)+5 {
			t.Errorf("%s: expected ~%d%%, got %.1f%%", action, pct, got)
		}
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
