package cmd

import (
	"fmt"
	"strings"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/discovery"
	"github.com/d-kuro/gwq/internal/finder"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
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
  gwq() {
    case "$1" in
      cd)
        # Check if -h or --help is passed
        if [[ " ${@:2} " =~ " -h " ]] || [[ " ${@:2} " =~ " --help " ]]; then
          command gwq "$@"
        else
          local dir=$(command gwq cd --print-path "${@:2}" 2>&1)
          # Check if the command succeeded
          if [ $? -eq 0 ] && [ -n "$dir" ]; then
            cd "$dir"
          else
            # If command failed, show the error message
            echo "$dir" >&2
            return 1
          fi
        fi
        ;;
      *)
        command gwq "$@"
        ;;
    esac
  }`,
	Example: `  # Select worktree using fuzzy finder
  gwq cd

  # Pattern matching selection
  gwq cd feature

  # Direct specification
  gwq cd feature/new-ui

  # Navigate to any worktree from base directory
  gwq cd -g myapp:feature`,
	RunE: runCd,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if cdGlobal {
			return getGlobalWorktreeCompletions(cmd, args, toComplete)
		}
		return getWorktreeCompletions(cmd, args, toComplete)
	},
}

func init() {
	rootCmd.AddCommand(cdCmd)

	cdCmd.Flags().BoolVar(&printPath, "print-path", true, "Print only the path (for shell integration)")
	_ = cdCmd.Flags().MarkHidden("print-path")
	cdCmd.Flags().BoolVarP(&cdGlobal, "global", "g", false, "Navigate to any worktree from the configured base directory")
}

func runCd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if we're in a git repository
	g, err := git.NewFromCwd()
	if err != nil || cdGlobal {
		// Not in a git repo or global flag set - use global worktrees
		return navigateGlobalWorktree(cfg, args)
	}

	// In a git repo - use local worktrees
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
			return fmt.Errorf("failed to list worktrees: %w", err)
		}

		if len(worktrees) == 0 {
			return fmt.Errorf("no worktrees found")
		}

		f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
		selected, err := f.SelectWorktree(worktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}

		path = selected.Path
	}

	if printPath {
		fmt.Println(path)
	} else {
		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}
		return showShellSetupInstructions(pattern)
	}

	return nil
}

func navigateGlobalWorktree(cfg *models.Config, args []string) error {
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
			
			f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
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
		
		f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
		selected, err := f.SelectWorktree(worktrees)
		if err != nil {
			return fmt.Errorf("worktree selection cancelled")
		}
		path = selected.Path
	}

	if printPath {
		fmt.Println(path)
	} else {
		pattern := ""
		if len(args) > 0 {
			pattern = args[0]
		}
		return showShellSetupInstructions(pattern)
	}

	return nil
}

func showShellSetupInstructions(pattern string) error {
	fmt.Println("Error: gwq cd requires a shell function to change directories.")
	fmt.Println("\nTo use this command, add the following to your shell configuration:")
	fmt.Println("\nFor Bash/Zsh (~/.bashrc or ~/.zshrc):")
	fmt.Println(`
gwq() {
  case "$1" in
    cd)
      # Check if -h or --help is passed
      if [[ " ${@:2} " =~ " -h " ]] || [[ " ${@:2} " =~ " --help " ]]; then
        command gwq "$@"
      else
        local dir=$(command gwq cd --print-path "${@:2}" 2>&1)
        # Check if the command succeeded
        if [ $? -eq 0 ] && [ -n "$dir" ]; then
          cd "$dir"
        else
          # If failed or cancelled, show the error message
          echo "$dir" >&2
          return 1
        fi
      fi
      ;;
    *)
      command gwq "$@"
      ;;
  esac
}`)
	fmt.Println("\nFor Fish (~/.config/fish/config.fish):")
	fmt.Println(`
function gwq
  if test "$argv[1]" = "cd"
    # Check if -h or --help is passed
    if contains -- -h $argv[2..-1]; or contains -- --help $argv[2..-1]
      command gwq $argv
    else
      set -l dir (command gwq cd --print-path $argv[2..-1] 2>&1)
      if test $status -eq 0; and test -n "$dir"
        cd $dir
      else
        echo "$dir" >&2
        return 1
      end
    end
  else
    command gwq $argv
  end
end`)
	fmt.Println("\nAlternatively, use one of these commands:")
	if pattern != "" {
		fmt.Printf("  cd $(gwq get %s)          # Simple path retrieval\n", pattern)
		fmt.Printf("  gwq exec %s -- command    # Execute command in worktree\n", pattern)
	} else {
		fmt.Println("  cd $(gwq get)             # Simple path retrieval")
		fmt.Println("  gwq exec -- command       # Execute command in worktree")
	}
	return fmt.Errorf("shell function not configured")
}