package cmd

import (
	"fmt"

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
	return ExecuteWithContext(true, func(ctx *CommandContext) error {
		if err := ctx.WorktreeManager.Prune(); err != nil {
			return fmt.Errorf("failed to prune worktrees: %w", err)
		}

		ctx.Printer.PrintSuccess("Pruned stale worktree information")
		return nil
	})(cmd, args)
}
