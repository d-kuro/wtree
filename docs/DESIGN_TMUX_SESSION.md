# tmux Session Management Design

## Overview

Design for tmux integration focused on session management and log saving during Claude Code execution. This design focuses on Claude Code process management and log recording without complex layout management.

## Core Concepts

### Session Management Goals

1. **Process Persistence**: Make Claude Code execution independent of terminal connections
2. **Log Preservation**: Automatically record all Claude Code output
3. **Monitoring**: Monitor the status of running Claude Code instances
4. **Detach/Attach**: Continue processing even when terminal is closed

### Session Naming Convention

```
gwq-claude-{task-id}-{timestamp}
```

Examples:
- `gwq-claude-abc123-20240115103045`
- `gwq-claude-def456-20240115110230`

## Architecture

### tmux Session Manager

```go
type SessionManager struct {
    config    *SessionConfig
    tmuxCmd   *TmuxCommand
    logger    *SessionLogger
}

type ClaudeSession struct {
    ID          string    `json:"id"`
    SessionName string    `json:"session_name"`
    TaskID      string    `json:"task_id"`
    WorktreePath string   `json:"worktree_path"`
    Command     string    `json:"command"`
    PID         int       `json:"pid"`
    StartTime   time.Time `json:"start_time"`
    Status      Status    `json:"status"`
    LogFile     string    `json:"log_file"`
}

type Status string

const (
    StatusRunning   Status = "running"
    StatusCompleted Status = "completed"
    StatusFailed    Status = "failed"
    StatusDetached  Status = "detached"
)
```

## Core Features

### Session Creation

Automatically create tmux sessions when executing Claude Code:

```go
func (s *SessionManager) CreateSession(taskID, worktreePath, command string) (*ClaudeSession, error) {
    sessionName := fmt.Sprintf("gwq-claude-%s-%s", taskID, time.Now().Format("20060102150405"))
    logFile := filepath.Join(s.config.LogDir, fmt.Sprintf("%s.log", sessionName))
    
    // Create tmux session
    tmuxCmd := fmt.Sprintf("tmux new-session -d -s %s -c %s", sessionName, worktreePath)
    
    // Execute Claude Code with logging
    claudeCmd := fmt.Sprintf("%s 2>&1 | tee %s", command, logFile)
    
    session := &ClaudeSession{
        ID:           generateID(),
        SessionName:  sessionName,
        TaskID:       taskID,
        WorktreePath: worktreePath,
        Command:      command,
        StartTime:    time.Now(),
        Status:       StatusRunning,
        LogFile:      logFile,
    }
    
    return session, nil
}
```

### Log Management

```go
type SessionLogger struct {
    baseDir string
}

func (l *SessionLogger) CreateLogFile(sessionName string) (string, error) {
    logFile := filepath.Join(l.baseDir, fmt.Sprintf("%s.log", sessionName))
    
    // Create log file and setup rotation
    file, err := os.Create(logFile)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    return logFile, nil
}

func (l *SessionLogger) TailLog(sessionName string, lines int) ([]string, error) {
    logFile := filepath.Join(l.baseDir, fmt.Sprintf("%s.log", sessionName))
    // tail implementation
}
```

## Command Design

### gwq session subcommands

#### `gwq session list`

List running Claude Code sessions (following existing status command patterns):

```bash
# Session list (simple table format)
gwq session list

# Output:
# TASK          WORKTREE        STATUS     DURATION
# ● auth-impl   feature/auth    running    1h 25m
#   api-dev     feature/api     running    45m
#   auth-review review/auth     completed  2h 15m

# Detailed information
gwq session list --verbose

# JSON output
gwq session list --json

# CSV output
gwq session list --csv

# Real-time monitoring
gwq session list --watch

# Status filter
gwq session list --filter running
gwq session list --filter completed

# Sort
gwq session list --sort duration
gwq session list --sort task
```

#### `gwq session attach`

Attach to sessions (following existing get/exec patterns):

```bash
# Attach with pattern matching
gwq session attach auth

# Attach with exact match
gwq session attach auth-impl

# Auto fuzzy finder when multiple matches
gwq session attach feature  # when feature/* matches

# Fuzzy finder selection for all sessions without arguments
gwq session attach

# Explicit fuzzy finder usage
gwq session attach -i
```

#### `gwq session logs`

Display logs:

```bash
# Display logs with pattern matching
gwq session logs auth

# Auto fuzzy finder when multiple matches
gwq session logs feature

# Fuzzy finder selection without arguments
gwq session logs

# Real-time logs (equivalent to tail)
gwq session logs auth -f
gwq session logs auth --follow

# Display last N lines
gwq session logs auth --tail 100

# Display all logs (no line limit)
gwq session logs auth --all
```

