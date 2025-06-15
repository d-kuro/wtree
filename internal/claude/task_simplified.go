package claude

import (
	"time"
)

// SimplifiedTask represents the essential 9-field task structure
// This replaces the 22-field Task struct with only core functionality
type SimplifiedTask struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Worktree  string      `json:"worktree"`
	Priority  Priority    `json:"priority"`
	Status    Status      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	Prompt    string      `json:"prompt"`
	DependsOn []string    `json:"depends_on"`
	Result    *TaskResult `json:"result,omitempty"`
}

// TaskExtensions holds optional/computed fields for backward compatibility
type TaskExtensions struct {
	StartedAt            *time.Time       `json:"started_at,omitempty"`
	CompletedAt          *time.Time       `json:"completed_at,omitempty"`
	WorktreePath         string           `json:"worktree_path,omitempty"`
	RepositoryRoot       string           `json:"repository_root,omitempty"`
	SessionID            string           `json:"session_id,omitempty"`
	VerificationCommands []string         `json:"verification_commands,omitempty"`
	BaseBranch           string           `json:"base_branch,omitempty"`
	AgentType            string           `json:"agent_type,omitempty"`
	Blocks               []string         `json:"blocks,omitempty"`
	DependencyPolicy     DependencyPolicy `json:"dependency_policy,omitempty"`
	FilesToFocus         []string         `json:"files_to_focus,omitempty"`
	Config               TaskConfig       `json:"config,omitempty"`
	AutoCreateWorktree   bool             `json:"auto_create_worktree,omitempty"`
}

// NewSimplifiedTask creates a new simplified task with essential fields
func NewSimplifiedTask(id, name, worktree, prompt string, priority Priority) *SimplifiedTask {
	return &SimplifiedTask{
		ID:        id,
		Name:      name,
		Worktree:  worktree,
		Priority:  priority,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		Prompt:    prompt,
		DependsOn: []string{},
	}
}

// GetDisplayName returns the display name, falling back to prompt if name is empty
func (st *SimplifiedTask) GetDisplayName() string {
	if st.Name != "" {
		return st.Name
	}
	if len(st.Prompt) > 50 {
		return st.Prompt[:47] + "..."
	}
	return st.Prompt
}

// IsCompleted checks if the task is in a completed state
func (st *SimplifiedTask) IsCompleted() bool {
	return st.Status == StatusCompleted
}

// IsFailed checks if the task is in a failed state
func (st *SimplifiedTask) IsFailed() bool {
	return st.Status == StatusFailed
}

// IsRunning checks if the task is currently running
func (st *SimplifiedTask) IsRunning() bool {
	return st.Status == StatusRunning
}

// GetDuration returns the task duration if available
func (st *SimplifiedTask) GetDuration() time.Duration {
	if st.Result != nil {
		return st.Result.Duration
	}
	return 0
}

// ToLegacyTask converts a SimplifiedTask to the legacy Task format for backward compatibility
func (st *SimplifiedTask) ToLegacyTask() *Task {
	task := &Task{
		ID:        st.ID,
		Name:      st.Name,
		Worktree:  st.Worktree,
		Priority:  st.Priority,
		Status:    st.Status,
		CreatedAt: st.CreatedAt,
		Prompt:    st.Prompt,
		DependsOn: st.DependsOn,
		Result:    st.Result,

		// Set reasonable defaults for legacy fields
		AgentType:        "claude",
		DependencyPolicy: DependencyPolicyWait,
		Config: TaskConfig{
			SkipPermissions: true,
			AutoCommit:      false,
			BackupFiles:     false,
		},
	}

	// Copy timing fields if result is available
	if st.Result != nil && st.Status == StatusCompleted {
		// Estimate completion time based on duration if not available
		completedAt := st.CreatedAt.Add(st.Result.Duration)
		task.CompletedAt = &completedAt
	}

	return task
}

// FromLegacyTask creates a SimplifiedTask from a legacy Task
func FromLegacyTask(task *Task) *SimplifiedTask {
	st := &SimplifiedTask{
		ID:        task.ID,
		Name:      task.Name,
		Worktree:  task.Worktree,
		Priority:  task.Priority,
		Status:    task.Status,
		CreatedAt: task.CreatedAt,
		Prompt:    task.Prompt,
		DependsOn: task.DependsOn,
		Result:    task.Result,
	}

	// Preserve timing information by calculating duration if available
	if task.StartedAt != nil && task.CompletedAt != nil && st.Result != nil {
		st.Result.Duration = task.CompletedAt.Sub(*task.StartedAt)
	}

	return st
}
