package claude

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/d-kuro/gwq/pkg/utils"
	"gopkg.in/yaml.v3"
)

// TaskManager handles task operations with simplified architecture
type TaskManager struct {
	storage   *Storage
	config    *models.Config
	gitClient *git.Git
}

// NewTaskManager creates a new task manager
func NewTaskManager(storage *Storage, config *models.Config) *TaskManager {
	// Initialize git client for current directory (will be updated per task)
	// We allow this to be nil since tasks can specify their own repositories
	gitClient, _ := git.NewFromCwd()

	return &TaskManager{
		storage:   storage,
		config:    config,
		gitClient: gitClient,
	}
}

// CreateTaskRequest represents a simplified task creation request
type CreateTaskRequest struct {
	Name                 string
	Worktree             string
	BaseBranch           string
	Priority             int
	DependsOn            []string
	Prompt               string
	FilesToFocus         []string
	VerificationCommands []string
	AutoCommit           bool
	Repository           string
}

// CreateTask creates a new task with simplified logic
func (tm *TaskManager) CreateTask(req *CreateTaskRequest) (*Task, error) {
	// Basic validation
	if req.Name == "" {
		return nil, fmt.Errorf("task name is required")
	}
	if req.Worktree == "" {
		return nil, fmt.Errorf("worktree must be specified")
	}
	if req.Priority < 1 || req.Priority > 100 {
		return nil, fmt.Errorf("priority must be between 1 and 100")
	}

	// Resolve repository using existing git package
	repoRoot, err := tm.resolveRepository(req.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repository: %w", err)
	}

	// Create simplified task with essential fields only
	simplifiedTask := NewSimplifiedTask(
		utils.GenerateShortID(),
		req.Name,
		req.Worktree,
		req.Prompt,
		Priority(req.Priority),
	)
	simplifiedTask.DependsOn = req.DependsOn

	// Convert to legacy format for storage compatibility
	task := simplifiedTask.ToLegacyTask()

	// Setup worktree information
	if err := tm.setupWorktree(task, req, repoRoot); err != nil {
		return nil, err
	}

	// Save task
	if err := tm.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return task, nil
}

// CreateTasksFromFile creates multiple tasks from a YAML file
func (tm *TaskManager) CreateTasksFromFile(filePath string) ([]*Task, error) {
	// Read YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	// Parse YAML
	var tasksDefinition TaskFile
	if err := yaml.Unmarshal(data, &tasksDefinition); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate version
	if tasksDefinition.Version != "1.0" {
		return nil, fmt.Errorf("unsupported task file version: %s (expected 1.0)", tasksDefinition.Version)
	}

	// Resolve default repository
	defaultRepo, err := tm.resolveRepository(tasksDefinition.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve default repository: %w", err)
	}

	var createdTasks []*Task

	// Process each task
	for _, entry := range tasksDefinition.Tasks {
		task, err := tm.createTaskFromEntry(entry, defaultRepo)
		if err != nil {
			return createdTasks, fmt.Errorf("failed to create task %s: %w", entry.ID, err)
		}
		createdTasks = append(createdTasks, task)
	}

	return createdTasks, nil
}

// FindTaskByPattern finds a task by ID or pattern matching
func (tm *TaskManager) FindTaskByPattern(pattern string) (*Task, error) {
	// Try exact ID match first
	if task, err := tm.storage.LoadTask(pattern); err == nil {
		return task, nil
	}

	// Try pattern matching
	tasks, err := tm.storage.ListTasks()
	if err != nil {
		return nil, err
	}

	var matches []*Task
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
func (tm *TaskManager) FilterTasksByStatus(tasks []*Task, status string) []*Task {
	var filtered []*Task
	for _, task := range tasks {
		if string(task.Status) == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterTasksByPriority filters tasks by minimum priority
func (tm *TaskManager) FilterTasksByPriority(tasks []*Task, minPriority int) []*Task {
	var filtered []*Task
	for _, task := range tasks {
		if int(task.Priority) >= minPriority {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// resolveRepository resolves repository path using existing git package
func (tm *TaskManager) resolveRepository(repo string) (string, error) {
	if repo == "" {
		// Use current directory
		g, err := git.NewFromCwd()
		if err != nil {
			return "", fmt.Errorf("not in a git repository: %w", err)
		}
		return g.GetRepositoryPath()
	}

	// Try to create git client from the specified repository
	g := git.New(repo)

	// Check if it's a repository by trying to get the root path
	rootPath, err := g.GetRepositoryPath()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %s", repo)
	}

	return rootPath, nil
}

// setupWorktree configures worktree information for a task
func (tm *TaskManager) setupWorktree(task *Task, req *CreateTaskRequest, repoRoot string) error {
	// Use existing worktree package for worktree management
	g := git.New(repoRoot)
	wm := worktree.New(g, tm.config)

	// Try to get existing worktree path
	worktreePath, err := wm.GetWorktreePath(req.Worktree)
	if err != nil {
		// Worktree doesn't exist - it will be created by the execution engine
		// Just store the worktree name
		task.Worktree = req.Worktree
		return nil
	}

	// Verify the worktree actually exists
	if _, statErr := os.Stat(worktreePath); statErr != nil {
		// Path configured but doesn't exist
		task.Worktree = req.Worktree
		return nil
	}

	// Worktree exists and is accessible
	task.Worktree = req.Worktree
	return nil
}

// createTaskFromEntry creates a task from a YAML file entry
func (tm *TaskManager) createTaskFromEntry(entry TaskFileEntry, defaultRepo string) (*Task, error) {
	// Basic validation
	if entry.ID == "" {
		return nil, fmt.Errorf("task ID is required")
	}
	if entry.Worktree == "" {
		return nil, fmt.Errorf("worktree must be specified")
	}

	// Determine repository for this task - use defaultRepo unless overridden
	if entry.Repository != "" {
		_, err := tm.resolveRepository(entry.Repository)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve repository: %w", err)
		}
	}

	// Create simplified task using the new model
	priority := Priority(entry.Priority)
	if priority == 0 {
		priority = 50
	}

	simplifiedTask := &SimplifiedTask{
		ID:        entry.ID,
		Name:      entry.Name,
		Worktree:  entry.Worktree,
		Priority:  priority,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		Prompt:    entry.Prompt,
		DependsOn: entry.DependsOn,
	}

	// Convert to legacy format for storage compatibility
	task := simplifiedTask.ToLegacyTask()

	// Save task
	if err := tm.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return task, nil
}
