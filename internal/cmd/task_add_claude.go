package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/d-kuro/gwq/internal/claude/presenters"
	"github.com/d-kuro/gwq/internal/config"
	"github.com/spf13/cobra"
)

var taskAddClaudeCmd = &cobra.Command{
	Use:   "claude [NAME]",
	Short: "Add a new Claude task",
	Long: `Add a new Claude Code task to the queue.

Tasks require a worktree name (-w flag). If the worktree doesn't exist,
it will be created automatically from the specified base branch (--base flag)
or from the current branch if no base is specified.

The task name is required and should be descriptive of the work to be done.

Tasks support:
- Priority levels (1-100, higher = more important)
- Dependencies on other tasks
- Detailed context and instructions
- Verification commands to ensure success
- Custom configuration options`,
	Example: `  # Basic task (creates worktree from current branch if needed)
  gwq task add claude -w feature/auth "Implement JWT authentication"

  # Task with specific base branch for worktree creation
  gwq task add claude -w feature/api --base develop "REST API endpoints" -p 80

  # Task with dependencies and detailed prompt
  gwq task add claude -w feature/tests "Add comprehensive tests" \
    --depends-on api-endpoints \
    --prompt "Add comprehensive unit tests. Target 90% coverage. Focus on error handling." \
    --verify "make test" \
    --verify "make coverage"`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runTaskAddClaude,
}

// Task add flags
var (
	taskAddClaudeWorktree     string
	taskAddClaudeBaseBranch   string
	taskAddClaudePriority     int
	taskAddClaudeDependsOn    []string
	taskAddClaudePrompt       string
	taskAddClaudeFilesToFocus []string
	taskAddClaudeVerify       []string
	taskAddClaudeAutoCommit   bool
	taskAddClaudeFile         string
)

func init() {
	taskAddCmd.AddCommand(taskAddClaudeCmd)

	// Task add flags
	taskAddClaudeCmd.Flags().StringVarP(&taskAddClaudeWorktree, "worktree", "w", "", "Worktree name (creates if doesn't exist)")
	taskAddClaudeCmd.Flags().StringVar(&taskAddClaudeBaseBranch, "base", "", "Base branch for worktree creation (defaults to current branch)")
	taskAddClaudeCmd.Flags().IntVarP(&taskAddClaudePriority, "priority", "p", 50, "Task priority (1-100, higher = more important)")
	taskAddClaudeCmd.Flags().StringSliceVar(&taskAddClaudeDependsOn, "depends-on", nil, "Task IDs this task depends on")
	taskAddClaudeCmd.Flags().StringVar(&taskAddClaudePrompt, "prompt", "", "Complete task prompt for Claude")
	taskAddClaudeCmd.Flags().StringSliceVar(&taskAddClaudeFilesToFocus, "files", nil, "Key files to focus on")
	taskAddClaudeCmd.Flags().StringSliceVar(&taskAddClaudeVerify, "verify", nil, "Commands to verify task completion")
	taskAddClaudeCmd.Flags().BoolVar(&taskAddClaudeAutoCommit, "auto-commit", false, "Enable automatic commits")
	taskAddClaudeCmd.Flags().StringVarP(&taskAddClaudeFile, "file", "f", "", "Load tasks from YAML file")
}

func runTaskAddClaude(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Initialize storage
	storage, err := claude.NewStorage(cfg.Claude.Queue.QueueDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create simplified task manager (no service layer)
	taskManager := claude.NewTaskManager(storage, cfg)
	presenter := presenters.NewTaskPresenter()

	// Handle file-based task creation
	if taskAddClaudeFile != "" {
		return handleTaskAddClaudeFileCreation(taskManager, presenter)
	}

	// Validate that NAME argument is provided for single task creation
	if len(args) == 0 {
		return fmt.Errorf("task name is required when not using --file flag")
	}

	// Handle single task creation
	return handleTaskAddClaudeSingleTaskCreation(args[0], taskManager, presenter)
}

func handleTaskAddClaudeFileCreation(taskManager *claude.TaskManager, presenter *presenters.TaskPresenter) error {
	tasks, err := taskManager.CreateTasksFromFile(taskAddClaudeFile)
	if err != nil {
		return err
	}

	presenter.OutputTaskFileCreationSummary(tasks, taskAddClaudeFile)
	return nil
}

func handleTaskAddClaudeSingleTaskCreation(name string, taskManager *claude.TaskManager, presenter *presenters.TaskPresenter) error {
	// Validate required flags
	if err := validateTaskAddClaudeFlags(); err != nil {
		return err
	}

	// Create task request
	req := &claude.CreateTaskRequest{
		Name:                 name,
		Worktree:             taskAddClaudeWorktree,
		BaseBranch:           taskAddClaudeBaseBranch,
		Priority:             taskAddClaudePriority,
		DependsOn:            taskAddClaudeDependsOn,
		Prompt:               taskAddClaudePrompt,
		FilesToFocus:         taskAddClaudeFilesToFocus,
		VerificationCommands: taskAddClaudeVerify,
		AutoCommit:           taskAddClaudeAutoCommit,
	}

	// Create task
	task, err := taskManager.CreateTask(req)
	if err != nil {
		return err
	}

	// Output summary
	presenter.OutputTaskCreationSummary(task)
	return nil
}

func validateTaskAddClaudeFlags() error {
	if taskAddClaudeWorktree == "" {
		return fmt.Errorf("--worktree must be specified")
	}

	if taskAddClaudePriority < 1 || taskAddClaudePriority > 100 {
		return fmt.Errorf("priority must be between 1 and 100")
	}

	return nil
}
