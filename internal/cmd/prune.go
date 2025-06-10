package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/spf13/cobra"
)

// pruneCmd represents the prune command.
var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up deleted worktree information",
	Long: `Clean up worktree information for directories that have been deleted.

This command removes administrative files from .git/worktrees for worktrees
whose working directories have been deleted from the filesystem.`,
	Example: `  # Clean up stale worktree information
  gwq prune`,
	RunE: runPrune,
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}

func runPrune(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	g, err := git.NewFromCwd()
	if err != nil {
		return fmt.Errorf("failed to initialize git: %w", err)
	}

	wm := worktree.New(g, cfg)
	printer := ui.New(&cfg.UI)

	if err := wm.Prune(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}

	printer.PrintSuccess("Pruned stale worktree information")
	return nil
}
