package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/d-kuro/gwq/internal/tmux"
	"github.com/d-kuro/gwq/pkg/models"
)

// ExecutionStatus represents the status of a Claude execution
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusAborted   ExecutionStatus = "aborted"
)

// ExecutionMetadata holds metadata about a Claude execution
type ExecutionMetadata struct {
	ExecutionID      string          `json:"execution_id"`
	SessionID        string          `json:"session_id"`
	Prompt           string          `json:"prompt"`
	StartTime        time.Time       `json:"start_time"`
	EndTime          *time.Time      `json:"end_time,omitempty"`
	Status           ExecutionStatus `json:"status"`
	ExitCode         int             `json:"exit_code"`
	Repository       string          `json:"repository"`
	WorkingDirectory string          `json:"working_directory"`
	TmuxSession      string          `json:"tmux_session"`
	CostUSD          float64         `json:"cost_usd"`
	DurationMS       int64           `json:"duration_ms"`
	Model            string          `json:"model"`
	Tags             []string        `json:"tags,omitempty"`
	Priority         string          `json:"priority"`
	Timeout          time.Duration   `json:"timeout"`
}

// ExecutionManager manages Claude executions
type ExecutionManager struct {
	config     *models.ClaudeConfig
	sessionMgr *tmux.SessionManager
	logDir     string
	mu         sync.RWMutex
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager(config *models.ClaudeConfig) (*ExecutionManager, error) {
	// Create log directory structure
	logDir := filepath.Join(config.ConfigDir, "logs")
	dirs := []string{
		filepath.Join(logDir, "executions"),
		filepath.Join(logDir, "metadata"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}
	}

	// Create session manager
	sessionMgr := tmux.NewSessionManager(&tmux.SessionConfig{
		Enabled:      true,
		TmuxCommand:  "tmux",
		HistoryLimit: 50000,
	}, config.ConfigDir)

	return &ExecutionManager{
		config:     config,
		sessionMgr: sessionMgr,
		logDir:     logDir,
	}, nil
}

// Execute starts a Claude execution
func (em *ExecutionManager) Execute(ctx context.Context, metadata *ExecutionMetadata) (*tmux.Session, error) {
	// Auto cleanup old logs if enabled
	if em.config.Execution.AutoCleanup {
		go func() {
			if err := em.autoCleanupLogs(); err != nil {
				fmt.Printf("Warning: auto cleanup failed: %v\n", err)
			}
		}()
	}

	// Build Claude command
	cmd := em.buildClaudeCommand(metadata.Prompt)

	// Create log file paths (no date subdirectory)
	execLogDir := filepath.Join(em.logDir, "executions")
	if err := os.MkdirAll(execLogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create execution log directory: %w", err)
	}

	// Generate timestamp-prefixed filenames for better sorting
	logFileName := GenerateLogFileName(metadata.StartTime, metadata.ExecutionID)
	metadataFileName := GenerateMetadataFileName(metadata.StartTime, metadata.ExecutionID)
	logFile := filepath.Join(execLogDir, logFileName)
	metadataFile := filepath.Join(em.logDir, "metadata", metadataFileName)

	// Save initial metadata
	if err := em.saveMetadata(metadata, metadataFile); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	// Create named pipe for capturing output
	pipePath := filepath.Join(os.TempDir(), fmt.Sprintf("gwq-claude-%s.pipe", metadata.ExecutionID))
	if err := syscallMkfifo(pipePath, 0600); err != nil {
		return nil, fmt.Errorf("failed to create named pipe: %w", err)
	}
	defer func() {
		if err := os.Remove(pipePath); err != nil {
			fmt.Printf("Warning: failed to remove pipe: %v\n", err)
		}
	}()

	// Start log capture goroutine
	logCaptureDone := make(chan error, 1)
	go func() {
		logCaptureDone <- em.captureLogOutput(pipePath, logFile, metadata)
	}()

	// Build command with output redirection
	fullCmd := fmt.Sprintf("%s | tee %s", cmd, pipePath)

	// Create tmux session
	sessionOpts := tmux.SessionOptions{
		Context:    "claude-exec",
		Identifier: metadata.ExecutionID,
		WorkingDir: metadata.WorkingDirectory,
		Command:    fullCmd,
		Metadata: map[string]string{
			"execution_id": metadata.ExecutionID,
			"session_id":   metadata.SessionID,
			"prompt":       truncateString(metadata.Prompt, 100),
			"repository":   metadata.Repository,
			"priority":     metadata.Priority,
			"type":         "task",
		},
	}

	session, err := em.sessionMgr.CreateSession(ctx, sessionOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	metadata.TmuxSession = session.SessionName

	// Update metadata with session info
	if err := em.saveMetadata(metadata, metadataFile); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to update metadata: %v\n", err)
	}

	// Start monitoring goroutine
	go em.monitorExecution(ctx, metadata, session, logCaptureDone)

	return session, nil
}

// buildClaudeCommand builds the Claude command for execution
func (em *ExecutionManager) buildClaudeCommand(prompt string) string {
	// Escape the prompt for shell
	escapedPrompt := strings.ReplaceAll(prompt, `"`, `\"`)
	escapedPrompt = strings.ReplaceAll(escapedPrompt, `$`, `\$`)
	escapedPrompt = strings.ReplaceAll(escapedPrompt, "`", "\\`")

	// Build command with required flags for execution
	args := []string{
		em.config.Executable,
		"--verbose",
		"--dangerously-skip-permissions",
		"--output-format", "stream-json",
		"-p", fmt.Sprintf(`"%s"`, escapedPrompt),
	}

	return strings.Join(args, " ")
}

// captureLogOutput captures the JSON output from Claude
func (em *ExecutionManager) captureLogOutput(pipePath, logFile string, metadata *ExecutionMetadata) error {
	// Open pipe for reading
	pipe, err := os.OpenFile(pipePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open pipe: %w", err)
	}
	defer func() {
		if err := pipe.Close(); err != nil {
			fmt.Printf("Warning: failed to close pipe: %v\n", err)
		}
	}()

	// Create log file
	log, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Printf("Warning: failed to close log file: %v\n", err)
		}
	}()

	// Read and process JSON stream
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Add timestamp to each JSON line
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
			jsonData["timestamp"] = time.Now().Format(time.RFC3339)

			// Extract cost and model info if available
			if jsonData["type"] == "result" {
				if cost, ok := jsonData["cost_usd"].(float64); ok {
					metadata.CostUSD = cost
				}
				if duration, ok := jsonData["duration_ms"].(float64); ok {
					metadata.DurationMS = int64(duration)
				}
			}

			if jsonData["type"] == "system" && jsonData["subtype"] == "init" {
				if model, ok := jsonData["model"].(string); ok {
					metadata.Model = model
				}
			}

			// Write enhanced JSON line
			enhancedLine, _ := json.Marshal(jsonData)
			if _, err := fmt.Fprintf(log, "%s\n", enhancedLine); err != nil {
				fmt.Printf("Warning: failed to write enhanced log line: %v\n", err)
			}
		} else {
			// If not valid JSON, write as-is
			if _, err := fmt.Fprintln(log, line); err != nil {
				fmt.Printf("Warning: failed to write log line: %v\n", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from pipe: %w", err)
	}

	return nil
}

// monitorExecution monitors the execution and updates metadata
func (em *ExecutionManager) monitorExecution(ctx context.Context, metadata *ExecutionMetadata, session *tmux.Session, logCaptureDone <-chan error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	metadataFileName := GenerateMetadataFileName(metadata.StartTime, metadata.ExecutionID)
	metadataFile := filepath.Join(em.logDir, "metadata", metadataFileName)

	for {
		select {
		case <-ctx.Done():
			metadata.Status = ExecutionStatusAborted
			endTime := time.Now()
			metadata.EndTime = &endTime
			metadata.DurationMS = int64(endTime.Sub(metadata.StartTime).Milliseconds())
			if err := em.saveMetadata(metadata, metadataFile); err != nil {
				fmt.Printf("Warning: failed to save metadata on abort: %v\n", err)
			}
			return

		case err := <-logCaptureDone:
			// Log capture completed
			if err != nil {
				fmt.Printf("Warning: log capture error: %v\n", err)
			}

			// Check final status
			if em.sessionMgr.HasSession(session.SessionName) {
				// Session still exists, wait a bit more
				time.Sleep(2 * time.Second)
			}

			// Determine final status
			metadata.Status = ExecutionStatusCompleted
			if err != nil {
				metadata.Status = ExecutionStatusFailed
			}

			endTime := time.Now()
			metadata.EndTime = &endTime
			metadata.DurationMS = int64(endTime.Sub(metadata.StartTime).Milliseconds())
			if err := em.saveMetadata(metadata, metadataFile); err != nil {
				fmt.Printf("Warning: failed to save metadata on completion: %v\n", err)
			}
			return

		case <-ticker.C:
			// Check if session still exists
			if !em.sessionMgr.HasSession(session.SessionName) {
				// Session ended, wait for log capture to complete
				select {
				case <-logCaptureDone:
					// Already handled above
				case <-time.After(10 * time.Second):
					// Timeout waiting for log capture
					metadata.Status = ExecutionStatusCompleted
					endTime := time.Now()
					metadata.EndTime = &endTime
					metadata.DurationMS = int64(endTime.Sub(metadata.StartTime).Milliseconds())
					if err := em.saveMetadata(metadata, metadataFile); err != nil {
						fmt.Printf("Warning: failed to save metadata on timeout: %v\n", err)
					}
				}
				return
			}
		}
	}
}

// WatchExecution watches the execution output in real-time
func (em *ExecutionManager) WatchExecution(ctx context.Context, executionID string) error {
	// Find the log file
	metadata, err := em.LoadMetadata(executionID)
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	logFile := FindLogFileByExecutionID(em.logDir, metadata.StartTime, executionID)

	// Open log file
	file, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close log file: %v\n", err)
		}
	}()

	// Follow the file
	reader := bufio.NewReader(file)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Check if execution is still running
					if metadata.Status == ExecutionStatusRunning {
						time.Sleep(100 * time.Millisecond)
						continue
					}
					return nil
				}
				return err
			}

			// Parse and format JSON for display
			em.displayLogLine(line)
		}
	}
}

