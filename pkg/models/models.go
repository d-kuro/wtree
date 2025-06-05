// Package models defines the core data structures used throughout the gwq application.
package models

import "time"

// Worktree represents a Git worktree with its associated metadata.
type Worktree struct {
	Path       string    `json:"path"`        // Absolute path to the worktree directory
	Branch     string    `json:"branch"`      // Branch name associated with this worktree
	CommitHash string    `json:"commit_hash"` // Current HEAD commit hash
	IsMain     bool      `json:"is_main"`     // Whether this is the main worktree
	CreatedAt  time.Time `json:"created_at"`  // Creation timestamp
}

// Branch represents a Git branch with its metadata.
type Branch struct {
	Name       string     `json:"name"`        // Branch name
	IsCurrent  bool       `json:"is_current"`  // Whether this is the current branch
	IsRemote   bool       `json:"is_remote"`   // Whether this is a remote branch
	LastCommit CommitInfo `json:"last_commit"` // Information about the last commit
}

// CommitInfo contains information about a Git commit.
type CommitInfo struct {
	Hash    string    `json:"hash"`    // Commit hash
	Message string    `json:"message"` // Commit message
	Author  string    `json:"author"`  // Commit author
	Date    time.Time `json:"date"`    // Commit date
}

// Config represents the application configuration.
type Config struct {
	Worktree WorktreeConfig `mapstructure:"worktree"` // Worktree-related configuration
	Finder   FinderConfig   `mapstructure:"finder"`   // Fuzzy finder configuration
	Naming   NamingConfig   `mapstructure:"naming"`   // Naming convention configuration
	UI       UIConfig       `mapstructure:"ui"`       // UI-related configuration
}

// WorktreeConfig contains worktree-specific configuration options.
type WorktreeConfig struct {
	BaseDir   string `mapstructure:"basedir"`    // Base directory for creating worktrees
	AutoMkdir bool   `mapstructure:"auto_mkdir"` // Automatically create directories
}

// FinderConfig contains fuzzy finder configuration options.
type FinderConfig struct {
	Preview       bool   `mapstructure:"preview"`        // Enable preview window
	PreviewSize   int    `mapstructure:"preview_size"`   // Preview window size
	KeybindSelect string `mapstructure:"keybind_select"` // Key binding for selection
	KeybindCancel string `mapstructure:"keybind_cancel"` // Key binding for cancellation
}

// NamingConfig contains worktree naming convention configuration.
type NamingConfig struct {
	Template      string            `mapstructure:"template"`       // Directory name template
	SanitizeChars map[string]string `mapstructure:"sanitize_chars"` // Character replacements for sanitization
}

// UIConfig contains UI-related configuration options.
type UIConfig struct {
	Icons     bool `mapstructure:"icons"`      // Enable icon display
	TildeHome bool `mapstructure:"tilde_home"` // Display home directory as ~
}

// WorktreeStatus represents the current status of a worktree.
type WorktreeStatus struct {
	Path           string           `json:"path"`             // Absolute path to the worktree
	Branch         string           `json:"branch"`           // Branch name
	Repository     string           `json:"repository"`       // Repository identifier
	Status         WorktreeState    `json:"status"`           // Current status (clean, modified, etc.)
	GitStatus      GitStatus        `json:"git_status"`       // Detailed git status
	LastActivity   time.Time        `json:"last_activity"`    // Last modification time
	ActiveProcess  []ProcessInfo    `json:"active_processes"` // Running processes
	IsCurrent      bool             `json:"is_current"`       // Whether this is the current worktree
}

// WorktreeState represents the overall state of a worktree.
type WorktreeState string

const (
	WorktreeStatusClean    WorktreeState = "clean"
	WorktreeStatusModified WorktreeState = "modified"
	WorktreeStatusStaged   WorktreeState = "staged"
	WorktreeStatusConflict WorktreeState = "conflict"
	WorktreeStatusStale    WorktreeState = "stale"
)

// GitStatus contains detailed git status information.
type GitStatus struct {
	Modified  int `json:"modified"`  // Number of modified files
	Added     int `json:"added"`     // Number of added files
	Deleted   int `json:"deleted"`   // Number of deleted files
	Untracked int `json:"untracked"` // Number of untracked files
	Staged    int `json:"staged"`    // Number of staged files
	Ahead     int `json:"ahead"`     // Number of commits ahead of remote
	Behind    int `json:"behind"`    // Number of commits behind remote
	Conflicts int `json:"conflicts"` // Number of files with conflicts
}

// ProcessInfo represents information about a running process.
type ProcessInfo struct {
	PID     int    `json:"pid"`     // Process ID
	Command string `json:"command"` // Command name
	Type    string `json:"type"`    // Process type (e.g., "ai_agent")
}