package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWorktreeJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	wt := Worktree{
		Path:       "/path/to/worktree",
		Branch:     "feature/test",
		CommitHash: "abc123def456",
		IsMain:     false,
		CreatedAt:  now,
	}

	// Test marshaling
	data, err := json.Marshal(wt)
	if err != nil {
		t.Fatalf("Failed to marshal Worktree: %v", err)
	}

	// Test unmarshaling
	var decoded Worktree
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Worktree: %v", err)
	}

	// Verify fields
	if decoded.Path != wt.Path {
		t.Errorf("Path mismatch: got %s, want %s", decoded.Path, wt.Path)
	}
	if decoded.Branch != wt.Branch {
		t.Errorf("Branch mismatch: got %s, want %s", decoded.Branch, wt.Branch)
	}
	if decoded.CommitHash != wt.CommitHash {
		t.Errorf("CommitHash mismatch: got %s, want %s", decoded.CommitHash, wt.CommitHash)
	}
	if decoded.IsMain != wt.IsMain {
		t.Errorf("IsMain mismatch: got %v, want %v", decoded.IsMain, wt.IsMain)
	}
	if !decoded.CreatedAt.Equal(wt.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, wt.CreatedAt)
	}
}

func TestBranchJSON(t *testing.T) {
	commitDate := time.Now().Truncate(time.Second)
	branch := Branch{
		Name:      "main",
		IsCurrent: true,
		IsRemote:  false,
		LastCommit: CommitInfo{
			Hash:    "abc123",
			Message: "Initial commit",
			Author:  "Test User",
			Date:    commitDate,
		},
	}

	// Test marshaling
	data, err := json.Marshal(branch)
	if err != nil {
		t.Fatalf("Failed to marshal Branch: %v", err)
	}

	// Test unmarshaling
	var decoded Branch
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Branch: %v", err)
	}

	// Verify fields
	if decoded.Name != branch.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, branch.Name)
	}
	if decoded.IsCurrent != branch.IsCurrent {
		t.Errorf("IsCurrent mismatch: got %v, want %v", decoded.IsCurrent, branch.IsCurrent)
	}
	if decoded.IsRemote != branch.IsRemote {
		t.Errorf("IsRemote mismatch: got %v, want %v", decoded.IsRemote, branch.IsRemote)
	}
	if decoded.LastCommit.Hash != branch.LastCommit.Hash {
		t.Errorf("LastCommit.Hash mismatch: got %s, want %s", decoded.LastCommit.Hash, branch.LastCommit.Hash)
	}
	if decoded.LastCommit.Message != branch.LastCommit.Message {
		t.Errorf("LastCommit.Message mismatch: got %s, want %s", decoded.LastCommit.Message, branch.LastCommit.Message)
	}
	if decoded.LastCommit.Author != branch.LastCommit.Author {
		t.Errorf("LastCommit.Author mismatch: got %s, want %s", decoded.LastCommit.Author, branch.LastCommit.Author)
	}
	if !decoded.LastCommit.Date.Equal(branch.LastCommit.Date) {
		t.Errorf("LastCommit.Date mismatch: got %v, want %v", decoded.LastCommit.Date, branch.LastCommit.Date)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		Worktree: WorktreeConfig{
			BaseDir:   "~/worktrees",
			AutoMkdir: true,
		},
		Finder: FinderConfig{
			Preview: true,
		},
		UI: UIConfig{
			Icons: true,
		},
	}

	// Verify default values
	if cfg.Worktree.BaseDir != "~/worktrees" {
		t.Errorf("Default BaseDir mismatch: got %s, want ~/worktrees", cfg.Worktree.BaseDir)
	}
	if !cfg.Worktree.AutoMkdir {
		t.Error("Default AutoMkdir should be true")
	}
	if !cfg.Finder.Preview {
		t.Error("Default Preview should be true")
	}
	if !cfg.UI.Icons {
		t.Error("Default Icons should be true")
	}
}
