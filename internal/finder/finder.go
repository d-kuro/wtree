// Package finder provides fuzzy finder integration for the wtree application.
package finder

import (
	"fmt"
	"strings"

	"github.com/d-kuro/wtree/internal/git"
	"github.com/d-kuro/wtree/pkg/models"
	"github.com/d-kuro/wtree/pkg/utils"
	"github.com/ktr0731/go-fuzzyfinder"
)

// Finder provides fuzzy finder functionality.
type Finder struct {
	git    *git.Git
	config *models.FinderConfig
}

// New creates a new Finder instance.
func New(g *git.Git, config *models.FinderConfig) *Finder {
	return &Finder{
		git:    g,
		config: config,
	}
}

// SelectWorktree displays a fuzzy finder for worktree selection.
func (f *Finder) SelectWorktree(worktrees []models.Worktree) (*models.Worktree, error) {
	if len(worktrees) == 0 {
		return nil, fmt.Errorf("no worktrees available")
	}

	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPromptString("Select worktree> "),
	}

	if f.config.Preview {
		opts = append(opts, fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return f.generateWorktreePreview(worktrees[i], h)
		}))
	}

	idx, err := fuzzyfinder.Find(
		worktrees,
		func(i int) string {
			wt := worktrees[i]
			marker := ""
			if wt.IsMain {
				marker = "[main] "
			}
			return fmt.Sprintf("%s%s (%s)", marker, wt.Branch, wt.Path)
		},
		opts...,
	)

	if err != nil {
		return nil, err
	}

	return &worktrees[idx], nil
}

// SelectBranch displays a fuzzy finder for branch selection.
func (f *Finder) SelectBranch(branches []models.Branch) (*models.Branch, error) {
	if len(branches) == 0 {
		return nil, fmt.Errorf("no branches available")
	}

	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPromptString("Select branch> "),
	}

	if f.config.Preview {
		opts = append(opts, fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return f.generateBranchPreview(branches[i], h)
		}))
	}

	idx, err := fuzzyfinder.Find(
		branches,
		func(i int) string {
			branch := branches[i]
			marker := ""
			if branch.IsCurrent {
				marker = "* "
			} else if branch.IsRemote {
				marker = "→ "
			}
			return fmt.Sprintf("%s%s", marker, branch.Name)
		},
		opts...,
	)

	if err != nil {
		return nil, err
	}

	return &branches[idx], nil
}

// SelectMultipleWorktrees displays a fuzzy finder for multiple worktree selection.
func (f *Finder) SelectMultipleWorktrees(worktrees []models.Worktree) ([]models.Worktree, error) {
	if len(worktrees) == 0 {
		return nil, fmt.Errorf("no worktrees available")
	}

	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPromptString("Select worktrees (Tab to select multiple)> "),
	}

	if f.config.Preview {
		opts = append(opts, fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return f.generateWorktreePreview(worktrees[i], h)
		}))
	}

	indices, err := fuzzyfinder.FindMulti(
		worktrees,
		func(i int) string {
			wt := worktrees[i]
			marker := ""
			if wt.IsMain {
				marker = "[main] "
			}
			return fmt.Sprintf("%s%s (%s)", marker, wt.Branch, wt.Path)
		},
		opts...,
	)

	if err != nil {
		return nil, err
	}

	selected := make([]models.Worktree, len(indices))
	for i, idx := range indices {
		selected[i] = worktrees[idx]
	}

	return selected, nil
}

// generateWorktreePreview generates preview content for a worktree.
func (f *Finder) generateWorktreePreview(wt models.Worktree, maxLines int) string {
	preview := []string{
		fmt.Sprintf("Branch: %s", wt.Branch),
		fmt.Sprintf("Path: %s", wt.Path),
		fmt.Sprintf("Commit: %s", truncateHash(wt.CommitHash)),
		fmt.Sprintf("Created: %s", wt.CreatedAt.Format("2006-01-02 15:04")),
	}

	if wt.IsMain {
		preview = append(preview, "Type: Main worktree")
	} else {
		preview = append(preview, "Type: Additional worktree")
	}

	remainingLines := maxLines - len(preview) - 2
	if remainingLines > 0 && f.git != nil {
		preview = append(preview, "", "Recent commits:")
		commits, err := f.git.GetRecentCommits(wt.Path, remainingLines)
		if err == nil {
			for _, commit := range commits {
				preview = append(preview, fmt.Sprintf("  %s %s",
					truncateHash(commit.Hash),
					truncateMessage(commit.Message, 50),
				))
			}
		}
	}

	return strings.Join(preview, "\n")
}

// generateBranchPreview generates preview content for a branch.
func (f *Finder) generateBranchPreview(branch models.Branch, maxLines int) string {
	branchType := "Local"
	if branch.IsCurrent {
		branchType = "Current"
	} else if branch.IsRemote {
		branchType = "Remote"
	}

	preview := []string{
		fmt.Sprintf("Branch: %s", branch.Name),
		fmt.Sprintf("Type: %s", branchType),
		fmt.Sprintf("Last commit: %s", truncateMessage(branch.LastCommit.Message, 60)),
		fmt.Sprintf("Author: %s", branch.LastCommit.Author),
		fmt.Sprintf("Date: %s", branch.LastCommit.Date.Format("2006-01-02 15:04")),
		fmt.Sprintf("Hash: %s", truncateHash(branch.LastCommit.Hash)),
	}

	return strings.Join(preview[:utils.Min(len(preview), maxLines)], "\n")
}

// truncateHash truncates a commit hash to 8 characters.
func truncateHash(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}

// truncateMessage truncates a message to the specified length.
func truncateMessage(message string, maxLen int) string {
	if len(message) > maxLen {
		return message[:maxLen-3] + "..."
	}
	return message
}

