package cmd

import (
	"testing"
	"time"

	"github.com/d-kuro/gwq/pkg/models"
)

func TestSortStatuses(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		statuses []*models.WorktreeStatus
		sortBy   string
		want     []string // expected branch order
	}{
		{
			name: "sort by branch",
			statuses: []*models.WorktreeStatus{
				{Branch: "feature/z"},
				{Branch: "feature/a"},
				{Branch: "main"},
			},
			sortBy: "branch",
			want:   []string{"feature/a", "feature/z", "main"},
		},
		{
			name: "sort by status",
			statuses: []*models.WorktreeStatus{
				{Branch: "clean", Status: models.WorktreeStatusClean},
				{Branch: "conflict", Status: models.WorktreeStatusConflict},
				{Branch: "modified", Status: models.WorktreeStatusModified},
			},
			sortBy: "status",
			want:   []string{"conflict", "modified", "clean"},
		},
		{
			name: "sort by changes",
			statuses: []*models.WorktreeStatus{
				{Branch: "few", GitStatus: models.GitStatus{Modified: 1}},
				{Branch: "many", GitStatus: models.GitStatus{Modified: 10, Added: 5}},
				{Branch: "none", GitStatus: models.GitStatus{}},
			},
			sortBy: "changes",
			want:   []string{"many", "few", "none"},
		},
		{
			name: "sort by activity",
			statuses: []*models.WorktreeStatus{
				{Branch: "old", LastActivity: now.Add(-72 * time.Hour)},
				{Branch: "recent", LastActivity: now.Add(-1 * time.Hour)},
				{Branch: "middle", LastActivity: now.Add(-24 * time.Hour)},
			},
			sortBy: "activity",
			want:   []string{"recent", "middle", "old"},
		},
		{
			name: "sort by ahead",
			statuses: []*models.WorktreeStatus{
				{Branch: "behind", GitStatus: models.GitStatus{Ahead: 0}},
				{Branch: "ahead", GitStatus: models.GitStatus{Ahead: 5}},
				{Branch: "more-ahead", GitStatus: models.GitStatus{Ahead: 10}},
			},
			sortBy: "ahead",
			want:   []string{"more-ahead", "ahead", "behind"},
		},
		{
			name: "sort by behind",
			statuses: []*models.WorktreeStatus{
				{Branch: "up-to-date", GitStatus: models.GitStatus{Behind: 0}},
				{Branch: "behind", GitStatus: models.GitStatus{Behind: 3}},
				{Branch: "far-behind", GitStatus: models.GitStatus{Behind: 10}},
			},
			sortBy: "behind",
			want:   []string{"far-behind", "behind", "up-to-date"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortStatuses(tt.statuses, tt.sortBy)
			
			for i, expected := range tt.want {
				if tt.statuses[i].Branch != expected {
					t.Errorf("sortStatuses() index %d = %s, want %s", i, tt.statuses[i].Branch, expected)
				}
			}
		})
	}
}

func TestGetStatusPriority(t *testing.T) {
	tests := []struct {
		status   models.WorktreeState
		expected int
	}{
		{models.WorktreeStatusConflict, 0},
		{models.WorktreeStatusModified, 1},
		{models.WorktreeStatusStaged, 2},
		{models.WorktreeStatusStale, 3},
		{models.WorktreeStatusClean, 4},
		{models.WorktreeState("unknown"), 999},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := getStatusPriority(tt.status)
			if got != tt.expected {
				t.Errorf("getStatusPriority(%s) = %d, want %d", tt.status, got, tt.expected)
			}
		})
	}
}

func TestCountTotalChanges(t *testing.T) {
	tests := []struct {
		name     string
		status   models.GitStatus
		expected int
	}{
		{
			name:     "no changes",
			status:   models.GitStatus{},
			expected: 0,
		},
		{
			name: "only modified",
			status: models.GitStatus{
				Modified: 5,
			},
			expected: 5,
		},
		{
			name: "all types",
			status: models.GitStatus{
				Modified:  5,
				Added:     3,
				Deleted:   2,
				Untracked: 4,
				Staged:    1,
			},
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countTotalChanges(tt.status)
			if got != tt.expected {
				t.Errorf("countTotalChanges() = %d, want %d", got, tt.expected)
			}
		})
	}
}
