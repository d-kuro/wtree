package cmd

import (
	"fmt"

	"github.com/d-kuro/gwq/internal/config"
	"github.com/d-kuro/gwq/internal/ui"
	"github.com/spf13/cobra"
)

// configCmd represents the config command.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Manage gwq configuration settings.`,
}

// configListCmd represents the config list command.
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show configuration",
	Long:  `Display all current configuration settings.`,
	Example: `  # Show all configuration
  gwq config list`,
	RunE: runConfigList,
}

// configSetCmd represents the config set command.
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Long: `Set a configuration value.

Configuration keys follow a dot notation format (e.g., worktree.basedir).`,
	Example: `  # Set worktree base directory
  gwq config set worktree.basedir ~/worktrees

  # Set naming template
  gwq config set naming.template "{{.Repository}}-{{.Branch}}"

  # Enable/disable colored output
  gwq config set ui.color true`,
	Args:              cobra.ExactArgs(2),
	RunE:              runConfigSet,
	ValidArgsFunction: getConfigKeyCompletions,
}

// configGetCmd represents the config get command.
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get configuration value",
	Long:  `Get a specific configuration value.`,
	Example: `  # Get worktree base directory
  gwq config get worktree.basedir

  # Get naming template
  gwq config get naming.template`,
	Args:              cobra.ExactArgs(1),
	RunE:              runConfigGet,
	ValidArgsFunction: getConfigKeyCompletions,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

func runConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	printer := ui.New(&cfg.UI)
	settings := config.AllSettings()
	printer.PrintConfig(settings)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Convert string values to appropriate types
	var typedValue interface{} = value
	switch value {
	case "true":
		typedValue = true
	case "false":
		typedValue = false
	default:
		// Try to convert to integer
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			typedValue = intVal
		}
	}

	if err := config.Set(key, typedValue); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	fmt.Printf("Set %s = %v\n", key, typedValue)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := config.GetValue(key)

	if value == nil {
		return fmt.Errorf("configuration key not found: %s", key)
	}

	fmt.Println(value)
	return nil
}
