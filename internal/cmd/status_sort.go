package cmd

import (
	"sort"
	"strings"

	"github.com/d-kuro/gwq/pkg/models"
)

// sortStatuses sorts worktree statuses based on the specified field.
func sortStatuses(statuses []*models.WorktreeStatus, sortBy string) {
	switch strings.ToLower(sortBy) {
	case "branch", "name":
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].Branch < statuses[j].Branch
		})
	case "status":
		sort.Slice(statuses, func(i, j int) bool {
			return getStatusPriority(statuses[i].Status) < getStatusPriority(statuses[j].Status)
		})
	case "modified", "changes":
		sort.Slice(statuses, func(i, j int) bool {
			iChanges := countTotalChanges(statuses[i].GitStatus)
			jChanges := countTotalChanges(statuses[j].GitStatus)
			return iChanges > jChanges
		})
	case "activity", "time":
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].LastActivity.After(statuses[j].LastActivity)
		})
	case "ahead":
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].GitStatus.Ahead > statuses[j].GitStatus.Ahead
		})
	case "behind":
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].GitStatus.Behind > statuses[j].GitStatus.Behind
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
