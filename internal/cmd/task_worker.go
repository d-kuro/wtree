package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/tmux"
	"github.com/spf13/cobra"
)

var taskWorkerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Manage Claude Code workers",
	Long: `Manage Claude Code workers that process tasks from the queue.

Workers are responsible for:
- Polling the task queue for executable tasks
- Resolving task dependencies
- Managing resource allocation and parallelism
- Executing Claude Code tasks in tmux sessions
- Handling task completion and cleanup

The worker system ensures efficient resource utilization while respecting
dependency constraints and priority ordering.`,
	Example: `  # Start worker with default settings
  gwq task worker start

  # Start worker with custom parallelism
  gwq task worker start --parallel 3

  # Check worker status
  gwq task worker status

  # Stop worker
  gwq task worker stop`,
}

var taskWorkerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Claude Code worker",
	Long: `Start a Claude Code worker to process tasks from the queue.

The worker will continuously poll the task queue for executable tasks,
resolve dependencies, and execute tasks using Claude Code in dedicated
tmux sessions. The worker respects resource limits and parallelism
constraints configured in the Claude settings.

The worker runs in the foreground by default and can be stopped with Ctrl+C.
All active tasks will be allowed to complete gracefully during shutdown.`,
	Example: `  # Start with default parallelism
  gwq task worker start

  # Start with custom parallelism
  gwq task worker start --parallel 3

  # Start in background (daemon mode)
  gwq task worker start --daemon`,
	RunE: runTaskWorkerStart,
}

var taskWorkerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Claude Code worker",
	Long: `Stop the currently running Claude Code worker.

This command will gracefully shut down the worker, allowing active tasks
to complete before terminating. If tasks are still running, the command
will wait up to the specified timeout before forcefully stopping.`,
	Example: `  # Stop worker gracefully
  gwq task worker stop

  # Stop with custom timeout
  gwq task worker stop --timeout 5m`,
	RunE: runTaskWorkerStop,
}

var taskWorkerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Claude Code worker status",
	Long: `Show the current status of Claude Code workers and active tasks.

Displays information about:
- Worker running state
- Active task count and resource utilization
- Queue statistics (pending, waiting, completed)
- Recent task activity
- Session management status`,
	Example: `  # Show basic status
  gwq task worker status

  # Show detailed status
  gwq task worker status --verbose

  # Show status in JSON format
  gwq task worker status --json`,
	RunE: runTaskWorkerStatus,
}

// Worker flags
var (
	taskWorkerParallel int
	taskWorkerDaemon   bool
	taskWorkerTimeout  time.Duration
	taskWorkerVerbose  bool
	taskWorkerJSON     bool
)

func init() {
	taskCmd.AddCommand(taskWorkerCmd)
	taskWorkerCmd.AddCommand(taskWorkerStartCmd, taskWorkerStopCmd, taskWorkerStatusCmd)

	// Start command flags
	taskWorkerStartCmd.Flags().IntVar(&taskWorkerParallel, "parallel", 0, "Maximum parallel tasks (0 = use config default)")
	taskWorkerStartCmd.Flags().BoolVar(&taskWorkerDaemon, "daemon", false, "Run in background as daemon")

	// Stop command flags
	taskWorkerStopCmd.Flags().DurationVar(&taskWorkerTimeout, "timeout", 5*time.Minute, "Graceful shutdown timeout")

	// Status command flags
	taskWorkerStatusCmd.Flags().BoolVarP(&taskWorkerVerbose, "verbose", "v", false, "Show detailed status information")
	taskWorkerStatusCmd.Flags().BoolVar(&taskWorkerJSON, "json", false, "Output status in JSON format")
}

func runTaskWorkerStart(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Use config defaults if not specified
	if taskWorkerParallel == 0 {
		taskWorkerParallel = cfg.Claude.MaxParallel
	}

	fmt.Printf("Starting Claude Code worker (max parallel: %d)\n", taskWorkerParallel)

	// Initialize components
	storage, err := claude.NewStorage(cfg.Claude.Queue.QueueDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create unified execution engine
	executionEngine, err := claude.NewExecutionEngine(&cfg.Claude)
	if err != nil {
		return fmt.Errorf("failed to create execution engine: %w", err)
	}

	resourceMgr := claude.NewResourceManager(
		cfg.Claude.MaxParallel,
		cfg.Claude.MaxDevelopmentTasks,
	)

	dependencyGraph := claude.NewDependencyGraph()

	// Create worker
	worker := NewTaskWorker(TaskWorkerConfig{
		Storage:         storage,
		ExecutionEngine: executionEngine,
		ResourceManager: resourceMgr,
		DependencyGraph: dependencyGraph,
		MaxParallel:     taskWorkerParallel,
		PollInterval:    5 * time.Second,
	})

	// Handle shutdown gracefully
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nReceived shutdown signal, stopping worker...")
		cancel()
	}()

	// Start worker
	if err := worker.Start(ctx); err != nil {
		return fmt.Errorf("worker failed: %w", err)
	}

	fmt.Println("Worker stopped.")
	return nil
}

