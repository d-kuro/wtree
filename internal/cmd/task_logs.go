package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/d-kuro/gwq/internal/claude"
	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/tui"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

var taskLogsCmd = &cobra.Command{
	Use:   "logs [EXECUTION_ID] [flags]",
	Short: "View and manage Claude execution logs",
	Long: `View and manage Claude execution logs with fuzzy finder selection.

This command provides access to all Claude execution logs, allowing you to:
- Browse execution history with interactive selection
- View formatted output from past executions  
- Filter logs by status, date, or content
- Clean up old logs

Logs are stored in ~/.config/gwq/claude/logs/ and include both raw JSON
output and formatted metadata for easy browsing.`,
	Example: `  # Interactive log selection (default)
  gwq task logs
  
  # Show specific execution log
  gwq task logs exec-a1b2c3
  
  # Filter by status
  gwq task logs --status running
  gwq task logs --status completed
  
  # Filter by date
  gwq task logs --date 2024-01-15
  
  # Search logs containing text
  gwq task logs --contains "authentication"
  
  # Clean up old logs
  gwq task logs clean --older-than 30d`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTaskLogsMain,
}

var taskLogsCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up old logs",
	Long: `Clean up old execution logs based on age or size limits.

This helps manage disk usage by removing old logs while preserving
recent execution history.`,
	RunE: runTaskLogsClean,
}

// Flags for logs command
var (
	taskLogsStatus    string
	taskLogsDate      string
	taskLogsContains  string
	taskLogsLimit     int
	taskLogsJSON      bool
	taskLogsOlderThan string
	taskLogsPlain     bool
)

func init() {
	taskCmd.AddCommand(taskLogsCmd)
	taskLogsCmd.AddCommand(taskLogsCleanCmd)

	// List command flags
	taskLogsCmd.Flags().StringVar(&taskLogsStatus, "status", "", "Filter by status (running, completed, failed)")
	taskLogsCmd.Flags().StringVar(&taskLogsDate, "date", "", "Filter by date (YYYY-MM-DD)")
	taskLogsCmd.Flags().StringVar(&taskLogsContains, "contains", "", "Filter by content containing text")
	taskLogsCmd.Flags().IntVar(&taskLogsLimit, "limit", 20, "Limit number of results")
	taskLogsCmd.Flags().BoolVar(&taskLogsJSON, "json", false, "Output in JSON format")
	taskLogsCmd.Flags().BoolVar(&taskLogsPlain, "plain", false, "Use plain text output instead of TUI")

	// Clean command flags
	taskLogsCleanCmd.Flags().StringVar(&taskLogsOlderThan, "older-than", "30d", "Remove logs older than specified duration (e.g., 30d, 1w)")
}

func runTaskLogsMain(cmd *cobra.Command, args []string) error {
	// If execution ID is provided as argument, show that specific execution
	if len(args) > 0 {
		return runTaskLogsShow(cmd, args)
	}

	// Otherwise, run the interactive list
	return runTaskLogsList(cmd, args)
}

func runTaskLogsList(cmd *cobra.Command, args []string) error {
	// Create execution manager
	execMgr, err := createTaskExecutionManager()
	if err != nil {
		return err
	}

	// Load executions from metadata directory
	executions, err := loadTaskExecutionsFromMetadata(execMgr)
	if err != nil {
		return fmt.Errorf("failed to load executions: %w", err)
	}

	// Apply filters
	if taskLogsStatus != "" {
		executions = filterTaskExecutionsByStatus(executions, taskLogsStatus)
	}
	if taskLogsDate != "" {
		executions = filterTaskExecutionsByDate(executions, taskLogsDate)
	}
	if taskLogsContains != "" {
		executions = filterTaskExecutionsByContent(executions, taskLogsContains, execMgr)
	}

	// Limit results
	if len(executions) > taskLogsLimit {
		executions = executions[:taskLogsLimit]
	}

	// Output format
	if taskLogsJSON {
		return outputTaskExecutionsJSON(executions)
	}

	// Interactive selection
	if len(executions) == 0 {
		fmt.Println("No executions found.")
		return nil
	}

	// Show fuzzy finder for selection
	selectedExecution, err := selectTaskExecutionWithFinder(executions)
	if err != nil {
		return fmt.Errorf("failed to select execution: %w", err)
	}

	if selectedExecution == nil {
		// User cancelled
		return nil
	}

	// Show the selected execution
	return showTaskExecution(selectedExecution, execMgr)
}

func runTaskLogsShow(cmd *cobra.Command, args []string) error {
	execMgr, err := createTaskExecutionManager()
	if err != nil {
		return err
	}

	var executionID string
	if len(args) > 0 {
		executionID = args[0]
	} else {
		// Interactive selection
		executions, err := loadTaskExecutionsFromMetadata(execMgr)
		if err != nil {
			return fmt.Errorf("failed to load executions: %w", err)
		}

		selectedExecution, err := selectTaskExecutionWithFinder(executions)
		if err != nil {
			return fmt.Errorf("failed to select execution: %w", err)
		}

		if selectedExecution == nil {
			return nil
		}

		executionID = selectedExecution.ExecutionID
	}

	// Load metadata
	metadata, err := execMgr.LoadMetadata(executionID)
	if err != nil {
		return fmt.Errorf("failed to load metadata for %s: %w", executionID, err)
	}

	return showTaskExecution(metadata, execMgr)
}

