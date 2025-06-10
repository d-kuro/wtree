package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/pkg/models"
)

// StatusCollectorOptions contains optional parameters for StatusCollector.
type StatusCollectorOptions struct {
	IncludeProcess bool
	FetchRemote    bool
	StaleThreshold time.Duration
	BaseDir        string
}

// StatusCollector collects status information for worktrees.
type StatusCollector struct {
	includeProcess bool
	fetchRemote    bool
	staleThreshold time.Duration
	basedir        string
}

// NewStatusCollector creates a new status collector instance.
func NewStatusCollector(includeProcess, fetchRemote bool) *StatusCollector {
	return &StatusCollector{
		includeProcess: includeProcess,
		fetchRemote:    fetchRemote,
		staleThreshold: 14 * 24 * time.Hour, // 14 days
	}
}

// NewStatusCollectorWithOptions creates a new status collector with custom options.
func NewStatusCollectorWithOptions(opts StatusCollectorOptions) *StatusCollector {
	// Default stale threshold to 14 days if not specified
	if opts.StaleThreshold == 0 {
		opts.StaleThreshold = 14 * 24 * time.Hour
	}

	return &StatusCollector{
		includeProcess: opts.IncludeProcess,
		fetchRemote:    opts.FetchRemote,
		staleThreshold: opts.StaleThreshold,
		basedir:        opts.BaseDir,
	}
}

// CollectAll collects status for all provided worktrees in parallel.
func (c *StatusCollector) CollectAll(ctx context.Context, worktrees []*models.Worktree) ([]*models.WorktreeStatus, error) {
	statuses := make([]*models.WorktreeStatus, len(worktrees))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	currentPath, _ := os.Getwd()

	for i, wt := range worktrees {
		wg.Add(1)
		go func(idx int, worktree *models.Worktree) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			status, err := c.collectOne(ctx, worktree)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			if strings.HasPrefix(currentPath, worktree.Path) {
				status.IsCurrent = true
			}

			statuses[idx] = status
		}(i, wt)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	var validStatuses []*models.WorktreeStatus
	for _, s := range statuses {
		if s != nil {
			validStatuses = append(validStatuses, s)
		}
	}

	return validStatuses, nil
}

func (c *StatusCollector) collectOne(ctx context.Context, worktree *models.Worktree) (*models.WorktreeStatus, error) {
	status := &models.WorktreeStatus{
		Path:       worktree.Path,
		Branch:     worktree.Branch,
		Repository: c.extractRepository(worktree.Path),
		Status:     models.WorktreeStatusClean,
	}

	g := git.New(worktree.Path)

	gitStatus, err := c.collectGitStatus(ctx, g)
	if err != nil {
		// Log error but continue with minimal status
		// fmt.Fprintf(os.Stderr, "Warning: Failed to collect git status for %s: %v\n", worktree.Path, err)
		status.GitStatus = models.GitStatus{}
		status.Status = models.WorktreeStatusUnknown
	} else {
		status.GitStatus = *gitStatus
		status.Status = c.determineWorktreeState(gitStatus)
	}

	lastActivity, err := c.getLastActivity(worktree.Path)
	if err == nil {
		status.LastActivity = lastActivity
		if time.Since(lastActivity) > c.staleThreshold {
			status.Status = models.WorktreeStatusStale
		}
	}

	if c.includeProcess {
		processes, err := c.collectProcesses(ctx, worktree.Path)
		if err == nil {
			status.ActiveProcess = processes
		}
	}

	return status, nil
}

func (c *StatusCollector) collectGitStatus(ctx context.Context, g *git.Git) (*models.GitStatus, error) {
	status := &models.GitStatus{}

	// Create a timeout context for git operations
	gitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := g.RunWithContext(gitCtx, "status", "--porcelain=v1", "-uno")
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(output, "\n") {
		if len(line) < 3 {
			continue
		}

		index := line[0]
		worktree := line[1]

		if index != ' ' && index != '?' {
			status.Staged++
		}

		switch worktree {
		case 'M':
			status.Modified++
		case 'A':
			status.Added++
		case 'D':
			status.Deleted++
		case '?':
			status.Untracked++
		case 'U':
			status.Conflicts++
		}
	}

	// Reset timeout context for the next operation
	gitCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()

	untrackedFiles, err := g.RunWithContext(gitCtx2, "ls-files", "--others", "--exclude-standard")
	if err == nil && untrackedFiles != "" {
		status.Untracked = len(strings.Split(strings.TrimSpace(untrackedFiles), "\n"))
	}

	if c.fetchRemote {
		// Errors are ignored as remote might not be available
		_ = c.fetchRemoteStatus(ctx, g, status)
	}

	return status, nil
}

