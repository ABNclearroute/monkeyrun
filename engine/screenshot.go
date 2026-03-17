package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"monkeyrun/device"
	"path/filepath"
	"sort"
	"sync"
)

// ScreenshotMode controls when screenshots are captured.
type ScreenshotMode string

const (
	ScreenshotMinimal  ScreenshotMode = "minimal"
	ScreenshotBalanced ScreenshotMode = "balanced"
	ScreenshotFull     ScreenshotMode = "full"
)

// ScreenshotConfig holds screenshot strategy settings.
type ScreenshotConfig struct {
	Mode     ScreenshotMode
	Interval int // capture every N events (balanced/full)
	Dir      string
}

// ScreenshotJob is a unit of work for the async worker pool.
type ScreenshotJob struct {
	EventNum int
	Path     string
	Priority bool // crash screenshots bypass the queue
}

// Screenshotter manages hybrid screenshot capture with an async worker pool.
type Screenshotter struct {
	dev      device.Device
	config   ScreenshotConfig
	prevHash string
	mu       sync.Mutex

	// async worker pool
	jobs   chan ScreenshotJob
	wg     sync.WaitGroup
	taken  map[int]string // eventNum → filename
	takenMu sync.Mutex
}

// NewScreenshotter creates a screenshotter with the given config.
// Workers controls max concurrent screenshot goroutines.
func NewScreenshotter(dev device.Device, cfg ScreenshotConfig, workers int) *Screenshotter {
	if workers <= 0 {
		workers = 2
	}
	s := &Screenshotter{
		dev:    dev,
		config: cfg,
		jobs:   make(chan ScreenshotJob, 64),
		taken:  make(map[int]string),
	}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}
	return s
}

func (s *Screenshotter) worker() {
	defer s.wg.Done()
	for job := range s.jobs {
		ctx := context.Background()
		if err := s.dev.Screenshot(ctx, job.Path); err == nil {
			s.takenMu.Lock()
			s.taken[job.EventNum] = filepath.Base(job.Path)
			s.takenMu.Unlock()
		}
	}
}

// ShouldCapture returns true if a screenshot should be taken for this event,
// given the current UI elements (for hash comparison).
func (s *Screenshotter) ShouldCapture(eventNum int, elements []device.UIElement) bool {
	switch s.config.Mode {
	case ScreenshotMinimal:
		return false // only crash triggers capture
	case ScreenshotFull:
		return true
	default: // balanced
		interval := s.config.Interval
		if interval <= 0 {
			interval = 25
		}
		if eventNum%interval == 0 {
			return true
		}
		return s.uiChanged(elements)
	}
}

// Enqueue submits a screenshot job (non-blocking). Returns the filename.
func (s *Screenshotter) Enqueue(eventNum int) string {
	name := fmt.Sprintf("event_%d.png", eventNum)
	path := filepath.Join(s.config.Dir, name)
	select {
	case s.jobs <- ScreenshotJob{EventNum: eventNum, Path: path}:
	default:
		// queue full — drop non-priority screenshot
	}
	return name
}

// EnqueueCrash captures a crash screenshot synchronously (never dropped).
func (s *Screenshotter) EnqueueCrash(eventNum int) string {
	name := fmt.Sprintf("crash_%d.png", eventNum)
	path := filepath.Join(s.config.Dir, name)
	ctx := context.Background()
	if err := s.dev.Screenshot(ctx, path); err == nil {
		s.takenMu.Lock()
		s.taken[eventNum] = name
		s.takenMu.Unlock()
	}
	return name
}

// Close drains the job queue and waits for workers to finish.
func (s *Screenshotter) Close() {
	close(s.jobs)
	s.wg.Wait()
}

// TakenScreenshots returns a sorted list of screenshot filenames that were actually captured.
func (s *Screenshotter) TakenScreenshots() []string {
	s.takenMu.Lock()
	defer s.takenMu.Unlock()
	var out []string
	seen := map[string]bool{}
	for _, name := range s.taken {
		if !seen[name] {
			out = append(out, name)
			seen[name] = true
		}
	}
	sort.Strings(out)
	return out
}

// ClosestScreenshot returns the filename of the screenshot nearest to eventNum.
func (s *Screenshotter) ClosestScreenshot(eventNum int) string {
	s.takenMu.Lock()
	defer s.takenMu.Unlock()
	if name, ok := s.taken[eventNum]; ok {
		return name
	}
	bestDist := int(^uint(0) >> 1)
	bestName := ""
	for ev, name := range s.taken {
		d := eventNum - ev
		if d < 0 {
			d = -d
		}
		if d < bestDist {
			bestDist = d
			bestName = name
		}
	}
	return bestName
}

// HasScreenshot returns true if a screenshot was captured for the exact event.
func (s *Screenshotter) HasScreenshot(eventNum int) bool {
	s.takenMu.Lock()
	defer s.takenMu.Unlock()
	_, ok := s.taken[eventNum]
	return ok
}

// --- UI hash ---

func (s *Screenshotter) uiChanged(elements []device.UIElement) bool {
	hash := hashElements(elements)
	s.mu.Lock()
	defer s.mu.Unlock()
	if hash == s.prevHash {
		return false
	}
	s.prevHash = hash
	return true
}

func hashElements(elements []device.UIElement) string {
	h := sha256.New()
	for _, el := range elements {
		fmt.Fprintf(h, "%s|%s|%d,%d,%d,%d|%v|%v|%v\n",
			el.Text, el.ResourceID,
			el.X, el.Y, el.Width, el.Height,
			el.Clickable, el.InputField, el.Scrollable)
	}
	return hex.EncodeToString(h.Sum(nil))
}
