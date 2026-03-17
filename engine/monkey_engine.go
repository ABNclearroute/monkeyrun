package engine

import (
	"context"
	"math/rand"
	"monkeyrun/device"
	"sync"
	"time"
)

// Weights for action selection (must sum to 100).
const (
	weightTap       = 40
	weightDoubleTap = 10
	weightLongPress = 10
	weightSwipe     = 20
	weightScroll    = 10
	weightType      = 5
	weightBack      = 5
)

// RunConfig holds options for a monkey run.
type RunConfig struct {
	Events      int
	ReportDir   string
	Verbose     bool
	OnEvent     func(EventLog)
	OnCrash     func(CrashInfo)
	ReplayFile  string // optional: replay from JSON
}

// EventLog is one logged event.
type EventLog struct {
	Event    int    `json:"event"`
	Platform string `json:"platform"`
	Action   string `json:"action"`
	Element  string `json:"element,omitempty"`
	Status   string `json:"status"`
	Time     string `json:"time,omitempty"`
}

// CrashInfo holds crash details.
type CrashInfo struct {
	Event     int    `json:"event"`
	Message   string `json:"message"`
	Screenshot string `json:"screenshot,omitempty"`
	LogSnippet string `json:"log_snippet,omitempty"`
}

// Monkey runs the chaos test loop.
type Monkey struct {
	dev    device.Device
	config RunConfig
	rand   *rand.Rand
	mu     sync.Mutex
}

// NewMonkey creates a monkey engine for the given device.
func NewMonkey(dev device.Device, config RunConfig) *Monkey {
	return &Monkey{
		dev:    dev,
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Run executes the monkey test for config.Events iterations.
func (m *Monkey) Run(ctx context.Context) (events int, crashes int, err error) {
	for i := 1; i <= m.config.Events; i++ {
		select {
		case <-ctx.Done():
			return i - 1, crashes, ctx.Err()
		default:
		}
		_, crashed, runErr := m.runOne(ctx, i)
		events = i
		if crashed {
			crashes++
		}
		if runErr != nil && runErr != context.Canceled {
			if m.config.OnEvent != nil {
				m.config.OnEvent(EventLog{Event: i, Platform: m.dev.Platform(), Action: "error", Status: runErr.Error()})
			}
		}
		if crashed {
			// Optional: stop on first crash or continue; we continue by default
		}
	}
	return events, crashes, nil
}

func (m *Monkey) runOne(ctx context.Context, eventNum int) (EventLog, bool, error) {
	elements, err := m.dev.GetUIHierarchy(ctx)
	if err != nil {
		return EventLog{Event: eventNum, Status: "hierarchy_error"}, false, err
	}
	action := m.selectAction(elements, eventNum)
	elDesc := ""
	if action.Element != nil {
		elDesc = action.Element.Text
		if elDesc == "" {
			elDesc = action.Element.ResourceID
		}
	}
	err = ExecuteAction(ctx, m.dev, action)
	status := "ok"
	if err != nil {
		status = err.Error()
	}
	ev := EventLog{
		Event:    eventNum,
		Platform: m.dev.Platform(),
		Action:   string(action.Type),
		Element:  elDesc,
		Status:   status,
		Time:     time.Now().Format(time.RFC3339),
	}
	if m.config.OnEvent != nil {
		m.config.OnEvent(ev)
	}
	return ev, false, err
}

// selectAction picks a random element and action type with weighted probability and smart filtering.
func (m *Monkey) selectAction(elements []device.UIElement, eventNum int) Action {
	m.mu.Lock()
	r := m.rand
	m.mu.Unlock()

	var el *device.UIElement
	if len(elements) > 0 {
		idx := r.Intn(len(elements))
		el = &elements[idx]
	}

	// Screen bounds for swipe when no element (use first element or default)
	x, y := 400, 600
	if el != nil {
		x, y = el.X+el.Width/2, el.Y+el.Height/2
	}

	actionType := m.weightedAction(r, el)
	a := Action{Type: actionType, Element: el, X: x, Y: y}
	switch actionType {
	case Swipe, Scroll:
		dx, dy := randomSwipeDelta()
		a.X2, a.Y2 = x+dx, y+dy
	case LongPress:
		a.Duration = 500 + r.Intn(500)
	case Type:
		a.Text = randomTypingSample()
	}
	return a
}

func (m *Monkey) weightedAction(r *rand.Rand, el *device.UIElement) ActionType {
	// Smart: prefer tap/double/long for clickable, type for input, swipe for scrollable
	if el != nil {
		if el.InputField {
			if r.Intn(100) < 60 {
				return Type
			}
		}
		if el.Scrollable {
			if r.Intn(100) < 50 {
				return Swipe
			}
		}
		if el.Clickable {
			// keep normal weights
		}
	}
	roll := r.Intn(100)
	if roll < weightTap {
		return Tap
	}
	roll -= weightTap
	if roll < weightDoubleTap {
		return DoubleTap
	}
	roll -= weightDoubleTap
	if roll < weightLongPress {
		return LongPress
	}
	roll -= weightLongPress
	if roll < weightSwipe {
		return Swipe
	}
	roll -= weightSwipe
	if roll < weightScroll {
		return Scroll
	}
	roll -= weightScroll
	if roll < weightType {
		return Type
	}
	return Back
}
