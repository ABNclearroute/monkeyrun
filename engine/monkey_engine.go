package engine

import (
	"context"
	"fmt"
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
	// DelayMinMs/DelayMaxMs control per-event human-like delay. If both are 0, defaults to 200–800ms.
	DelayMinMs int
	DelayMaxMs int
	// HierarchyEvery controls how often UI hierarchy is refreshed. If 0, defaults to 1 (every event).
	HierarchyEvery int
	// StopOnCrash cancels the run when a fatal crash is detected (default: true).
	StopOnCrash bool
	// Screenshot strategy configuration.
	ScreenshotCfg ScreenshotConfig
	OnEvent     func(EventLog)
	OnCrash     func(CrashInfo)
	ReplayFile  string // optional: replay from JSON
}

// EventLog is one logged event.
type EventLog struct {
	Event      int    `json:"event"`
	Platform   string `json:"platform"`
	Action     string `json:"action"`
	Element    string `json:"element,omitempty"`
	X          int    `json:"x,omitempty"`
	Y          int    `json:"y,omitempty"`
	Status     string `json:"status"`
	Time       string `json:"time,omitempty"`
	Screenshot bool   `json:"screenshot"`
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
	dev          device.Device
	config       RunConfig
	rand         *rand.Rand
	mu           sync.Mutex
	screenshotter *Screenshotter
}

// NewMonkey creates a monkey engine for the given device.
func NewMonkey(dev device.Device, config RunConfig) *Monkey {
	return &Monkey{
		dev:    dev,
		config: config,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Screenshotter returns the engine's screenshotter (may be nil).
func (m *Monkey) Screenshotter() *Screenshotter { return m.screenshotter }

// Run executes the monkey test for config.Events iterations.
func (m *Monkey) Run(ctx context.Context) (events int, crashes int, err error) {
	hEvery := m.config.HierarchyEvery
	if hEvery <= 0 {
		hEvery = 1
	}

	if m.config.ScreenshotCfg.Dir != "" {
		m.screenshotter = NewScreenshotter(m.dev, m.config.ScreenshotCfg, 2)
		defer m.screenshotter.Close()
	}

	var cached []device.UIElement
	var cachedAt int
	consecutiveHierarchyErrors := 0
	const maxConsecutiveErrors = 5

	for i := 1; i <= m.config.Events; i++ {
		select {
		case <-ctx.Done():
			return i - 1, crashes, ctx.Err()
		default:
		}

		if cached == nil || (i-cachedAt) >= hEvery {
			elements, hErr := m.dev.GetUIHierarchy(ctx)
			if hErr != nil {
				consecutiveHierarchyErrors++
				if m.config.OnEvent != nil {
					m.config.OnEvent(EventLog{Event: i, Platform: device.Platform(m.dev), Action: "hierarchy", Status: hErr.Error(), Time: time.Now().Format(time.RFC3339)})
				}
				if consecutiveHierarchyErrors >= maxConsecutiveErrors {
					return i - 1, crashes, fmt.Errorf("stopped: %d consecutive UI hierarchy errors (app may have crashed or left foreground)", maxConsecutiveErrors)
				}
			} else {
				consecutiveHierarchyErrors = 0
				cached = elements
				cachedAt = i
			}
		}

		_, crashed, runErr := m.runOneWithElements(ctx, i, cached)
		events = i
		if crashed {
			crashes++
		}
		if runErr != nil && runErr != context.Canceled {
			if m.config.OnEvent != nil {
				m.config.OnEvent(EventLog{Event: i, Platform: device.Platform(m.dev), Action: "error", Status: runErr.Error(), Time: time.Now().Format(time.RFC3339)})
			}
		}
	}
	return events, crashes, nil
}

func (m *Monkey) runOneWithElements(ctx context.Context, eventNum int, elements []device.UIElement) (EventLog, bool, error) {
	action := m.selectAction(elements, eventNum)
	elDesc := ""
	if action.Element != nil {
		elDesc = action.Element.Text
		if elDesc == "" {
			elDesc = action.Element.ResourceID
		}
	}
	err := ExecuteAction(ctx, m.dev, action, m.config.DelayMinMs, m.config.DelayMaxMs)
	status := "ok"
	if err != nil {
		status = err.Error()
	}

	tookScreenshot := false
	if m.screenshotter != nil && m.screenshotter.ShouldCapture(eventNum, elements) {
		m.screenshotter.Enqueue(eventNum)
		tookScreenshot = true
	}

	ev := EventLog{
		Event:      eventNum,
		Platform:   device.Platform(m.dev),
		Action:     string(action.Type),
		Element:    elDesc,
		X:          action.X,
		Y:          action.Y,
		Status:     status,
		Time:       time.Now().Format(time.RFC3339),
		Screenshot: tookScreenshot,
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
