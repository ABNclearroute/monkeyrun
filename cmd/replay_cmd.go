package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"monkeyrun/device"
	"monkeyrun/engine"
	"monkeyrun/report"

	"github.com/spf13/cobra"
)

var (
	replayPath     string
	replayPlatform string
	replayLimit    int
)

var replayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Replay events from a previous run's events.json",
	Long:  "Load events.json from report dir and re-execute the same action sequence on a connected device.",
	RunE:  runReplay,
}

func init() {
	rootCmd.AddCommand(replayCmd)
	replayCmd.Flags().StringVar(&replayPath, "report", "report", "Report directory containing events.json")
	replayCmd.Flags().StringVar(&replayPlatform, "platform", "android", "Platform: android or ios")
	replayCmd.Flags().IntVar(&replayLimit, "events", 0, "Max events to replay (0 = all)")
}

func runReplay(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	platform := strings.ToLower(replayPlatform)
	dev, err := device.New(ctx, platform, device.Options{})
	if err != nil {
		return err
	}

	path := filepath.Join(replayPath, "events.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read events: %w", err)
	}
	var entries []report.EventEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse events.json: %w", err)
	}
	events := make([]engine.EventLog, 0, len(entries))
	for _, e := range entries {
		events = append(events, engine.EventLog{
			Event: e.Event, Platform: e.Platform, Action: e.Action,
			Element: e.Element, Status: e.Status, Time: e.Time,
		})
	}
	if replayLimit > 0 && len(events) > replayLimit {
		events = events[:replayLimit]
	}

	info := dev.Info()
	fmt.Printf("Replaying %d events on %s (%s)\n", len(events), info.Name, info.ID)
	return engine.Replay(ctx, dev, events, func(ev engine.EventLog) {
		if ev.Event%100 == 0 {
			fmt.Printf("  event %d\n", ev.Event)
		}
	})
}
