// Package git provides Git operations for the wtree application.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/d-kuro/wtree/pkg/models"
)

// Git provides Git command operations.
type Git struct {
	workDir string
}

// New creates a new Git instance.
func New(workDir string) *Git {
	return &Git{
		workDir: workDir,
	}
}

// NewFromCwd creates a new Git instance using the current working directory.
func NewFromCwd() (*Git, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	return New(cwd), nil
}

// ListWorktrees returns a list of all worktrees in the repository.
func (g *Git) ListWorktrees() ([]models.Worktree, error) {
	output, err := g.run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []models.Worktree
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "worktree ") {
			path := strings.TrimPrefix(lines[i], "worktree ")
			
			var branch, commitHash string
			isMain := false

			for j := i + 1; j < len(lines) && !strings.HasPrefix(lines[j], "worktree "); j++ {
				if strings.HasPrefix(lines[j], "branch ") {
					branch = strings.TrimPrefix(lines[j], "branch ")
					// Remove refs/heads/ prefix if present
					if strings.HasPrefix(branch, "refs/heads/") {
						branch = strings.TrimPrefix(branch, "refs/heads/")
					}
				} else if strings.HasPrefix(lines[j], "HEAD ") {
					commitHash = strings.TrimPrefix(lines[j], "HEAD ")
				} else if strings.HasPrefix(lines[j], "bare") {
					continue
				}
				i = j
			}

			if branch == "" {
				branch = g.getCurrentBranch(path)
			}

			info, err := os.Stat(path)
			var createdAt time.Time
			if err == nil {
				createdAt = info.ModTime()
			}

			worktrees = append(worktrees, models.Worktree{
				Path:       path,
				Branch:     branch,
				CommitHash: commitHash,
				IsMain:     isMain,
				CreatedAt:  createdAt,
			})
		}
	}

	if len(worktrees) > 0 {
		mainDir, err := g.getMainWorktreeDir()
		if err == nil {
			for i := range worktrees {
				if worktrees[i].Path == mainDir {
					worktrees[i].IsMain = true
					break
				}
			}
		}
	}

	return worktrees, nil
}

// AddWorktree creates a new worktree.
func (g *Git) AddWorktree(path, branch string, createBranch bool) error {
	args := []string{"worktree", "add"}
	
	if createBranch {
		args = append(args, "-b", branch, path)
	} else {
		args = append(args, path, branch)
	}

	if _, err := g.run(args...); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	return nil
}

// RemoveWorktree removes a worktree.
func (g *Git) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	if _, err := g.run(args...); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	return nil
}

// PruneWorktrees removes worktree information for deleted directories.
func (g *Git) PruneWorktrees() error {
	if _, err := g.run("worktree", "prune"); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}
	return nil
}

// ListBranches returns a list of all branches.
func (g *Git) ListBranches(includeRemote bool) ([]models.Branch, error) {
	args := []string{"branch", "-v", "--format=%(refname:short)|%(HEAD)|%(committerdate:iso)|%(objectname)|%(subject)|%(authorname)"}
	if includeRemote {
		args = append(args, "-a")
	}

	output, err := g.run(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []models.Branch
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}

		name := parts[0]
		isCurrent := parts[1] == "*"
		dateStr := parts[2]
		hash := parts[3]
		message := parts[4]
		author := parts[5]

		isRemote := strings.HasPrefix(name, "remotes/")
		if isRemote {
			name = strings.TrimPrefix(name, "remotes/")
		}

		date, _ := time.Parse("2006-01-02 15:04:05 -0700", dateStr)

		branches = append(branches, models.Branch{
			Name:      name,
			IsCurrent: isCurrent,
			IsRemote:  isRemote,
			LastCommit: models.CommitInfo{
				Hash:    hash,
				Message: message,
				Author:  author,
				Date:    date,
			},
		})
	}

	return branches, nil
}

// GetRepositoryName returns the name of the repository.
func (g *Git) GetRepositoryName() (string, error) {
	rootDir, err := g.getRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Base(rootDir), nil
}

// GetRepositoryPath returns the root path of the git repository.
func (g *Git) GetRepositoryPath() (string, error) {
	return g.getRootDir()
}

// GetRecentCommits returns recent commits for a specific path.
func (g *Git) GetRecentCommits(path string, limit int) ([]models.CommitInfo, error) {
	oldWorkDir := g.workDir
	g.workDir = path
	defer func() { g.workDir = oldWorkDir }()

	args := []string{"log", fmt.Sprintf("-%d", limit), "--pretty=format:%H|%s|%an|%ai"}
	output, err := g.run(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent commits: %w", err)
	}

	var commits []models.CommitInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		date, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])

		commits = append(commits, models.CommitInfo{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    date,
		})
	}

	return commits, nil
}

// getCurrentBranch returns the current branch name for a specific worktree.
func (g *Git) getCurrentBranch(worktreePath string) string {
	oldWorkDir := g.workDir
	g.workDir = worktreePath
	defer func() { g.workDir = oldWorkDir }()

	output, err := g.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

// getMainWorktreeDir returns the main worktree directory.
func (g *Git) getMainWorktreeDir() (string, error) {
	return g.getRootDir()
}

// getRootDir returns the repository root directory.
func (g *Git) getRootDir() (string, error) {
	output, err := g.run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// GetRepositoryURL returns the remote origin URL of the repository.
func (g *Git) GetRepositoryURL() (string, error) {
	output, err := g.run("remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("failed to get repository URL: %w", err)
	}
	return strings.TrimSpace(output), nil
}


// RunCommand executes a git command and returns the output (public method).
func (g *Git) RunCommand(args ...string) (string, error) {
	return g.run(args...)
}

// run executes a git command.
func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if g.workDir != "" {
		cmd.Dir = g.workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), stderr.String())
	}

	return stdout.String(), nil
}