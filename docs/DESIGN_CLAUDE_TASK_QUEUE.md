# Claude Task Queue Design

## Overview

Design for an automated task queue system with Claude Code integration. This system enables developers to leverage idle time (such as during sleep) to automatically execute tasks with Claude Code and complete code reviews.

The design is Claude Code-focused while maintaining extensibility for future AI agents (Cursor, GitHub Copilot, etc.).

## Core Concepts

### Git Worktree-First Design

- **Repository Root Execution**: All `gwq claude` commands executed from git repository root
- **Automatic Worktree Management**: Tasks always use git worktrees for isolation
- **Dedicated Commands**: All functionality provided through `gwq claude` subcommands
- **Automatic Review**: Automatically execute code reviews upon task completion
- **tmux Integration**: Process persistence using tmux session management
- **Parallelism Control**: System-wide Claude Code parallel execution control

### Future Extensibility

```go
// Agent interface for future extensibility
type Agent interface {
    Name() string
    Execute(ctx context.Context, task *Task) (*Result, error)
    HealthCheck() error
    Capabilities() []Capability
}

// Claude Code implementation
type ClaudeAgent struct {
    config *ClaudeConfig
    session *tmux.Session
}

// Future agents
type CursorAgent struct { /* ... */ }
type CopilotAgent struct { /* ... */ }
```

## Architecture

### Overall Structure

```mermaid
graph TD
    A[gwq claude] --> B[Task Manager]
    B --> C[Claude Agent]
    B --> D[Review Agent] 
    B --> E[tmux Manager]
    
    C --> F[Claude Code Process]
    D --> G[Review Process]
    E --> H[tmux Sessions]
    
    B --> I[Global Resource Manager]
    I --> J[Semaphore Control]
    
    F --> K[Worktree 1]
    G --> L[Worktree 2]
    H --> M[tmux History]
```

### Data Model

```go
type Task struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Branch       string            `json:"branch"`              // Target branch for worktree
    Priority     Priority          `json:"priority"`            // 1-100, higher = more important
    Status       Status            `json:"status"`
    CreatedAt    time.Time         `json:"created_at"`
    StartedAt    *time.Time        `json:"started_at,omitempty"`
    CompletedAt  *time.Time        `json:"completed_at,omitempty"`
    
    // Git worktree information (required)
    RepositoryRoot   string            `json:"repository_root"`        // Git repository root path
    WorktreePath     string            `json:"worktree_path"`          // Dedicated worktree path for this task
    WorktreeName     string            `json:"worktree_name"`          // Worktree identifier
    
    SessionID    string            `json:"session_id,omitempty"`
    AgentType    string            `json:"agent_type"`
    
    // Task dependencies
    DependsOn        []string          `json:"depends_on"`             // Task IDs this task depends on
    Blocks           []string          `json:"blocks,omitempty"`       // Task IDs blocked by this task (auto-populated)
    DependencyPolicy DependencyPolicy  `json:"dependency_policy"`      // How to handle dependency failures
    
    // Enhanced task definition based on Claude Code best practices
    Context              string            `json:"context"`                // Background and problem description
    Objectives           []string          `json:"objectives"`             // Clear, measurable goals
    Instructions         string            `json:"instructions"`           // Detailed implementation guidance
    Constraints          []string          `json:"constraints"`            // Limitations and requirements
    FilesToFocus         []string          `json:"files_to_focus"`         // Key files to work on (relative to worktree)
    VerificationCommands []string          `json:"verification_commands"`  // Commands to verify success (run in worktree)
    
    // Task configuration
    Config TaskConfig `json:"config"`
    
    // Results
    Result       *TaskResult       `json:"result,omitempty"`
    ReviewResult *ReviewResult     `json:"review_result,omitempty"`
}

type DependencyPolicy string

const (
    DependencyPolicyWait DependencyPolicy = "wait"  // Wait for dependencies to complete (default)
    DependencyPolicySkip DependencyPolicy = "skip"  // Skip this task if dependency fails
    DependencyPolicyFail DependencyPolicy = "fail"  // Fail this task if dependency fails
)

type TaskConfig struct {
    SkipPermissions bool          `json:"skip_permissions"`
    Timeout         string        `json:"timeout"`
    MaxIterations   int           `json:"max_iterations"`
    AutoReview      bool          `json:"auto_review"`
    AutoCommit      bool          `json:"auto_commit"`
    BackupFiles     bool          `json:"backup_files"`
}

// Numeric priority system (0-100, higher numbers = higher priority)
type Priority int

const (
    PriorityVeryLow  Priority = 10   // Background tasks
    PriorityLow      Priority = 25   // Nice-to-have features
    PriorityNormal   Priority = 50   // Standard development tasks
    PriorityHigh     Priority = 75   // Important features
    PriorityUrgent   Priority = 90   // Critical fixes
    PriorityCritical Priority = 100  // Blocking issues
)

// Custom priorities can use any value 1-100
// Examples: 65 (above normal), 42 (below normal), 88 (very urgent)

type Status string
const (
    StatusPending     Status = "pending"
    StatusRunning     Status = "running"
    StatusReviewing   Status = "reviewing"
    StatusCompleted   Status = "completed"
    StatusFailed      Status = "failed"
)

type TaskResult struct {
    ExitCode           int              `json:"exit_code"`
    Duration           time.Duration    `json:"duration"`
    FilesChanged       []string         `json:"files_changed"`
    CommitHash         string           `json:"commit_hash,omitempty"`
    DependenciesWaitTime time.Duration  `json:"dependencies_wait_time"`  // Time spent waiting for dependencies
    DependencyFailures []string         `json:"dependency_failures"`     // Failed dependencies that affected this task
}

type ReviewResult struct {
    Issues       []ReviewIssue    `json:"issues"`
    FixesApplied []string         `json:"fixes_applied"`
    Summary      string           `json:"summary"`
}
```

