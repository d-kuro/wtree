package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/d-kuro/gwq/pkg/models"
)

func TestNewUnifiedLogManager(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	if ulm == nil {
		t.Fatal("NewUnifiedLogManager() returned nil")
	}

	// Check directory structure was created
	dirs := []string{
		filepath.Join(tempDir, "logs", "executions"),
		filepath.Join(tempDir, "logs", "metadata"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}
}

func TestNewUnifiedLogManagerError(t *testing.T) {
	// Use an invalid path to trigger error
	config := &models.ClaudeConfig{
		ConfigDir: "/root/invalid/path/that/should/fail",
	}

	_, err := NewUnifiedLogManager(config)
	if err == nil {
		t.Error("Expected error for invalid config directory")
	}
}

func TestStartLogging(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	execution := &UnifiedExecution{
		ExecutionID:   "task-123",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now(),
		Status:        ExecutionStatusRunning,
	}

	logFile, err := ulm.StartLogging(execution)
	if err != nil {
		t.Fatalf("StartLogging() failed: %v", err)
	}

	// Check log file path format
	if !strings.Contains(logFile, "task-123") {
		t.Errorf("Log file path should contain execution ID, got: %s", logFile)
	}

	// Check timestamp format in filename
	baseName := filepath.Base(logFile)
	if !strings.HasSuffix(baseName, ".jsonl") {
		t.Errorf("Log file should have .jsonl extension, got: %s", baseName)
	}

	// Check metadata file was created
	metadataFiles, err := os.ReadDir(filepath.Join(tempDir, "logs", "metadata"))
	if err != nil {
		t.Fatalf("Failed to read metadata directory: %v", err)
	}

	if len(metadataFiles) != 1 {
		t.Errorf("Expected 1 metadata file, got %d", len(metadataFiles))
	}
}

func TestSaveExecution(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	execution := &UnifiedExecution{
		ExecutionID:   "task-456",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now(),
		Status:        ExecutionStatusCompleted,
		Prompt:        "Process this task",
		Repository:    "test-repo",
	}

	err = ulm.SaveExecution(execution)
	if err != nil {
		t.Fatalf("SaveExecution() failed: %v", err)
	}

	// Verify metadata file was created
	metadataFile := filepath.Join(tempDir, "logs", "metadata",
		fmt.Sprintf("%s-%s.json", execution.StartTime.Format("20060102-150405"), execution.ExecutionID))

	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		t.Error("Metadata file was not created")
	}

	// Verify content
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	var saved UnifiedExecution
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	if saved.ExecutionID != execution.ExecutionID {
		t.Errorf("ExecutionID mismatch: expected %s, got %s", execution.ExecutionID, saved.ExecutionID)
	}

	if saved.ExecutionType != execution.ExecutionType {
		t.Errorf("ExecutionType mismatch: expected %s, got %s", execution.ExecutionType, saved.ExecutionType)
	}
}

func TestLoadExecution(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Create and save an execution
	original := &UnifiedExecution{
		ExecutionID:   "task-789",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now(),
		Status:        ExecutionStatusRunning,
		Prompt:        "Test prompt",
		WorkingDir:    "/test/dir",
		Tags:          []string{"test", "integration"},
	}

	err = ulm.SaveExecution(original)
	if err != nil {
		t.Fatalf("SaveExecution() failed: %v", err)
	}

	// Load it back
	loaded, err := ulm.LoadExecution("task-789")
	if err != nil {
		t.Fatalf("LoadExecution() failed: %v", err)
	}

	// Verify fields
	if loaded.ExecutionID != original.ExecutionID {
		t.Errorf("ExecutionID mismatch: expected %s, got %s", original.ExecutionID, loaded.ExecutionID)
	}

	if loaded.Prompt != original.Prompt {
		t.Errorf("Prompt mismatch: expected %s, got %s", original.Prompt, loaded.Prompt)
	}

	if len(loaded.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: expected %d, got %d", len(original.Tags), len(loaded.Tags))
	}
}

func TestLoadExecutionNotFound(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	_, err = ulm.LoadExecution("nonexistent-id")
	if err == nil {
		t.Error("Expected error for non-existent execution")
	}
}

