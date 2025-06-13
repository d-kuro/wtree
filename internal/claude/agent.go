package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/d-kuro/gwq/internal/tmux"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
)

// ClaudeAgent implements the Agent interface for Claude Code
type ClaudeAgent struct {
	config      *models.ClaudeConfig
	sessionMgr  *tmux.SessionManager
	worktreeMgr *worktree.Manager
}

// NewClaudeAgent creates a new Claude agent
func NewClaudeAgent(config *models.ClaudeConfig, sessionMgr *tmux.SessionManager, worktreeMgr *worktree.Manager) *ClaudeAgent {
	return &ClaudeAgent{
		config:      config,
		sessionMgr:  sessionMgr,
		worktreeMgr: worktreeMgr,
	}
}

// Name returns the agent name
func (c *ClaudeAgent) Name() string {
	return "claude"
}

// Version returns the Claude Code version (if available)
func (c *ClaudeAgent) Version() string {
	cmd := exec.Command(c.config.Executable, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// Capabilities returns the capabilities of the Claude agent
func (c *ClaudeAgent) Capabilities() []Capability {
	return []Capability{
		CapabilityCodeGeneration,
		CapabilityTesting,
		CapabilityRefactoring,
		CapabilityDocumentation,
	}
}

// Execute runs a development task using Claude Code
func (c *ClaudeAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
	startTime := time.Now()

	// Ensure worktree exists for the task
	if err := c.ensureWorktree(task); err != nil {
		return &TaskResult{
			ExitCode: 1,
			Duration: time.Since(startTime),
			Error:    fmt.Sprintf("failed to prepare worktree: %v", err),
		}, err
	}

	// Build Claude Code command
	cmd := c.buildClaudeCommand(task)

	// Create tmux session for persistent execution in worktree
	sessionID, err := c.CreateSession(ctx, task)
	if err != nil {
		return &TaskResult{
			ExitCode: 1,
			Duration: time.Since(startTime),
			Error:    fmt.Sprintf("failed to create session: %v", err),
		}, err
	}

	task.SessionID = sessionID

	// Monitor execution and handle results
	result, err := c.monitorExecution(ctx, sessionID, task, cmd, startTime)
	if err != nil {
		return result, err
	}

	return result, nil
}

// HealthCheck verifies Claude Code is available and working
func (c *ClaudeAgent) HealthCheck() error {
	cmd := exec.Command(c.config.Executable, "--help")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude Code executable not available: %w", err)
	}
	return nil
}

// IsAvailable checks if Claude Code is available
func (c *ClaudeAgent) IsAvailable() bool {
	return c.HealthCheck() == nil
}

// CreateSession creates a tmux session for the task
func (c *ClaudeAgent) CreateSession(ctx context.Context, task *Task) (string, error) {
	cmd := c.buildClaudeCommand(task)

	sessionOpts := tmux.SessionOptions{
		Context:    "claude",
		Identifier: fmt.Sprintf("%s-%s", task.Worktree, task.ID),
		WorkingDir: task.WorktreePath,
		Command:    cmd,
		Metadata: map[string]string{
			"task_id":       task.ID,
			"task_name":     task.Name,
			"worktree":      task.Worktree,
			"worktree_path": task.WorktreePath,
			"repo_root":     task.RepositoryRoot,
			"type":          "development",
		},
	}

	session, err := c.sessionMgr.CreateSession(ctx, sessionOpts)
	if err != nil {
		return "", err
	}

	return session.SessionName, nil
}

// AttachSession attaches to an existing session
func (c *ClaudeAgent) AttachSession(ctx context.Context, sessionID string) error {
	return c.sessionMgr.AttachSession(sessionID)
}

// ensureWorktree ensures the worktree exists for the task
func (c *ClaudeAgent) ensureWorktree(task *Task) error {
	// If no worktree manager is available, skip worktree handling
	if c.worktreeMgr == nil {
		if task.WorktreePath != "" && task.RepositoryRoot != "" {
			// Use provided paths if available
			return nil
		}
		return fmt.Errorf("worktree manager not available")
	}

	// If worktree path is already set, verify it exists
	if task.WorktreePath != "" {
		if _, err := os.Stat(task.WorktreePath); err == nil {
			return nil // Worktree exists
		}
	}

	// Try to find existing worktree
	if task.Worktree == "" {
		return fmt.Errorf("worktree field is required")
	}

	worktreePath, err := c.worktreeMgr.GetWorktreePath(task.Worktree)
	if err != nil {
		return fmt.Errorf("worktree '%s' not found, please create it first using 'gwq add %s': %w", task.Worktree, task.Worktree, err)
	}

	task.WorktreePath = worktreePath
	return nil
}

// buildClaudeCommand builds the Claude Code command for execution
func (c *ClaudeAgent) buildClaudeCommand(task *Task) string {
	// Core automation options (always included)
	args := []string{
		c.config.Executable,
		"--dangerously-skip-permissions", // REQUIRED for automation
		"--print",                        // Non-interactive mode
	}

	// Add any additional configured arguments (only supported Claude options)
	args = append(args, c.config.AdditionalArgs...)

	// Build comprehensive task prompt
	prompt := c.buildTaskPrompt(task)
	args = append(args, prompt)

	return strings.Join(args, " ")
}

// buildTaskPrompt builds a comprehensive prompt for the task
func (c *ClaudeAgent) buildTaskPrompt(task *Task) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))

	if task.Prompt != "" {
		prompt.WriteString(fmt.Sprintf("%s\n\n", task.Prompt))
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

	return fmt.Sprintf(`"%s"`, prompt.String())
}

