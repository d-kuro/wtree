// Package utils provides generic utility functions for the gwq application.
package utils

import (
	"cmp"
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