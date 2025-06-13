package presenters

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/d-kuro/gwq/internal/claude"
)

// TaskPresenter handles task display formatting
type TaskPresenter struct{}

// NewTaskPresenter creates a new task presenter
func NewTaskPresenter() *TaskPresenter {
	return &TaskPresenter{}
}

// OutputTasksTable outputs tasks in table format
func (p *TaskPresenter) OutputTasksTable(tasks []*claude.Task, verbose bool) error {
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	// Create table with lipgloss
	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers("TASK", "WORKTREE", "STATUS", "PRIORITY", "DEPS", "DURATION")

	// Add rows to table
	for _, task := range tasks {
		status := string(task.Status)
		statusIcon := p.getStatusIcon(task.Status)

		worktree := task.Worktree

		deps := strconv.Itoa(len(task.DependsOn))
		if len(task.DependsOn) == 0 {
			deps = "-"
		}

		duration := "-"
		if task.Result != nil {
			duration = p.formatDuration(task.Result.Duration)
		} else if task.StartedAt != nil {
			duration = p.formatDuration(time.Since(*task.StartedAt))
		}

		t.Row(
			statusIcon+" "+task.ID,
			worktree,
			status,
			strconv.Itoa(int(task.Priority)),
			deps,
			duration,
		)

		if verbose {
			if task.Prompt != "" {
				fmt.Printf("  Prompt: %s\n", p.truncateString(task.Prompt, 60))
			}
		}
	}

	fmt.Println(t)
	return nil
}

// OutputTaskDetails outputs detailed information about a task
func (p *TaskPresenter) OutputTaskDetails(task *claude.Task) error {
	fmt.Printf("Task: %s (ID: %s)\n", task.Name, task.ID)
	fmt.Printf("Status: %s\n", task.Status)
	fmt.Printf("Priority: %d\n", task.Priority)

	if task.Worktree != "" {
		fmt.Printf("Worktree: %s\n", task.Worktree)
	}

	fmt.Printf("Created: %s\n", task.CreatedAt.Format(time.RFC3339))

	if task.StartedAt != nil {
		fmt.Printf("Started: %s\n", task.StartedAt.Format(time.RFC3339))
	}

	if task.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", task.CompletedAt.Format(time.RFC3339))
	}

	if len(task.DependsOn) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(task.DependsOn, ", "))
	}

	if task.Prompt != "" {
		fmt.Printf("\nPrompt:\n%s\n", task.Prompt)
	}

	if len(task.VerificationCommands) > 0 {
		fmt.Printf("\nVerification Commands:\n")
		for _, cmd := range task.VerificationCommands {
			fmt.Printf("- %s\n", cmd)
		}
	}

	if task.Result != nil {
		fmt.Printf("\nExecution Result:\n")
		fmt.Printf("  Exit Code: %d\n", task.Result.ExitCode)
		fmt.Printf("  Duration: %s\n", p.formatDuration(task.Result.Duration))
		if task.Result.Error != "" {
			fmt.Printf("  Error: %s\n", task.Result.Error)
		}
		if len(task.Result.FilesChanged) > 0 {
			fmt.Printf("  Files Changed: %s\n", strings.Join(task.Result.FilesChanged, ", "))
		}
	}

	return nil
}

// OutputTaskLogs outputs task logs with detailed formatting
func (p *TaskPresenter) OutputTaskLogs(task *claude.Task) error {
	fmt.Printf("╭─ Task: %s ─────────────────────────────────────────╮\n", task.Name)
	fmt.Printf("│ ID: %-48s │\n", task.ID)
	fmt.Printf("│ Status: %-42s │\n", task.Status)
	fmt.Printf("│ Priority: %-40d │\n", task.Priority)
	fmt.Printf("│ Created: %-41s │\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

	if task.StartedAt != nil {
		fmt.Printf("│ Started: %-41s │\n", task.StartedAt.Format("2006-01-02 15:04:05"))
	}

	if task.CompletedAt != nil {
		fmt.Printf("│ Completed: %-39s │\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if task.Worktree != "" {
		fmt.Printf("│ Worktree: %-40s │\n", task.Worktree)
	}

	if task.WorktreePath != "" {
		fmt.Printf("│ Worktree: %-40s │\n", task.WorktreePath)
	}

	if task.SessionID != "" {
		fmt.Printf("│ Session: %-41s │\n", task.SessionID)
	}

	fmt.Printf("╰───────────────────────────────────────────────────────╯\n")

	if task.Prompt != "" {
		fmt.Printf("\n💬 Prompt:\n%s\n", task.Prompt)
	}

	if len(task.VerificationCommands) > 0 {
		fmt.Printf("\n✅ Verification Commands:\n")
		for _, cmd := range task.VerificationCommands {
			fmt.Printf("  • %s\n", cmd)
		}
	}

	if task.Result != nil {
		fmt.Printf("\n📊 Execution Result:\n")
		fmt.Printf("  Exit Code: %d\n", task.Result.ExitCode)
		fmt.Printf("  Duration: %s\n", p.formatDuration(task.Result.Duration))
		if task.Result.Error != "" {
			fmt.Printf("  Error: %s\n", task.Result.Error)
		}
		if len(task.Result.FilesChanged) > 0 {
			fmt.Printf("  Files Changed: %s\n", strings.Join(task.Result.FilesChanged, ", "))
		}
	}

	return nil
}

// OutputTasksJSON outputs tasks in JSON format
func (p *TaskPresenter) OutputTasksJSON(tasks []*claude.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// OutputTaskJSON outputs a single task in JSON format
func (p *TaskPresenter) OutputTaskJSON(task *claude.Task) error {
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// OutputTaskCreationSummary outputs a summary of a created task
func (p *TaskPresenter) OutputTaskCreationSummary(task *claude.Task) {
	displayName := task.Name
	if displayName == "" && task.Prompt != "" {
		// Truncate prompt to 60 characters if no name is available
		if len(task.Prompt) > 60 {
			displayName = task.Prompt[:57] + "..."
		} else {
			displayName = task.Prompt
		}
	}
	fmt.Printf("Task '%s' added successfully (ID: %s)\n", displayName, task.ID)
	fmt.Printf("Worktree: %s, Priority: %d\n", task.Worktree, task.Priority)

	if len(task.DependsOn) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(task.DependsOn, ", "))
	}
}

// OutputTaskFileCreationSummary outputs summary for multiple tasks created from file
func (p *TaskPresenter) OutputTaskFileCreationSummary(tasks []*claude.Task, fileName string) {
	successCount := len(tasks)

	for _, task := range tasks {
		displayName := task.Name
		if displayName == "" && task.Prompt != "" {
			// Truncate prompt to 60 characters if no name is available
			if len(task.Prompt) > 60 {
				displayName = task.Prompt[:57] + "..."
			} else {
				displayName = task.Prompt
			}
		}
		fmt.Printf("Task '%s' (%s) added successfully\n", displayName, task.ID)
		fmt.Printf("  Repository: %s\n", task.RepositoryRoot)
		fmt.Printf("  Worktree: %s, Priority: %d\n", task.Worktree, task.Priority)
		if len(task.DependsOn) > 0 {
			fmt.Printf("  Dependencies: %s\n", strings.Join(task.DependsOn, ", "))
		}
		fmt.Println()
	}

	fmt.Printf("Successfully added %d tasks from %s\n", successCount, fileName)
}

// getStatusIcon returns an icon for the task status
func (p *TaskPresenter) getStatusIcon(status claude.Status) string {
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

// formatDuration formats a duration for display
func (p *TaskPresenter) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// truncateString truncates a string to a maximum length
func (p *TaskPresenter) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