func runTaskLogsClean(cmd *cobra.Command, args []string) error {
	execMgr, err := createTaskExecutionManager()
	if err != nil {
		return err
	}

	// Parse duration
	duration, err := time.ParseDuration(taskLogsOlderThan)
	if err != nil {
		// Try parsing as days (e.g., "30d")
		if strings.HasSuffix(taskLogsOlderThan, "d") {
			days := strings.TrimSuffix(taskLogsOlderThan, "d")
			if d, err := time.ParseDuration(days + "h"); err == nil {
				duration = d * 24
			} else {
				return fmt.Errorf("invalid duration format: %s", taskLogsOlderThan)
			}
		} else {
			return fmt.Errorf("invalid duration format: %s", taskLogsOlderThan)
		}
	}

	cutoff := time.Now().Add(-duration)

	fmt.Printf("Cleaning logs older than %v (before %s)\n", duration, cutoff.Format("2006-01-02 15:04:05"))

	// Load executions and filter old ones
	executions, err := loadTaskExecutionsFromMetadata(execMgr)
	if err != nil {
		return fmt.Errorf("failed to load executions: %w", err)
	}

	var toDelete []claude.ExecutionMetadata
	for _, exec := range executions {
		if exec.StartTime.Before(cutoff) && exec.Status != claude.ExecutionStatusRunning {
			toDelete = append(toDelete, exec)
		}
	}

	if len(toDelete) == 0 {
		fmt.Println("No old logs found to clean.")
		return nil
	}

	fmt.Printf("Found %d old executions to clean. Continue? [y/N]: ", len(toDelete))
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		return err
	}

	if strings.ToLower(response) != "y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete log files and metadata
	cfg := config.Get()
	logDir := filepath.Join(cfg.Claude.ConfigDir, "logs")
	deletedCount := 0

	for _, exec := range toDelete {
		// Delete log file using new helper function
		logFile := claude.FindLogFileByExecutionID(logDir, exec.StartTime, exec.ExecutionID)
		if err := os.Remove(logFile); err == nil {
			deletedCount++
		}

		// Delete metadata file - try both new and old formats
		newMetadataFile := filepath.Join(logDir, "metadata", claude.GenerateMetadataFileName(exec.StartTime, exec.ExecutionID))
		oldMetadataFile := filepath.Join(logDir, "metadata", fmt.Sprintf("%s.json", exec.ExecutionID))

		// Try new format first, then old format
		metadataFile := newMetadataFile
		if _, err := os.Stat(newMetadataFile); os.IsNotExist(err) {
			metadataFile = oldMetadataFile
		}
		if err := os.Remove(metadataFile); err != nil {
			// Ignore errors for metadata files as they're not critical
			fmt.Printf("Warning: failed to delete metadata file %s: %v\n", metadataFile, err)
		}
	}

	fmt.Printf("Cleaned %d log files.\n", deletedCount)

	// TODO: Update index to remove deleted entries

	return nil
}

// Helper functions

// createTaskExecutionManager creates a new execution manager with error handling
func createTaskExecutionManager() (*claude.ExecutionManager, error) {
	cfg := config.Get()
	execMgr, err := claude.NewExecutionManager(&cfg.Claude)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution manager: %w", err)
	}
	return execMgr, nil
}

func loadTaskExecutionsFromMetadata(execMgr *claude.ExecutionManager) ([]claude.ExecutionMetadata, error) {
	// Load executions directly from metadata directory - no index file needed
	metadataDir := filepath.Join(execMgr.GetLogDir(), "metadata")

	// Read all metadata files
	files, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []claude.ExecutionMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var executions []claude.ExecutionMetadata
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		metadataFile := filepath.Join(metadataDir, file.Name())
		data, err := os.ReadFile(metadataFile)
		if err != nil {
			fmt.Printf("Warning: failed to read metadata file %s: %v\n", metadataFile, err)
			continue
		}

		var execution claude.ExecutionMetadata
		if err := json.Unmarshal(data, &execution); err != nil {
			fmt.Printf("Warning: failed to unmarshal metadata file %s: %v\n", metadataFile, err)
			continue
		}

		// Verify that corresponding log file exists using new helper function
		logFile := claude.FindLogFileByExecutionID(execMgr.GetLogDir(), execution.StartTime, execution.ExecutionID)
		if _, err := os.Stat(logFile); err == nil {
			executions = append(executions, execution)
		} else {
			// Update status to indicate missing log file
			execution.Status = claude.ExecutionStatusAborted
			executions = append(executions, execution)
		}
	}

	// Sort by start time (newest first)
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].StartTime.After(executions[j].StartTime)
	})

	return executions, nil
}

