package tmux

import (
	"testing"
)

func TestDefaultSessionConfig(t *testing.T) {
	config := DefaultSessionConfig()
	
	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}
	
	if config.TmuxCommand != "tmux" {
		t.Errorf("Expected TmuxCommand to be 'tmux', got '%s'", config.TmuxCommand)
	}
	
	if config.HistoryLimit != 50000 {
		t.Errorf("Expected HistoryLimit to be 50000, got %d", config.HistoryLimit)
	}
}


func TestSessionOptionsCreation(t *testing.T) {
	opts := SessionOptions{
		Context:    "test",
		Identifier: "test-session",
		WorkingDir: "/tmp",
		Command:    "echo hello",
		Metadata: map[string]string{
			"created_by": "test",
		},
	}
	
	if opts.Context != "test" {
		t.Errorf("Expected Context to be 'test', got '%s'", opts.Context)
	}
	
	if opts.Identifier != "test-session" {
		t.Errorf("Expected Identifier to be 'test-session', got '%s'", opts.Identifier)
	}
	
	if opts.Command != "echo hello" {
		t.Errorf("Expected Command to be 'echo hello', got '%s'", opts.Command)
	}
}