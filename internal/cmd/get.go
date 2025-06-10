package cmd

import (
	"fmt"
	"os"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/discovery"
	"github.com/d-kuro/gwq/internal/finder"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/cobra"
)

var (
	getGlobal        bool
	getNullTerminate bool
)

var getCmd = &cobra.Command{
	Use:   "get [pattern]",
	Short: "Get worktree path",
	Long: `Get worktree path matching the pattern.

If multiple worktrees match the pattern, an interactive fuzzy finder will be shown.
If no pattern is provided, all worktrees will be shown in the fuzzy finder.

The path is printed to stdout, making it suitable for shell command substitution:
  cd $(gwq get feature)`,
	Example: `  # Get path and change directory
  cd $(gwq get feature)
  
  # Use with other commands
  ls -la $(gwq get main)
  
  # Use null-terminated output with xargs
  gwq get -0 feature | xargs -0 -I {} echo "Path: {}"
  
  # Get global worktree path
  gwq get -g project:feature`,
	RunE: runGet,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getWorktreeCompletions(cmd, args, toComplete)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	
	getCmd.Flags().BoolVarP(&getGlobal, "global", "g", false, "Get from all repositories")
	getCmd.Flags().BoolVarP(&getNullTerminate, "null", "0", false, "Output null-terminated path")
}

func runGet(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if getGlobal {
		return getGlobalWorktreePath(cfg, args)
	}

	g, err := git.NewFromCwd()
	if err != nil {
		// Not in a git repo, try global
		return getGlobalWorktreePath(cfg, args)
	}

	wm := worktree.New(g, cfg)
	var path string

	if len(args) > 0 {
		// Get all matching worktrees
		matches, err := wm.GetMatchingWorktrees(args[0])
		if err != nil {
			return err
		}
		
		if len(matches) == 0 {
			return fmt.Errorf("no worktree found matching pattern: %s", args[0])
		} else if len(matches) == 1 {
			path = matches[0].Path
		} else {
			// Multiple matches - use fuzzy finder
			f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
			selected, err := f.SelectWorktree(matches)
			if err != nil {
				return fmt.Errorf("worktree selection cancelled")
			}
			path = selected.Path
		}
	} else {
		worktrees, err := wm.List()
		if err != nil {
			return err
		}

		if len(worktrees) == 0 {
			return fmt.Errorf("no worktrees found")
		}

		if len(worktrees) == 1 {
			path = worktrees[0].Path
		} else {
			f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
			selected, err := f.SelectWorktree(worktrees)
			if err != nil {
				return fmt.Errorf("worktree selection cancelled")
			}
			path = selected.Path
		}
	}

	// Output the path
	if getNullTerminate {
		_, _ = fmt.Fprintf(os.Stdout, "%s\x00", path)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, path)
	}

	return nil
}

func getGlobalWorktreePath(cfg *models.Config, args []string) error {
	entries, err := discovery.DiscoverGlobalWorktrees(cfg.Worktree.BaseDir)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		return fmt.Errorf("no worktrees found across all repositories")
	}

	var selected *discovery.GlobalWorktreeEntry

	if len(args) > 0 {
		// Pattern matching
		pattern := args[0]
		matches := discovery.FilterGlobalWorktrees(entries, pattern)
		
		if len(matches) == 0 {
			return fmt.Errorf("no worktree matches pattern: %s", pattern)
		} else if len(matches) == 1 {
			selected = matches[0]
		} else {
			// Multiple matches - use fuzzy finder
			worktrees := discovery.ConvertToWorktreeModels(matches, true)
			
			// Create a temporary git instance for finder
			g := &git.Git{}
			f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
			selectedWT, err := f.SelectWorktree(worktrees)
			if err != nil {
				return fmt.Errorf("worktree selection cancelled")
			}

			// Find the corresponding entry
			for _, entry := range matches {
				if entry.Path == selectedWT.Path {
					selected = entry
					break
				}
			}
		}
	} else {
		// No pattern - show all in fuzzy finder
		worktrees := discovery.ConvertToWorktreeModels(entries, true)
		
		g := &git.Git{}
		f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
		selectedWT, err := f.SelectWorktree(worktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}

		// Find the corresponding entry
		for _, entry := range entries {
			if entry.Path == selectedWT.Path {
				selected = entry
				break
			}
		}
	}

	if selected == nil {
		return fmt.Errorf("no worktree selected")
	}

	// Output the path
	if getNullTerminate {
		_, _ = fmt.Fprintf(os.Stdout, "%s\x00", selected.Path)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, selected.Path)
	}

	return nil
}