// monitorExecution monitors the Claude Code execution and returns results
func (c *ClaudeAgent) monitorExecution(ctx context.Context, sessionID string, task *Task, cmd string, startTime time.Time) (*TaskResult, error) {
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return &TaskResult{
				ExitCode: 1,
				Duration: time.Since(startTime),
				Error:    "execution cancelled",
			}, ctx.Err()

		case <-ticker.C:
			// Check if tmux session still exists
			sessionExists, err := c.checkSessionExists(sessionID)
			if err != nil {
				return &TaskResult{
					ExitCode: 1,
					Duration: time.Since(startTime),
					Error:    fmt.Sprintf("failed to check session: %v", err),
				}, err
			}

			if !sessionExists {
				// Session ended - check if it was successful
				return c.determineTaskResult(sessionID, task, startTime)
			}

			// Check if Claude Code process is still running
			claudeRunning, err := c.checkClaudeProcessRunning(sessionID)
			if err != nil {
				fmt.Printf("Warning: failed to check Claude process: %v\n", err)
				continue
			}

			if !claudeRunning {
				// Claude process finished - check final state
				return c.determineTaskResult(sessionID, task, startTime)
			}

			// Check for completion patterns in session output
			completed, exitCode, err := c.checkSessionCompletion(sessionID)
			if err != nil {
				fmt.Printf("Warning: failed to check session completion: %v\n", err)
				continue
			}

			if completed {
				return &TaskResult{
					ExitCode:     exitCode,
					Duration:     time.Since(startTime),
					FilesChanged: c.detectChangedFiles(task.WorktreePath),
				}, nil
			}
		}
	}
}

// checkSessionExists checks if a tmux session exists
func (c *ClaudeAgent) checkSessionExists(sessionID string) (bool, error) {
	return c.sessionMgr.HasSession(sessionID), nil
}

// checkClaudeProcessRunning checks if Claude process is still running in session
func (c *ClaudeAgent) checkClaudeProcessRunning(sessionID string) (bool, error) {
	// Use tmux to check if there are running processes
	cmd := exec.Command("tmux", "list-panes", "-t", sessionID, "-F", "#{pane_current_command}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Check if claude is still running
	processes := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, process := range processes {
		if strings.Contains(strings.ToLower(process), "claude") {
			return true, nil
		}
	}

	return false, nil
}

// checkSessionCompletion checks if session has completion indicators
func (c *ClaudeAgent) checkSessionCompletion(sessionID string) (bool, int, error) {
	// Capture recent session output to check for completion patterns
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionID, "-p", "-S", "-10")
	output, err := cmd.Output()
	if err != nil {
		return false, 1, err
	}

	content := string(output)

	// Look for completion patterns in Claude output
	if strings.Contains(content, `"type":"result"`) {
		// Found result indicator - task completed
		if strings.Contains(content, `"subtype":"success"`) {
			return true, 0, nil
		} else if strings.Contains(content, `"subtype":"error"`) {
			return true, 1, nil
		}
		return true, 0, nil
	}

	return false, 0, nil
}

// determineTaskResult determines the final result of a task
func (c *ClaudeAgent) determineTaskResult(sessionID string, task *Task, startTime time.Time) (*TaskResult, error) {
	// Check session output for results
	completed, exitCode, err := c.checkSessionCompletion(sessionID)
	if err != nil {
		return &TaskResult{
			ExitCode: 1,
			Duration: time.Since(startTime),
			Error:    fmt.Sprintf("failed to determine completion status: %v", err),
		}, err
	}

	result := &TaskResult{
		ExitCode:     exitCode,
		Duration:     time.Since(startTime),
		FilesChanged: c.detectChangedFiles(task.WorktreePath),
	}

	if !completed {
		result.Error = "task execution incomplete"
	}

	return result, nil
}

// detectChangedFiles detects files that were changed during task execution
func (c *ClaudeAgent) detectChangedFiles(worktreePath string) []string {
	if worktreePath == "" {
		return []string{}
	}

	// Use git to find changed files
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}
	}

	return files
}
