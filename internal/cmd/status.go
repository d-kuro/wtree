package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/discovery"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/d-kuro/gwq/internal/worktree"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/cobra"
)

var (
	statusWatch        bool
	statusInterval     int
	statusFilter       string
	statusSort         string
	statusJSON         bool
	statusCSV          bool
	statusVerbose      bool
	statusGlobal       bool
	statusShowProcess  bool
	statusNoFetch      bool
	statusStaleDays    int
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all worktrees",
	Long: `Show status of all worktrees including git status, recent activity, and optional process information.

This command provides a comprehensive view of all worktrees' current state, which is essential
for managing multiple AI coding agents working in parallel across different worktrees.`,
	Example: `  # Table view with basic status
  gwq status
  
  # JSON output for scripting
  gwq status --json
  
  # Watch mode with 5 second interval
  gwq status --watch
  
  # Include process information
  gwq status --show-processes
  
  # Filter modified worktrees
  gwq status --filter modified
  
  # Global status from anywhere
  gwq status --global`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVarP(&statusWatch, "watch", "w", false, "Auto-refresh mode")
	statusCmd.Flags().IntVarP(&statusInterval, "interval", "i", 5, "Refresh interval in seconds for watch mode")
	statusCmd.Flags().StringVarP(&statusFilter, "filter", "f", "", "Filter by status (changed, up to date, inactive)")
	statusCmd.Flags().StringVarP(&statusSort, "sort", "s", "", "Sort by field (branch, modified, activity)")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
	statusCmd.Flags().BoolVar(&statusCSV, "csv", false, "Output as CSV")
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show additional information")
	statusCmd.Flags().BoolVarP(&statusGlobal, "global", "g", false, "Show all worktrees from base directory")
	statusCmd.Flags().BoolVar(&statusShowProcess, "show-processes", false, "Include running processes (slower)")
	statusCmd.Flags().BoolVar(&statusNoFetch, "no-fetch", false, "Skip remote status check (faster)")
	statusCmd.Flags().IntVar(&statusStaleDays, "stale-days", 14, "Days of inactivity before marking as stale")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusWatch {
		return runStatusWatch(cmd, time.Duration(statusInterval)*time.Second)
	}

	return runStatusOnce(cmd)
}

func runStatusOnce(cmd *cobra.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)
	ctx := context.Background()

	statuses, err := collectWorktreeStatuses(ctx, cfg, printer)
	if err != nil {
		return fmt.Errorf("failed to collect worktree statuses: %w", err)
	}

	statuses = applyFiltersAndSort(statuses)

	return outputStatuses(statuses, printer, cfg)
}

func runStatusWatch(cmd *cobra.Command, interval time.Duration) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)
	
	hideCursor := "\033[?25l"
	showCursor := "\033[?25h"
	clearScreen := "\033[H\033[2J"
	
	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		cancel()
	}()
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	refresh := func() error {
		fmt.Print(clearScreen)
		
		statuses, err := collectWorktreeStatuses(ctx, cfg, printer)
		if err != nil {
			return fmt.Errorf("failed to collect worktree statuses: %w", err)
		}
		
		statuses = applyFiltersAndSort(statuses)
		
		summary := calculateSummary(statuses)
		currentRepo := getCurrentRepository()
		
		fmt.Printf("Worktrees Status (%s) - Updated: %s\n", 
			currentRepo, time.Now().Format("15:04:05"))
		fmt.Printf("Total: %d | Changed: %d | Up to date: %d | Inactive: %d\n\n",
			summary.Total, summary.Modified, summary.Clean, summary.Stale)
		
		if err := outputStatuses(statuses, printer, cfg); err != nil {
			return err
		}
		
		fmt.Println("\n[Press Ctrl+C to exit]")
		return nil
	}
	
	if err := refresh(); err != nil {
		return err
	}
	
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := refresh(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

func collectWorktreeStatuses(ctx context.Context, cfg *models.Config, printer *ui.Printer) ([]*models.WorktreeStatus, error) {
	var worktrees []*models.Worktree
	
	g, err := git.NewFromCwd()
	if err != nil || statusGlobal {
		globalEntries, err := discovery.DiscoverGlobalWorktrees(cfg.Worktree.BaseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to discover worktrees: %w", err)
		}
		// Convert []*GlobalWorktreeEntry to []*models.Worktree
		for _, entry := range globalEntries {
			worktrees = append(worktrees, &models.Worktree{
				Path:       entry.Path,
				Branch:     entry.Branch,
				CommitHash: entry.CommitHash,
				IsMain:     entry.IsMain,
			})
		}
	} else {
		wm := worktree.New(g, cfg)
		localWorktrees, err := wm.List()
		if err != nil {
			return nil, fmt.Errorf("failed to list worktrees: %w", err)
		}
		// Convert []models.Worktree to []*models.Worktree
		for i := range localWorktrees {
			worktrees = append(worktrees, &localWorktrees[i])
		}
	}
	
	collector := NewStatusCollectorWithOptions(StatusCollectorOptions{
		IncludeProcess: statusShowProcess,
		FetchRemote:    !statusNoFetch,
		StaleThreshold: time.Duration(statusStaleDays) * 24 * time.Hour,
		BaseDir:        cfg.Worktree.BaseDir,
	})
	return collector.CollectAll(ctx, worktrees)
}

func applyFiltersAndSort(statuses []*models.WorktreeStatus) []*models.WorktreeStatus {
	if statusFilter != "" {
		statuses = filterStatuses(statuses, statusFilter)
	}
	
	if statusSort != "" {
		sortStatuses(statuses, statusSort)
	}
	
	return statuses
}

func outputStatuses(statuses []*models.WorktreeStatus, printer *ui.Printer, cfg *models.Config) error {
	switch {
	case statusJSON:
		return outputJSON(statuses)
	case statusCSV:
		return outputCSV(statuses)
	default:
		return outputTable(statuses, printer, statusVerbose)
	}
}

func getCurrentRepository() string {
	g, err := git.NewFromCwd()
	if err != nil {
		return "all repositories"
	}
	
	remote, err := g.GetRepositoryURL()
	if err != nil {
		return "local"
	}
	
	return remote
}

type statusSummary struct {
	Total    int
	Modified int
	Clean    int
	Stale    int
}

func calculateSummary(statuses []*models.WorktreeStatus) statusSummary {
	summary := statusSummary{Total: len(statuses)}
	
	for _, s := range statuses {
		switch s.Status {
		case models.WorktreeStatusModified:
			summary.Modified++
		case models.WorktreeStatusClean:
			summary.Clean++
		case models.WorktreeStatusStale:
			summary.Stale++
		}
	}
	
	return summary
}

func filterStatuses(statuses []*models.WorktreeStatus, filter string) []*models.WorktreeStatus {
	var filtered []*models.WorktreeStatus
	
	for _, s := range statuses {
		switch filter {
		case "modified", "changed":
			if s.Status == models.WorktreeStatusModified {
				filtered = append(filtered, s)
			}
		case "clean", "up to date":
			if s.Status == models.WorktreeStatusClean {
				filtered = append(filtered, s)
			}
		case "stale", "inactive":
			if s.Status == models.WorktreeStatusStale {
				filtered = append(filtered, s)
			}
		case "staged":
			if s.Status == models.WorktreeStatusStaged {
				filtered = append(filtered, s)
			}
		case "conflict", "conflicted":
			if s.Status == models.WorktreeStatusConflict {
				filtered = append(filtered, s)
			}
		}
	}
	
	return filtered
}
