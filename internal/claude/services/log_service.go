package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/d-kuro/gwq/internal/claude"
)

// LogService handles log processing and filtering operations
type LogService struct {
	execManager *claude.ExecutionManager
}

// NewLogService creates a new log service
func NewLogService(execManager *claude.ExecutionManager) *LogService {
	return &LogService{
		execManager: execManager,
	}
}

// LoadExecutions loads executions from metadata directory
func (l *LogService) LoadExecutions() ([]claude.ExecutionMetadata, error) {
	// Load executions directly from metadata directory
	metadataDir := filepath.Join(l.execManager.GetLogDir(), "metadata")

	// Read all metadata files
	files, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []claude.ExecutionMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var executions []claude.ExecutionMetadata
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		execution, err := l.loadExecutionFromFile(metadataDir, file.Name())
		if err != nil {
			continue // Skip invalid files
		}

		// Verify that corresponding log file exists
		if l.logFileExists(execution) {
			executions = append(executions, execution)
		} else {
			// Update status to indicate missing log file
			execution.Status = claude.ExecutionStatusAborted
			executions = append(executions, execution)
		}
	}

	// Sort by start time (newest first)
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].StartTime.After(executions[j].StartTime)
	})

	return executions, nil
}

// FilterExecutionsByStatus filters executions by status
func (l *LogService) FilterExecutionsByStatus(executions []claude.ExecutionMetadata, status string) []claude.ExecutionMetadata {
	var filtered []claude.ExecutionMetadata
	for _, exec := range executions {
		if string(exec.Status) == status {
			filtered = append(filtered, exec)
		}
	}
	return filtered
}

// FilterExecutionsByDate filters executions by date
func (l *LogService) FilterExecutionsByDate(executions []claude.ExecutionMetadata, date string) []claude.ExecutionMetadata {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return executions // Return unfiltered if invalid date
	}

	var filtered []claude.ExecutionMetadata
	for _, exec := range executions {
		if exec.StartTime.Format("2006-01-02") == date {
			filtered = append(filtered, exec)
		}
	}
	return filtered
}

// FilterExecutionsByContent filters executions by content (prompt/tags)
func (l *LogService) FilterExecutionsByContent(executions []claude.ExecutionMetadata, text string) []claude.ExecutionMetadata {
	var filtered []claude.ExecutionMetadata
	lowerText := strings.ToLower(text)

	for _, exec := range executions {
		// Check prompt
		if strings.Contains(strings.ToLower(exec.Prompt), lowerText) {
			filtered = append(filtered, exec)
			continue
		}

		// Check tags
		for _, tag := range exec.Tags {
			if strings.Contains(strings.ToLower(tag), lowerText) {
				filtered = append(filtered, exec)
				break
			}
		}
	}

	return filtered
}