func runTaskWorkerStop(cmd *cobra.Command, args []string) error {
	// TODO: Implement worker stop logic
	// This would typically involve:
	// 1. Finding the running worker process
	// 2. Sending a graceful shutdown signal
	// 3. Waiting for completion with timeout

	fmt.Println("Worker stop not yet implemented.")
	fmt.Println("Use Ctrl+C to stop a running worker.")
	return nil
}

func runTaskWorkerStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Initialize storage to get task statistics
	storage, err := claude.NewStorage(cfg.Claude.Queue.QueueDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Get task statistics
	tasks, err := storage.ListTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Count tasks by status
	statusCounts := make(map[claude.Status]int)
	for _, task := range tasks {
		statusCounts[task.Status]++
	}

	// Get tmux session information
	sessionMgr := tmux.NewSessionManager(&tmux.SessionConfig{
		Enabled:      true,
		TmuxCommand:  "tmux",
		HistoryLimit: 50000,
	}, cfg.Claude.ConfigDir)

	sessions, err := sessionMgr.ListSessions()
	if err != nil {
		sessions = []*tmux.Session{} // Don't fail if tmux is not available
	}

	// Filter Claude sessions
	claudeSessions := filterTaskClaudeSessions(sessions)

	// Output status
	if taskWorkerJSON {
		return outputTaskWorkerStatusJSON(statusCounts, claudeSessions)
	}

	return outputTaskWorkerStatusTable(statusCounts, claudeSessions, taskWorkerVerbose)
}

// TaskWorker manages the execution of Claude tasks
type TaskWorker struct {
	config          TaskWorkerConfig
	storage         *claude.Storage
	executionEngine *claude.ExecutionEngine
	resourceMgr     *claude.ResourceManager
	dependencyGraph *claude.DependencyGraph
	running         bool
	mu              sync.RWMutex
}

type TaskWorkerConfig struct {
	Storage         *claude.Storage
	ExecutionEngine *claude.ExecutionEngine
	ResourceManager *claude.ResourceManager
	DependencyGraph *claude.DependencyGraph
	MaxParallel     int
	PollInterval    time.Duration
}

func NewTaskWorker(config TaskWorkerConfig) *TaskWorker {
	return &TaskWorker{
		config:          config,
		storage:         config.Storage,
		executionEngine: config.ExecutionEngine,
		resourceMgr:     config.ResourceManager,
		dependencyGraph: config.DependencyGraph,
	}
}

func (w *TaskWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	// Load existing tasks into dependency graph
	if err := w.loadTasks(); err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Start worker loop
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	fmt.Println("Worker started, polling for tasks...")

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Worker shutting down...")
			return w.shutdown(ctx)
		case <-ticker.C:
			if err := w.processTasks(ctx); err != nil {
				fmt.Printf("Error processing tasks: %v\n", err)
			}
		}
	}
}

func (w *TaskWorker) loadTasks() error {
	tasks, err := w.storage.ListTasks()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err := w.dependencyGraph.AddTask(task); err != nil {
			fmt.Printf("Warning: failed to add task %s to dependency graph: %v\n", task.ID, err)
		}
	}

	return nil
}

func (w *TaskWorker) processTasks(ctx context.Context) error {
	// Get executable tasks
	readyTasks := w.dependencyGraph.GetReadyTasks()

	for _, task := range readyTasks {
		// Check if we can acquire a resource slot
		if !w.resourceMgr.CanAcquire(claude.TaskTypeDevelopment) {
			break // No more resources available
		}

		// Try to acquire slot
		slot, err := w.resourceMgr.TryAcquireSlot(claude.TaskTypeDevelopment, task.ID)
		if err != nil {
			continue // Skip if can't acquire slot
		}

		// Start task execution
		go w.executeTask(ctx, task, slot)
	}

	return nil
}

