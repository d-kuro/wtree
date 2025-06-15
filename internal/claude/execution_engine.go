package claude

import (
	"context"
	"fmt"
	"time"

	"github.com/d-kuro/gwq/pkg/models"
	"github.com/d-kuro/gwq/pkg/utils"
)

// ExecutionType represents the type of execution
type ExecutionType string

const (
	ExecutionTypeTask ExecutionType = "task"
)

// UnifiedExecution represents a unified execution record
type UnifiedExecution struct {
	// Core identification
	ExecutionID   string        `json:"execution_id"`   // Unique across all execution types
	SessionID     string        `json:"session_id"`     // tmux session identifier
	ExecutionType ExecutionType `json:"execution_type"` // "task"

	// Timing and status
	StartTime time.Time       `json:"start_time"`
	EndTime   *time.Time      `json:"end_time,omitempty"`
	Status    ExecutionStatus `json:"status"`

	// Execution context
	Repository  string `json:"repository"`
	WorkingDir  string `json:"working_directory"`
	TmuxSession string `json:"tmux_session"`

	// Content and results
	Prompt string           `json:"prompt"` // User prompt or generated prompt for tasks
	Result *ExecutionResult `json:"result,omitempty"`

	// Task-specific information (when ExecutionType == "task")
	TaskInfo *TaskExecutionInfo `json:"task_info,omitempty"`

	// Metadata
	Tags       []string      `json:"tags,omitempty"`
	Priority   string        `json:"priority"`
	Model      string        `json:"model,omitempty"`
	CostUSD    float64       `json:"cost_usd"`
	DurationMS int64         `json:"duration_ms"`
	Timeout    time.Duration `json:"timeout"`
}

// TaskExecutionInfo contains task-specific execution information
type TaskExecutionInfo struct {
	TaskID             string   `json:"task_id"`
	TaskName           string   `json:"task_name"`
	Worktree           string   `json:"worktree"` // Worktree name or path
	WorktreePath       string   `json:"worktree_path,omitempty"`
	BaseBranch         string   `json:"base_branch,omitempty"`          // Base branch for worktree creation
	AutoCreateWorktree bool     `json:"auto_create_worktree,omitempty"` // Whether to create worktree if it doesn't exist
	Dependencies       []string `json:"dependencies,omitempty"`
	TaskPriority       int      `json:"task_priority"`
	Prompt             string   `json:"prompt,omitempty"`
}

// ExecutionResult contains detailed execution results
type ExecutionResult struct {
	Success      bool     `json:"success"`
	ExitCode     int      `json:"exit_code"`
	Error        string   `json:"error,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`

	// Detailed analysis
	TokensUsed int      `json:"tokens_used,omitempty"`
	ToolsUsed  []string `json:"tools_used,omitempty"`
	Summary    string   `json:"summary,omitempty"`
}

// ExecutionRequest represents a request to execute Claude Code
type ExecutionRequest struct {
	Type       ExecutionType
	Prompt     string
	Repository string
	WorkingDir string
	TaskInfo   *TaskExecutionInfo // For task executions
	Tags       []string
	Priority   string
	Timeout    time.Duration
}

// ExecutionEngine provides unified execution of Claude Code for all execution types
type ExecutionEngine struct {
	config         *models.ClaudeConfig
	sessionManager *UnifiedSessionManager
	logManager     *UnifiedLogManager
	claudeExecutor *ClaudeCodeExecutor
}

// NewExecutionEngine creates a new unified execution engine
func NewExecutionEngine(config *models.ClaudeConfig) (*ExecutionEngine, error) {
	// Create unified session manager
	sessionManager, err := NewUnifiedSessionManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create unified log manager
	logManager, err := NewUnifiedLogManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create log manager: %w", err)
	}

	// Create Claude executor
	claudeExecutor := NewClaudeCodeExecutor(config)

	return &ExecutionEngine{
		config:         config,
		sessionManager: sessionManager,
		logManager:     logManager,
		claudeExecutor: claudeExecutor,
	}, nil
}

