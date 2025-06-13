package presenters

import (
	"encoding/json"
	"fmt"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/mattn/go-runewidth"
)

// LogPresenter handles log display formatting
type LogPresenter struct{}

// NewLogPresenter creates a new log presenter
func NewLogPresenter() *LogPresenter {
	return &LogPresenter{}
}

// OutputExecutionsJSON outputs executions in JSON format
func (p *LogPresenter) OutputExecutionsJSON(executions []claude.ExecutionMetadata) error {
	data, err := json.MarshalIndent(executions, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// ShowExecution displays a formatted execution log
func (p *LogPresenter) ShowExecution(metadata *claude.ExecutionMetadata, logContent string) error {
	if logContent != "" {
		fmt.Print(logContent)
	} else {
		// Show metadata-only view for executions without log files
		p.showMetadataOnly(metadata)
	}
	return nil
}

// ShowExecutionSummary displays a brief summary of executions
func (p *LogPresenter) ShowExecutionSummary(executions []claude.ExecutionMetadata) {
	if len(executions) == 0 {
		fmt.Println("No executions found.")
		return
	}

	fmt.Printf("Found %d execution(s):\n\n", len(executions))

	for _, exec := range executions {
		status := p.getStatusIcon(exec.Status)
		relativeTime := p.formatRelativeTime(exec.StartTime)

		fmt.Printf("%s %s (%s) - %s\n",
			status, exec.ExecutionID, exec.Repository, relativeTime)

		if exec.Prompt != "" {
			prompt := p.truncateString(exec.Prompt, 80)
			fmt.Printf("   %s\n", prompt)
		}
		fmt.Println()
	}
}

// ShowCleanupSummary displays cleanup operation results
func (p *LogPresenter) ShowCleanupSummary(deletedCount int, duration string, cutoffTime string) {
	fmt.Printf("Cleaning logs older than %s (before %s)\n", duration, cutoffTime)

	if deletedCount == 0 {
		fmt.Println("No old logs found to clean.")
	} else {
		fmt.Printf("Cleaned %d log files.\n", deletedCount)
	}
}

// ShowAttachInfo displays information about attaching to a session
func (p *LogPresenter) ShowAttachInfo(executionID, sessionName string) {
	fmt.Printf("Attaching to execution: %s\n", executionID)
	fmt.Printf("Session: %s\n", sessionName)
	fmt.Println("Press Ctrl+B, D to detach")
}

// ShowKillInfo displays information about terminating an execution
func (p *LogPresenter) ShowKillInfo(executionID string) {
	fmt.Printf("Terminating execution: %s\n", executionID)
}

// ShowKillSuccess displays successful termination
func (p *LogPresenter) ShowKillSuccess() {
	fmt.Println("Execution terminated.")
}

// ShowWorkerStatus displays worker status information
func (p *LogPresenter) ShowWorkerStatus(statusCounts map[claude.Status]int, activeSessions int, verbose bool) {
	fmt.Println("Claude Worker Status")
	fmt.Println("===================")

	// Show running status
	if activeSessions > 0 {
		fmt.Printf("Status: Running (%d active sessions)\n", activeSessions)
	} else {
		fmt.Println("Status: Not running")
	}

	// Show task queue statistics
	fmt.Println("\nQueue Statistics:")
	fmt.Printf("  Pending:   %d\n", statusCounts[claude.StatusPending])
	fmt.Printf("  Waiting:   %d\n", statusCounts[claude.StatusWaiting])
	fmt.Printf("  Running:   %d\n", statusCounts[claude.StatusRunning])
	fmt.Printf("  Completed: %d\n", statusCounts[claude.StatusCompleted])
	fmt.Printf("  Failed:    %d\n", statusCounts[claude.StatusFailed])
}

// ShowWorkerStatusVerbose displays detailed worker status
func (p *LogPresenter) ShowWorkerStatusVerbose(statusCounts map[claude.Status]int, sessions interface{}, activeSessions int) {
	p.ShowWorkerStatus(statusCounts, activeSessions, false)

	// Note: sessions interface{} would need to be typed properly based on session structure
	// This is a placeholder for detailed session information
	if activeSessions > 0 {
		fmt.Println("\nActive Sessions:")
		fmt.Printf("  %d sessions are currently active\n", activeSessions)
		// Additional session details would be displayed here
	}
}

// showMetadataOnly displays metadata when log file is missing
func (p *LogPresenter) showMetadataOnly(metadata *claude.ExecutionMetadata) {
	fmt.Printf("╭─ Execution: %s ─────────────────────────────────────────╮\n", metadata.ExecutionID)
	fmt.Printf("│ Status: ⊘ Aborted (log file missing)                     │\n")
	fmt.Printf("│ Started: %-42s │\n", metadata.StartTime.Format("2006-01-02 15:04:05"))
	if metadata.Repository != "" {
		fmt.Printf("│ Repository: %-38s │\n", p.truncateStringWidth(metadata.Repository, 38))
	}
	fmt.Printf("╰───────────────────────────────────────────────────────────╯\n")
	fmt.Printf("\n💬 Prompt:\n%s\n", metadata.Prompt)
	fmt.Printf("\n⚠️  Log file not found. This execution may have been interrupted or not properly initialized.\n")
}

// getStatusIcon returns an icon for execution status
func (p *LogPresenter) getStatusIcon(status claude.ExecutionStatus) string {
	switch status {
	case claude.ExecutionStatusRunning:
		return "●"
	case claude.ExecutionStatusCompleted:
		return "✓"
	case claude.ExecutionStatusFailed:
		return "✗"
	case claude.ExecutionStatusAborted:
		return "⊘"
	default:
		return "?"
	}
}

// formatRelativeTime formats time as relative duration
func (p *LogPresenter) formatRelativeTime(startTime interface{}) string {
	// This would need proper time handling based on the actual time type
	// Placeholder implementation
	return "some time ago"
}

// truncateString truncates a string to maximum length
func (p *LogPresenter) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// truncateStringWidth truncates a string to a given visual width, handling Unicode properly
func (p *LogPresenter) truncateStringWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	width := 0
	var result []rune
	for _, r := range s {
		// Replace newlines and tabs with spaces for display
		if r == '\n' || r == '\t' {
			r = ' '
		}

		runeWidth := runewidth.RuneWidth(r)
		if width+runeWidth > maxWidth {
			// Add ellipsis if truncating
			if maxWidth >= 3 {
				// Remove characters to make room for ellipsis
				for width+3 > maxWidth && len(result) > 0 {
					lastRune := result[len(result)-1]
					width -= runewidth.RuneWidth(lastRune)
					result = result[:len(result)-1]
				}
				result = append(result, '.')
				result = append(result, '.')
				result = append(result, '.')
			}
			break
		}
		width += runeWidth
		result = append(result, r)
	}

	// Pad with spaces to ensure consistent width
	for width < maxWidth {
		result = append(result, ' ')
		width++
	}

	return string(result)
}
