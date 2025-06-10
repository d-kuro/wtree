package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/tmux"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/d-kuro/gwq/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	tmuxListVerbose bool
	tmuxListJSON    bool
	tmuxListCSV     bool
	tmuxListWatch   bool
	tmuxListFilter  string
	tmuxListSort    string
)

var tmuxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active tmux sessions",
	Long: `List active tmux sessions with their status and information.

Shows running tmux sessions with context, identifier, status, duration and command.
Supports various output formats and filtering options.`,
	Example: `  # Simple session list
  gwq tmux list

  # Detailed information
  gwq tmux list --verbose

  # JSON output for scripting  
  gwq tmux list --json

  # Filter by status
  gwq tmux list --filter running
  gwq tmux list --filter completed

  # Sort by duration
  gwq tmux list --sort duration`,
	RunE: runTmuxList,
}

func init() {
	tmuxCmd.AddCommand(tmuxListCmd)

	tmuxListCmd.Flags().BoolVarP(&tmuxListVerbose, "verbose", "v", false, "Show detailed information")
	tmuxListCmd.Flags().BoolVar(&tmuxListJSON, "json", false, "Output as JSON")
	tmuxListCmd.Flags().BoolVar(&tmuxListCSV, "csv", false, "Output as CSV")
	tmuxListCmd.Flags().BoolVarP(&tmuxListWatch, "watch", "w", false, "Real-time monitoring")
	tmuxListCmd.Flags().StringVarP(&tmuxListFilter, "filter", "f", "", "Filter by status (running, completed, failed)")
	tmuxListCmd.Flags().StringVarP(&tmuxListSort, "sort", "s", "", "Sort by field (duration, context, identifier)")
}

func runTmuxList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dataDir := filepath.Join(cfg.Worktree.BaseDir, ".gwq")
	sessionManager := tmux.NewSessionManager(nil, dataDir)

	if tmuxListWatch {
		return runTmuxListWatch(sessionManager, cfg)
	}

	return runTmuxListOnce(sessionManager, cfg)
}

func runTmuxListOnce(sessionManager *tmux.SessionManager, cfg *models.Config) error {
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	filteredSessions := applySessionFilters(sessions, tmuxListFilter)
	sortedSessions := applySessionSort(filteredSessions, tmuxListSort)

	switch {
	case tmuxListJSON:
		return outputSessionsJSON(sortedSessions)
	case tmuxListCSV:
		return outputSessionsCSV(sortedSessions)
	default:
		printer := ui.New(&cfg.UI)
		return outputSessionsTable(sortedSessions, printer, tmuxListVerbose)
	}
}

func runTmuxListWatch(sessionManager *tmux.SessionManager, cfg *models.Config) error {
	printer := ui.New(&cfg.UI)
	
	hideCursor := "\033[?25l"
	showCursor := "\033[?25h"
	clearScreen := "\033[H\033[2J"
	
	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	refresh := func() error {
		fmt.Print(clearScreen)
		
		sessions, err := sessionManager.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		filteredSessions := applySessionFilters(sessions, tmuxListFilter)
		sortedSessions := applySessionSort(filteredSessions, tmuxListSort)

		fmt.Printf("tmux Sessions - Updated: %s\n", time.Now().Format("15:04:05"))
		fmt.Printf("Total: %d | Running: %d\n\n",
			len(sessions), countByStatus(sessions, tmux.StatusRunning))

		if err := outputSessionsTable(sortedSessions, printer, tmuxListVerbose); err != nil {
			return err
		}

		fmt.Println("\n[Press Ctrl+C to exit]")
		return nil
	}

	if err := refresh(); err != nil {
		return err
	}

	for range ticker.C {
		if err := refresh(); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

func applySessionFilters(sessions []*tmux.Session, filter string) []*tmux.Session {
	if filter == "" || filter != "running" {
		return sessions
	}

	var filtered []*tmux.Session
	for _, session := range sessions {
		if session.Status == tmux.StatusRunning {
			filtered = append(filtered, session)
		}
	}

	return filtered
}

func applySessionSort(sessions []*tmux.Session, sortBy string) []*tmux.Session {
	if sortBy == "" {
		return sessions
	}

	// Sorting is not implemented - return as-is
	// TODO: Implement sorting if needed in the future
	return sessions
}

func outputSessionsJSON(sessions []*tmux.Session) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sessions)
}

func outputSessionsCSV(sessions []*tmux.Session) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"Context", "Identifier", "Status", "Duration", "Command", "WorkingDir", "SessionName"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, session := range sessions {
		duration := time.Since(session.StartTime).Round(time.Second).String()
		record := []string{
			session.Context,
			session.Identifier,
			string(session.Status),
			duration,
			session.Command,
			session.WorkingDir,
			session.SessionName,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func outputSessionsTable(sessions []*tmux.Session, printer *ui.Printer, verbose bool) error {
	if len(sessions) == 0 {
		printer.PrintInfo("No tmux sessions found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer func() { _ = w.Flush() }()

	if verbose {
		_, _ = fmt.Fprintln(w, "SESSION\tSTATUS\tDURATION\tWORKING_DIR")
	} else {
		_, _ = fmt.Fprintln(w, "SESSION\tSTATUS\tDURATION")
	}

	for _, session := range sessions {
		// Format session identifier with marker for running sessions
		var sessionWithMarker string
		sessionIdentifier := session.Context + "/" + session.Identifier
		if session.Status == tmux.StatusRunning && printer != nil && printer.UseIcons() {
			sessionWithMarker = "● " + sessionIdentifier
		} else {
			sessionWithMarker = "  " + sessionIdentifier
		}

		status := formatSessionStatus(session.Status)
		duration := formatSessionDuration(session.StartTime)

		if verbose {
			workdir := formatWorkingDir(session.WorkingDir, printer)
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				sessionWithMarker, status, duration, workdir)
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n",
				sessionWithMarker, status, duration)
		}
	}

	return nil
}

func formatSessionStatus(status tmux.Status) string {
	if status == tmux.StatusRunning {
		return "running"
	}
	return string(status)
}

func formatSessionDuration(startTime time.Time) string {
	duration := time.Since(startTime)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min"
		}
		return fmt.Sprintf("%d mins", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	default:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	}
}


func formatWorkingDir(workdir string, printer *ui.Printer) string {
	// Apply tilde home replacement first if enabled
	if printer != nil && printer.UseTildeHome() {
		workdir = utils.TildePath(workdir)
	}
	
	// Then apply truncation if needed
	if len(workdir) > 30 {
		return "..." + workdir[len(workdir)-27:]
	}
	return workdir
}

func countByStatus(sessions []*tmux.Session, status tmux.Status) int {
	count := 0
	for _, session := range sessions {
		if session.Status == status {
			count++
		}
	}
	return count
}