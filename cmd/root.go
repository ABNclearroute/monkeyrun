package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "monkeyrun",
	Short: "Mobile chaos (monkey) testing for Android and iOS",
	Long:  "Lightweight, CI-friendly CLI for gesture-based UI chaos testing on already running emulators/simulators.",
}

// SetVersionInfo sets version/commit/date from main (injected by ldflags).
func SetVersionInfo(version, commit, date string) {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