// displayLogLine formats and displays a log line
func (em *ExecutionManager) displayLogLine(line string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		fmt.Print(line)
		return
	}

	// Format based on type
	switch data["type"] {
	case "assistant":
		if msg, ok := data["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						if contentItem["type"] == "text" {
							if text, ok := contentItem["text"].(string); ok {
								fmt.Printf("🤖 %s\n", text)
							}
						}
					}
				}
			}
		}
	case "user":
		// Tool results
		if msg, ok := data["message"].(map[string]interface{}); ok {
			if content, ok := msg["content"].([]interface{}); ok {
				for _, item := range content {
					if contentItem, ok := item.(map[string]interface{}); ok {
						if contentItem["type"] == "tool_result" {
							if result, ok := contentItem["content"].(string); ok {
								fmt.Printf("📊 Tool Result:\n%s\n", result)
							}
						}
					}
				}
			}
		}
	case "result":
		if result, ok := data["result"].(string); ok {
			fmt.Printf("\n✅ Result: %s\n", result)
		}
		if cost, ok := data["cost_usd"].(float64); ok {
			fmt.Printf("💰 Cost: $%.4f\n", cost)
		}
	}
}

// saveMetadata saves execution metadata
func (em *ExecutionManager) saveMetadata(metadata *ExecutionMetadata, path string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadMetadata loads execution metadata by searching for files containing the executionID
func (em *ExecutionManager) LoadMetadata(executionID string) (*ExecutionMetadata, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	metadataDir := filepath.Join(em.logDir, "metadata")

	// First try exact match for backward compatibility
	exactFile := filepath.Join(metadataDir, fmt.Sprintf("%s.json", executionID))
	if data, err := os.ReadFile(exactFile); err == nil {
		var metadata ExecutionMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			return nil, err
		}
		return &metadata, nil
	}

	// If exact match fails, search for timestamp-prefixed files
	files, err := os.ReadDir(metadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), fmt.Sprintf("-%s.json", executionID)) {
			metadataFile := filepath.Join(metadataDir, file.Name())
			data, err := os.ReadFile(metadataFile)
			if err != nil {
				continue
			}

			var metadata ExecutionMetadata
			if err := json.Unmarshal(data, &metadata); err != nil {
				continue
			}

			return &metadata, nil
		}
	}

	return nil, fmt.Errorf("metadata not found for execution ID: %s", executionID)
}

