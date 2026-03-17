package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"monkeyrun/crash"
	"monkeyrun/device"
	"monkeyrun/engine"
	"monkeyrun/report"

	"github.com/spf13/cobra"
)

var (
	runPlatform  string
	runApp       string
	runEvents    int
	runReportDir string
	runDevice    string
	runVerbose   bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run monkey test on a device",
	Long:  "Run gesture-based chaos test. Use --platform android|ios and --app package/bundleId. Events run on already connected device.",
	RunE:  runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&runPlatform, "platform", "", "Platform: android or ios (required)")
	runCmd.Flags().StringVar(&runApp, "app", "", "App package (Android) or bundle ID (iOS) - for focus; device must have app running")
	runCmd.Flags().IntVar(&runEvents, "events", 1000, "Number of events to run")
	runCmd.Flags().StringVar(&runReportDir, "report", "report", "Report output directory")
	runCmd.Flags().StringVar(&runDevice, "device", "", "Device ID override (Android: serial; iOS: UDID)")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Verbose output")
	runCmd.MarkFlagRequired("platform")
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	platform := strings.ToLower(runPlatform)
	if platform != "android" && platform != "ios" {
		return fmt.Errorf("platform must be android or ios")
	}

	var dev device.Device
	if platform == "android" {
		deviceID := runDevice
		if deviceID == "" {
			ids, err := device.DetectAndroidDevices(ctx)
			if err != nil || len(ids) == 0 {
				return fmt.Errorf("no Android device: run 'adb devices' and connect one: %w", err)
			}
			deviceID = ids[0]
		}
		dev = device.NewAndroidDevice(deviceID)
	} else {
		udid := runDevice
		if udid == "" {
			var err error
			udid, err = device.DetectIOSBootedSimulator(ctx)
			if err != nil || udid == "" {
				return fmt.Errorf("no booted iOS simulator: run 'xcrun simctl list devices' and boot one: %w", err)
			}
		}
		dev = device.NewIOSDevice(udid, "")
	}

	reportDir, err := filepath.Abs(runReportDir)
	if err != nil {
		reportDir = runReportDir
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	screenshotsDir := filepath.Join(reportDir, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
		return err
	}

	det := crash.NewDetector(platform)
	var eventsMu sync.Mutex
	var events []report.EventEntry
	var crashes []report.CrashEntry
	var lastEventMu sync.Mutex
	var lastEventNum int

	// Start log stream for crash detection
	logCh := make(chan string, 100)
	startLogStream := dev.StartLogStream(ctx, logCh)
	if startLogStream != nil {
		if runVerbose {
			fmt.Fprintln(os.Stderr, "Log stream failed:", startLogStream)
		}
	}

	cfg := engine.RunConfig{
		Events:    runEvents,
		ReportDir: reportDir,
		Verbose:   runVerbose,
		OnEvent: func(ev engine.EventLog) {
			lastEventMu.Lock()
			lastEventNum = ev.Event
			lastEventMu.Unlock()
			eventsMu.Lock()
			events = append(events, report.EventEntry{
				Event: ev.Event, Platform: ev.Platform, Action: ev.Action,
				Element: ev.Element, Status: ev.Status, Time: ev.Time,
			})
			eventsMu.Unlock()
		},
		OnCrash: func(c engine.CrashInfo) {
			eventsMu.Lock()
			crashes = append(crashes, report.CrashEntry{
				Event: c.Event, Message: c.Message,
				Screenshot: c.Screenshot, LogSnippet: c.LogSnippet,
			})
			eventsMu.Unlock()
		},
	}

	if startLogStream == nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case line, ok := <-logCh:
					if !ok {
						return
					}
					if isCrash, msg := det.Check(line); isCrash && msg != "" {
						lastEventMu.Lock()
						evNum := lastEventNum
						lastEventMu.Unlock()
						screenshotPath := filepath.Join(screenshotsDir, fmt.Sprintf("crash_%d_%d.png", evNum, time.Now().Unix()))
						if err := dev.Screenshot(ctx, screenshotPath); err != nil && runVerbose {
							fmt.Fprintln(os.Stderr, "Screenshot failed:", err)
						}
						logSnippet := strings.Join(det.LastLines(), "\n")
						if len(logSnippet) > 4096 {
							logSnippet = logSnippet[len(logSnippet)-4096:]
						}
						cfg.OnCrash(engine.CrashInfo{
							Event:       evNum,
							Message:     msg,
							Screenshot:  screenshotPath,
							LogSnippet:  logSnippet,
						})
					}
				}
			}
		}()
	}

	monkey := engine.NewMonkey(dev, cfg)
	start := time.Now()
	n, crashCount, err := monkey.Run(ctx)
	elapsed := time.Since(start)

	// Build report
	rep := report.Report{
		Dir:          reportDir,
		Events:       events,
		Crashes:      crashes,
		StartTime:    start,
		EndTime:      time.Now(),
		TotalEvents: n,
		TotalCrashes: crashCount,
		LogLines:     det.LastLines(),
	}
	for i := range crashes {
		if crashes[i].Screenshot != "" {
			rep.Screenshots = append(rep.Screenshots, filepath.Base(crashes[i].Screenshot))
		}
	}

	_ = rep.WriteEventsJSON()
	_ = rep.WriteLogs()
	_ = rep.WriteHTML()

	fmt.Printf("Done: %d events in %v, %d crashes. Report: %s\n", n, elapsed, crashCount, reportDir)
	if err != nil {
		return err
	}
	return nil
}
