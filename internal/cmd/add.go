package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/finder"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	addBranch      bool
	addInteractive bool
	addForce       bool
)

// addCmd represents the add command.
var addCmd = &cobra.Command{
	Use:   "add [branch] [path]",
	Short: "Create a new worktree",
	Long: `Create a new worktree for the specified branch.

If no path is provided, it will be generated based on the configuration template.
Use -i flag to interactively select a branch using fuzzy finder.`,
	Example: `  # Create worktree from existing branch
  gwq add feature/new-ui

  # Create at specific path
  gwq add feature/new-ui ~/projects/myapp-feature

  # Create new branch and worktree
  gwq add -b feature/api-v2

  # Interactive branch selection
  gwq add -i`,
	RunE: runAdd,
	ValidArgsFunction: getBranchCompletions,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().BoolVarP(&addBranch, "branch", "b", false, "Create new branch")
	addCmd.Flags().BoolVarP(&addInteractive, "interactive", "i", false, "Select branch using fuzzy finder")
	addCmd.Flags().BoolVarP(&addForce, "force", "f", false, "Overwrite existing directory")
}

func runAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	g, err := git.NewFromCwd()
	if err != nil {
		return fmt.Errorf("failed to initialize git: %w", err)
	}

	printer := ui.New(&cfg.UI)
	wm := worktree.New(g, cfg)

	var branch string
	var path string

	if addInteractive {
		if len(args) > 0 {
			return fmt.Errorf("cannot specify branch name with -i flag")
		}

		branches, err := g.ListBranches(true)
		if err != nil {
			return fmt.Errorf("failed to list branches: %w", err)
		}

		f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)
		selectedBranch, err := f.SelectBranch(branches)
		if err != nil {
			return fmt.Errorf("branch selection cancelled")
		}

		branch = selectedBranch.Name
		if selectedBranch.IsRemote {
			branch = selectedBranch.Name[len("origin/"):]
			addBranch = true
		}
	} else {
		if len(args) < 1 {
			return fmt.Errorf("branch name is required")
		}
		branch = args[0]
		if len(args) > 1 {
			path = args[1]
		}
	}

	if path != "" && !addForce {
		if err := wm.ValidateWorktreePath(path); err != nil {
			return err
		}
	}

	if err := wm.Add(branch, path, addBranch); err != nil {
		return err
	}

	printer.PrintSuccess(fmt.Sprintf("Created worktree for branch '%s'", branch))
	return nil
}
