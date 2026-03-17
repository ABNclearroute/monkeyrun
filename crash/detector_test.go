package crash

import (
	"testing"
)

func TestDetectorAndroidFatal(t *testing.T) {
	d := NewDetector("android")
	sev, msg := d.Check("E AndroidRuntime: FATAL EXCEPTION: main")
	if sev != SeverityFatal {
		t.Errorf("expected SeverityFatal, got %d", sev)
	}
	if msg == "" {
		t.Error("expected non-empty message")
	}
	fatal, _ := d.Counts()
	if fatal != 1 {
		t.Errorf("expected 1 fatal, got %d", fatal)
	}
}

func TestDetectorAndroidMinor(t *testing.T) {
	d := NewDetector("android")
	sev, _ := d.Check("W ActivityManager: Force finishing activity com.demo.app/.MainActivity")
	if sev != SeverityMinor {
		t.Errorf("expected SeverityMinor, got %d", sev)
	}
	_, minor := d.Counts()
	if minor != 1 {
		t.Errorf("expected 1 minor, got %d", minor)
	}
}

func TestDetectorAndroidNone(t *testing.T) {
	d := NewDetector("android")
	sev, _ := d.Check("I ActivityManager: Start proc 12345:com.demo.app")
	if sev != SeverityNone {
		t.Errorf("expected SeverityNone, got %d", sev)
	}
}

func TestDetectorIOSFatal(t *testing.T) {
	d := NewDetector("ios")
	sev, _ := d.Check("Terminating app due to uncaught exception 'NSInvalidArgumentException'")
	if sev != SeverityFatal {
		t.Errorf("expected SeverityFatal, got %d", sev)
	}
}

func TestDetectorIOSMinor(t *testing.T) {
	d := NewDetector("ios")
	sev, _ := d.Check("Assertion failed: (condition), function foo, file bar.m, line 42")
	if sev != SeverityMinor {
		t.Errorf("expected SeverityMinor, got %d", sev)
	}
}

func TestDetectorLastLines(t *testing.T) {
	d := NewDetector("android")
	d.Check("line 1")
	d.Check("line 2")
	d.Check("line 3")
	lines := d.LastLines()
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line 1" || lines[2] != "line 3" {
		t.Error("lines out of order")
	}
}

func TestDetectorLastLinesMaxCap(t *testing.T) {
	d := NewDetector("android")
	for i := 0; i < 150; i++ {
		d.Check("log line")
	}
	lines := d.LastLines()
	if len(lines) != 100 {
		t.Errorf("expected max 100 lines, got %d", len(lines))
	}
}
