package cmd

import (
	"fmt"
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
	printPath bool
	cdGlobal  bool
)

// cdCmd represents the cd command.
var cdCmd = &cobra.Command{
	Use:   "cd [pattern]",
	Short: "Navigate to worktree directory",
	Long: `Navigate to a worktree directory.

This command outputs the worktree path, which should be used with a shell function
to actually change directories. If no pattern is provided, shows a fuzzy finder
to select the worktree.

When run inside a git repository, shows worktrees for the current repository.
When run outside a git repository, shows all worktrees from the configured base directory.
Use -g flag to always show all worktrees from the base directory.

To use this command effectively, add this function to your shell configuration:

For Bash/Zsh:
  wtree() {
    case "$1" in
      cd)
        local dir=$(command wtree cd --print-path "${@:2}")
        if [ -n "$dir" ]; then
          cd "$dir"
        fi
        ;;
      *)
        command wtree "$@"
        ;;
    esac
  }`,
	Example: `  # Select worktree using fuzzy finder
  wtree cd

  # Pattern matching selection
  wtree cd feature

  # Direct specification
  wtree cd feature/new-ui

  # Navigate to any worktree from base directory
  wtree cd -g myapp:feature`,
	RunE: runCd,
}

func init() {
	rootCmd.AddCommand(cdCmd)

	cdCmd.Flags().BoolVar(&printPath, "print-path", true, "Print only the path (for shell integration)")
	cdCmd.Flags().MarkHidden("print-path")
	cdCmd.Flags().BoolVarP(&cdGlobal, "global", "g", false, "Navigate to any worktree from the configured base directory")
}

func runCd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)

	// Check if we're in a git repository
	g, err := git.NewFromCwd()
	if err != nil || cdGlobal {
		// Not in a git repo or global flag set - use global worktrees
		return navigateGlobalWorktree(cfg, printer, args)
	}

	// In a git repo - use local worktrees
	wm := worktree.New(g, cfg)

	var path string

	if len(args) > 0 {
		path, err = wm.GetWorktreePath(args[0])
		if err != nil {
			return err
		}
	} else {
		worktrees, err := wm.List()
		if err != nil {
			return fmt.Errorf("failed to list worktrees: %w", err)
		}

		if len(worktrees) == 0 {
			return fmt.Errorf("no worktrees found")
		}

		f := finder.New(g, &cfg.Finder)
		selected, err := f.SelectWorktree(worktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}

		path = selected.Path
	}

	if printPath {
		fmt.Println(path)
	} else {
		printer.PrintSuccess(fmt.Sprintf("Navigate to: %s", path))
	}

	return nil
}

func navigateGlobalWorktree(cfg *models.Config, printer *ui.Printer, args []string) error {
	entries, err := discovery.DiscoverGlobalWorktrees(cfg.Worktree.BaseDir)
	if err != nil {
		return fmt.Errorf("failed to discover worktrees: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no worktrees found in %s", cfg.Worktree.BaseDir)
	}

	var path string

	if len(args) > 0 {
		// Pattern matching
		pattern := strings.ToLower(args[0])
		var matches []*discovery.GlobalWorktreeEntry
		
		for _, entry := range entries {
			branchLower := strings.ToLower(entry.Branch)
			var repoName string
			if entry.RepositoryInfo != nil {
				repoName = strings.ToLower(entry.RepositoryInfo.Repository)
			}
			
			// Match against branch name, repo name, or repo:branch pattern
			if strings.Contains(branchLower, pattern) || 
			   strings.Contains(repoName, pattern) ||
			   strings.Contains(repoName+":"+branchLower, pattern) {
				matches = append(matches, entry)
			}
		}

		if len(matches) == 0 {
			return fmt.Errorf("no worktree matches pattern: %s", args[0])
		} else if len(matches) == 1 {
			path = matches[0].Path
		} else {
			// Multiple matches - use fuzzy finder
			worktrees := discovery.ConvertToWorktreeModels(matches, true)

			// Create a temporary git instance for finder (won't be used for git operations)
			g, _ := git.NewFromCwd()
			if g == nil {
				g = &git.Git{}
			}
			
			f := finder.New(g, &cfg.Finder)
			selected, err := f.SelectWorktree(worktrees)
			if err != nil {
				return fmt.Errorf("worktree selection cancelled")
			}
			path = selected.Path
		}
	} else {
		// No pattern - show all in fuzzy finder
		worktrees := discovery.ConvertToWorktreeModels(entries, true)

		// Create a temporary git instance for finder
		g, _ := git.NewFromCwd()
		if g == nil {
			g = &git.Git{}
		}
		
		f := finder.New(g, &cfg.Finder)
		selected, err := f.SelectWorktree(worktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}
		path = selected.Path
	}

	if printPath {
		fmt.Println(path)
	} else {
		printer.PrintSuccess(fmt.Sprintf("Navigate to: %s", path))
	}

	return nil
}