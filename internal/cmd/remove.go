package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/d-kuro/wtree/internal/config"
	"github.com/d-kuro/wtree/internal/discovery"
	"github.com/d-kuro/wtree/internal/finder"
	"github.com/d-kuro/wtree/internal/git"
	"github.com/d-kuro/wtree/internal/ui"
	"github.com/d-kuro/wtree/internal/worktree"
	"github.com/d-kuro/wtree/pkg/models"
	"github.com/spf13/cobra"
)

var (
	removeForce  bool
	removeDryRun bool
	removeGlobal bool
)

// removeCmd represents the remove command.
var removeCmd = &cobra.Command{
	Use:     "remove [pattern]",
	Aliases: []string{"rm"},
	Short:   "Delete worktree",
	Long: `Delete a worktree from the repository.

If no pattern is provided, shows a fuzzy finder to select the worktree.
The pattern can match against branch name or path.

When run inside a git repository, shows worktrees for the current repository.
When run outside a git repository, shows all worktrees from the configured base directory.
Use -g flag to always show all worktrees from the base directory.`,
	Example: `  # Select and delete using fuzzy finder
  wtree remove

  # Delete by pattern matching
  wtree remove feature/old

  # Force delete even if dirty
  wtree remove -f feature/broken

  # Show what would be deleted
  wtree remove --dry-run feature/old

  # Remove from all worktrees in base directory
  wtree remove -g myapp:feature/old`,
	RunE: runRemove,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if removeGlobal {
			return getGlobalWorktreeCompletions(cmd, args, toComplete)
		}
		return getRemoveCompletions(cmd, args, toComplete)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force delete even if dirty")
	removeCmd.Flags().BoolVarP(&removeDryRun, "dry-run", "d", false, "Show deletion targets only")
	removeCmd.Flags().BoolVarP(&removeGlobal, "global", "g", false, "Remove from any worktree in the configured base directory")
}

func runRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)

	// Check if we're in a git repository
	g, err := git.NewFromCwd()
	if err != nil || removeGlobal {
		// Not in a git repo or global flag set - use global worktrees
		return removeGlobalWorktree(cfg, printer, args)
	}

	// In a git repo - use local worktrees
	wm := worktree.New(g, cfg)

	worktrees, err := wm.List()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	nonMainWorktrees := filterNonMainWorktrees(worktrees)
	if len(nonMainWorktrees) == 0 {
		return fmt.Errorf("no removable worktrees found")
	}

	var toRemove []models.Worktree

	if len(args) > 0 {
		pattern := strings.ToLower(args[0])
		for _, wt := range nonMainWorktrees {
			if strings.Contains(strings.ToLower(wt.Branch), pattern) ||
				strings.Contains(strings.ToLower(wt.Path), pattern) {
				toRemove = append(toRemove, wt)
			}
		}
		if len(toRemove) == 0 {
			return fmt.Errorf("no worktree found matching pattern: %s", args[0])
		}
	} else {
		f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
		selected, err := f.SelectMultipleWorktrees(nonMainWorktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}
		toRemove = selected
	}

	if removeDryRun {
		fmt.Println("Would remove the following worktrees:")
		for _, wt := range toRemove {
			fmt.Printf("  %s (%s)\n", wt.Branch, wt.Path)
		}
		return nil
	}

	for _, wt := range toRemove {
		if err := wm.Remove(wt.Path, removeForce); err != nil {
			printer.PrintError(fmt.Errorf("failed to remove %s: %v", wt.Branch, err))
			continue
		}
		printer.PrintSuccess(fmt.Sprintf("Removed worktree: %s", wt.Branch))
	}

	return nil
}

func filterNonMainWorktrees(worktrees []models.Worktree) []models.Worktree {
	var filtered []models.Worktree
	for _, wt := range worktrees {
		if !wt.IsMain {
			filtered = append(filtered, wt)
		}
	}
	return filtered
}

