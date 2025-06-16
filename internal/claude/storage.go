package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/d-kuro/gwq/pkg/filesystem"
)

// Storage provides persistent storage for Claude tasks
type Storage struct {
	queueDir string
	mu       sync.RWMutex
	fs       filesystem.FileSystemInterface
}

// NewStorage creates a new storage instance
func NewStorage(queueDir string) (*Storage, error) {
	return NewStorageWithFS(queueDir, filesystem.NewStandardFileSystem())
}

// NewStorageWithFS creates a new storage instance with custom filesystem
func NewStorageWithFS(queueDir string, fs filesystem.FileSystemInterface) (*Storage, error) {
	// Ensure queue directory exists
	if err := fs.MkdirAll(queueDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create queue directory: %w", err)
	}

	return &Storage{
		queueDir: queueDir,
		fs:       fs,
	}, nil
}

// SaveTask persists a task to storage
func (s *Storage) SaveTask(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	filename := s.taskFilename(task.ID)
	if err := s.fs.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// LoadTask loads a task from storage by ID
func (s *Storage) LoadTask(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filename := s.taskFilename(taskID)
	data, err := s.fs.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

// DeleteTask removes a task from storage
func (s *Storage) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := s.taskFilename(taskID)
	if err := s.fs.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("task not found: %s", taskID)
		}
		return fmt.Errorf("failed to delete task file: %w", err)
	}

	return nil
}

// ListTasks returns all tasks from storage
func (s *Storage) ListTasks() ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := s.fs.ReadDir(s.queueDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read queue directory: %w", err)
	}

	var tasks []*Task
	for _, entry := range entries {
		if entry.IsDir() || !isTaskFile(entry.Name()) {
			continue
		}

		filename := filepath.Join(s.queueDir, entry.Name())
		data, err := s.fs.ReadFile(filename)
		if err != nil {
			// Skip files that can't be read
			continue
		}

		var task Task
		if err := json.Unmarshal(data, &task); err != nil {
			// Skip files that can't be unmarshalled
			continue
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// UpdateTaskStatus updates the status of a task
func (s *Storage) UpdateTaskStatus(taskID string, status Status) error {
	task, err := s.LoadTask(taskID)
	if err != nil {
		return err
	}

	task.Status = status

	// Update timestamps based on status
	now := time.Now()
	switch status {
	case StatusRunning:
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
	case StatusCompleted, StatusFailed, StatusCancelled, StatusSkipped:
		if task.CompletedAt == nil {
			task.CompletedAt = &now
		}
	}

	return s.SaveTask(task)
}

// UpdateTaskResult updates the result of a task
func (s *Storage) UpdateTaskResult(taskID string, result *TaskResult) error {
	task, err := s.LoadTask(taskID)
	if err != nil {
		return err
	}

	task.Result = result
	return s.SaveTask(task)
}

// UpdateTaskSessionID updates the session ID of a task
func (s *Storage) UpdateTaskSessionID(taskID string, sessionID string) error {
	task, err := s.LoadTask(taskID)
	if err != nil {
		return err
	}

	task.SessionID = sessionID
	return s.SaveTask(task)
}

// FindTaskBySessionID finds a task by its session ID
func (s *Storage) FindTaskBySessionID(sessionID string) (*Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if task.SessionID == sessionID {
			return task, nil
		}
	}

	return nil, fmt.Errorf("task not found for session: %s", sessionID)
}

// GetTasksByStatus returns all tasks with a specific status
func (s *Storage) GetTasksByStatus(status Status) ([]*Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}

	var filtered []*Task
	for _, task := range tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}

	return filtered, nil
}

// GetPendingTasks returns all tasks that are pending or waiting
func (s *Storage) GetPendingTasks() ([]*Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}

	var pending []*Task
	for _, task := range tasks {
		if task.Status == StatusPending || task.Status == StatusWaiting {
			pending = append(pending, task)
		}
	}

	return pending, nil
}

// Cleanup removes completed/failed/cancelled tasks older than the specified duration
func (s *Storage) Cleanup(olderThan time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, err := s.ListTasks()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-olderThan)
	removed := 0

	for _, task := range tasks {
		// Only cleanup terminal states
		if task.Status != StatusCompleted && task.Status != StatusFailed &&
			task.Status != StatusCancelled && task.Status != StatusSkipped {
			continue
		}

		// Check if task is old enough
		if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
			if err := s.fs.Remove(s.taskFilename(task.ID)); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

// taskFilename returns the filename for a task
func (s *Storage) taskFilename(taskID string) string {
	return filepath.Join(s.queueDir, fmt.Sprintf("task-%s.json", taskID))
}

// isTaskFile checks if a filename is a task file
func isTaskFile(filename string) bool {
	return filepath.Ext(filename) == ".json" && len(filename) > 5 && filename[:5] == "task-"
}