func (w *TaskWorker) executeTask(ctx context.Context, task *claude.Task, slot *claude.Slot) {
	defer slot.Release()

	// Update task status
	task.Status = claude.StatusRunning
	startTime := time.Now()
	task.StartedAt = &startTime

	if err := w.storage.SaveTask(task); err != nil {
		fmt.Printf("Error updating task status: %v\n", err)
		return
	}

	displayName := task.Name
	if displayName == "" && task.Prompt != "" {
		// Truncate prompt to 60 characters if no name is available
		if len(task.Prompt) > 60 {
			displayName = task.Prompt[:57] + "..."
		} else {
			displayName = task.Prompt
		}
	}
	fmt.Printf("Starting task: %s (ID: %s)\n", displayName, task.ID)

	// Execute task through unified execution engine
	execution, err := w.executionEngine.ExecuteTask(ctx, task)

	// Update task with execution results
	if execution != nil {
		task.SessionID = execution.TmuxSession
		if execution.Result != nil {
			task.Result = &claude.TaskResult{
				ExitCode:     execution.Result.ExitCode,
				Duration:     time.Duration(execution.DurationMS) * time.Millisecond,
				FilesChanged: execution.Result.FilesChanged,
				Error:        execution.Result.Error,
			}
		}
	}

	if err != nil {
		task.Status = claude.StatusFailed
		if task.Result == nil {
			task.Result = &claude.TaskResult{}
		}
		task.Result.Error = err.Error()
		fmt.Printf("Task failed: %s - %v\n", task.ID, err)
	} else {
		task.Status = claude.StatusCompleted
		fmt.Printf("Task completed: %s\n", task.ID)
	}

	completedTime := time.Now()
	task.CompletedAt = &completedTime

	// Update dependency graph and storage
	if err := w.dependencyGraph.UpdateTask(task); err != nil {
		fmt.Printf("Error updating dependency graph: %v\n", err)
	}
	if err := w.storage.SaveTask(task); err != nil {
		fmt.Printf("Error saving task result: %v\n", err)
	}

}

func (w *TaskWorker) shutdown(ctx context.Context) error {
	fmt.Println("Waiting for active tasks to complete...")

	// TODO: Implement graceful shutdown
	// 1. Stop accepting new tasks
	// 2. Wait for active tasks to complete
	// 3. Clean up resources

	return nil
}

func filterTaskClaudeSessions(sessions []*tmux.Session) []*tmux.Session {
	var claudeSessions []*tmux.Session
	for _, session := range sessions {
		if session.Context == "claude" {
			claudeSessions = append(claudeSessions, session)
		}
	}
	return claudeSessions
}

func outputTaskWorkerStatusJSON(statusCounts map[claude.Status]int, sessions []*tmux.Session) error {
	// TODO: Implement JSON output
	return fmt.Errorf("JSON output not yet implemented")
}

func outputTaskWorkerStatusTable(statusCounts map[claude.Status]int, sessions []*tmux.Session, verbose bool) error {
	fmt.Println("Claude Worker Status")
	fmt.Println("===================")

	// Show running status
	running := len(sessions) > 0
	if running {
		fmt.Printf("Status: Running (%d active sessions)\n", len(sessions))
	} else {
		fmt.Println("Status: Not running")
	}

	// Show task queue statistics
	fmt.Println("\nQueue Statistics:")
	fmt.Printf("  Pending:   %d\n", statusCounts[claude.StatusPending])
	fmt.Printf("  Waiting:   %d\n", statusCounts[claude.StatusWaiting])
	fmt.Printf("  Running:   %d\n", statusCounts[claude.StatusRunning])
	fmt.Printf("  Completed: %d\n", statusCounts[claude.StatusCompleted])
	fmt.Printf("  Failed:    %d\n", statusCounts[claude.StatusFailed])

	// Show active sessions if verbose
	if verbose && len(sessions) > 0 {
		fmt.Println("\nActive Sessions:")
		for _, session := range sessions {
			taskID := session.Metadata["task_id"]
			taskName := session.Metadata["task_name"]
			duration := time.Since(session.StartTime)

			fmt.Printf("  %s: %s (%s) - %s\n",
				taskID, taskName, session.Context, formatTaskWorkerDuration(duration))
		}
	}

	return nil
}

// formatTaskWorkerDuration formats a duration for display
func formatTaskWorkerDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
