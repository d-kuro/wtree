package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/discovery"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/cobra"
)

var (
	listVerbose bool
	listJSON    bool
	listGlobal  bool
)

// listCmd represents the list command.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Display worktree list",
	Long: `Display a list of worktrees.

When run inside a git repository, shows worktrees for the current repository.
When run outside a git repository, shows all worktrees in the configured base directory.
Use -g flag to always show all worktrees from the base directory.
Use -v flag for detailed information including commit hashes and creation times.
Use --json flag to output in JSON format for scripting.`,
	Example: `  # Simple list
  gwq list

  # Detailed information
  gwq list -v

  # JSON format for scripting
  gwq list --json

  # Show all worktrees from base directory (from anywhere)
  gwq list -g`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listVerbose, "verbose", "v", false, "Show detailed information")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVarP(&listGlobal, "global", "g", false, "Show all worktrees from the configured base directory")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)

	// Check if we're in a git repository
	g, err := git.NewFromCwd()
	if err != nil || listGlobal {
		// Not in a git repo or global flag set - show all worktrees from base directory
		return showGlobalWorktrees(cfg, printer)
	}

	// In a git repo - show local worktrees
	wm := worktree.New(g, cfg)
	worktrees, err := wm.List()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if listJSON {
		return printer.PrintWorktreesJSON(worktrees)
	}

	printer.PrintWorktrees(worktrees, listVerbose)
	return nil
}

func showGlobalWorktrees(cfg *models.Config, printer *ui.Printer) error {
	entries, err := discovery.DiscoverGlobalWorktrees(cfg.Worktree.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to discover worktrees: %w", err)
	}

	if len(entries) == 0 {
		printer.PrintInfo("No worktrees found in " + cfg.Worktree.BaseDir)
		return nil
	}

	// Convert to worktree models with repository names for clarity
	worktrees := discovery.ConvertToWorktreeModels(entries, !listVerbose)

	if listJSON {
		return printer.PrintWorktreesJSON(worktrees)
	}

	printer.PrintWorktrees(worktrees, listVerbose)
	return nil
}