## Command Design

### gwq claude subcommands

#### `gwq claude task`

Task management functionality with dependencies (following existing patterns):

```bash
# Add tasks (executed from repository root)
gwq claude task add -b feature/auth "Authentication system implementation" -p 75
  # → Creates worktree at .worktrees/feature-auth-<task-id>
gwq claude task add -b feature/api "REST API implementation" -p 85 --depends-on auth-system
  # → Creates worktree at .worktrees/feature-api-<task-id>, waits for auth-system
gwq claude task add -f tasks.yaml  # Batch registration from YAML
  # → Creates worktrees for each task with dependency resolution

# Task list (status command pattern)
gwq claude task list

# Output:
# TASK          BRANCH        WORKTREE                    STATUS      PRIORITY   DEPS     DURATION
# ● auth-impl   feature/auth  .worktrees/feature-auth-abc running     75         -        45m
#   api-dev     feature/api   .worktrees/feature-api-def  waiting     85         auth     -
#   bug-fix     bugfix/login  .worktrees/bugfix-login-ghi completed   90         -        2h 15m
#   setup-db    feature/db    .worktrees/feature-db-123   completed   80         -        1h 30m

# Detailed information
gwq claude task list --verbose
gwq claude task list --json
gwq claude task list --csv

# Filter tasks (tree structure maintained)
gwq claude task list --filter running
gwq claude task list --filter waiting
gwq claude task list --priority-min 75

# Real-time monitoring
gwq claude task list --watch

# Specific task details
gwq claude task show auth-impl
gwq claude task show auth  # Pattern matching
gwq claude task show       # Fuzzy finder

# Dependency information (included in tree view)
gwq claude task show auth-impl           # Show detailed task info with dependencies
```

#### `gwq claude worker`

Worker management:

```bash
# Start worker
gwq claude worker start
gwq claude worker start --parallel 3

# Check worker status
gwq claude worker status

# Output:
# Worker Status: Running
# Active Tasks: 2/3
# Queue: 5 pending
# Sessions: 2 running, 1 reviewing

# Stop worker
gwq claude worker stop

# Check configuration
gwq claude worker config
```

#### `gwq claude tmux`

tmux session management (delegates to generic tmux commands):

```bash
# Session list (filtered for Claude sessions)
gwq tmux list --context claude

# Attach to Claude session (pattern matching)
gwq tmux attach claude-auth
gwq tmux attach --context claude  # Fuzzy finder for Claude sessions

# Terminate Claude session
gwq tmux kill claude-auth
gwq tmux kill --context claude --status completed
```

#### `gwq claude review`

Review functionality:

```bash
# Check review results
gwq claude review show auth-impl

# Output:
# Review Summary for auth-impl:
# Status: Completed
# Issues Found: 3 (1 error, 2 warnings)
# Fixes Applied: 2
# 
# Issues:
# [ERROR] auth.go:45 - SQL injection vulnerability
# [WARN]  auth.go:78 - Missing error handling
# [INFO]  auth_test.go:23 - Test coverage improvement

# Review list
gwq claude review list
gwq claude review list --filter error

# Manual review execution
gwq claude review run auth-impl
```

#### `gwq claude start/stop`

Direct execution commands (executed from repository root):

```bash
# Start Claude with structured task (creates worktree)
gwq claude start --task-file task.yaml
  # → Creates worktree and executes Claude inside it

# Quick start with inline context (from repository root)
gwq claude start -b feature/bugfix \
  --objective "Fix all failing tests" \
  --verify "make test" \
  --constraint "Don't modify public APIs"
  # → Creates .worktrees/feature-bugfix-<id>/ and runs Claude there

# Start with existing worktree and dependencies
gwq claude start -b feature/auth \
  --context "$(cat docs/auth-spec.md)" \
  --objective "Complete JWT authentication" \
  --verify "make test" \
  --verify "make security-scan" \
  --depends-on setup-database,user-model \
  --priority 80
  # → Uses existing .worktrees/feature-auth-*/ or creates new one
  # → Waits for dependencies before starting

# List running Claude instances (shows worktree paths)
gwq claude list

# Output:
# TASK         BRANCH        WORKTREE                    STATUS     DURATION
# auth-impl    feature/auth  .worktrees/feature-auth-abc running    45m
# api-dev      feature/api   .worktrees/feature-api-def  running    1h 20m

# Stop Claude
gwq claude stop auth
gwq claude stop --all

# Cleanup specific worktree
gwq claude cleanup auth-impl
  # → Removes .worktrees/feature-auth-abc/
```

