package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/d-kuro/gwq/internal/claude/presenters"
	"github.com/d-kuro/gwq/internal/claude/services"
	"github.com/d-kuro/gwq/internal/config"
	"github.com/spf13/cobra"
)

var taskShowCmd = &cobra.Command{
	Use:   "show [TASK_ID]",
	Short: "Show detailed task information",
	Long: `Show detailed information about a specific Claude task.

Displays complete task details including context, objectives, instructions,
constraints, dependencies, and execution results. If no task ID is provided,
a fuzzy finder will be shown to select a task.`,
	Example: `  # Show specific task
  gwq task show auth-impl

  # Show task with pattern matching
  gwq task show auth

  # Interactive task selection
  gwq task show`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTaskShow,
}

func init() {
	taskCmd.AddCommand(taskShowCmd)
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Initialize storage
	storage, err := claude.NewStorage(cfg.Claude.Queue.QueueDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create simplified task manager (no service layer)
	taskManager := claude.NewTaskManager(storage, cfg)
	finderService := services.NewFuzzyFinderService()
	presenter := presenters.NewTaskPresenter()

	var task *claude.Task

	if len(args) > 0 {
		// Find task by ID or pattern
		task, err = taskManager.FindTaskByPattern(args[0])
		if err != nil {
			return err
		}
	} else {
		// Interactive task selection
		task, err = selectTaskShowInteractively(storage, finderService)
		if err != nil {
			return err
		}

		if task == nil {
			return nil // User cancelled
		}
	}

	return presenter.OutputTaskDetails(task)
}

func selectTaskShowInteractively(storage *claude.Storage, finderService *services.FuzzyFinderService) (*claude.Task, error) {
	// Load all tasks
	tasks, err := storage.ListTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil, nil
	}

	// Use fuzzy finder for selection
	return finderService.SelectTask(tasks)
}
