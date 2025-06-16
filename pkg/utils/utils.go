// Package utils provides generic utility functions for the gwq application.
package utils

import (
	"cmp"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Min returns the minimum of two ordered values.
func Min[T cmp.Ordered](a, b T) T {
	return min(a, b)
}

// Max returns the maximum of two ordered values.
func Max[T cmp.Ordered](a, b T) T {
	return max(a, b)
}

// Filter returns a new slice containing only elements that match the predicate.
func Filter[T any](slice []T, predicate func(T) bool) []T {
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms a slice of one type to a slice of another type.
func Map[T, U any](slice []T, transform func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = transform(item)
	}
	return result
}

// Find returns the first element in the slice that matches the predicate,
// along with a boolean indicating whether such an element was found.
func Find[T any](slice []T, predicate func(T) bool) (T, bool) {
	var zero T
	for _, item := range slice {
		if predicate(item) {
			return item, true
		}
	}
	return zero, false
}

// Contains checks if a slice contains a specific element.
func Contains[T comparable](slice []T, element T) bool {
	return slices.Contains(slice, element)
}

// Unique returns a new slice with duplicate elements removed.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0, len(slice))
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// ExpandPath expands environment variables, tilde (~), and converts relative paths to absolute paths.
// It returns an error if the path cannot be expanded.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Step 1: Expand environment variables
	path = os.ExpandEnv(path)

	// Step 2: Expand tilde (~)
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = home
	}

	// Step 3: Convert to absolute path if relative
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		path = absPath
	}

	return path, nil
}

// TildePath replaces the home directory portion of a path with ~.
// If the path doesn't start with the home directory, it returns the original path.
func TildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	// Ensure we have clean paths for comparison
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(home)

	// Check if the path starts with the home directory
	if strings.HasPrefix(cleanPath, cleanHome) {
		// Check if it's exactly the home directory or has a path separator after it
		if len(cleanPath) == len(cleanHome) {
			return "~"
		}
		if len(cleanPath) > len(cleanHome) && cleanPath[len(cleanHome)] == filepath.Separator {
			return "~" + cleanPath[len(cleanHome):]
		}
	}

	return path
}

// GenerateID generates a random ID (12 characters).
func GenerateID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a basic ID if crypto/rand fails
		return fmt.Sprintf("%012x", len(b)*1000000)
	}
	return hex.EncodeToString(b)
}

// GenerateShortID generates a short random ID (6 characters).
func GenerateShortID() string {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a basic short ID if crypto/rand fails
		return fmt.Sprintf("%06x", len(b)*1000000)
	}
	return hex.EncodeToString(b)
}

// GenerateUUID generates a UUID-like string.
func GenerateUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a deterministic UUID-like string if crypto/rand fails
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			0, 0, 0, 0, 0)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// SanitizeForFilesystem converts strings to filesystem-safe names by replacing problematic characters.
func SanitizeForFilesystem(input string) string {
	// Replace problematic characters
	replacements := map[string]string{
		"/":  "-",
		"\\": "-",
		":":  "-",
		"*":  "-",
		"?":  "-",
		"\"": "-",
		"<":  "-",
		">":  "-",
		"|":  "-",
	}

	result := input
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

// EscapeForShell escapes a string for safe shell usage by escaping special characters.
func EscapeForShell(s string) string {
	// Replace problematic characters with escaped versions
	s = strings.ReplaceAll(s, `\`, `\\`)  // Escape backslashes first
	s = strings.ReplaceAll(s, `"`, `\"`)  // Escape double quotes
	s = strings.ReplaceAll(s, `$`, `\$`)  // Escape dollar signs (variable expansion)
	s = strings.ReplaceAll(s, "`", "\\`") // Escape backticks (command substitution)
	return s
}
