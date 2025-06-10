package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/finder"
	"github.com/d-kuro/gwq/internal/git"
	"github.com/d-kuro/gwq/internal/tmux"
	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/cobra"
)

var (
	tmuxKillInteractive bool
	tmuxKillAll         bool
	tmuxKillCompleted   bool
	tmuxKillForce       bool
)

var tmuxKillCmd = &cobra.Command{
	Use:   "kill [pattern]",
	Short: "Terminate tmux sessions",
	Long: `Terminate tmux sessions matching the given pattern.

If multiple sessions match the pattern, an interactive selection will be shown.
If no pattern is provided, an interactive session selector will be displayed.`,
	Example: `  # Terminate session matching 'auth'
  gwq tmux kill auth

  # Use interactive selection
  gwq tmux kill -i

  # Terminate all sessions (with confirmation)
  gwq tmux kill --all

  # Cleanup only completed sessions
  gwq tmux kill --completed

  # Force kill without confirmation
  gwq tmux kill --all --force`,
	RunE: runTmuxKill,
}

func init() {
	tmuxCmd.AddCommand(tmuxKillCmd)

	tmuxKillCmd.Flags().BoolVarP(&tmuxKillInteractive, "interactive", "i", false, "Always use interactive selection")
	tmuxKillCmd.Flags().BoolVar(&tmuxKillAll, "all", false, "Terminate all sessions")
	tmuxKillCmd.Flags().BoolVar(&tmuxKillCompleted, "completed", false, "Terminate only completed sessions")
	tmuxKillCmd.Flags().BoolVar(&tmuxKillForce, "force", false, "Skip confirmation prompts")
}

func runTmuxKill(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dataDir := filepath.Join(cfg.Worktree.BaseDir, ".gwq")
	sessionManager := tmux.NewSessionManager(nil, dataDir)

	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No tmux sessions found")
		return nil
	}

	var sessionsToKill []*tmux.Session

	switch {
	case tmuxKillAll:
		sessionsToKill = sessions
	case tmuxKillCompleted:
		sessionsToKill = filterCompletedSessions(sessions)
	case len(args) == 0 || tmuxKillInteractive:
		// Interactive selection using fuzzy finder
		selected, err := selectSessionsToKillWithFinder(sessions, cfg)
		if err != nil {
			return fmt.Errorf("session selection cancelled: %w", err)
		}
		sessionsToKill = selected
	default:
		// Pattern matching
		pattern := args[0]
		matches := findMatchingSessions(sessions, pattern)
		if len(matches) == 0 {
			return fmt.Errorf("no session found matching pattern: %s", pattern)
		} else if len(matches) == 1 {
			sessionsToKill = matches
		} else {
			// Multiple matches - use fuzzy finder
			selected, err := selectSessionsToKillWithFinder(matches, cfg)
			if err != nil {
				return fmt.Errorf("session selection cancelled: %w", err)
			}
			sessionsToKill = selected
		}
	}

	if len(sessionsToKill) == 0 {
		fmt.Println("No sessions selected for termination")
		return nil
	}

	// Confirmation unless force flag is used
	if !tmuxKillForce {
		if !confirmKillSessions(sessionsToKill) {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Kill selected sessions
	return killSessions(sessionManager, sessionsToKill)
}

func filterCompletedSessions(sessions []*tmux.Session) []*tmux.Session {
	var completed []*tmux.Session
	for _, session := range sessions {
		if session.Status == tmux.StatusCompleted || session.Status == tmux.StatusFailed {
			completed = append(completed, session)
		}
	}
	return completed
}

func selectSessionsToKillWithFinder(sessions []*tmux.Session, cfg *models.Config) ([]*tmux.Session, error) {
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions available")
	}

	// Create finder
	g := &git.Git{} // Temporary git instance (not used for sessions)
	f := finder.NewWithUI(g, &cfg.Finder, &cfg.UI)

	// Use fuzzy finder for multiple session selection
	selected, err := f.SelectMultipleSessions(sessions)
	if err != nil {
		return nil, err
	}

	return selected, nil
}

func confirmKillSessions(sessions []*tmux.Session) bool {
	fmt.Printf("\nThis will terminate %d session(s):\n", len(sessions))
	for _, session := range sessions {
		statusIndicator := getStatusIndicator(session.Status)
		fmt.Printf("  %s%s/%s (%s)\n",
			statusIndicator, session.Context, session.Identifier, session.Status)
	}

	fmt.Print("\nAre you sure? (y/N): ")
	var response string
	_, _ = fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func killSessions(sessionManager *tmux.SessionManager, sessions []*tmux.Session) error {
	var errors []string

	for _, session := range sessions {
		fmt.Printf("Terminating session %s/%s...", session.Context, session.Identifier)

		err := sessionManager.KillSessionDirect(session)
		if err != nil {
			fmt.Printf(" FAILED: %v\n", err)
			errors = append(errors, fmt.Sprintf("%s: %v", session.SessionName, err))
		} else {
			fmt.Printf(" OK\n")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some sessions failed to terminate:\n%s", strings.Join(errors, "\n"))
	}

	fmt.Printf("\nSuccessfully terminated %d session(s)\n", len(sessions))
	return nil
}

func getStatusIndicator(status tmux.Status) string {
	switch status {
	case tmux.StatusRunning:
		return "● "
	case tmux.StatusCompleted:
		return "✓ "
	case tmux.StatusFailed:
		return "✗ "
	case tmux.StatusDetached:
		return "◦ "
	default:
		return "  "
	}
}