## Task Dependency System

### Dependency Resolution Algorithm

```go
type DependencyGraph struct {
    tasks map[string]*Task
    edges map[string][]string  // task_id -> dependencies
}

func (dg *DependencyGraph) GetExecutableTask(availableWorkers int) (*Task, error) {
    // Find tasks that are ready to run (no pending dependencies)
    readyTasks := dg.getReadyTasks()
    
    if len(readyTasks) == 0 {
        return nil, ErrNoExecutableTasks
    }
    
    // Sort by priority (highest first), then by creation time (oldest first)
    sort.Slice(readyTasks, func(i, j int) bool {
        if readyTasks[i].Priority == readyTasks[j].Priority {
            return readyTasks[i].CreatedAt.Before(readyTasks[j].CreatedAt)
        }
        return readyTasks[i].Priority > readyTasks[j].Priority
    })
    
    return readyTasks[0], nil
}

func (dg *DependencyGraph) getReadyTasks() []*Task {
    var ready []*Task
    
    for _, task := range dg.tasks {
        if task.Status != StatusPending {
            continue
        }
        
        // Check if all dependencies are completed
        if dg.areDependenciesCompleted(task) {
            ready = append(ready, task)
        }
    }
    
    return ready
}

func (dg *DependencyGraph) areDependenciesCompleted(task *Task) bool {
    for _, depID := range task.DependsOn {
        depTask, exists := dg.tasks[depID]
        if !exists {
            // Dependency task not found - treat as failed dependency
            return false
        }
        
        switch depTask.Status {
        case StatusCompleted:
            continue // OK
        case StatusFailed:
            // Handle based on dependency policy
            switch task.DependencyPolicy {
            case DependencyPolicyFail:
                task.Status = StatusFailed
                return false
            case DependencyPolicySkip:
                task.Status = StatusSkipped
                return false
            case DependencyPolicyWait:
                return false // Keep waiting
            }
        default:
            return false // Still pending/running
        }
    }
    
    return true
}

// Detect circular dependencies using depth-first search
func (dg *DependencyGraph) ValidateDependencies() error {
    visited := make(map[string]bool)
    recursionStack := make(map[string]bool)
    
    for taskID := range dg.tasks {
        if !visited[taskID] {
            if dg.hasCycle(taskID, visited, recursionStack) {
                return fmt.Errorf("circular dependency detected involving task: %s", taskID)
            }
        }
    }
    
    return nil
}

func (dg *DependencyGraph) hasCycle(taskID string, visited, recursionStack map[string]bool) bool {
    visited[taskID] = true
    recursionStack[taskID] = true
    
    task := dg.tasks[taskID]
    for _, depID := range task.DependsOn {
        if !visited[depID] {
            if dg.hasCycle(depID, visited, recursionStack) {
                return true
            }
        } else if recursionStack[depID] {
            return true
        }
    }
    
    recursionStack[taskID] = false
    return false
}
```

### Enhanced Task Status

```go
type Status string

const (
    StatusPending     Status = "pending"     // Task is queued
    StatusWaiting     Status = "waiting"     // Waiting for dependencies
    StatusRunning     Status = "running"     // Currently executing
    StatusReviewing   Status = "reviewing"   // Under review
    StatusCompleted   Status = "completed"   // Successfully finished
    StatusFailed      Status = "failed"      // Execution failed
    StatusSkipped     Status = "skipped"     // Skipped due to dependency policy
    StatusCancelled   Status = "cancelled"   // Manually cancelled
)
```

## Automatic Review Feature

### Review Flow

```mermaid
sequenceDiagram
    participant TM as Task Manager
    participant CA as Claude Agent
    participant RA as Review Agent
    participant SM as Session Manager

    TM->>CA: Execute Development Task
    CA->>SM: Create Session
    CA->>CA: Run Claude Code
    CA-->>TM: Task Completed
    
    TM->>RA: Start Review
    RA->>SM: Create Review Session
    RA->>RA: Run Review Claude
    RA-->>TM: Review Completed
    
    TM->>TM: Apply Fixes (if any)
    TM-->>TM: Task Finalized
```

### Review Configuration

