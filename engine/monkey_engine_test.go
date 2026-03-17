package engine

import (
	"math/rand"
	"monkeyrun/device"
	"testing"
)

// newTestMonkey creates a Monkey with all actions allowed (default behaviour).
func newTestMonkey(seed int64) *Monkey {
	return NewMonkey(nil, RunConfig{Events: 1})
}

// newTestMonkeyWithSeed creates a test Monkey with a fixed random seed and all actions.
func newTestMonkeyWithSeed(seed int64) *Monkey {
	m := NewMonkey(nil, RunConfig{Events: 1})
	m.rand = rand.New(rand.NewSource(seed))
	return m
}

// newFilteredMonkey creates a Monkey restricted to the given actions.
func newFilteredMonkey(seed int64, actions []ActionType) *Monkey {
	m := NewMonkey(nil, RunConfig{Events: 1, AllowedActions: actions})
	m.rand = rand.New(rand.NewSource(seed))
	return m
}

func TestWeightedActionDistribution(t *testing.T) {
	m := newTestMonkeyWithSeed(42)
	counts := map[ActionType]int{}
	n := 10000
	for i := 0; i < n; i++ {
		a := m.weightedAction(m.rand, nil)
		counts[a]++
	}
	expect := map[ActionType]int{
		Tap: 36, DoubleTap: 8, LongPress: 8,
		Swipe: 16, Scroll: 8, Type: 5, Back: 4,
		PinchIn: 4, PinchOut: 4, Home: 3,
		ClearText: 2, RotateDevice: 2,
	}
	for action, pct := range expect {
		got := float64(counts[action]) / float64(n) * 100
		if got < float64(pct)-5 || got > float64(pct)+5 {
			t.Errorf("%s: expected ~%d%%, got %.1f%%", action, pct, got)
		}
	}
}

func TestAllActionTypesAppear(t *testing.T) {
	m := newTestMonkeyWithSeed(99)
	seen := map[ActionType]bool{}
	for i := 0; i < 5000; i++ {
		a := m.weightedAction(m.rand, nil)
		seen[a] = true
	}
	all := AllActionTypes()
	for _, a := range all {
		if !seen[a] {
			t.Errorf("action %s never appeared in 5000 rolls", a)
		}
	}
}

func TestInputFieldPrefersClearText(t *testing.T) {
	m := newTestMonkeyWithSeed(42)
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
	m := newTestMonkeyWithSeed(42)
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
	m := newTestMonkeyWithSeed(42)
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
	m := newTestMonkeyWithSeed(42)
	a := m.selectAction(nil, 1)
	if a.Element != nil {
		t.Error("expected nil element when no elements available")
	}
	if a.X != 400 || a.Y != 600 {
		t.Errorf("expected default coords (400,600), got (%d,%d)", a.X, a.Y)
	}
}

func TestSelectActionUsesElementCenter(t *testing.T) {
	m := newTestMonkeyWithSeed(1)
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

// --- Filtered action tests ---

func TestFilteredActionsOnlyProduceAllowed(t *testing.T) {
	allowed := []ActionType{Tap, Swipe}
	m := newFilteredMonkey(42, allowed)
	set := map[ActionType]bool{Tap: true, Swipe: true}
	for i := 0; i < 5000; i++ {
		a := m.weightedAction(m.rand, nil)
		if !set[a] {
			t.Fatalf("got disallowed action %s when only %v allowed", a, allowed)
		}
	}
}

func TestFilteredActionsWeightRedistribution(t *testing.T) {
	allowed := []ActionType{Tap, Swipe}
	m := newFilteredMonkey(42, allowed)
	counts := map[ActionType]int{}
	n := 10000
	for i := 0; i < n; i++ {
		counts[m.weightedAction(m.rand, nil)]++
	}
	// Tap weight=36, Swipe weight=16 → Tap ~69%, Swipe ~31%
	tapPct := float64(counts[Tap]) / float64(n) * 100
	if tapPct < 55 || tapPct > 85 {
		t.Errorf("expected Tap ~69%%, got %.1f%%", tapPct)
	}
}

func TestFilteredActionsInputFieldFallback(t *testing.T) {
	// Only allow Tap — Type/ClearText not allowed, so input field should still produce Tap.
	m := newFilteredMonkey(42, []ActionType{Tap})
	el := &device.UIElement{InputField: true, X: 10, Y: 10, Width: 100, Height: 40}
	for i := 0; i < 500; i++ {
		a := m.weightedAction(m.rand, el)
		if a != Tap {
			t.Fatalf("expected only Tap when it's the sole allowed action, got %s", a)
		}
	}
}

func TestParseActionsValid(t *testing.T) {
	actions, err := ParseActions("tap,swipe,type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
}

func TestParseActionsInvalid(t *testing.T) {
	_, err := ParseActions("tap,invalidAction")
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestParseActionsEmpty(t *testing.T) {
	_, err := ParseActions("")
	if err == nil {
		t.Fatal("expected error for empty actions")
	}
}

func TestParseActionsCaseInsensitive(t *testing.T) {
	actions, err := ParseActions("Tap,SWIPE,DoubleTap")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
}

func TestParseActionsDeduplication(t *testing.T) {
	actions, err := ParseActions("tap,tap,swipe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 unique actions, got %d", len(actions))
	}
}
