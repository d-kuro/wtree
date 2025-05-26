// Package cmd provides CLI commands for the wtree application.
package cmd

import (
	"fmt"
	"os"

	"github.com/d-kuro/wtree/internal/config"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "wtree",
	Short: "Git worktree manager",
	Long: `wtree is a CLI tool for efficiently managing Git worktrees.

Like how 'ghq' manages repository clones, wtree provides intuitive 
operations for creating, switching, and deleting worktrees using 
a fuzzy finder interface.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}
}