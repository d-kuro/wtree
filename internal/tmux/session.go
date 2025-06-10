package tmux

import (
	"time"
)

type Session struct {
	ID           string            `json:"id"`
	SessionName  string            `json:"session_name"`
	Context      string            `json:"context"`
	Identifier   string            `json:"identifier"`
	WorkingDir   string            `json:"working_dir"`
	Command      string            `json:"command"`
	StartTime    time.Time         `json:"start_time"`
	HistorySize  int               `json:"history_size"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type SessionOptions struct {
	Context    string
	Identifier string
	WorkingDir string
	Command    string
	Metadata   map[string]string
}

type SessionConfig struct {
	Enabled      bool   `toml:"enabled" json:"enabled"`
	TmuxCommand  string `toml:"tmux_command" json:"tmux_command"`
	HistoryLimit int    `toml:"history_limit" json:"history_limit"`
}

func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		Enabled:      true,
		TmuxCommand:  "tmux",
		HistoryLimit: 50000,
	}
}