// DetermineExecutionState determines the state of an execution
func (em *ExecutionManager) DetermineExecutionState(executionID string) ExecutionStatus {
	metadata, err := em.LoadMetadata(executionID)
	if err != nil {
		return ExecutionStatusAborted
	}

	// Check if tmux session exists
	if em.sessionMgr.HasSession(metadata.TmuxSession) {
		return ExecutionStatusRunning
	}

	// Check if log file exists and has content
	logFile := FindLogFileByExecutionID(em.logDir, metadata.StartTime, executionID)

	if _, err := os.Stat(logFile); err == nil {
		if metadata.Status == ExecutionStatusRunning {
			// Session ended but status not updated
			return ExecutionStatusCompleted
		}
		return metadata.Status
	}

	return ExecutionStatusAborted
}

// syscallMkfifo creates a named pipe (FIFO)
func syscallMkfifo(path string, mode uint32) error {
	// Use mkfifo command as a portable solution
	cmd := exec.Command("mkfifo", path)
	return cmd.Run()
}

// GetLogDir returns the log directory path
func (em *ExecutionManager) GetLogDir() string {
	return em.logDir
}

// autoCleanupLogs automatically cleans up old log files based on retention policy
func (em *ExecutionManager) autoCleanupLogs() error {
	// Use default retention of 30 days
	const defaultRetentionDays = 30
	cutoff := time.Now().AddDate(0, 0, -defaultRetentionDays)

	// Clean up execution logs
	executionsDir := filepath.Join(em.logDir, "executions")
	if err := em.cleanupExecutionLogs(executionsDir, cutoff); err != nil {
		return fmt.Errorf("failed to cleanup execution logs: %w", err)
	}

	// Clean up metadata files
	metadataDir := filepath.Join(em.logDir, "metadata")
	if err := em.cleanupMetadataFiles(metadataDir, cutoff); err != nil {
		return fmt.Errorf("failed to cleanup metadata files: %w", err)
	}

	// Remove obsolete index.json file if it exists
	em.cleanupObsoleteIndexFile()

	return nil
}