// Execute runs a unified Claude Code execution
func (ee *ExecutionEngine) Execute(ctx context.Context, req *ExecutionRequest) (*UnifiedExecution, error) {
	// Generate IDs
	executionID := ee.generateExecutionID(req.Type)
	sessionID := ee.generateSessionID()

	// Create unified execution record
	execution := &UnifiedExecution{
		ExecutionID:   executionID,
		SessionID:     sessionID,
		ExecutionType: req.Type,
		StartTime:     time.Now(),
		Status:        ExecutionStatusRunning,
		Repository:    req.Repository,
		WorkingDir:    req.WorkingDir,
		Prompt:        req.Prompt,
		TaskInfo:      req.TaskInfo,
		Tags:          req.Tags,
		Priority:      req.Priority,
		Timeout:       req.Timeout,
	}

	// Create tmux session with unified naming
	session, err := ee.sessionManager.CreateSession(ctx, execution)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	execution.TmuxSession = session.SessionName

	// Start unified logging
	logFile, err := ee.logManager.StartLogging(execution)
	if err != nil {
		return nil, fmt.Errorf("failed to start logging: %w", err)
	}

	// Execute Claude Code with unified monitoring
	result, err := ee.claudeExecutor.Execute(ctx, execution, logFile)

	// Update execution record
	execution.Result = result
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.DurationMS = int64(endTime.Sub(execution.StartTime).Milliseconds())

	if err != nil {
		execution.Status = ExecutionStatusFailed
		if execution.Result == nil {
			execution.Result = &ExecutionResult{}
		}
		execution.Result.Error = err.Error()
	} else {
		execution.Status = ExecutionStatusCompleted
	}

	// Save to unified storage
	if saveErr := ee.logManager.SaveExecution(execution); saveErr != nil {
		return nil, fmt.Errorf("failed to save execution: %w", saveErr)
	}

	return execution, err
}

// ExecuteTask is a convenience method for executing tasks through the unified engine
func (ee *ExecutionEngine) ExecuteTask(ctx context.Context, task *Task) (*UnifiedExecution, error) {
	// Convert task to execution request
	req := &ExecutionRequest{
		Type:       ExecutionTypeTask,
		Repository: task.RepositoryRoot,
		WorkingDir: task.WorktreePath,
		Priority:   fmt.Sprintf("%d", task.Priority),
		Timeout:    2 * time.Hour, // Default timeout for tasks
		TaskInfo: &TaskExecutionInfo{
			TaskID:             task.ID,
			TaskName:           task.Name,
			Worktree:           task.Worktree,
			WorktreePath:       task.WorktreePath,
			BaseBranch:         task.BaseBranch,
			AutoCreateWorktree: task.AutoCreateWorktree,
			Dependencies:       task.DependsOn,
			TaskPriority:       int(task.Priority),
			Prompt:             task.Prompt,
		},
	}

	// Build task prompt
	req.Prompt = ee.buildTaskPrompt(task)

	// Execute through unified engine
	execution, err := ee.Execute(ctx, req)
	if err != nil {
		return execution, err
	}

	// Update task with execution results
	if execution.Result != nil {
		task.Result = &TaskResult{
			ExitCode:     execution.Result.ExitCode,
			Duration:     time.Duration(execution.DurationMS) * time.Millisecond,
			FilesChanged: execution.Result.FilesChanged,
			Error:        execution.Result.Error,
		}
	}

	return execution, nil
}

// GetExecution retrieves a unified execution by ID
func (ee *ExecutionEngine) GetExecution(executionID string) (*UnifiedExecution, error) {
	return ee.logManager.LoadExecution(executionID)
}

// ListExecutions lists all executions with optional filtering
func (ee *ExecutionEngine) ListExecutions(filters ...ExecutionFilter) ([]*UnifiedExecution, error) {
	return ee.logManager.ListExecutions(filters...)
}

// ExecutionFilter represents a filter for listing executions
type ExecutionFilter func(*UnifiedExecution) bool

// FilterByType filters executions by type
func FilterByType(execType ExecutionType) ExecutionFilter {
	return func(exec *UnifiedExecution) bool {
		return exec.ExecutionType == execType
	}
}

// FilterByStatus filters executions by status
func FilterByStatus(status ExecutionStatus) ExecutionFilter {
	return func(exec *UnifiedExecution) bool {
		return exec.Status == status
	}
}

// generateExecutionID generates a unique execution ID with type prefix
func (ee *ExecutionEngine) generateExecutionID(execType ExecutionType) string {
	return fmt.Sprintf("%s-%s", execType, utils.GenerateShortID())
}

// generateSessionID generates a unique session ID
func (ee *ExecutionEngine) generateSessionID() string {
	return utils.GenerateUUID()
}

// buildTaskPrompt builds a comprehensive prompt for tasks
func (ee *ExecutionEngine) buildTaskPrompt(task *Task) string {
	if task.Prompt != "" {
		return task.Prompt
	}
	return task.Name
}
