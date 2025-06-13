package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/d-kuro/gwq/internal/claude"
	"gopkg.in/yaml.v3"
)

// TaskService handles task business logic
type TaskService struct {
	storage    *claude.Storage
	repository *RepositoryService
}

// NewTaskService creates a new task service
func NewTaskService(storage *claude.Storage) *TaskService {
	return &TaskService{
		storage:    storage,
		repository: NewRepositoryService(),
	}
}

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	Name                 string
	Worktree             string
	BaseBranch           string
	Priority             int
	DependsOn            []string
	Prompt               string
	FilesToFocus         []string
	VerificationCommands []string
	AutoReview           bool
	AutoCommit           bool
	Repository           string
}

// CreateTask creates a new task with validation
func (s *TaskService) CreateTask(req *CreateTaskRequest) (*claude.Task, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Resolve repository
	repoRoot, err := s.resolveRepository(req.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repository: %w", err)
	}

	// Create task
	task := &claude.Task{
		ID:                   s.generateTaskID(),
		Name:                 req.Name,
		Priority:             claude.Priority(req.Priority),
		Status:               claude.StatusPending,
		CreatedAt:            time.Now(),
		RepositoryRoot:       repoRoot,
		DependsOn:            req.DependsOn,
		DependencyPolicy:     claude.DependencyPolicyWait,
		AgentType:            "claude",
		Prompt:               req.Prompt,
		FilesToFocus:         req.FilesToFocus,
		VerificationCommands: req.VerificationCommands,
		Config: claude.TaskConfig{
			SkipPermissions: true,
			AutoCommit:      req.AutoCommit,
		},
	}

	// Handle branch or worktree setup
	if err := s.setupTaskWorktree(task, req); err != nil {
		return nil, err
	}

	// Save task
	if err := s.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return task, nil
}

// CreateTasksFromFile creates multiple tasks from a YAML file
func (s *TaskService) CreateTasksFromFile(filePath string) ([]*claude.Task, error) {
	// Read YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	// Parse YAML
	var tasksDefinition claude.TaskFile
	if err := yaml.Unmarshal(data, &tasksDefinition); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate version
	if tasksDefinition.Version != "1.0" {
		return nil, fmt.Errorf("unsupported task file version: %s (expected 1.0)", tasksDefinition.Version)
	}

	// Resolve default repository
	defaultRepo, err := s.resolveDefaultRepository(tasksDefinition.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve default repository: %w", err)
	}

	var createdTasks []*claude.Task

	// Process each task
	for _, entry := range tasksDefinition.Tasks {
		task, err := s.createTaskFromFileEntry(entry, defaultRepo, tasksDefinition.DefaultConfig)
		if err != nil {
			return createdTasks, fmt.Errorf("failed to create task %s: %w", entry.ID, err)
		}
		createdTasks = append(createdTasks, task)
	}

	return createdTasks, nil
}