// cleanupExecutionLogs cleans up old execution log files
func (em *ExecutionManager) cleanupExecutionLogs(executionsDir string, cutoff time.Time) error {
	if _, err := os.Stat(executionsDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}

	entries, err := os.ReadDir(executionsDir)
	if err != nil {
		return err
	}

	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		// Try to extract timestamp from filename
		fileTime, err := ParseFileNameTimestamp(entry.Name())
		if err != nil {
			// For backward compatibility, check file modification time
			info, err := entry.Info()
			if err != nil {
				continue
			}
			fileTime = info.ModTime()
		}

		if fileTime.Before(cutoff) {
			filePath := filepath.Join(executionsDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("Warning: failed to remove old log file %s: %v\n", entry.Name(), err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("Auto cleanup: removed %d old execution log files\n", deletedCount)
	}

	return nil
}

// cleanupMetadataFiles cleans up old metadata files
func (em *ExecutionManager) cleanupMetadataFiles(metadataDir string, cutoff time.Time) error {
	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}

	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		return err
	}

	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Try to extract timestamp from filename
		fileTime, err := ParseFileNameTimestamp(entry.Name())
		if err != nil {
			// For backward compatibility, check file modification time
			info, err := entry.Info()
			if err != nil {
				continue
			}
			fileTime = info.ModTime()
		}

		if fileTime.Before(cutoff) {
			filePath := filepath.Join(metadataDir, entry.Name())

			// Check if execution is still running before deleting
			if !em.isExecutionRunningFromMetadataFile(filePath) {
				if err := os.Remove(filePath); err != nil {
					fmt.Printf("Warning: failed to remove old metadata file %s: %v\n", entry.Name(), err)
				} else {
					deletedCount++
				}
			}
		}
	}

	if deletedCount > 0 {
		fmt.Printf("Auto cleanup: removed %d old metadata files\n", deletedCount)
	}

	return nil
}

