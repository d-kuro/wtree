package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/d-kuro/gwq/pkg/models"
)

// UnifiedLogManager manages logs for all execution types
type UnifiedLogManager struct {
	config *models.ClaudeConfig
	logDir string
}

// NewUnifiedLogManager creates a new unified log manager
func NewUnifiedLogManager(config *models.ClaudeConfig) (*UnifiedLogManager, error) {
	logDir := filepath.Join(config.ConfigDir, "logs")

	// Create unified log directory structure
	dirs := []string{
		filepath.Join(logDir, "executions"),
		filepath.Join(logDir, "metadata"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}
	}

	return &UnifiedLogManager{
		config: config,
		logDir: logDir,
	}, nil
}

// StartLogging starts logging for a unified execution
func (ulm *UnifiedLogManager) StartLogging(execution *UnifiedExecution) (string, error) {
	// Create log file paths in flat structure (design-compliant)
	execLogDir := filepath.Join(ulm.logDir, "executions")
	if err := os.MkdirAll(execLogDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create execution log directory: %w", err)
	}

	// Use timestamp-first format: YYYYMMDD-HHMMSS-{executionID}.jsonl
	// ExecutionID already includes type prefix (e.g., "task-{id}")
	timestamp := execution.StartTime.Format("20060102-150405")
	logFileName := fmt.Sprintf("%s-%s.jsonl", timestamp, execution.ExecutionID)
	logFile := filepath.Join(execLogDir, logFileName)

	// Save initial metadata
	if err := ulm.saveExecutionMetadata(execution); err != nil {
		return "", fmt.Errorf("failed to save initial metadata: %w", err)
	}

	// No index file needed - metadata directory is the source of truth

	return logFile, nil
}

// SaveExecution saves execution data to unified storage
func (ulm *UnifiedLogManager) SaveExecution(execution *UnifiedExecution) error {
	// Save metadata
	if err := ulm.saveExecutionMetadata(execution); err != nil {
		return fmt.Errorf("failed to save execution metadata: %w", err)
	}

	// No index file needed - metadata directory is the source of truth

	return nil
}

// LoadExecution loads a unified execution by ID
func (ulm *UnifiedLogManager) LoadExecution(executionID string) (*UnifiedExecution, error) {
	metadataDir := filepath.Join(ulm.logDir, "metadata")

	// Find the metadata file with timestamp prefix
	files, err := os.ReadDir(metadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var metadataFile string
	suffix := fmt.Sprintf("-%s.json", executionID)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			metadataFile = filepath.Join(metadataDir, file.Name())
			break
		}
	}

	if metadataFile == "" {
		return nil, fmt.Errorf("metadata file not found for execution ID: %s", executionID)
	}

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var execution UnifiedExecution
	if err := json.Unmarshal(data, &execution); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution metadata: %w", err)
	}

	return &execution, nil
}

// ListExecutions lists all executions with optional filtering
func (ulm *UnifiedLogManager) ListExecutions(filters ...ExecutionFilter) ([]*UnifiedExecution, error) {
	// Load executions directly from metadata directory
	metadataDir := filepath.Join(ulm.logDir, "metadata")

	files, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*UnifiedExecution{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var executions []*UnifiedExecution
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		metadataFile := filepath.Join(metadataDir, file.Name())
		data, err := os.ReadFile(metadataFile)
		if err != nil {
			fmt.Printf("Warning: failed to read metadata file %s: %v\n", metadataFile, err)
			continue
		}

		var execution UnifiedExecution
		if err := json.Unmarshal(data, &execution); err != nil {
			fmt.Printf("Warning: failed to unmarshal metadata file %s: %v\n", metadataFile, err)
			continue
		}

		executions = append(executions, &execution)
	}

	// Apply filters
	var filtered []*UnifiedExecution
	for _, exec := range executions {
		include := true
		for _, filter := range filters {
			if !filter(exec) {
				include = false
				break
			}
		}
		if include {
			filtered = append(filtered, exec)
		}
	}

	// Sort by start time (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.After(filtered[j].StartTime)
	})

	return filtered, nil
}

// GetLogFile returns the log file path for an execution
func (ulm *UnifiedLogManager) GetLogFile(execution *UnifiedExecution) string {
	dateDir := execution.StartTime.Format("2006-01-02")
	logFileName := fmt.Sprintf("%s-%s.jsonl", execution.ExecutionType, execution.ExecutionID)
	return filepath.Join(ulm.logDir, "executions", dateDir, logFileName)
}

// saveExecutionMetadata saves execution metadata to file using timestamp-first format
func (ulm *UnifiedLogManager) saveExecutionMetadata(execution *UnifiedExecution) error {
	// Use timestamp-first format: YYYYMMDD-HHMMSS-{executionID}.json
	timestamp := execution.StartTime.Format("20060102-150405")
	metadataFile := filepath.Join(ulm.logDir, "metadata", fmt.Sprintf("%s-%s.json", timestamp, execution.ExecutionID))

	data, err := json.MarshalIndent(execution, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution metadata: %w", err)
	}

	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// CleanupOldLogs removes old log files and metadata
func (ulm *UnifiedLogManager) CleanupOldLogs(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Load executions and filter old ones
	executions, err := ulm.ListExecutions()
	if err != nil {
		return fmt.Errorf("failed to list executions: %w", err)
	}

	var toDelete []*UnifiedExecution
	for _, exec := range executions {
		if exec.StartTime.Before(cutoff) && exec.Status != ExecutionStatusRunning {
			toDelete = append(toDelete, exec)
		}
	}

	deletedCount := 0
	for _, exec := range toDelete {
		// Delete log file
		logFile := ulm.GetLogFile(exec)
		if err := os.Remove(logFile); err == nil {
			deletedCount++
		}

		// Delete metadata file with timestamp prefix
		metadataDir := filepath.Join(ulm.logDir, "metadata")
		files, _ := os.ReadDir(metadataDir)
		suffix := fmt.Sprintf("-%s.json", exec.ExecutionID)
		for _, file := range files {
			if strings.HasSuffix(file.Name(), suffix) {
				metadataFile := filepath.Join(metadataDir, file.Name())
				if err := os.Remove(metadataFile); err != nil {
					fmt.Printf("Warning: failed to delete metadata file %s: %v\n", metadataFile, err)
				}
				break
			}
		}
	}

	// No index rebuilding needed - metadata directory is the source of truth

	fmt.Printf("Cleaned %d old execution logs\n", deletedCount)
	return nil
}

// GetLogDir returns the log directory path
func (ulm *UnifiedLogManager) GetLogDir() string {
	return ulm.logDir
}
