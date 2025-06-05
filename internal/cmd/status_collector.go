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

// StatusCollector collects status information for worktrees.
type StatusCollector struct {
	includeProcess  bool
	fetchRemote     bool
	staleThreshold  time.Duration
	basedir         string
}

// NewStatusCollector creates a new status collector instance.
func NewStatusCollector(includeProcess, fetchRemote bool) *StatusCollector {
	return &StatusCollector{
		includeProcess: includeProcess,
		fetchRemote:    fetchRemote,
		staleThreshold: 14 * 24 * time.Hour, // 14 days
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
		return nil, fmt.Errorf("failed to collect git status: %w", err)
	}
	status.GitStatus = *gitStatus

	status.Status = c.determineWorktreeState(gitStatus)

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

	output, err := g.Run("status", "--porcelain=v1", "-uno")
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

	untrackedFiles, err := g.Run("ls-files", "--others", "--exclude-standard")
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
	currentBranch, err := g.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	currentBranch = strings.TrimSpace(currentBranch)
	if err != nil {
		return err
	}

	upstream, err := g.Run("rev-parse", "--abbrev-ref", currentBranch+"@{upstream}")
	if err != nil {
		return err
	}
	upstream = strings.TrimSpace(upstream)

	if upstream == "" {
		return nil
	}

	ahead, err := g.Run("rev-list", "--count", upstream+"..HEAD")
	if err == nil {
		_, _ = fmt.Sscanf(strings.TrimSpace(ahead), "%d", &status.Ahead)
	}

	behind, err := g.Run("rev-list", "--count", "HEAD.."+upstream)
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
	var latestTime time.Time

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if strings.Contains(p, "/.git/") || strings.Contains(p, "node_modules") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
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
	baseDir := filepath.Clean(c.basedir)
	cleanPath := filepath.Clean(path)

	if strings.HasPrefix(cleanPath, baseDir) {
		rel, err := filepath.Rel(baseDir, cleanPath)
		if err == nil {
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) >= 3 {
				return filepath.Join(parts[0], parts[1], parts[2])
			}
		}
	}

	return filepath.Base(path)
}

func (c *StatusCollector) collectProcesses(ctx context.Context, worktreePath string) ([]models.ProcessInfo, error) {
	// Process collection would be implemented here
	// For now, return empty slice
	return []models.ProcessInfo{}, nil
}