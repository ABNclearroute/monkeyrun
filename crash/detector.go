package crash

import (
	"strings"
	"sync"
)

// Severity indicates how critical a crash is.
type Severity int

const (
	SeverityNone  Severity = 0
	SeverityMinor Severity = 1
	SeverityFatal Severity = 2
)

// Detector watches log lines and signals on crash keywords.
type Detector struct {
	platform     string
	fatalKeys    []string
	minorKeys    []string
	mu           sync.Mutex
	lastLines    []string
	maxLines     int
	fatalCount   int
	minorCount   int
}

// NewDetector creates a crash detector for the given platform.
func NewDetector(platform string) *Detector {
	d := &Detector{platform: platform, maxLines: 100}
	if platform == "android" {
		d.fatalKeys = []string{"FATAL EXCEPTION", "SIGSEGV", "Fatal signal", "ANR in"}
		d.minorKeys = []string{"AndroidRuntime", "Force finishing", "has died"}
	} else {
		d.fatalKeys = []string{"SIGABRT", "SIGSEGV", "Terminating app", "Exception Type", "fatal error"}
		d.minorKeys = []string{"Assertion failed", "crash"}
	}
	return d
}

// Check returns severity and a short message if the line indicates a crash.
func (d *Detector) Check(line string) (severity Severity, message string) {
	d.mu.Lock()
	d.lastLines = append(d.lastLines, line)
	if len(d.lastLines) > d.maxLines {
		d.lastLines = d.lastLines[len(d.lastLines)-d.maxLines:]
	}
	d.mu.Unlock()

	upper := strings.ToUpper(line)
	for _, k := range d.fatalKeys {
		if strings.Contains(upper, strings.ToUpper(k)) {
			d.mu.Lock()
			d.fatalCount++
			d.mu.Unlock()
			return SeverityFatal, line
		}
	}
	for _, k := range d.minorKeys {
		if strings.Contains(upper, strings.ToUpper(k)) {
			d.mu.Lock()
			d.minorCount++
			d.mu.Unlock()
			return SeverityMinor, line
		}
	}
	return SeverityNone, ""
}

// Counts returns (fatal, minor) crash counts.
func (d *Detector) Counts() (fatal, minor int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fatalCount, d.minorCount
}

// LastLines returns recent log lines for inclusion in crash report.
func (d *Detector) LastLines() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.lastLines))
	copy(out, d.lastLines)
	return out
}
