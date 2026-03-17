package crash

import (
	"strings"
	"sync"
)

// Detector watches log lines and signals on crash keywords.
type Detector struct {
	platform string
	keywords []string
	mu       sync.Mutex
	lastLines []string
	maxLines  int
}

// NewDetector creates a crash detector for the given platform.
func NewDetector(platform string) *Detector {
	d := &Detector{platform: platform, maxLines: 100}
	if platform == "android" {
		d.keywords = []string{"FATAL EXCEPTION", "AndroidRuntime", "ANR in", "SIGSEGV", "Fatal signal", "Force finishing"}
	} else {
		d.keywords = []string{"Assertion failed", "fatal error", "SIGABRT", "SIGSEGV", "crash", "Terminating app", "Exception Type"}
	}
	return d
}

// Check returns true if the line indicates a crash, and a short message.
func (d *Detector) Check(line string) (isCrash bool, message string) {
	d.mu.Lock()
	d.lastLines = append(d.lastLines, line)
	if len(d.lastLines) > d.maxLines {
		d.lastLines = d.lastLines[len(d.lastLines)-d.maxLines:]
	}
	d.mu.Unlock()
	upper := strings.ToUpper(line)
	for _, k := range d.keywords {
		if strings.Contains(upper, strings.ToUpper(k)) {
			return true, line
		}
	}
	return false, ""
}

// LastLines returns recent log lines for inclusion in crash report.
func (d *Detector) LastLines() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.lastLines))
	copy(out, d.lastLines)
	return out
}
