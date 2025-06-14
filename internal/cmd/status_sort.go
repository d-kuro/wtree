package cmd

import (
	"slices"
	"strings"

	"github.com/d-kuro/gwq/pkg/models"
)

// sortStatuses sorts worktree statuses based on the specified field.
func sortStatuses(statuses []*models.WorktreeStatus, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "branch", "name":
		slices.SortFunc(statuses, func(a, b *models.WorktreeStatus) int {
			if a.Branch < b.Branch {
				return -1
			} else if a.Branch > b.Branch {
				return 1
			}
			return 0
		})
	case "status":
		slices.SortFunc(statuses, func(a, b *models.WorktreeStatus) int {
			aPriority := getStatusPriority(a.Status)
			bPriority := getStatusPriority(b.Status)
			if aPriority < bPriority {
				return -1
			} else if aPriority > bPriority {
				return 1
			}
			return 0
		})
	case "modified", "changes":
		slices.SortFunc(statuses, func(a, b *models.WorktreeStatus) int {
			aChanges := countTotalChanges(a.GitStatus)
			bChanges := countTotalChanges(b.GitStatus)
			if aChanges > bChanges {
				return -1
			} else if aChanges < bChanges {
				return 1
			}
			return 0
		})
	case "activity", "time":
		slices.SortFunc(statuses, func(a, b *models.WorktreeStatus) int {
			if a.LastActivity.After(b.LastActivity) {
				return -1
			} else if a.LastActivity.Before(b.LastActivity) {
				return 1
			}
			return 0
		})
	case "ahead":
		slices.SortFunc(statuses, func(a, b *models.WorktreeStatus) int {
			if a.GitStatus.Ahead > b.GitStatus.Ahead {
				return -1
			} else if a.GitStatus.Ahead < b.GitStatus.Ahead {
				return 1
			}
			return 0
		})
	case "behind":
		slices.SortFunc(statuses, func(a, b *models.WorktreeStatus) int {
			if a.GitStatus.Behind > b.GitStatus.Behind {
				return -1
			} else if a.GitStatus.Behind < b.GitStatus.Behind {
				return 1
			}
			return 0
		})
	}
}

// getStatusPriority returns a priority value for sorting statuses.
// Lower values appear first in the sorted list.
func getStatusPriority(status models.WorktreeState) int {
	priorities := map[models.WorktreeState]int{
		models.WorktreeStatusConflict: 0,
		models.WorktreeStatusModified: 1,
		models.WorktreeStatusStaged:   2,
		models.WorktreeStatusStale:    3,
		models.WorktreeStatusClean:    4,
	}

	if priority, ok := priorities[status]; ok {
		return priority
	}
	return 999
}

// countTotalChanges calculates the total number of changes in a git status.
func countTotalChanges(gs models.GitStatus) int {
	return gs.Modified + gs.Added + gs.Deleted + gs.Untracked + gs.Staged
}
