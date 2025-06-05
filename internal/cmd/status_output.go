package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/pkg/models"
)

// outputJSON outputs worktree statuses in JSON format.
func outputJSON(statuses []*models.WorktreeStatus) error {
	summary := calculateSummary(statuses)
	
	output := struct {
		Summary   statusSummary             `json:"summary"`
		Worktrees []*models.WorktreeStatus `json:"worktrees"`
	}{
		Summary:   summary,
		Worktrees: statuses,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputCSV outputs worktree statuses in CSV format.
func outputCSV(statuses []*models.WorktreeStatus) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	headers := []string{
		"branch", "status", "modified", "added", "deleted", 
		"ahead", "behind", "last_activity", "process",
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for _, s := range statuses {
		process := ""
		if len(s.ActiveProcess) > 0 {
			processes := make([]string, len(s.ActiveProcess))
			for i, p := range s.ActiveProcess {
				processes[i] = fmt.Sprintf("%s:%d", p.Command, p.PID)
			}
			process = strings.Join(processes, ",")
		}

		record := []string{
			s.Branch,
			string(s.Status),
			strconv.Itoa(s.GitStatus.Modified),
			strconv.Itoa(s.GitStatus.Added),
			strconv.Itoa(s.GitStatus.Deleted),
			strconv.Itoa(s.GitStatus.Ahead),
			strconv.Itoa(s.GitStatus.Behind),
			s.LastActivity.Format(time.RFC3339),
			process,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// outputTable outputs worktree statuses in table format.
func outputTable(statuses []*models.WorktreeStatus, printer *ui.Printer, verbose bool) error {
	if len(statuses) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer func() { _ = w.Flush() }()

	if verbose {
		_, _ = fmt.Fprintln(w, "BRANCH\tSTATUS\tCHANGES\tAHEAD/BEHIND\tACTIVITY\tPROCESS")
	} else {
		_, _ = fmt.Fprintln(w, "BRANCH\tSTATUS\tCHANGES\tACTIVITY")
	}

	for _, s := range statuses {
		// Apply marker for current worktree, with consistent spacing
		var branchWithMarker string
		if s.IsCurrent && printer != nil && printer.UseIcons() {
			branchWithMarker = "● " + s.Branch
		} else {
			branchWithMarker = "  " + s.Branch  // Two spaces to match "● " width
		}

		status := formatStatusNoColor(s.Status)
		changes := formatChanges(s.GitStatus)
		activity := formatActivity(s.LastActivity)

		if verbose {
			aheadBehind := formatAheadBehind(s.GitStatus.Ahead, s.GitStatus.Behind)
			process := formatProcess(s.ActiveProcess)
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				branchWithMarker, status, changes, aheadBehind, activity, process)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				branchWithMarker, status, changes, activity)
		}
	}

	return nil
}

func formatStatusNoColor(status models.WorktreeState) string {
	switch status {
	case models.WorktreeStatusClean:
		return "up to date"
	case models.WorktreeStatusModified:
		return "changed"
	case models.WorktreeStatusStaged:
		return "staged"
	case models.WorktreeStatusConflict:
		return "conflicted"
	case models.WorktreeStatusStale:
		return "inactive"
	default:
		return string(status)
	}
}

func formatChanges(gs models.GitStatus) string {
	if gs.Modified == 0 && gs.Added == 0 && gs.Deleted == 0 && gs.Untracked == 0 {
		return "-"
	}

	parts := []string{}
	if gs.Added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", gs.Added))
	}
	if gs.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", gs.Modified))
	}
	if gs.Deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", gs.Deleted))
	}
	if gs.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", gs.Untracked))
	}

	return strings.Join(parts, ", ")
}

func formatAheadBehind(ahead, behind int) string {
	return fmt.Sprintf("↑%d ↓%d", ahead, behind)
}

func formatActivity(lastActivity time.Time) string {
	if lastActivity.IsZero() {
		return "unknown"
	}

	duration := time.Since(lastActivity)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

func formatProcess(processes []models.ProcessInfo) string {
	if len(processes) == 0 {
		return "-"
	}

	procs := make([]string, len(processes))
	for i, p := range processes {
		procs[i] = fmt.Sprintf("%s:%d", p.Command, p.PID)
	}
	return strings.Join(procs, ",")
}

