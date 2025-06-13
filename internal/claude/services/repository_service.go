package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
)

// RepositoryService handles Git repository operations
type RepositoryService struct {
	config *models.Config
}

// NewRepositoryService creates a new repository service
func NewRepositoryService() *RepositoryService {
	return &RepositoryService{
		config: getConfig(),
	}
}

// getConfig is a helper to get config - this will be refactored later
func getConfig() *models.Config {
	// This imports the actual config package locally to avoid circular imports
	// In a real implementation, config would be passed in via dependency injection
	return &models.Config{
		Worktree: models.WorktreeConfig{
			BaseDir: os.Getenv("HOME") + "/worktrees", // fallback default
		},
	}
}

// FindRepoRoot finds the git repository root
func (r *RepositoryService) FindRepoRoot(path string) (string, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		path = filepath.Join(wd, path)
	}

	// Find git repository root
	dir := path
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a git repository: %s", path)
		}
		dir = parent
	}
}

// ResolveRepository resolves repository path from various formats
func (r *RepositoryService) ResolveRepository(repo string) (string, error) {
	if repo == "" {
		// Use current directory
		return r.FindRepoRoot("")
	}

	// Check if it's an absolute path
	if filepath.IsAbs(repo) {
		return r.FindRepoRoot(repo)
	}

	// Check if it's a relative path
	if strings.HasPrefix(repo, "./") || strings.HasPrefix(repo, "../") {
		return r.FindRepoRoot(repo)
	}

	// Check if it's a gwq-style repository identifier (e.g., github.com/owner/repo)
	possiblePath := filepath.Join(r.config.Worktree.BaseDir, repo)
	if _, err := os.Stat(possiblePath); err == nil {
		return r.FindRepoRoot(possiblePath)
	}

	// Try as a direct path
	return r.FindRepoRoot(repo)
}

// GetCurrentBranch gets the current branch name from the repository
func (r *RepositoryService) GetCurrentBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("no current branch (detached HEAD?)")
	}

	return branch, nil
}

// FindWorktreeByName finds a worktree by its branch name using gwq worktree management
func (r *RepositoryService) FindWorktreeByName(repoRoot, name string) (string, error) {
	// Load config to use gwq worktree management
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize git and worktree manager
	g := git.New(repoRoot)
	wm := worktree.New(g, cfg)

	// Use gwq worktree logic to find worktree path
	worktreePath, err := wm.GetWorktreePath(name)
	if err != nil {
		return "", fmt.Errorf("worktree not found: %s (using gwq worktree management)", name)
	}

	// Verify the worktree actually exists
	if _, statErr := os.Stat(worktreePath); statErr != nil {
		return "", fmt.Errorf("worktree path does not exist: %s", worktreePath)
	}

	return worktreePath, nil
}

// ResolveWorktreePath resolves a worktree path from various formats using gwq worktree management
func (r *RepositoryService) ResolveWorktreePath(repoRoot, worktree string) (string, error) {
	// If it's an absolute path, verify it exists
	if filepath.IsAbs(worktree) {
		if _, err := os.Stat(worktree); err != nil {
			return "", fmt.Errorf("worktree path does not exist: %s", worktree)
		}
		return worktree, nil
	}

	// Check if it's a relative path from repository root
	if strings.HasPrefix(worktree, "./") || strings.HasPrefix(worktree, "../") {
		fullPath := filepath.Join(repoRoot, worktree)
		if _, err := os.Stat(fullPath); err != nil {
			return "", fmt.Errorf("worktree path does not exist: %s", fullPath)
		}
		return fullPath, nil
	}

	// Try to find worktree by name using gwq worktree management
	return r.FindWorktreeByName(repoRoot, worktree)
}

// ValidateRepository checks if the path is a valid git repository
func (r *RepositoryService) ValidateRepository(path string) error {
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return fmt.Errorf("not a git repository: %s", path)
	}
	return nil
}

// ValidateBranch checks if a branch exists in the repository
func (r *RepositoryService) ValidateBranch(repoRoot, branch string) error {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch does not exist: %s", branch)
	}
	return nil
}