func TestListExecutions(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Create multiple executions
	executions := []*UnifiedExecution{
		{
			ExecutionID:   "task-001",
			ExecutionType: ExecutionTypeTask,
			StartTime:     time.Now().Add(-2 * time.Hour),
			Status:        ExecutionStatusCompleted,
		},
		{
			ExecutionID:   "task-002",
			ExecutionType: ExecutionTypeTask,
			StartTime:     time.Now().Add(-1 * time.Hour),
			Status:        ExecutionStatusFailed,
		},
		{
			ExecutionID:   "task-003",
			ExecutionType: ExecutionTypeTask,
			StartTime:     time.Now(),
			Status:        ExecutionStatusRunning,
		},
	}

	for _, exec := range executions {
		if err := ulm.SaveExecution(exec); err != nil {
			t.Fatalf("SaveExecution() failed: %v", err)
		}
	}

	// List all executions
	listed, err := ulm.ListExecutions()
	if err != nil {
		t.Fatalf("ListExecutions() failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("Expected 3 executions, got %d", len(listed))
	}

	// Verify sort order (newest first)
	if listed[0].ExecutionID != "task-003" {
		t.Errorf("Expected newest execution first, got %s", listed[0].ExecutionID)
	}
}

func TestListExecutionsWithFilters(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Create executions with different types and statuses
	executions := []*UnifiedExecution{
		{
			ExecutionID:   "task-001",
			ExecutionType: ExecutionTypeTask,
			StartTime:     time.Now(),
			Status:        ExecutionStatusCompleted,
		},
		{
			ExecutionID:   "task-002",
			ExecutionType: ExecutionTypeTask,
			StartTime:     time.Now(),
			Status:        ExecutionStatusRunning,
		},
		{
			ExecutionID:   "task-003",
			ExecutionType: ExecutionTypeTask,
			StartTime:     time.Now(),
			Status:        ExecutionStatusCompleted,
		},
	}

	for _, exec := range executions {
		if err := ulm.SaveExecution(exec); err != nil {
			t.Fatalf("SaveExecution() failed: %v", err)
		}
	}

	// Test type filter
	taskFilter := func(e *UnifiedExecution) bool {
		return e.ExecutionType == ExecutionTypeTask
	}

	taskExecutions, err := ulm.ListExecutions(taskFilter)
	if err != nil {
		t.Fatalf("ListExecutions() with filter failed: %v", err)
	}

	if len(taskExecutions) != 3 {
		t.Errorf("Expected 3 task executions, got %d", len(taskExecutions))
	}

	// Test status filter
	completedFilter := func(e *UnifiedExecution) bool {
		return e.Status == ExecutionStatusCompleted
	}

	completedExecutions, err := ulm.ListExecutions(completedFilter)
	if err != nil {
		t.Fatalf("ListExecutions() with filter failed: %v", err)
	}

	if len(completedExecutions) != 2 {
		t.Errorf("Expected 2 completed executions, got %d", len(completedExecutions))
	}

	// Test multiple filters
	taskCompletedExecutions, err := ulm.ListExecutions(taskFilter, completedFilter)
	if err != nil {
		t.Fatalf("ListExecutions() with multiple filters failed: %v", err)
	}

	if len(taskCompletedExecutions) != 2 {
		t.Errorf("Expected 2 completed task executions, got %d", len(taskCompletedExecutions))
	}
}

func TestListExecutionsEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	executions, err := ulm.ListExecutions()
	if err != nil {
		t.Fatalf("ListExecutions() failed: %v", err)
	}

	if len(executions) != 0 {
		t.Errorf("Expected empty list, got %d executions", len(executions))
	}
}

func TestGetLogFile(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	execution := &UnifiedExecution{
		ExecutionID:   "task-123",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Date(2024, 12, 6, 10, 30, 0, 0, time.UTC),
	}

	logFile := ulm.GetLogFile(execution)

	// Verify path structure
	expectedDate := "2024-12-06"
	if !strings.Contains(logFile, expectedDate) {
		t.Errorf("Log file path should contain date directory %s, got: %s", expectedDate, logFile)
	}

	if !strings.Contains(logFile, "task-task-123.jsonl") {
		t.Errorf("Log file should contain execution type and ID, got: %s", logFile)
	}
}

func TestCleanupOldLogs(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Create old and new executions
	oldExecution := &UnifiedExecution{
		ExecutionID:   "old-001",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now().Add(-48 * time.Hour),
		Status:        ExecutionStatusCompleted,
	}

	newExecution := &UnifiedExecution{
		ExecutionID:   "new-002",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now(),
		Status:        ExecutionStatusCompleted,
	}

	runningExecution := &UnifiedExecution{
		ExecutionID:   "running-003",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now().Add(-48 * time.Hour),
		Status:        ExecutionStatusRunning,
	}

	for _, exec := range []*UnifiedExecution{oldExecution, newExecution, runningExecution} {
		if err := ulm.SaveExecution(exec); err != nil {
			t.Fatalf("SaveExecution() failed: %v", err)
		}

		// Create dummy log files
		logFile := ulm.GetLogFile(exec)
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			t.Fatalf("Failed to create log directory: %v", err)
		}
		if err := os.WriteFile(logFile, []byte("test log"), 0644); err != nil {
			t.Fatalf("Failed to create log file: %v", err)
		}
	}

	// Cleanup logs older than 24 hours
	err = ulm.CleanupOldLogs(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupOldLogs() failed: %v", err)
	}

	// Verify results
	executions, err := ulm.ListExecutions()
	if err != nil {
		t.Fatalf("ListExecutions() failed: %v", err)
	}

	// Should have 2 executions left (new and running)
	if len(executions) != 2 {
		t.Errorf("Expected 2 executions after cleanup, got %d", len(executions))
	}

	// Verify old completed execution was deleted
	_, err = ulm.LoadExecution("old-001")
	if err == nil {
		t.Error("Old execution should have been deleted")
	}

	// Verify running execution was not deleted
	_, err = ulm.LoadExecution("running-003")
	if err != nil {
		t.Error("Running execution should not have been deleted")
	}
}

