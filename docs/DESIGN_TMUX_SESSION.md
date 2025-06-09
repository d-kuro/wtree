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
    HistorySize int       `json:"history_size"`
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
    
    // Create tmux session with increased history limit
    tmuxCmd := fmt.Sprintf("tmux new-session -d -s %s -c %s", sessionName, worktreePath)
    if err := exec.Command("sh", "-c", tmuxCmd).Run(); err != nil {
        return nil, err
    }
    
    // Set history limit for this session
    historyCmd := fmt.Sprintf("tmux set-option -t %s history-limit %d", sessionName, s.config.HistoryLimit)
    exec.Command("sh", "-c", historyCmd).Run()
    
    // Execute Claude Code in the session
    claudeCmd := fmt.Sprintf("tmux send-keys -t %s '%s' Enter", sessionName, command)
    exec.Command("sh", "-c", claudeCmd).Run()
    
    session := &ClaudeSession{
        ID:           generateID(),
        SessionName:  sessionName,
        TaskID:       taskID,
        WorktreePath: worktreePath,
        Command:      command,
        StartTime:    time.Now(),
        Status:       StatusRunning,
        HistorySize:  s.config.HistoryLimit,
    }
    
    return session, nil
}
```

### tmux History Management

Uses tmux's built-in history and buffer functionality instead of custom logging:

```go
type SessionManager struct {
    config    *SessionConfig
    tmuxCmd   *TmuxCommand
}

func (s *SessionManager) CaptureHistory(sessionName string, lines int) ([]string, error) {
    // Use tmux capture-pane to get session history
    cmd := fmt.Sprintf("tmux capture-pane -t %s -p -S -%d", sessionName, lines)
    output, err := exec.Command("sh", "-c", cmd).Output()
    if err != nil {
        return nil, err
    }
    return strings.Split(string(output), "\n"), nil
}

func (s *SessionManager) SaveHistory(sessionName, filename string) error {
    // Save current buffer to file using tmux save-buffer
    cmd := fmt.Sprintf("tmux capture-pane -t %s && tmux save-buffer %s", sessionName, filename)
    return exec.Command("sh", "-c", cmd).Run()
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

## tmux History Features

### History Configuration

tmux provides built-in history management that is more efficient than custom logging:

```bash
# Set global history limit in tmux.conf
set-option -g history-limit 50000

# Per-session history limit (set by gwq)
tmux set-option -t session-name history-limit 50000

# Capture session history
tmux capture-pane -t session-name -p -S -1000

# Save captured content to file
tmux save-buffer ~/session-history.txt
```

### History Search

Use tmux's built-in search functionality:

```bash
# Enter copy mode and search (within tmux session)
Ctrl+B [         # Enter copy mode
Ctrl+S           # Search forward
Ctrl+R           # Search backward

# Capture and search outside tmux
tmux capture-pane -t auth-session -p -S -1000 | grep "error"
tmux capture-pane -t auth-session && tmux save-buffer /tmp/history.txt
grep "authentication" /tmp/history.txt
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

[session.history]
# tmux history configuration
history_limit = 50000
history_auto_save = true
history_save_dir = "~/.gwq/history"
```

## Usage Examples

### Basic Usage Flow

```bash
# Execute task (auto session creation)
gwq task add -b feature/auth "Authentication system implementation"

# Check session status (status command pattern)
gwq session list
gwq status --verbose  # includes session information

# Attach to session to check progress
gwq session attach auth

# Detach from session (Ctrl+B, D)
# → Claude Code continues execution

# Next morning, check results
gwq session list --filter completed
# Use tmux history to check output
tmux capture-pane -t gwq-claude-auth-* -p -S -100
```

### History Analysis Examples

```bash
# Capture and search session history for errors
tmux capture-pane -t session-name -p -S -1000 | grep "error\|failed\|exception"

# Track changes related to specific files
tmux capture-pane -t session-name -p -S -1000 | grep "auth.go"

# Identify long-running tasks
gwq session list --sort duration

# Display only running sessions
gwq session list --filter running
```

## Benefits

1. **Process Persistence**: Claude Code continues execution even after terminal disconnection
2. **History Recording**: All output is automatically saved in tmux history
3. **Debugging Support**: tmux history search and analysis capabilities
4. **Monitoring Functionality**: Detailed understanding of execution status
5. **Simple Design**: Lightweight implementation using native tmux features

## Limitations

1. Works only in environments where tmux is installed
2. Resource consumption increases as the number of sessions grows
3. tmux history is limited by configured history-limit

## Summary

This tmux session management feature significantly improves the stability and monitorability of Claude Code execution. By focusing purely on process management and using tmux's native history features without complex layout management, it provides a simple and reliable system.