// CleanOldLogs removes execution logs older than the specified duration
func (l *LogService) CleanOldLogs(olderThan time.Duration) (int, error) {
	executions, err := l.LoadExecutions()
	if err != nil {
		return 0, fmt.Errorf("failed to load executions: %w", err)
	}

	cutoff := time.Now().Add(-olderThan)
	var toDelete []claude.ExecutionMetadata

	for _, exec := range executions {
		if exec.StartTime.Before(cutoff) && exec.Status != claude.ExecutionStatusRunning {
			toDelete = append(toDelete, exec)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete log files and metadata
	logDir := l.execManager.GetLogDir()
	deletedCount := 0

	for _, exec := range toDelete {
		// Delete log file
		logFile := claude.FindLogFileByExecutionID(logDir, exec.StartTime, exec.ExecutionID)
		if err := os.Remove(logFile); err == nil {
			deletedCount++
		}

		// Delete metadata file - try both new and old formats
		newMetadataFile := filepath.Join(logDir, "metadata", claude.GenerateMetadataFileName(exec.StartTime, exec.ExecutionID))
		oldMetadataFile := filepath.Join(logDir, "metadata", fmt.Sprintf("%s.json", exec.ExecutionID))

		// Try new format first, then old format
		metadataFile := newMetadataFile
		if _, err := os.Stat(newMetadataFile); os.IsNotExist(err) {
			metadataFile = oldMetadataFile
		}
		// Remove metadata file - ignore errors as they're not critical
		_ = os.Remove(metadataFile)
	}

	return deletedCount, nil
}

// GetExecution loads a specific execution by ID
func (l *LogService) GetExecution(executionID string) (*claude.ExecutionMetadata, error) {
	metadata, err := l.execManager.LoadMetadata(executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata for %s: %w", executionID, err)
	}
	return metadata, nil
}

// ProcessExecution processes and formats an execution for display
func (l *LogService) ProcessExecution(metadata *claude.ExecutionMetadata) (string, error) {
	// Check if log file exists
	logFile := claude.FindLogFileByExecutionID(l.execManager.GetLogDir(), metadata.StartTime, metadata.ExecutionID)

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		// Return metadata-only view for executions without log files
		return l.formatMetadataOnly(metadata), nil
	}

	// Load and format the log
	processor := claude.NewLogProcessor()
	formatted, err := processor.ProcessExecution(metadata, l.execManager)
	if err != nil {
		return "", fmt.Errorf("failed to process log: %w", err)
	}

	return formatted, nil
}

// GetRunningExecutions returns all currently running executions
func (l *LogService) GetRunningExecutions() ([]claude.ExecutionMetadata, error) {
	executions, err := l.LoadExecutions()
	if err != nil {
		return nil, err
	}

	return l.FilterExecutionsByStatus(executions, "running"), nil
}

// ParseDuration parses duration string with support for days
func (l *LogService) ParseDuration(durationStr string) (time.Duration, error) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// Try parsing as days (e.g., "30d")
		if strings.HasSuffix(durationStr, "d") {
			days := strings.TrimSuffix(durationStr, "d")
			if d, err := time.ParseDuration(days + "h"); err == nil {
				duration = d * 24
			} else {
				return 0, fmt.Errorf("invalid duration format: %s", durationStr)
			}
		} else {
			return 0, fmt.Errorf("invalid duration format: %s", durationStr)
		}
	}
	return duration, nil
}

// loadExecutionFromFile loads execution metadata from a file
func (l *LogService) loadExecutionFromFile(metadataDir, fileName string) (claude.ExecutionMetadata, error) {
	var execution claude.ExecutionMetadata

	metadataFile := filepath.Join(metadataDir, fileName)
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return execution, fmt.Errorf("failed to read metadata file %s: %w", metadataFile, err)
	}

	if err := json.Unmarshal(data, &execution); err != nil {
		return execution, fmt.Errorf("failed to unmarshal metadata file %s: %w", metadataFile, err)
	}

	return execution, nil
}

// logFileExists checks if the log file exists for an execution
func (l *LogService) logFileExists(execution claude.ExecutionMetadata) bool {
	logFile := claude.FindLogFileByExecutionID(l.execManager.GetLogDir(), execution.StartTime, execution.ExecutionID)
	_, err := os.Stat(logFile)
	return err == nil
}

// formatMetadataOnly returns a formatted view for executions without log files
func (l *LogService) formatMetadataOnly(metadata *claude.ExecutionMetadata) string {
	return fmt.Sprintf(`╭─ Execution: %s ─────────────────────────────────────────╮
│ Status: ⊘ Aborted (log file missing)                     │
│ Started: %-42s │
│ Repository: %-38s │
╰───────────────────────────────────────────────────────────╯

💬 Prompt:
%s

⚠️  Log file not found. This execution may have been interrupted or not properly initialized.
`,
		metadata.ExecutionID,
		metadata.StartTime.Format("2006-01-02 15:04:05"),
		l.truncateString(metadata.Repository, 38),
		metadata.Prompt)
}

// truncateString truncates a string to a maximum length
func (l *LogService) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