```toml
[claude.review]
# Enable automatic review
enabled = true

# Review targets
review_patterns = ["*.go", "*.js", "*.ts", "*.py"]
exclude_patterns = ["*_test.go", "vendor/*"]

# Review prompt
review_prompt = """
Please focus on reviewing the following points:
1. Security vulnerabilities
2. Bugs and potential issues
3. Performance concerns
4. Code readability
5. Missing tests

If fixes are needed, please apply them automatically.
"""

# Automatic fixes
auto_fix = true
max_fix_attempts = 3
```

## Parallelism Control and Resource Management

### Global Parallelism Control

```go
type ResourceManager struct {
    maxClaude          int
    maxDevelopment     int  
    maxReview         int
    activeDevelopment int
    activeReview      int
    semaphore         *Semaphore
}

func (r *ResourceManager) AcquireSlot(taskType TaskType) (*Slot, error) {
    switch taskType {
    case TaskTypeDevelopment:
        if r.activeDevelopment >= r.maxDevelopment {
            return nil, ErrSlotUnavailable
        }
    case TaskTypeReview:
        if r.activeReview >= r.maxReview {
            return nil, ErrSlotUnavailable
        }
    }
    return r.semaphore.Acquire(), nil
}
```

### Configuration

```toml
[claude]
# Claude Code executable
executable = "claude"
default_args = []

# Global parallelism control
max_parallel = 5
max_development_tasks = 3
max_review_tasks = 2

# Resource limits
max_cpu_percent = 80
max_memory_mb = 4096
task_timeout = "2h"

[claude.queue]
# Queue management
max_queue_size = 100
queue_dir = "~/.gwq/claude/queue"

# Priority processing
priority_boost_after = "1h"               # Boost priority of waiting tasks
starvation_prevention = true               # Prevent low priority tasks from starving

# Dependency processing
dependency_timeout = "30m"                 # Max time to wait for dependencies
max_dependency_depth = 5                   # Max depth of dependency chains
validate_dependencies = true               # Validate dependency graph on task creation

# Task validation
validate_task_files = true
required_fields = ["objectives", "instructions", "verification_commands"]

[claude.tmux]
# tmux session configuration
auto_create_session = true
session_prefix = "gwq-claude"
history_limit = 50000
```

## Agent Abstraction Design

### Agent Interface

