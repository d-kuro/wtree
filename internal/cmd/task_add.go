package cmd

import (
	"github.com/spf13/cobra"
)

var taskAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new tasks to the queue",
	Long: `Add new tasks to the queue for automated processing.

This command provides different task types that can be added to the queue:
- Claude Code tasks for AI-assisted development
- Other task types may be added in the future

Each task type has its own specific options and configuration.`,
	Example: `  # Add a Claude Code task
  gwq task add claude -b feature/auth "Implement JWT authentication"

  # Add a Claude task with priority and dependencies
  gwq task add claude -b feature/api "REST API endpoints" -p 80 --depends-on auth-task`,
}

func init() {
	taskCmd.AddCommand(taskAddCmd)
}
