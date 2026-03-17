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
	runPlatform       string
	runApp            string
	runEvents         int
	runReportDir      string
	runDevice         string
	runVerbose        bool
	runDelayMin       int
	runDelayMax       int
	runHierarchyEvery int
	runShowTouches    bool
	runStopOnCrash    bool
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
	runCmd.Flags().StringVar(&runApp, "app", "", "App package (Android) or bundle ID (iOS)")
	runCmd.Flags().IntVar(&runEvents, "events", 1000, "Number of events to run")
	runCmd.Flags().StringVar(&runReportDir, "report", "report", "Report output directory")
	runCmd.Flags().StringVar(&runDevice, "device", "", "Device ID override (Android: serial; iOS: UDID)")
	runCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Verbose output")
	runCmd.Flags().IntVar(&runDelayMin, "delay-min", 200, "Min delay between actions in ms")
	runCmd.Flags().IntVar(&runDelayMax, "delay-max", 800, "Max delay between actions in ms")
	runCmd.Flags().IntVar(&runHierarchyEvery, "hierarchy-every", 1, "Refresh UI hierarchy every N events")
	runCmd.Flags().BoolVar(&runShowTouches, "show-touches", false, "Enable visual touch indicators while running")
	runCmd.Flags().BoolVar(&runStopOnCrash, "stop-on-crash", true, "Stop execution on fatal crash")
	runCmd.MarkFlagRequired("platform")
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	platform := strings.ToLower(runPlatform)
	dev, err := device.New(ctx, platform, device.Options{DeviceID: runDevice})
	if err != nil {
		return err
	}

	info := dev.Info()
	if runVerbose {
		fmt.Printf("Device: %s (%s) [%s]\n", info.Name, info.ID, info.Platform)
		if info.ScreenWidth > 0 {
			fmt.Printf("Screen: %dx%d\n", info.ScreenWidth, info.ScreenHeight)
		}
	}

	if runShowTouches {
		_ = dev.SetTouchVisuals(ctx, true)
		defer func() { _ = dev.SetTouchVisuals(context.Background(), false) }()
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

	logCh := make(chan string, 100)
	logStreamErr := dev.StartLogStream(ctx, logCh)
	if logStreamErr != nil && runVerbose {
		fmt.Fprintln(os.Stderr, "Log stream failed:", logStreamErr)
	}

	cfg := engine.RunConfig{
		Events:         runEvents,
		ReportDir:      reportDir,
		Verbose:        runVerbose,
		DelayMinMs:     runDelayMin,
		DelayMaxMs:     runDelayMax,
		HierarchyEvery: runHierarchyEvery,
		StopOnCrash:    runStopOnCrash,
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

	if logStreamErr == nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case line, ok := <-logCh:
					if !ok {
						return
					}
					severity, msg := det.Check(line)
					if severity == crash.SeverityNone || msg == "" {
						continue
					}
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
						Event: evNum, Message: msg,
						Screenshot: screenshotPath, LogSnippet: logSnippet,
					})
					if severity == crash.SeverityFatal && runStopOnCrash {
						fmt.Fprintf(os.Stderr, "\n*** FATAL CRASH at event %d — stopping ***\n  %s\n\n", evNum, msg)
						cancel()
						return
					}
					if severity == crash.SeverityMinor && runVerbose {
						fmt.Fprintf(os.Stderr, "  [minor crash] event %d: %s\n", evNum, msg)
					}
				}
			}
		}()
	}

	monkey := engine.NewMonkey(dev, cfg)
	start := time.Now()
	n, crashCount, runErr := monkey.Run(ctx)
	elapsed := time.Since(start)

	rep := report.Report{
		Dir: reportDir, Events: events, Crashes: crashes,
		StartTime: start, EndTime: time.Now(),
		TotalEvents: n, TotalCrashes: crashCount,
		LogLines: det.LastLines(),
		Platform: info.Platform, DeviceName: info.Name,
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
	return runErr
}