func (c *StatusCollector) fetchRemoteStatus(ctx context.Context, g *git.Git, status *models.GitStatus) error {
	// Create a timeout context for remote operations
	gitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	currentBranch, err := g.RunWithContext(gitCtx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	currentBranch = strings.TrimSpace(currentBranch)

	gitCtx2, cancel2 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel2()

	upstream, err := g.RunWithContext(gitCtx2, "rev-parse", "--abbrev-ref", currentBranch+"@{upstream}")
	if err != nil {
		return err
	}
	upstream = strings.TrimSpace(upstream)

	if upstream == "" {
		return nil
	}

	gitCtx3, cancel3 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel3()

	ahead, err := g.RunWithContext(gitCtx3, "rev-list", "--count", upstream+"..HEAD")
	if err == nil {
		_, _ = fmt.Sscanf(strings.TrimSpace(ahead), "%d", &status.Ahead)
	}

	gitCtx4, cancel4 := context.WithTimeout(ctx, 5*time.Second)
	defer cancel4()

	behind, err := g.RunWithContext(gitCtx4, "rev-list", "--count", "HEAD.."+upstream)
	if err == nil {
		_, _ = fmt.Sscanf(strings.TrimSpace(behind), "%d", &status.Behind)
	}

	return nil
}

func (c *StatusCollector) determineWorktreeState(status *models.GitStatus) models.WorktreeState {
	if status.Conflicts > 0 {
		return models.WorktreeStatusConflict
	}
	if status.Staged > 0 {
		return models.WorktreeStatusStaged
	}
	if status.Modified > 0 || status.Added > 0 || status.Deleted > 0 || status.Untracked > 0 {
		return models.WorktreeStatusModified
	}
	return models.WorktreeStatusClean
}

func (c *StatusCollector) getLastActivity(path string) (time.Time, error) {
	// Use git ls-files to get tracked files efficiently
	// This approach respects .gitignore patterns automatically and is much faster
	// than walking the entire directory tree
	g := git.New(path)

	// Get list of tracked files
	// Using -z for null-terminated output to handle filenames with spaces
	output, err := g.Run("ls-files", "-z")
	if err != nil {
		// Fallback to directory walk if git command fails
		return c.getLastActivityFallback(path)
	}

	var latestTime time.Time
	files := strings.Split(strings.TrimRight(output, "\x00"), "\x00")

	for _, file := range files {
		if file == "" {
			continue
		}

		fullPath := filepath.Join(path, file)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue // Skip files we can't stat
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}
	}

	// Also check untracked files that are not ignored
	untrackedOutput, err := g.Run("ls-files", "-z", "--others", "--exclude-standard")
	if err == nil {
		untrackedFiles := strings.Split(strings.TrimRight(untrackedOutput, "\x00"), "\x00")
		for _, file := range untrackedFiles {
			if file == "" {
				continue
			}

			fullPath := filepath.Join(path, file)
			info, err := os.Stat(fullPath)
			if err != nil || info.IsDir() {
				continue
			}

			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
			}
		}
	}

	if latestTime.IsZero() {
		// If no files found, use the directory's own modification time
		info, err := os.Stat(path)
		if err == nil {
			latestTime = info.ModTime()
		}
	}

	return latestTime, nil
}

// getLastActivityFallback is the fallback method when git commands fail
func (c *StatusCollector) getLastActivityFallback(path string) (time.Time, error) {
	var latestTime time.Time

	// Common large directories to skip
	skipDirs := map[string]bool{
		".git":          true,
		"node_modules":  true,
		"vendor":        true,
		".next":         true,
		"dist":          true,
		"build":         true,
		"target":        true,
		".cache":        true,
		"coverage":      true,
		"__pycache__":   true,
		".pytest_cache": true,
		".venv":         true,
		"venv":          true,
		".idea":         true,
		".vscode":       true,
	}

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue even if we can't access a file
		}

		// Skip directories
		if info.IsDir() {
			dirName := filepath.Base(p)
			if skipDirs[dirName] {
				return filepath.SkipDir
			}
			// Also skip hidden directories (except the root)
			if dirName != "." && strings.HasPrefix(dirName, ".") && p != path {
				return filepath.SkipDir
			}
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
		}

		return nil
	})

	if err != nil {
		return time.Time{}, err
	}

	return latestTime, nil
}

func (c *StatusCollector) extractRepository(path string) string {
	// Return basename if basedir is not set
	if c.basedir == "" {
		return filepath.Base(path)
	}

	baseDir := filepath.Clean(c.basedir)
	cleanPath := filepath.Clean(path)

	// Check if the path is under the base directory
	if !strings.HasPrefix(cleanPath, baseDir) {
		// Path is not under base directory, return basename
		return filepath.Base(path)
	}

	rel, err := filepath.Rel(baseDir, cleanPath)
	if err != nil {
		// Failed to get relative path, fallback to basename
		return filepath.Base(path)
	}

	// Split the relative path into components
	parts := strings.Split(rel, string(filepath.Separator))

	// Expected structure: host/owner/repository/branch
	// Return the first 3 components if available
	if len(parts) >= 3 {
		return filepath.Join(parts[0], parts[1], parts[2])
	}

	// If we don't have enough parts, return what we have or the basename
	if len(parts) > 0 {
		return rel
	}

	return filepath.Base(path)
}

func (c *StatusCollector) collectProcesses(ctx context.Context, worktreePath string) ([]models.ProcessInfo, error) {
	// TODO: Implement process detection for AI agents and other tools
	// This would involve:
	// 1. Scanning for processes with working directory in worktreePath
	// 2. Identifying known AI agent processes (claude, copilot, cursor, etc.)
	// 3. Detecting common development tools (npm, cargo, python, etc.)
	// 4. Platform-specific process enumeration (ps on Unix, tasklist on Windows)
	//
	// For now, this is a stub that returns an empty slice.
	// The actual implementation is deferred as it requires platform-specific code
	// and careful consideration of performance implications.
	return []models.ProcessInfo{}, nil
}