// FindTaskByPattern finds a task by ID or pattern matching
func (s *TaskService) FindTaskByPattern(pattern string) (*claude.Task, error) {
	// Try exact ID match first
	if task, err := s.storage.LoadTask(pattern); err == nil {
		return task, nil
	}

	// Try pattern matching
	tasks, err := s.storage.ListTasks()
	if err != nil {
		return nil, err
	}

	var matches []*claude.Task
	for _, task := range tasks {
		if strings.Contains(task.ID, pattern) ||
			strings.Contains(strings.ToLower(task.Name), strings.ToLower(pattern)) ||
			strings.Contains(task.Worktree, pattern) {
			matches = append(matches, task)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no task found matching pattern: %s", pattern)
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple tasks match pattern '%s': %d matches", pattern, len(matches))
	}

	return matches[0], nil
}

// FilterTasksByStatus filters tasks by status
func (s *TaskService) FilterTasksByStatus(tasks []*claude.Task, status string) []*claude.Task {
	var filtered []*claude.Task
	for _, task := range tasks {
		if string(task.Status) == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterTasksByPriority filters tasks by minimum priority
func (s *TaskService) FilterTasksByPriority(tasks []*claude.Task, minPriority int) []*claude.Task {
	var filtered []*claude.Task
	for _, task := range tasks {
		if int(task.Priority) >= minPriority {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterTasksByDate filters tasks by creation date
func (s *TaskService) FilterTasksByDate(tasks []*claude.Task, date string) []*claude.Task {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return tasks // Return unfiltered if invalid date
	}

	var filtered []*claude.Task
	for _, task := range tasks {
		if task.CreatedAt.Format("2006-01-02") == date {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// validateCreateRequest validates a task creation request
func (s *TaskService) validateCreateRequest(req *CreateTaskRequest) error {
	if req.Name == "" {
		return fmt.Errorf("task name is required")
	}

	if req.Worktree == "" {
		return fmt.Errorf("worktree must be specified")
	}

	if req.Priority < 1 || req.Priority > 100 {
		return fmt.Errorf("priority must be between 1 and 100")
	}

	return nil
}

// resolveRepository resolves repository path
func (s *TaskService) resolveRepository(repo string) (string, error) {
	if repo == "" {
		return s.repository.FindRepoRoot("")
	}
	return s.repository.ResolveRepository(repo)
}

// resolveDefaultRepository resolves default repository from task file
func (s *TaskService) resolveDefaultRepository(repo string) (string, error) {
	if repo == "" {
		return s.repository.FindRepoRoot("")
	}
	return s.repository.ResolveRepository(repo)
}

// setupTaskWorktree sets up the worktree configuration for a task
func (s *TaskService) setupTaskWorktree(task *claude.Task, req *CreateTaskRequest) error {
	// Try to resolve worktree path
	worktreePath, err := s.repository.ResolveWorktreePath(task.RepositoryRoot, req.Worktree)
	if err != nil {
		// Worktree doesn't exist, mark for creation
		task.AutoCreateWorktree = true

		// Set base branch
		if req.BaseBranch != "" {
			task.BaseBranch = req.BaseBranch
		} else {
			// Use current branch as base
			currentBranch, err := s.repository.GetCurrentBranch(task.RepositoryRoot)
			if err != nil {
				task.BaseBranch = "main" // fallback
			} else {
				task.BaseBranch = currentBranch
			}
		}
	} else {
		// Worktree exists
		task.WorktreePath = worktreePath
	}

	task.Worktree = req.Worktree
	return nil
}

// createTaskFromFileEntry creates a task from a YAML file entry
func (s *TaskService) createTaskFromFileEntry(entry claude.TaskFileEntry, defaultRepo string, defaultConfig *claude.TaskConfig) (*claude.Task, error) {
	// Validate entry
	if err := s.validateFileEntry(entry); err != nil {
		return nil, err
	}

	// Determine repository for this task
	repoRoot := defaultRepo
	if entry.Repository != "" {
		var err error
		repoRoot, err = s.repository.ResolveRepository(entry.Repository)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve repository: %w", err)
		}
	}

	// Create task
	taskName := entry.Name
	// Keep taskName empty if not provided to allow displayName logic to use prompt

	task := &claude.Task{
		ID:                   entry.ID,
		Name:                 taskName,
		Priority:             claude.Priority(entry.Priority),
		Status:               claude.StatusPending,
		CreatedAt:            time.Now(),
		RepositoryRoot:       repoRoot,
		DependsOn:            entry.DependsOn,
		DependencyPolicy:     entry.DependencyPolicy,
		AgentType:            "claude",
		Prompt:               entry.Prompt,
		FilesToFocus:         entry.FilesToFocus,
		VerificationCommands: entry.VerificationCommands,
	}

	// Set worktree information
	if entry.Worktree != "" {
		task.Worktree = entry.Worktree
		task.BaseBranch = entry.BaseBranch

		// Try to resolve worktree path
		worktreePath, err := s.repository.ResolveWorktreePath(repoRoot, entry.Worktree)
		if err != nil {
			// Worktree doesn't exist, mark for creation
			task.AutoCreateWorktree = true
		} else {
			// Worktree exists
			task.WorktreePath = worktreePath
		}
	}

	// Apply default config
	if defaultConfig != nil {
		task.Config = *defaultConfig
	}

	// Override with task-specific config
	if entry.Config != nil {
		s.mergeTaskConfig(&task.Config, entry.Config)
	}

	// Set defaults
	if task.Priority == 0 {
		task.Priority = 50
	}
	if task.DependencyPolicy == "" {
		task.DependencyPolicy = claude.DependencyPolicyWait
	}

	// Save task
	if err := s.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return task, nil
}

// validateFileEntry validates a task file entry
func (s *TaskService) validateFileEntry(entry claude.TaskFileEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("task ID is required")
	}

	if entry.Worktree == "" {
		return fmt.Errorf("worktree must be specified")
	}

	if entry.BaseBranch == "" {
		return fmt.Errorf("base_branch is required when specifying worktree")
	}

	return nil
}

// mergeTaskConfig merges task-specific config with default config
func (s *TaskService) mergeTaskConfig(defaultConfig *claude.TaskConfig, taskConfig *claude.TaskConfig) {
	if taskConfig.SkipPermissions {
		defaultConfig.SkipPermissions = taskConfig.SkipPermissions
	}
	defaultConfig.AutoCommit = taskConfig.AutoCommit
	defaultConfig.BackupFiles = taskConfig.BackupFiles
}

// generateTaskID generates a unique task ID
func (s *TaskService) generateTaskID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
