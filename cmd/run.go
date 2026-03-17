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

	"github.com/spf13/cobra"
)

var (
	runPlatform            string
	runApp                 string
	runEvents              int
	runReportDir           string
	runDevice              string
	runVerbose             bool
	runDelayMin            int
	runDelayMax            int
	runHierarchyEvery      int
	runShowTouches         bool
	runStopOnCrash         bool
	runScreenshotMode      string
	runScreenshotInterval  int
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
	runCmd.Flags().StringVar(&runScreenshotMode, "screenshot-mode", "balanced", "Screenshot strategy: minimal, balanced, or full")
	runCmd.Flags().IntVar(&runScreenshotInterval, "screenshot-interval", 25, "Capture screenshot every N events (balanced/full mode)")
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
	var events []engine.EventEntry
	var crashes []engine.CrashEntry
	var lastEventMu sync.Mutex
	var lastEventNum int

	logCh := make(chan string, 100)
	logStreamErr := dev.StartLogStream(ctx, logCh)
	if logStreamErr != nil && runVerbose {
		fmt.Fprintln(os.Stderr, "Log stream failed:", logStreamErr)
	}

	ssMode := engine.ScreenshotBalanced
	switch strings.ToLower(runScreenshotMode) {
	case "minimal":
		ssMode = engine.ScreenshotMinimal
	case "full":
		ssMode = engine.ScreenshotFull
	}

	cfg := engine.RunConfig{
		Events:         runEvents,
		ReportDir:      reportDir,
		Verbose:        runVerbose,
		DelayMinMs:     runDelayMin,
		DelayMaxMs:     runDelayMax,
		HierarchyEvery: runHierarchyEvery,
		StopOnCrash:    runStopOnCrash,
		ScreenshotCfg: engine.ScreenshotConfig{
			Mode:     ssMode,
			Interval: runScreenshotInterval,
			Dir:      screenshotsDir,
		},
		OnEvent: func(ev engine.EventLog) {
			lastEventMu.Lock()
			lastEventNum = ev.Event
			lastEventMu.Unlock()
			eventsMu.Lock()
			events = append(events, engine.EventEntry{
				Event: ev.Event, Platform: ev.Platform, Action: ev.Action,
				Element: ev.Element, X: ev.X, Y: ev.Y,
				Status: ev.Status, Time: ev.Time,
				Screenshot: ev.Screenshot,
			})
			eventsMu.Unlock()
		},
		OnCrash: func(c engine.CrashInfo) {
			eventsMu.Lock()
			crashes = append(crashes, engine.CrashEntry{
				Event: c.Event, Message: c.Message,
				Screenshot: c.Screenshot, LogSnippet: c.LogSnippet,
			})
			eventsMu.Unlock()
		},
	}

	monkey := engine.NewMonkey(dev, cfg)

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

					screenshotFile := ""
					if ss := monkey.Screenshotter(); ss != nil {
						screenshotFile = ss.EnqueueCrash(evNum)
					} else {
						screenshotFile = fmt.Sprintf("monkeyrun_crash_evt%04d.png", evNum)
						spath := filepath.Join(screenshotsDir, screenshotFile)
						if err := dev.Screenshot(ctx, spath); err != nil && runVerbose {
							fmt.Fprintln(os.Stderr, "Screenshot failed:", err)
						}
					}

					logSnippet := strings.Join(det.LastLines(), "\n")
					if len(logSnippet) > 4096 {
						logSnippet = logSnippet[len(logSnippet)-4096:]
					}
					cfg.OnCrash(engine.CrashInfo{
						Event: evNum, Message: msg,
						Screenshot: screenshotFile, LogSnippet: logSnippet,
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
	start := time.Now()
	n, _, runErr := monkey.Run(ctx)
	elapsed := time.Since(start)

	// Use the crash list from the log-stream goroutine as the source of truth
	eventsMu.Lock()
	totalCrashes := len(crashes)
	eventsMu.Unlock()

	rep := engine.Report{
		Dir: reportDir, Events: events, Crashes: crashes,
		StartTime: start, EndTime: time.Now(),
		TotalEvents: n, TotalCrashes: totalCrashes,
		LogLines: det.LastLines(),
		Platform: info.Platform, DeviceName: info.Name,
	}

	if ss := monkey.Screenshotter(); ss != nil {
		rep.Screenshots = ss.TakenScreenshots()
		rep.ClosestScreenshot = make(map[int]string)
		for _, e := range events {
			if e.Screenshot {
				rep.ClosestScreenshot[e.Event] = ss.ClosestScreenshot(e.Event)
			}
		}
	} else {
		for i := range crashes {
			if crashes[i].Screenshot != "" {
				rep.Screenshots = append(rep.Screenshots, filepath.Base(crashes[i].Screenshot))
			}
		}
	}

	_ = rep.WriteEventsJSON()
	_ = rep.WriteLogs()
	_ = rep.WriteHTML()

	fmt.Printf("Done: %d events in %v, %d crashes. Report: %s\n", n, elapsed, totalCrashes, reportDir)

	// Don't propagate context.Canceled as an error — it's expected when
	// stopping on crash or user interrupt (Ctrl+C).
	if runErr == context.Canceled {
		return nil
	}
	return runErr
}