#### `gwq session kill`

Terminate sessions (following existing remove patterns):

```bash
# Terminate with pattern matching
gwq session kill auth

# Auto fuzzy finder when multiple matches
gwq session kill feature

# Fuzzy finder selection without arguments
gwq session kill

# Explicit fuzzy finder usage
gwq session kill -i

# Terminate all sessions (with confirmation)
gwq session kill --all

# Cleanup only completed sessions
gwq session kill --completed
```

## Task Queue Integration

### Automatic Session Creation During Task Execution

```go
func (w *Worker) executeTaskWithSession(task *Task) error {
    // Create session
    session, err := w.sessionManager.CreateSession(
        task.ID,
        task.WorktreePath,
        w.buildClaudeCommand(task),
    )
    if err != nil {
        return err
    }
    
    // Record session information in task
    task.SessionID = session.ID
    task.SessionName = session.SessionName
    
    // Monitor session
    go w.monitorSession(session)
    
    return nil
}
```

### Integration with Task Status

Extend existing status command to integrate session information:

```bash
# Add session information to existing status command
gwq status --verbose

# Output:
# BRANCH          STATUS       CHANGES           ACTIVITY     SESSION
# ● main          up to date   -                2 hours ago  -
#   feature/auth  changed      5 added, 3 mod   running      auth-impl
#   feature/api   changed      12 added, 8 mod  running      api-dev
#   review/auth   clean        -                completed    auth-review

# Filter session information only
gwq status --filter session
gwq status --filter "no session"

# Check session information in task command
gwq task list --verbose

# Output:
# TASK         BRANCH        STATUS     SESSION      DURATION
# auth-impl    feature/auth  running    attached     1h 25m
# api-dev      feature/api   running    attached     45m
# auth-review  review/auth   completed  detached     2h 15m
```

## Log Management Features

### Log File Structure

```
~/.gwq/logs/sessions/
├── gwq-claude-abc123-20240115103045.log
├── gwq-claude-def456-20240115110230.log
└── gwq-claude-ghi789-20240115120015.log
```

### Log Rotation

```toml
[session.logging]
# Log directory
log_dir = "~/.gwq/logs/sessions"

# Log rotation
max_log_files = 100
log_retention_days = 30
max_log_size_mb = 100

# Log level
log_level = "info"
```

### Log Search

Following existing grep patterns for log search:

```bash
# Search keywords across all sessions
gwq session logs --grep "error"

# Pattern search in specific session
gwq session logs auth --grep "authentication.*failed"

# Multiple keyword search
gwq session logs --grep "error|failed|exception"

# Time range specification
gwq session logs auth --since "1h"
gwq session logs auth --since "2024-01-15 10:00"

# Log level filter
gwq session logs auth --filter error
gwq session logs auth --filter warn
```

## Configuration

### tmux Session Configuration

```toml
[session]
# Enable tmux integration
enabled = true

# Auto session creation
auto_create_session = true

# Behavior on session creation
detach_on_create = true
auto_cleanup_completed = true

[session.tmux]
# tmux configuration
tmux_command = "tmux"
default_shell = "/bin/bash"

# Session configuration
session_timeout = "24h"
keep_alive = true

[session.logging]
log_dir = "~/.gwq/logs/sessions"
max_log_files = 100
log_retention_days = 30
```

## Usage Examples

### Basic Usage Flow

```bash
# Execute task (auto session creation)
gwq task add -b feature/auth "Authentication system implementation"

# Check session status (status command pattern)
gwq session list
gwq status --verbose  # includes session information

# Check logs (pattern matching)
gwq session logs auth --follow

# Attach to session to check progress
gwq session attach auth

# Detach from session (Ctrl+B, D)
# → Claude Code continues execution

# Next morning, check results
gwq session list --filter completed
gwq session logs auth --tail 50
```

### Log Analysis Examples

```bash
# Identify sessions with errors
gwq session logs --grep "error|failed|exception"

# Track changes related to specific files
gwq session logs --grep "auth.go"

# Identify long-running tasks
gwq session list --sort duration

# Display only running sessions
gwq session list --filter running
```

## Benefits

1. **Process Persistence**: Claude Code continues execution even after terminal disconnection
2. **Complete Log Recording**: All output is automatically saved
3. **Debugging Support**: Log search and analysis capabilities
4. **Monitoring Functionality**: Detailed understanding of execution status
5. **Simple Design**: Lightweight implementation focused on minimal features

## Limitations

1. Works only in environments where tmux is installed
2. Resource consumption increases as the number of sessions grows
3. Disk space management for log files is required

## Summary

This tmux session management feature significantly improves the stability and monitorability of Claude Code execution. By focusing purely on process management and log recording without complex layout management, it provides a simple and reliable system.