func filterTaskExecutionsByStatus(executions []claude.ExecutionMetadata, status string) []claude.ExecutionMetadata {
	var filtered []claude.ExecutionMetadata
	for _, exec := range executions {
		if string(exec.Status) == status {
			filtered = append(filtered, exec)
		}
	}
	return filtered
}

func filterTaskExecutionsByDate(executions []claude.ExecutionMetadata, date string) []claude.ExecutionMetadata {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return executions // Return unfiltered if invalid date
	}

	var filtered []claude.ExecutionMetadata
	for _, exec := range executions {
		if exec.StartTime.Format("2006-01-02") == date {
			filtered = append(filtered, exec)
		}
	}
	return filtered
}

func filterTaskExecutionsByContent(executions []claude.ExecutionMetadata, text string, execMgr *claude.ExecutionManager) []claude.ExecutionMetadata {
	var filtered []claude.ExecutionMetadata
	lowerText := strings.ToLower(text)

	for _, exec := range executions {
		// Check prompt
		if strings.Contains(strings.ToLower(exec.Prompt), lowerText) {
			filtered = append(filtered, exec)
			continue
		}

		// Check tags
		for _, tag := range exec.Tags {
			if strings.Contains(strings.ToLower(tag), lowerText) {
				filtered = append(filtered, exec)
				break
			}
		}
	}

	return filtered
}

func selectTaskExecutionWithFinder(executions []claude.ExecutionMetadata) (*claude.ExecutionMetadata, error) {
	if len(executions) == 0 {
		return nil, nil
	}

	if len(executions) == 1 {
		return &executions[0], nil
	}

	// Use go-fuzzyfinder directly
	opts := []fuzzyfinder.Option{
		fuzzyfinder.WithPromptString("Select Execution> "),
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			exec := executions[i]
			return fmt.Sprintf("Execution: %s\nStatus: %s\nStarted: %s\nPrompt: %s",
				exec.ExecutionID,
				exec.Status,
				exec.StartTime.Format("2006-01-02 15:04:05"),
				exec.Prompt)
		}),
	}

	idx, err := fuzzyfinder.Find(
		executions,
		func(i int) string {
			exec := executions[i]
			status := string(exec.Status)
			relativeTime := formatTaskRelativeTime(exec.StartTime)

			// Get branch info from working directory or use "no-branch"
			branch := "no-branch"
			if strings.Contains(exec.WorkingDirectory, "/.worktrees/") {
				// Extract branch from worktree path
				parts := strings.Split(exec.WorkingDirectory, "/.worktrees/")
				if len(parts) > 1 {
					branchParts := strings.Split(parts[1], "-")
					if len(branchParts) > 0 {
						branch = strings.Join(branchParts[:len(branchParts)-1], "-")
					}
				}
			} else if exec.WorkingDirectory != "" {
				// Assume we're on the default branch if not in a worktree
				branch = "main"
			}

			// Format: [status] exec-id (~/path/to/repo on branch) - time ago
			return fmt.Sprintf("[%s] %s (%s on %s) - %s",
				status, exec.ExecutionID, exec.WorkingDirectory, branch, relativeTime)
		},
		opts...,
	)

	if err != nil {
		return nil, err
	}

	return &executions[idx], nil
}

func showTaskExecution(metadata *claude.ExecutionMetadata, execMgr *claude.ExecutionManager) error {
	// Check if log file exists using new helper function
	logFile := claude.FindLogFileByExecutionID(execMgr.GetLogDir(), metadata.StartTime, metadata.ExecutionID)

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		// Show metadata-only view for executions without log files
		fmt.Printf("Execution: %s\n", metadata.ExecutionID)
		fmt.Printf("Status: ⊘ Aborted (log file missing) • Started: %s", metadata.StartTime.Format("2006-01-02 15:04:05"))
		if metadata.Repository != "" {
			fmt.Printf(" • Repository: %s", metadata.Repository)
		}
		fmt.Printf("\n")
		fmt.Printf("\n💬 Prompt:\n%s\n", metadata.Prompt)
		fmt.Printf("\n⚠️  Log file not found. This execution may have been interrupted or not properly initialized.\n")
		return nil
	}

	// Load and format the log
	processor := claude.NewLogProcessor()
	formatted, err := processor.ProcessExecution(metadata, execMgr)
	if err != nil {
		return fmt.Errorf("failed to process log: %w", err)
	}

	// Use TUI if not plain mode and if we're in a terminal
	if !taskLogsPlain && os.Getenv("TERM") != "" {
		return tui.RunLogViewer(metadata, formatted)
	}

	// Fallback to plain text output
	fmt.Print(formatted)
	return nil
}

func outputTaskExecutionsJSON(executions []claude.ExecutionMetadata) error {
	data, err := json.MarshalIndent(executions, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func formatTaskRelativeTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}