```go
// Agent abstraction for future extensions
type Agent interface {
    // Basic information
    Name() string
    Version() string
    Capabilities() []Capability
    
    // Task execution
    Execute(ctx context.Context, task *Task) (*TaskResult, error)
    Review(ctx context.Context, task *Task) (*ReviewResult, error)
    
    // Health check
    HealthCheck() error
    IsAvailable() bool
    
    // Session management
    CreateSession(task *Task) (*Session, error)
    AttachSession(sessionID string) error
}

type Capability string

const (
    CapabilityCodeGeneration Capability = "code_generation"
    CapabilityCodeReview     Capability = "code_review"
    CapabilityTesting        Capability = "testing"
    CapabilityRefactoring    Capability = "refactoring"
    CapabilityDocumentation  Capability = "documentation"
)

// Claude Code implementation
type ClaudeAgent struct {
    config      *ClaudeConfig
    sessionMgr  *SessionManager
    resourceMgr *ResourceManager
}

func (c *ClaudeAgent) Name() string { return "claude" }

func (c *ClaudeAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // Ensure worktree exists for the task
    if err := c.ensureWorktree(task); err != nil {
        return nil, fmt.Errorf("failed to prepare worktree: %w", err)
    }
    
    // Build Claude Code command with automation flags
    cmd := c.buildCommand(task)
    
    // Create tmux session for persistent execution in worktree
    session, err := c.sessionMgr.CreateSession(ctx, SessionOptions{
        Context:    "claude",
        Identifier: fmt.Sprintf("%s-%s", task.Branch, task.ID),
        WorkingDir: task.WorktreePath,  // Execute Claude inside worktree
        Command:    cmd,
        Metadata: map[string]string{
            "task_id":      task.ID,
            "task_name":    task.Name,
            "branch":       task.Branch,
            "worktree":     task.WorktreePath,
            "repo_root":    task.RepositoryRoot,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    
    // Monitor execution and handle results
    result, err := c.monitorExecution(ctx, session, task)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

func (c *ClaudeAgent) ensureWorktree(task *Task) error {
    // Check if worktree already exists
    if _, err := os.Stat(task.WorktreePath); err == nil {
        return nil // Worktree already exists
    }
    
    // Create new worktree from repository root
    cmd := exec.Command("git", "worktree", "add", task.WorktreePath, task.Branch)
    cmd.Dir = task.RepositoryRoot
    
    if err := cmd.Run(); err != nil {
        // If branch doesn't exist, create it
        createBranchCmd := exec.Command("git", "worktree", "add", "-b", task.Branch, task.WorktreePath)
        createBranchCmd.Dir = task.RepositoryRoot
        if err := createBranchCmd.Run(); err != nil {
            return fmt.Errorf("failed to create worktree with new branch: %w", err)
        }
    }
    
    return nil
}

func (c *ClaudeAgent) buildCommand(task *Task) string {
    args := []string{
        c.config.Executable,
        "--dangerously-skip-permissions",
        "--json-output",
    }
    
    // Add timeout if specified
    if task.Config.Timeout != "" {
        args = append(args, "--timeout", task.Config.Timeout)
    }
    
    // Add max iterations
    if task.Config.MaxIterations > 0 {
        args = append(args, "--max-iterations", strconv.Itoa(task.Config.MaxIterations))
    }
    
    // Add task context as a comprehensive prompt
    prompt := c.buildTaskPrompt(task)
    args = append(args, "--task", prompt)
    
    return strings.Join(args, " ")
}

func (c *ClaudeAgent) buildTaskPrompt(task *Task) string {
    var prompt strings.Builder
    
    prompt.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))
    
    if task.Context != "" {
        prompt.WriteString(fmt.Sprintf("## Context\n%s\n\n", task.Context))
    }
    
    if len(task.Objectives) > 0 {
        prompt.WriteString("## Objectives\n")
        for _, obj := range task.Objectives {
            prompt.WriteString(fmt.Sprintf("- %s\n", obj))
        }
        prompt.WriteString("\n")
    }
    
    if task.Instructions != "" {
        prompt.WriteString(fmt.Sprintf("## Instructions\n%s\n\n", task.Instructions))
    }
    
    if len(task.Constraints) > 0 {
        prompt.WriteString("## Constraints\n")
        for _, constraint := range task.Constraints {
            prompt.WriteString(fmt.Sprintf("- %s\n", constraint))
        }
        prompt.WriteString("\n")
    }
    
    if len(task.FilesToFocus) > 0 {
        prompt.WriteString("## Files to Focus On\n")
        for _, file := range task.FilesToFocus {
            prompt.WriteString(fmt.Sprintf("- %s\n", file))
        }
        prompt.WriteString("\n")
    }
    
    if len(task.VerificationCommands) > 0 {
        prompt.WriteString("## Verification Commands\n")
        prompt.WriteString("Please run these commands to verify your work:\n")
        for _, cmd := range task.VerificationCommands {
            prompt.WriteString(fmt.Sprintf("- `%s`\n", cmd))
        }
        prompt.WriteString("\n")
    }
    
    prompt.WriteString("## Success Criteria\n")
    prompt.WriteString("Task is complete when:\n")
    prompt.WriteString("- All objectives are met\n")
    prompt.WriteString("- All verification commands pass\n")
    prompt.WriteString("- Code follows project conventions\n")
    prompt.WriteString("- No security issues introduced\n")
    
    return prompt.String()
}

func (c *ClaudeAgent) Capabilities() []Capability {
    return []Capability{
        CapabilityCodeGeneration,
        CapabilityCodeReview,
        CapabilityTesting,
        CapabilityRefactoring,
        CapabilityDocumentation,
    }
}
```

### Future Extension Examples

```bash
# Future support for other agents with dependencies
gwq cursor task add -b feature/ui "UI implementation" --depends-on api-endpoints -p 60
gwq copilot task add -b feature/api "API implementation" --depends-on auth-system -p 70

# Agent-specific features with priority
gwq claude review run task-123 -p 80
gwq cursor pair-programming start --priority 85
gwq copilot suggest improvements --depends-on code-review

# Integrated view with dependencies
gwq agent list --with-deps
gwq agent status --all --show-priority
gwq agent deps --graph --all-agents
```

## Integration and Workflow

### Integration with Existing Commands

```bash
# Integration with existing gwq commands (executed from repository root)

# Create worktree and Claude task together
gwq add -b feature/auth --with-claude-task "Authentication system implementation"
  # → Creates worktree AND registers Claude task

# Display Claude information in status command
gwq status --verbose

# Output:
# BRANCH            STATUS       CHANGES        ACTIVITY      CLAUDE TASKS
# ● main            clean        -             2 hours ago   -
#   feature/auth    changed      5 added, 3 mod running       auth-impl (.worktrees/feature-auth-abc)
#   feature/api     clean        -             pending       api-dev (queued)
# 
# Claude Worktrees:
# .worktrees/feature-auth-abc123   running   auth-impl        1h 25m
# .worktrees/feature-api-def456    pending   api-dev          -
```

### Task File Format

