package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"monkeyrun/report"

	"github.com/spf13/cobra"
)

var reportPath string

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate or view report from a previous run",
	Long:  "Generate HTML report from report directory (default: ./report). If events.json exists, builds index.html.",
	RunE:  runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVar(&reportPath, "path", "report", "Report directory (contains events.json)")
}

func runReport(cmd *cobra.Command, args []string) error {
	dir, err := filepath.Abs(reportPath)
	if err != nil {
		dir = reportPath
	}
	eventsPath := filepath.Join(dir, "events.json")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		return fmt.Errorf("read events: %w (run monkeyrun run first)", err)
	}
	var events []report.EventEntry
	if err := json.Unmarshal(data, &events); err != nil {
		return fmt.Errorf("parse events.json: %w", err)
	}
	screenshotsDir := filepath.Join(dir, "screenshots")
	entries, _ := os.ReadDir(screenshotsDir)
	var screenshots []string
	for _, e := range entries {
		if !e.IsDir() && (filepath.Ext(e.Name()) == ".png" || filepath.Ext(e.Name()) == ".jpg") {
			screenshots = append(screenshots, e.Name())
		}
	}
	logPath := filepath.Join(dir, "logs", "crash.log")
	logLines := []string{}
	if b, err := os.ReadFile(logPath); err == nil {
		logLines = splitLines(string(b))
	}
	rep := report.Report{
		Dir:         dir,
		Events:      events,
		Screenshots: screenshots,
		LogLines:    logLines,
		TotalEvents: len(events),
	}
	if err := rep.WriteHTML(); err != nil {
		return err
	}
	fmt.Println("Report written to", filepath.Join(dir, "index.html"))
	return nil
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		out = append(out, line)
	}
	return out
}