func removeGlobalWorktree(cfg *models.Config, printer *ui.Printer, args []string) error {
	entries, err := discovery.DiscoverGlobalWorktrees(cfg.Worktree.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to discover worktrees: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no worktrees found in %s", cfg.Worktree.BaseDir)
	}

	// Filter out main worktrees
	var nonMainEntries []*discovery.GlobalWorktreeEntry
	for _, entry := range entries {
		if !entry.IsMain {
			nonMainEntries = append(nonMainEntries, entry)
		}
	}

	if len(nonMainEntries) == 0 {
		return fmt.Errorf("no removable worktrees found")
	}

	var toRemove []*discovery.GlobalWorktreeEntry

	if len(args) > 0 {
		// Pattern matching
		pattern := strings.ToLower(args[0])
		
		for _, entry := range nonMainEntries {
			branchLower := strings.ToLower(entry.Branch)
			var repoName string
			if entry.RepositoryInfo != nil {
				repoName = strings.ToLower(entry.RepositoryInfo.Repository)
			}
			
			// Match against branch name, path, repo name, or repo:branch pattern
			if strings.Contains(branchLower, pattern) || 
			   strings.Contains(strings.ToLower(entry.Path), pattern) ||
			   strings.Contains(repoName, pattern) ||
			   strings.Contains(repoName+":"+branchLower, pattern) {
				toRemove = append(toRemove, entry)
			}
		}

		if len(toRemove) == 0 {
			return fmt.Errorf("no worktree matches pattern: %s", args[0])
		}
	} else {
		// No pattern - show all in fuzzy finder
		worktrees := discovery.ConvertToWorktreeModels(nonMainEntries, true)

		// Create a temporary git instance for finder
		g, _ := git.NewFromCwd()
		if g == nil {
			g = &git.Git{}
		}
		
		f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
		selected, err := f.SelectMultipleWorktrees(worktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}

		// Map selected worktrees back to entries
		selectedPaths := make(map[string]bool)
		for _, wt := range selected {
			selectedPaths[wt.Path] = true
		}

		for _, entry := range nonMainEntries {
			if selectedPaths[entry.Path] {
				toRemove = append(toRemove, entry)
			}
		}
	}

	if removeDryRun {
		fmt.Println("Would remove the following worktrees:")
		for _, entry := range toRemove {
			repoName := "unknown"
			if entry.RepositoryInfo != nil {
				repoName = entry.RepositoryInfo.Repository
			}
			fmt.Printf("  %s:%s (%s)\n", repoName, entry.Branch, entry.Path)
		}
		return nil
	}

	// Remove each worktree by changing to its repository directory
	for _, entry := range toRemove {
		// Change to the repository directory to run git commands
		originalDir, err := os.Getwd()
		if err != nil {
			printer.PrintError(fmt.Errorf("failed to get current directory: %v", err))
			continue
		}

		// Change to repository directory (need to find git repository root from URL)
		repoPath := entry.RepositoryURL
		if entry.RepositoryInfo != nil {
			// Try to find repository in common locations
			g := git.New(entry.Path)
			if repoRootPath, err := g.GetRepositoryPath(); err == nil {
				repoPath = repoRootPath
			}
		}
		
		if err := os.Chdir(repoPath); err != nil {
			printer.PrintError(fmt.Errorf("failed to change to repository %s: %v", repoPath, err))
			continue
		}

		// Create git instance for the repository
		g := git.New(repoPath)
		wm := worktree.New(g, cfg)

		if err := wm.Remove(entry.Path, removeForce); err != nil {
			repoName := "unknown"
			if entry.RepositoryInfo != nil {
				repoName = entry.RepositoryInfo.Repository
			}
			printer.PrintError(fmt.Errorf("failed to remove %s:%s: %v", repoName, entry.Branch, err))
			_ = os.Chdir(originalDir)
			continue
		}

		repoName := "unknown"
		if entry.RepositoryInfo != nil {
			repoName = entry.RepositoryInfo.Repository
		}
		printer.PrintSuccess(fmt.Sprintf("Removed worktree: %s:%s", repoName, entry.Branch))
		
		// Change back to original directory
		_ = os.Chdir(originalDir)
	}

	return nil
}