```yaml
# tasks.yaml (executed from repository root)
version: "1.0"
default_config:
  skip_permissions: true
  timeout: "2h"
  auto_review: true
  max_iterations: 3
  worktree_base: ".worktrees"        # Relative to repository root
  dependency_policy: "wait"          # Default dependency handling
  priority: 50                       # Default priority (normal)

tasks:
  - id: "setup-database"
    name: "Database Schema Setup"
    branch: "feature/database"
    priority: 80                        # High priority - foundational
    
    # No dependencies - can run immediately
    
  - id: "auth-system-impl"
    name: "Authentication System Implementation"
    branch: "feature/auth"              # Target branch for worktree
    priority: 75                        # High priority
    depends_on: ["setup-database"]      # Wait for database setup
    dependency_policy: "fail"           # Fail if database setup fails
    
    # Worktree will be created at: .worktrees/feature-auth-auth-system-impl/
    
  - id: "api-endpoints"
    name: "REST API Implementation"
    branch: "feature/api"
    priority: 70                        # Lower than auth system
    depends_on: ["auth-system-impl"]    # Wait for auth to complete
    dependency_policy: "wait"           # Wait even if auth has issues
    
  - id: "frontend-integration"
    name: "Frontend API Integration"
    branch: "feature/frontend"
    priority: 65
    depends_on: ["api-endpoints", "auth-system-impl"]  # Multiple dependencies
    dependency_policy: "skip"           # Skip if either dependency fails
    
    # Clear context and objectives
    context: |
      We need to implement a JWT-based authentication system for our web application.
      The existing codebase uses Go with Gin framework and PostgreSQL database.
      
    objectives:
      - "Implement secure JWT token generation and validation"
      - "Create login/logout endpoints with proper error handling"
      - "Add middleware for route protection"
      - "Write comprehensive tests with >90% coverage"
      - "Update API documentation"
    
    # Specific instructions and constraints
    instructions: |
      Please implement the authentication system following these guidelines:
      
      1. **Security Requirements:**
         - Use RS256 for JWT signing
         - Implement proper password hashing with bcrypt
         - Add rate limiting for login attempts
         - Include CSRF protection
      
      2. **Code Structure:**
         - Follow existing project patterns in `internal/auth/`
         - Use the existing database connection pool
         - Implement proper error handling with structured logging
      
      3. **Testing:**
         - Write unit tests for all auth functions
         - Add integration tests for endpoints
         - Include edge cases and security scenarios
    
    constraints:
      - "Do not modify existing user table schema"
      - "Maintain backward compatibility with current session handling"
      - "Follow OWASP security guidelines"
    
    files_to_focus:
      - "internal/auth/**"
      - "internal/middleware/**"
      - "cmd/server/routes.go"
      - "docs/api.md"
    
    verification_commands:
      - "make test"
      - "make lint"
      - "make security-check"
      - "go mod tidy"
    
    config:
      timeout: "3h"
      max_iterations: 5
```

## Claude Code Integration

### Claude Agent Implementation

