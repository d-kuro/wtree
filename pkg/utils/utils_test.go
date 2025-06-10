package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTildePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Home directory",
			input:    home,
			expected: "~",
		},
		{
			name:     "Home subdirectory",
			input:    filepath.Join(home, "Documents"),
			expected: "~/Documents",
		},
		{
			name:     "Deep subdirectory",
			input:    filepath.Join(home, "ghq", "github.com", "d-kuro", "gwq"),
			expected: "~/ghq/github.com/d-kuro/gwq",
		},
		{
			name:     "Non-home path",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "Path similar to home but not home",
			input:    home + "extra",
			expected: home + "extra",
		},
		{
			name:     "Relative path",
			input:    "./relative/path",
			expected: "./relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TildePath(tt.input)
			if result != tt.expected {
				t.Errorf("TildePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTildePathWithDifferentSeparators(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	// Test with unclean paths that might have extra separators
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Home with trailing slash",
			input:    home + string(filepath.Separator),
			expected: "~",
		},
		{
			name:     "Home subdirectory with double separators",
			input:    home + string(filepath.Separator) + string(filepath.Separator) + "Documents",
			expected: "~/Documents",
		},
		{
			name:     "Path with ./ elements",
			input:    filepath.Join(home, ".", "Documents", ".", "Projects"),
			expected: "~/Documents/Projects",
		},
		{
			name:     "Path with ../ elements",
			input:    filepath.Join(home, "Documents", "..", "Downloads"),
			expected: "~/Downloads",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TildePath(tt.input)
			if result != tt.expected {
				t.Errorf("TildePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