// isExecutionRunningFromMetadataFile checks if execution is still running
func (em *ExecutionManager) isExecutionRunningFromMetadataFile(metadataFile string) bool {
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return false
	}

	var metadata ExecutionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return false
	}

	// Check if status is running and tmux session exists
	if metadata.Status == ExecutionStatusRunning && metadata.TmuxSession != "" {
		return em.sessionMgr.HasSession(metadata.TmuxSession)
	}

	return false
}

// cleanupObsoleteIndexFile removes the obsolete index.json file
func (em *ExecutionManager) cleanupObsoleteIndexFile() {
	indexFile := filepath.Join(em.logDir, "index.json")
	if _, err := os.Stat(indexFile); err == nil {
		if err := os.Remove(indexFile); err == nil {
			fmt.Printf("Auto cleanup: removed obsolete index.json file\n")
		} else {
			fmt.Printf("Warning: failed to remove obsolete index.json file: %v\n", err)
		}
	}
}

// Helper functions

// GenerateLogFileName creates a timestamp-prefixed log file name
func GenerateLogFileName(startTime time.Time, executionID string) string {
	timestamp := startTime.Format("20060102-150405")
	return fmt.Sprintf("%s-%s.jsonl", timestamp, executionID)
}

// GenerateMetadataFileName creates a timestamp-prefixed metadata file name
func GenerateMetadataFileName(startTime time.Time, executionID string) string {
	timestamp := startTime.Format("20060102-150405")
	return fmt.Sprintf("%s-%s.json", timestamp, executionID)
}

// ParseFileNameTimestamp extracts timestamp from filename for sorting and cleanup
func ParseFileNameTimestamp(filename string) (time.Time, error) {
	parts := strings.Split(filename, "-")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid filename format: %s", filename)
	}
	return time.Parse("20060102-150405", parts[0]+"-"+parts[1])
}

// FindLogFileByExecutionID finds a log file by execution ID following the design specification:
// Primary: Flat structure with timestamp-first naming (YYYYMMDD-HHMMSS-{type}-{id}.jsonl)
// Fallback: Legacy formats in flat structure
func FindLogFileByExecutionID(logDir string, startTime time.Time, executionID string) string {
	execLogDir := filepath.Join(logDir, "executions")

	// 1. Try design-compliant timestamp-first format in flat structure
	timestamp := startTime.Format("20060102-150405")

	// Try timestamp-first format
	pattern := fmt.Sprintf("%s-%s.jsonl", timestamp, executionID)
	filePath := filepath.Join(execLogDir, pattern)
	if _, err := os.Stat(filePath); err == nil {
		return filePath
	}

	// 2. Try to find any file in flat structure containing the execution ID
	if entries, err := os.ReadDir(execLogDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.Contains(entry.Name(), executionID) && strings.HasSuffix(entry.Name(), ".jsonl") {
				filePath := filepath.Join(execLogDir, entry.Name())
				// Verify the file actually exists before returning
				if _, err := os.Stat(filePath); err == nil {
					return filePath
				}
			}
		}
	}

	// 3. Legacy fallback: old format in flat structure (no date subdirectory)
	oldFileName := fmt.Sprintf("%s.jsonl", executionID)
	oldPath := filepath.Join(execLogDir, oldFileName)
	if _, err := os.Stat(oldPath); err == nil {
		return oldPath
	}


	// Return design-compliant path as default for new file creation
	// This path will be used for new files but won't cause errors for missing files
	return filepath.Join(execLogDir, fmt.Sprintf("%s-%s.jsonl", timestamp, executionID))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