```go
type ClaudeAgent struct {
    config      *ClaudeConfig
    sessionMgr  *SessionManager
}

type ClaudeConfig struct {
    Executable           string   `toml:"executable"`
    SkipPermissions     bool     `toml:"skip_permissions"`      // Always true for automation
    Timeout             string   `toml:"timeout"`
    MaxIterations       int      `toml:"max_iterations"`
    WorktreeBase        string   `toml:"worktree_base"`
    AdditionalArgs      []string `toml:"additional_args"`
}

func (c *ClaudeAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // Ensure worktree exists for the task
    if err := c.ensureWorktree(task); err != nil {
        return nil, fmt.Errorf("failed to prepare worktree: %w", err)
    }
    
    // Build Claude Code command with automation-friendly options
    cmd := c.buildClaudeCommand(task)
    
    // Create tmux session for persistent execution in worktree
    session, err := c.sessionMgr.CreateSession(ctx, SessionOptions{
        Context:    "claude",
        Identifier: fmt.Sprintf("%s-%s", task.Branch, task.ID),
        WorkingDir: task.WorktreePath,  // Execute Claude inside worktree
        Command:    cmd,
        Metadata: map[string]string{
            "task_id":      task.ID,
            "task_name":    task.Name,
            "branch":       task.Branch,
            "worktree":     task.WorktreePath,
            "repo_root":    task.RepositoryRoot,
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    
    // Monitor execution and handle results
    result, err := c.monitorExecution(ctx, session, task)
    if err != nil {
        return nil, err
    }
    
    return result, nil
}

func (c *ClaudeAgent) buildClaudeCommand(task *Task) string {
    // Core automation options (always included)
    args := []string{
        c.config.Executable,
        "--dangerously-skip-permissions",  // REQUIRED for automation
    }
    
    // Add task-specific options
    if task.Config.Timeout != "" {
        args = append(args, "--timeout", task.Config.Timeout)
    } else if c.config.Timeout != "" {
        args = append(args, "--timeout", c.config.Timeout)
    }
    
    if task.Config.MaxIterations > 0 {
        args = append(args, "--max-iterations", strconv.Itoa(task.Config.MaxIterations))
    } else if c.config.MaxIterations > 0 {
        args = append(args, "--max-iterations", strconv.Itoa(c.config.MaxIterations))
    }
    
    // Add any additional configured arguments
    args = append(args, c.config.AdditionalArgs...)
    
    // Build comprehensive task prompt
    prompt := c.buildTaskPrompt(task)
    args = append(args, "--task", fmt.Sprintf("\"%s\"", prompt))
    
    return strings.Join(args, " ")
}

func (c *ClaudeAgent) buildTaskPrompt(task *Task) string {
    var prompt strings.Builder
    
    prompt.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))
    
    if task.Context != "" {
        prompt.WriteString(fmt.Sprintf("## Context\n%s\n\n", task.Context))
    }
    
    if len(task.Objectives) > 0 {
        prompt.WriteString("## Objectives\n")
        for _, obj := range task.Objectives {
            prompt.WriteString(fmt.Sprintf("- %s\n", obj))
        }
        prompt.WriteString("\n")
    }
    
    if task.Instructions != "" {
        prompt.WriteString(fmt.Sprintf("## Instructions\n%s\n\n", task.Instructions))
    }
    
    if len(task.Constraints) > 0 {
        prompt.WriteString("## Constraints\n")
        for _, constraint := range task.Constraints {
            prompt.WriteString(fmt.Sprintf("- %s\n", constraint))
        }
        prompt.WriteString("\n")
    }
    
    if len(task.FilesToFocus) > 0 {
        prompt.WriteString("## Files to Focus On\n")
        for _, file := range task.FilesToFocus {
            prompt.WriteString(fmt.Sprintf("- %s\n", file))
        }
        prompt.WriteString("\n")
    }
    
    if len(task.VerificationCommands) > 0 {
        prompt.WriteString("## Verification Commands\n")
        prompt.WriteString("Please run these commands to verify your work:\n")
        for _, cmd := range task.VerificationCommands {
            prompt.WriteString(fmt.Sprintf("- `%s`\n", cmd))
        }
        prompt.WriteString("\n")
    }
    
    prompt.WriteString("## Success Criteria\n")
    prompt.WriteString("Task is complete when:\n")
    prompt.WriteString("- All objectives are met\n")
    prompt.WriteString("- All verification commands pass\n")
    prompt.WriteString("- Code follows project conventions\n")
    prompt.WriteString("- No security issues introduced\n")
    
    return prompt.String()
}

func (c *ClaudeAgent) ensureWorktree(task *Task) error {
    // Check if worktree already exists
    if _, err := os.Stat(task.WorktreePath); err == nil {
        return nil // Worktree already exists
    }
    
    // Create new worktree from repository root
    cmd := exec.Command("git", "worktree", "add", task.WorktreePath, task.Branch)
    cmd.Dir = task.RepositoryRoot
    
    if err := cmd.Run(); err != nil {
        // If branch doesn't exist, create it
        createBranchCmd := exec.Command("git", "worktree", "add", "-b", task.Branch, task.WorktreePath)
        createBranchCmd.Dir = task.RepositoryRoot
        if err := createBranchCmd.Run(); err != nil {
            return fmt.Errorf("failed to create worktree with new branch: %w", err)
        }
    }
    
    return nil
}
```

## Configuration

### Claude Code Configuration

```toml
[claude]
# Claude Code executable and core options
executable = "claude"
skip_permissions = true                # ALWAYS true for automation
timeout = "2h"                         # Default timeout
max_iterations = 3                     # Default max iterations
worktree_base = ".worktrees"           # Relative to repository root

# Additional arguments passed to Claude Code
additional_args = [
    "--json-output",                   # For structured result parsing (optional)
]

# Global parallelism control
max_parallel = 3
max_development_tasks = 2
max_review_tasks = 1

# Resource limits
max_cpu_percent = 80
max_memory_mb = 4096

[claude.worktree]
# Git worktree settings
auto_create_worktree = true
worktree_naming = "{branch}-{task_id}" 
cleanup_on_completion = false
require_clean_repo = false
validate_branch_exists = true

[claude.queue]
# Task queue management
max_queue_size = 50
queue_dir = "~/.gwq/claude/queue"

# Priority processing
priority_boost_after = "1h"           # Boost priority of waiting tasks
starvation_prevention = true           # Prevent low priority tasks from starving

# Dependency processing
dependency_timeout = "30m"             # Max time to wait for dependencies
max_dependency_depth = 5               # Max depth of dependency chains
validate_dependencies = true           # Validate dependency graph on task creation

# Task validation
validate_task_files = true
required_fields = ["objectives", "instructions", "verification_commands"]

[claude.tmux]
# tmux session configuration
auto_create_session = true
session_prefix = "gwq-claude"
history_limit = 50000

[claude.review]
# Automatic review configuration
enabled = true
auto_fix = true
max_fix_attempts = 2
```

### Key Configuration Notes

- **`skip_permissions = true`**: REQUIRED - always passes `--dangerously-skip-permissions`
- **`executable = "claude"`**: Path to Claude Code executable
- **`additional_args`**: Extra arguments for specific needs (optional)
- **Worktree isolation**: Each task gets dedicated worktree
- **Dependency limits**: Max 5-level depth prevents complex chains