func TestGetLogDir(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	logDir := ulm.GetLogDir()
	expectedDir := filepath.Join(tempDir, "logs")

	if logDir != expectedDir {
		t.Errorf("Expected log dir %s, got %s", expectedDir, logDir)
	}
}

func TestCorruptedMetadataHandling(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Create a valid execution
	validExecution := &UnifiedExecution{
		ExecutionID:   "valid-001",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now(),
		Status:        ExecutionStatusCompleted,
	}

	if err := ulm.SaveExecution(validExecution); err != nil {
		t.Fatalf("SaveExecution() failed: %v", err)
	}

	// Create corrupted metadata file
	corruptedFile := filepath.Join(tempDir, "logs", "metadata", "corrupted.json")
	if err := os.WriteFile(corruptedFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// ListExecutions should handle corrupted files gracefully
	executions, err := ulm.ListExecutions()
	if err != nil {
		t.Fatalf("ListExecutions() failed with corrupted file: %v", err)
	}

	// Should still return the valid execution
	if len(executions) != 1 {
		t.Errorf("Expected 1 valid execution, got %d", len(executions))
	}
}

func TestTimestampFirstFormat(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Fixed time for predictable testing
	fixedTime := time.Date(2024, 12, 6, 14, 30, 45, 0, time.UTC)

	execution := &UnifiedExecution{
		ExecutionID:   "task-test-123",
		ExecutionType: ExecutionTypeTask,
		StartTime:     fixedTime,
		Status:        ExecutionStatusRunning,
	}

	logFile, err := ulm.StartLogging(execution)
	if err != nil {
		t.Fatalf("StartLogging() failed: %v", err)
	}

	// Check log file name format
	expectedLogName := "20241206-143045-task-test-123.jsonl"
	if !strings.Contains(logFile, expectedLogName) {
		t.Errorf("Expected log file name to contain %s, got: %s", expectedLogName, filepath.Base(logFile))
	}

	// Check metadata file name format
	metadataFiles, err := os.ReadDir(filepath.Join(tempDir, "logs", "metadata"))
	if err != nil {
		t.Fatalf("Failed to read metadata directory: %v", err)
	}

	expectedMetadataName := "20241206-143045-task-test-123.json"
	found := false
	for _, file := range metadataFiles {
		if file.Name() == expectedMetadataName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected metadata file name %s not found", expectedMetadataName)
	}
}

func TestTaskExecutionInfo(t *testing.T) {
	tempDir := t.TempDir()
	config := &models.ClaudeConfig{
		ConfigDir: tempDir,
	}

	ulm, err := NewUnifiedLogManager(config)
	if err != nil {
		t.Fatalf("NewUnifiedLogManager() failed: %v", err)
	}

	// Create execution with task-specific info
	taskInfo := &TaskExecutionInfo{
		TaskID:       "task-abc",
		TaskName:     "Build Project",
		Dependencies: []string{"task-xyz"},
		TaskPriority: 2,
	}

	execution := &UnifiedExecution{
		ExecutionID:   "task-exec-001",
		ExecutionType: ExecutionTypeTask,
		StartTime:     time.Now(),
		Status:        ExecutionStatusRunning,
		TaskInfo:      taskInfo,
	}

	err = ulm.SaveExecution(execution)
	if err != nil {
		t.Fatalf("SaveExecution() failed: %v", err)
	}

	// Load and verify
	loaded, err := ulm.LoadExecution("task-exec-001")
	if err != nil {
		t.Fatalf("LoadExecution() failed: %v", err)
	}

	if loaded.TaskInfo == nil {
		t.Fatal("TaskInfo should not be nil")
	}

	if loaded.TaskInfo.TaskName != "Build Project" {
		t.Errorf("TaskName mismatch: expected 'Build Project', got '%s'", loaded.TaskInfo.TaskName)
	}

	if len(loaded.TaskInfo.Dependencies) != 1 || loaded.TaskInfo.Dependencies[0] != "task-xyz" {
		t.Error("Dependencies field not correctly loaded")
	}
}
