package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/d-kuro/gwq/internal/claude/presenters"
	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/table"
	"github.com/spf13/cobra"
)

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Claude tasks",
	Long: `List all Claude Code tasks with their current status and dependencies.

The list shows tasks in a tree structure when dependencies exist, making it
easy to understand the relationship between tasks. The display includes:
- Task status (pending, waiting, running, completed, failed)
- Priority level (1-100)
- Dependencies and dependent tasks
- Duration for completed tasks`,
	Example: `  # List all tasks
  gwq task list

  # Filter by status
  gwq task list --filter running

  # Show only high priority tasks
  gwq task list --priority-min 75

  # Watch for real-time updates
  gwq task list --watch`,
	RunE: runTaskList,
}

// Task list flags
var (
	taskListFilter      string
	taskListPriorityMin int
	taskListWatch       bool
	taskListVerbose     bool
	taskListJSON        bool
	taskListCSV         bool
)

func init() {
	taskCmd.AddCommand(taskListCmd)

	// Task list flags
	taskListCmd.Flags().StringVar(&taskListFilter, "filter", "", "Filter by status (pending, running, completed, failed)")
	taskListCmd.Flags().IntVar(&taskListPriorityMin, "priority-min", 0, "Show only tasks with priority >= value")
	taskListCmd.Flags().BoolVar(&taskListWatch, "watch", false, "Watch for real-time updates")
	taskListCmd.Flags().BoolVarP(&taskListVerbose, "verbose", "v", false, "Show detailed information")
	taskListCmd.Flags().BoolVar(&taskListJSON, "json", false, "Output in JSON format")
	taskListCmd.Flags().BoolVar(&taskListCSV, "csv", false, "Output in CSV format")
}

func runTaskList(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Initialize storage
	storage, err := claude.NewStorage(cfg.Claude.Queue.QueueDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create simplified task manager (no service layer)
	taskManager := claude.NewTaskManager(storage, cfg)
	presenter := presenters.NewTaskPresenter()

	// Load tasks
	tasks, err := storage.ListTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Apply filters
	tasks = applyTaskListFilters(tasks, taskManager)

	// Output tasks based on format
	return outputTaskList(tasks, presenter)
}

func applyTaskListFilters(tasks []*claude.Task, taskManager *claude.TaskManager) []*claude.Task {
	// Apply status filter
	if taskListFilter != "" {
		tasks = taskManager.FilterTasksByStatus(tasks, taskListFilter)
	}

	// Apply priority filter
	if taskListPriorityMin > 0 {
		tasks = taskManager.FilterTasksByPriority(tasks, taskListPriorityMin)
	}

	return tasks
}

func outputTaskList(tasks []*claude.Task, presenter *presenters.TaskPresenter) error {
	if taskListJSON {
		return presenter.OutputTasksJSON(tasks)
	}

	if taskListCSV {
		return outputTaskListCSV(tasks)
	}

	if taskListWatch {
		return watchTaskList()
	}

	return presenter.OutputTasksTable(tasks, taskListVerbose)
}

func outputTaskListCSV(tasks []*claude.Task) error {
	// Create table with CSV-friendly data
	t := table.New().Headers("task_id", "worktree", "status", "priority", "dependencies", "duration")

	// Add rows to table
	for _, task := range tasks {
		status := string(task.Status)

		worktree := task.Worktree

		deps := strconv.Itoa(len(task.DependsOn))
		if len(task.DependsOn) == 0 {
			deps = "0"
		}

		duration := ""
		if task.Result != nil {
			duration = formatDurationForCSV(task.Result.Duration)
		} else if task.StartedAt != nil {
			duration = formatDurationForCSV(time.Since(*task.StartedAt))
		}

		t.Row(
			task.ID,
			worktree,
			status,
			strconv.Itoa(int(task.Priority)),
			deps,
			duration,
		)
	}

	return t.WriteCSV()
}

// formatDurationForCSV formats a duration for CSV output (no special characters)
func formatDurationForCSV(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func watchTaskList() error {
	// TODO: Implement watch mode
	return fmt.Errorf("watch mode not yet implemented")
}