## Usage Examples

### Daily Development Flow

```bash
# Execute ALL commands from repository root
cd /path/to/your/project

# Morning work preparation - load structured tasks
gwq claude task add -f daily-tasks.yaml
  # → Creates worktrees for each task under .worktrees/

# Or add individual tasks with context
gwq claude task add -b feature/auth \
  --name "Authentication System" \
  --context "$(cat docs/auth-requirements.md)" \
  --verify "make test" \
  --verify "make security-check" \
  --depends-on setup-database \
  --priority 75
  # → Creates .worktrees/feature-auth-<id>/ (waits for setup-database)

# Start worker with dependency resolution (from repository root)
gwq claude worker start --parallel 2
  # → Worker manages worktrees, resolves dependencies, and executes Claude in each

# Check work status (shows worktree paths and dependencies)
gwq claude task list --watch

# Check dependency status
gwq claude task deps --status
  # → Shows dependency graph and current blocking relationships

# Visualize dependency graph
gwq claude task deps --graph --format svg > task-deps.svg

# Monitor sessions (shows worktree info)
gwq claude tmux list

# Check worktree status integration
gwq status --verbose
  # Shows both regular worktrees and Claude task worktrees

# Evening, check review results
gwq claude review list
gwq claude review show auth --verbose

# Next morning, check completed tasks
gwq claude task list --filter completed

# Cleanup completed task worktrees (respects dependencies)
gwq claude task cleanup --completed
  # → Removes .worktrees/feature-*-<completed-ids>/ safely

# Cleanup with dependency checking
gwq claude task cleanup --safe
  # → Only removes worktrees if no pending dependents exist
```

### Error Handling Flow

```bash
# Check failed tasks and their impact on dependencies
gwq claude task list --filter failed
gwq claude task deps --blocked-by failed-task-id

# Check session directly for debugging
gwq claude tmux attach auth-impl

# Retry failed task (may unblock dependent tasks)
gwq claude task retry auth-impl

# Force execution ignoring dependencies (emergency override)
gwq claude task start auth-impl --ignore-dependencies

# Cancel task and all dependents
gwq claude task cancel auth-impl --cascade

# Show dependency chain for debugging
gwq claude task deps auth-impl --trace
```

## Benefits

1. **Git Worktree Integration**: Automatic worktree management for task isolation
2. **Repository Root Execution**: Consistent execution from git repository root
3. **Tree Structure Visualization**: Clear dependency relationships in task list
4. **Numeric Priority System**: Fine-grained priority control (1-100 scale)
5. **Simple Dependency Management**: Essential dependency resolution without complexity
6. **Effective Time Utilization**: Automated development during sleep with structured tasks
7. **Quality Assurance**: Quality improvement through automatic reviews and verification
8. **Best Practice Integration**: Follows Claude Code automation best practices
9. **Structured Task Definition**: Clear objectives, constraints, and success criteria
10. **Automated Verification**: Built-in command execution for validation
11. **Efficient Parallel Processing**: Maximum resource utilization with isolated worktrees
12. **Clean UX**: Focused on essential features without feature bloat
13. **Integrated Management**: Complete functionality in a single command system
14. **Extensibility**: Future support for other agents
15. **UX Consistency**: Maintains gwq usability and patterns

## Limitations

1. Requires Claude Code execution environment
2. Requires tmux installation
3. Requires git repository with worktree support
4. Requires execution from repository root
5. Requires network connection
6. Important to manage resource usage and worktree disk space
7. Worktree cleanup may be needed for long-running systems
8. Deep dependency chains (>5 levels) may impact performance

## Summary

This design enables gwq to function as a git worktree-based Claude Code automated development platform with sophisticated dependency management, effectively utilizing developer idle time. The worktree-first approach ensures task isolation and maintains clean repository state, while execution from repository root provides consistent behavior. The numeric priority system (1-100) and dependency resolution enable complex workflows with proper sequencing. The design considers future extensibility and can support other AI agents.

## Key Workflow

1. **Repository Root**: Execute all `gwq claude` commands from git repository root
2. **Dependency Analysis**: Validate dependency graph and detect circular dependencies
3. **Worktree Creation**: Automatic worktree creation at `.worktrees/{branch}-{task-id}/`
4. **Dependency Resolution**: Wait for required tasks before execution
5. **Task Execution**: Claude Code runs inside isolated worktree environment
6. **Verification**: Commands executed within worktree context
7. **Session Management**: tmux sessions track worktree-based execution
8. **Dependency Updates**: Real-time dependency status monitoring
9. **Cleanup**: Safe cleanup respecting dependency relationships

## Related Documentation

- [DESIGN_TMUX_SESSION.md](./DESIGN_TMUX_SESSION.md) - Generic tmux session management foundation
