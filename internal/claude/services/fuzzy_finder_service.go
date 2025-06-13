package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/ktr0731/go-fuzzyfinder"
)

// FuzzyFinderService provides unified interactive selection functionality
type FuzzyFinderService struct{}

// NewFuzzyFinderService creates a new fuzzy finder service
func NewFuzzyFinderService() *FuzzyFinderService {
	return &FuzzyFinderService{}
}

// SelectExecution allows interactive selection of executions
func (f *FuzzyFinderService) SelectExecution(executions []claude.ExecutionMetadata) (*claude.ExecutionMetadata, error) {
	if len(executions) == 0 {
		return nil, nil
	}

	if len(executions) == 1 {
		return &executions[0], nil
	}

	// Use go-fuzzyfinder
	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPromptString("Select Execution> "),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			exec := executions[i]
			return fmt.Sprintf("Execution: %s\nStatus: %s\nStarted: %s\nPrompt: %s",
				exec.ExecutionID,
				exec.Status,
				exec.StartTime.Format("2006-01-02 15:04:05"),
				exec.Prompt)
		}),
	}

	idx, err := fuzzyfinder.Find(
		executions,
		func(i int) string {
			exec := executions[i]
			status := string(exec.Status)
			relativeTime := f.formatRelativeTime(exec.StartTime)

			// Get branch info from working directory or use "no-branch"
			branch := f.extractBranchFromPath(exec.WorkingDirectory)

			// Format: [status] exec-id (~/path/to/repo on branch) - time ago
			return fmt.Sprintf("[%s] %s (%s on %s) - %s",
				status, exec.ExecutionID, exec.WorkingDirectory, branch, relativeTime)
		},
		opts...,
	)

	if err != nil {
		return nil, err
	}

	return &executions[idx], nil
}

// SelectTask allows interactive selection of tasks
func (f *FuzzyFinderService) SelectTask(tasks []*claude.Task) (*claude.Task, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	if len(tasks) == 1 {
		return tasks[0], nil
	}

	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPromptString("Select Task> "),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			task := tasks[i]
			return f.formatTaskPreview(task)
		}),
	}

	idx, err := fuzzyfinder.Find(
		tasks,
		func(i int) string {
			task := tasks[i]
			status := string(task.Status)
			statusIcon := f.getStatusIcon(task.Status)

			worktree := task.Worktree

			relativeTime := f.formatRelativeTime(task.CreatedAt)

			return fmt.Sprintf("%s [%s] %s (%s) - %s - %s",
				statusIcon, status, task.ID, worktree, task.Name, relativeTime)
		},
		opts...,
	)

	if err != nil {
		return nil, err
	}

	return tasks[idx], nil
}

// formatTaskPreview formats a task for preview display
func (f *FuzzyFinderService) formatTaskPreview(task *claude.Task) string {
	var preview strings.Builder

	preview.WriteString(fmt.Sprintf("Task: %s\n", task.Name))
	preview.WriteString(fmt.Sprintf("ID: %s\n", task.ID))
	preview.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
	preview.WriteString(fmt.Sprintf("Priority: %d\n", task.Priority))
	preview.WriteString(fmt.Sprintf("Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05")))

	if task.Worktree != "" {
		preview.WriteString(fmt.Sprintf("Worktree: %s\n", task.Worktree))
	}

	if task.Prompt != "" {
		preview.WriteString(fmt.Sprintf("\nPrompt: %s\n", f.truncateString(task.Prompt, 200)))
	}

	if len(task.DependsOn) > 0 {
		preview.WriteString(fmt.Sprintf("\nDependencies: %s\n", strings.Join(task.DependsOn, ", ")))
	}

	return preview.String()
}

// extractBranchFromPath extracts branch information from a working directory path
func (f *FuzzyFinderService) extractBranchFromPath(workingDir string) string {
	branch := "no-branch"
	if strings.Contains(workingDir, "/.worktrees/") {
		// Extract branch from worktree path
		parts := strings.Split(workingDir, "/.worktrees/")
		if len(parts) > 1 {
			branchParts := strings.Split(parts[1], "-")
			if len(branchParts) > 0 {
				branch = strings.Join(branchParts[:len(branchParts)-1], "-")
			}
		}
	} else if workingDir != "" {
		// Assume we're on the default branch if not in a worktree
		branch = "main"
	}
	return branch
}

// formatRelativeTime formats a time as a relative duration
func (f *FuzzyFinderService) formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}

// getStatusIcon returns an icon for the task status
func (f *FuzzyFinderService) getStatusIcon(status claude.Status) string {
	switch status {
	case claude.StatusPending:
		return "○"
	case claude.StatusWaiting:
		return "⏳"
	case claude.StatusRunning:
		return "●"
	case claude.StatusCompleted:
		return "✓"
	case claude.StatusFailed:
		return "✗"
	case claude.StatusSkipped:
		return "⤵"
	case claude.StatusCancelled:
		return "✕"
	default:
		return "?"
	}
}

// truncateString truncates a string to a maximum length
func (f *FuzzyFinderService) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
