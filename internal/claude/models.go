// Package claude provides task queue management and execution for Claude Code integration.
// It includes models, services, and executors for managing AI-assisted development tasks
// with support for priorities, dependencies, and session management.
package claude

import (
	"context"
	"time"
)

// Priority represents task priority (1-100, higher = more important)
type Priority int

const (
	// PriorityVeryLow is used for background tasks that can wait indefinitely.
	PriorityVeryLow Priority = 10
	// PriorityLow is used for nice-to-have features that are not time-sensitive.
	PriorityLow Priority = 25
	// PriorityNormal is the default priority for standard development tasks.
	PriorityNormal Priority = 50
	// PriorityHigh is used for important features that should be prioritized.
	PriorityHigh Priority = 75
	// PriorityUrgent is used for critical fixes that need immediate attention.
	PriorityUrgent Priority = 90
	// PriorityCritical is used for blocking issues that must be resolved first.
	PriorityCritical Priority = 100
)

// Status represents the current state of a task
type Status string

const (
	// StatusPending indicates a task is queued and waiting to be executed.
	StatusPending Status = "pending"
	// StatusWaiting indicates a task is waiting for its dependencies to complete.
	StatusWaiting Status = "waiting"
	// StatusRunning indicates a task is currently being executed.
	StatusRunning Status = "running"
	// StatusCompleted indicates a task has been successfully finished.
	StatusCompleted Status = "completed"
	// StatusFailed indicates a task execution has failed.
	StatusFailed Status = "failed"
	// StatusSkipped indicates a task was skipped due to dependency policy.
	StatusSkipped Status = "skipped"
	// StatusCancelled indicates a task was manually cancelled.
	StatusCancelled Status = "cancelled"
)

// DependencyPolicy defines how to handle dependency failures
type DependencyPolicy string

const (
	// DependencyPolicyWait waits for dependencies to complete regardless of their status (default).
	DependencyPolicyWait DependencyPolicy = "wait"
	// DependencyPolicySkip skips this task if any dependency fails.
	DependencyPolicySkip DependencyPolicy = "skip"
	// DependencyPolicyFail fails this task immediately if any dependency fails.
	DependencyPolicyFail DependencyPolicy = "fail"
)

// TaskType represents the type of task
type TaskType string

const (
	// TaskTypeDevelopment represents standard development tasks.
	TaskTypeDevelopment TaskType = "development"
)

// Capability represents what an agent can do
type Capability string

const (
	// CapabilityCodeGeneration indicates the ability to generate new code.
	CapabilityCodeGeneration Capability = "code_generation"
	// CapabilityTesting indicates the ability to write and run tests.
	CapabilityTesting Capability = "testing"
	// CapabilityRefactoring indicates the ability to refactor existing code.
	CapabilityRefactoring Capability = "refactoring"
	// CapabilityDocumentation indicates the ability to write documentation.
	CapabilityDocumentation Capability = "documentation"
)

// Task represents a Claude Code development task
type Task struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Worktree    string     `json:"worktree"`    // Worktree name or path
	BaseBranch  string     `json:"base_branch"` // Base branch for worktree creation
	Priority    Priority   `json:"priority"`    // 1-100, higher = more important
	Status      Status     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Git worktree information (uses existing gwq worktrees)
	RepositoryRoot string `json:"repository_root"` // Git repository root path
	WorktreePath   string `json:"worktree_path"`   // Path to gwq worktree

	SessionID string `json:"session_id,omitempty"`
	AgentType string `json:"agent_type"`

	// Task dependencies
	DependsOn        []string         `json:"depends_on"`        // Task IDs this task depends on
	Blocks           []string         `json:"blocks,omitempty"`  // Task IDs blocked by this task (auto-populated)
	DependencyPolicy DependencyPolicy `json:"dependency_policy"` // How to handle dependency failures

	// Enhanced task definition based on Claude Code best practices
	Prompt               string   `json:"prompt"`                // Complete task prompt for Claude
	FilesToFocus         []string `json:"files_to_focus"`        // Key files to work on (relative to worktree)
	VerificationCommands []string `json:"verification_commands"` // Commands to verify success (run in worktree)

	// Task configuration
	Config TaskConfig `json:"config"`

	// Results
	Result *TaskResult `json:"result,omitempty"`

	// Internal flags
	AutoCreateWorktree bool `json:"auto_create_worktree,omitempty"` // Whether to create worktree if it doesn't exist
}

// TaskConfig holds configuration for a task
type TaskConfig struct {
	SkipPermissions bool `json:"skip_permissions"`
	AutoCommit      bool `json:"auto_commit"`
	BackupFiles     bool `json:"backup_files"`
}

// TaskResult represents the outcome of task execution
type TaskResult struct {
	ExitCode             int           `json:"exit_code"`
	Duration             time.Duration `json:"duration"`
	FilesChanged         []string      `json:"files_changed"`
	CommitHash           string        `json:"commit_hash,omitempty"`
	DependenciesWaitTime time.Duration `json:"dependencies_wait_time"` // Time spent waiting for dependencies
	DependencyFailures   []string      `json:"dependency_failures"`    // Failed dependencies that affected this task
	Error                string        `json:"error,omitempty"`        // Error message if task failed
}

// TaskFile represents the YAML structure for batch task creation
type TaskFile struct {
	Version       string          `yaml:"version"`
	Repository    string          `yaml:"repository,omitempty"` // Target repository (path, URL, or gwq format)
	DefaultConfig *TaskConfig     `yaml:"default_config,omitempty"`
	Tasks         []TaskFileEntry `yaml:"tasks"`
}

// TaskFileEntry represents a single task in the YAML file
type TaskFileEntry struct {
	ID                   string           `yaml:"id"`
	Name                 string           `yaml:"name"`
	Repository           string           `yaml:"repository,omitempty"` // Override repository for this specific task
	Worktree             string           `yaml:"worktree"`             // Worktree name or path
	BaseBranch           string           `yaml:"base_branch"`          // Base branch for worktree creation (required)
	Priority             int              `yaml:"priority,omitempty"`
	DependsOn            []string         `yaml:"depends_on,omitempty"`
	DependencyPolicy     DependencyPolicy `yaml:"dependency_policy,omitempty"`
	Prompt               string           `yaml:"prompt,omitempty"`
	FilesToFocus         []string         `yaml:"files_to_focus,omitempty"`
	VerificationCommands []string         `yaml:"verification_commands,omitempty"`
	Config               *TaskConfig      `yaml:"config,omitempty"`
}

// Agent interface for future extensibility
type Agent interface {
	// Basic information
	Name() string
	Version() string
	Capabilities() []Capability

	// Task execution
	Execute(ctx context.Context, task *Task) (*TaskResult, error)

	// Health check
	HealthCheck() error
	IsAvailable() bool

	// Session management
	CreateSession(ctx context.Context, task *Task) (string, error) // Returns session ID
	AttachSession(ctx context.Context, sessionID string) error
}
