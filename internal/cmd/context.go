package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/discovery"
	"github.com/d-kuro/gwq/internal/finder"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/cobra"
)

// CommandContext encapsulates common dependencies used across commands.
// This eliminates boilerplate code and provides consistent initialization.
type CommandContext struct {
	Config          *models.Config
	Git             *git.Git
	Printer         *ui.Printer
	WorktreeManager *worktree.Manager
	finder          *finder.Finder // Lazy-loaded
	IsGitRepo       bool
}

// NewCommandContext creates a new command context for commands that don't require git.
func NewCommandContext() (*CommandContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)

	return &CommandContext{
		Config:    cfg,
		Printer:   printer,
		IsGitRepo: false,
	}, nil
}

// NewGitCommandContext creates a new command context for commands that require git repository.
func NewGitCommandContext() (*CommandContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	g, err := git.NewFromCwd()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git: %w", err)
	}

	printer := ui.New(&cfg.UI)
	wm := worktree.New(g, cfg)

	return &CommandContext{
		Config:          cfg,
		Git:             g,
		Printer:         printer,
		WorktreeManager: wm,
		IsGitRepo:       true,
	}, nil
}

// GetFinder returns a finder instance, creating it if needed.
// This provides lazy initialization to avoid creating finders for commands that don't need them.
func (ctx *CommandContext) GetFinder() *finder.Finder {
	if ctx.finder == nil && ctx.Git != nil {
		ctx.finder = finder.NewWithUI(ctx.Git, &ctx.Config.Finder, &ctx.Config.UI)
	}
	return ctx.finder
}

// DiscoverGlobalWorktrees discovers global worktrees when -g flag is used.
func (ctx *CommandContext) DiscoverGlobalWorktrees() ([]*models.Worktree, error) {
	entries, err := discovery.DiscoverGlobalWorktrees(ctx.Config.Worktree.BaseDir)
	if err != nil {
		return nil, err
	}

	// Convert GlobalWorktreeEntry to models.Worktree
	var worktrees []*models.Worktree
	for _, entry := range entries {
		worktrees = append(worktrees, &models.Worktree{
			Path:       entry.Path,
			Branch:     entry.Branch,
			CommitHash: entry.CommitHash,
			IsMain:     entry.IsMain,
		})
	}

	return worktrees, nil
}

// WithGlobalLocalSupport handles commands that support both global and local modes.
func (ctx *CommandContext) WithGlobalLocalSupport(
	forceGlobal bool,
	localFn func(*CommandContext) error,
	globalFn func(*CommandContext) error,
) error {
	if forceGlobal || !ctx.IsGitRepo {
		return globalFn(ctx)
	}
	return localFn(ctx)
}

// ExecuteWithContext creates a command context and executes the provided function.
// This is the main wrapper function that eliminates boilerplate in command implementations.
func ExecuteWithContext(requiresGit bool, fn func(*CommandContext) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var ctx *CommandContext
		var err error

		if requiresGit {
			ctx, err = NewGitCommandContext()
		} else {
			ctx, err = NewCommandContext()
		}

		if err != nil {
			return err
		}

		return fn(ctx)
	}
}

// ExecuteWithArgs is a variant that passes command arguments to the function.
func ExecuteWithArgs(requiresGit bool, fn func(*CommandContext, *cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var ctx *CommandContext
		var err error

		if requiresGit {
			ctx, err = NewGitCommandContext()
		} else {
			ctx, err = NewCommandContext()
		}

		if err != nil {
			return err
		}

		return fn(ctx, cmd, args)
	}
}
