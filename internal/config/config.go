// Package config provides configuration management for the gwq application.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d-kuro/gwq/pkg/models"
	"github.com/spf13/viper"
)

const (
	configName = "config"
	configType = "toml"
)

// getConfigDir returns the configuration directory path.
func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home is not available
		return filepath.Join(".", ".config", "gwq")
	}
	return filepath.Join(home, ".config", "gwq")
}

// Init initializes the configuration system, creating default config if needed.
func Init() error {
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configDir)

	viper.SetDefault("worktree.basedir", "~/worktrees")
	viper.SetDefault("worktree.auto_mkdir", true)
	viper.SetDefault("finder.preview", true)
	viper.SetDefault("finder.preview_size", 3)
	viper.SetDefault("finder.keybind_select", "enter")
	viper.SetDefault("finder.keybind_cancel", "esc")
	viper.SetDefault("naming.template", "{{.Host}}/{{.Owner}}/{{.Repository}}/{{.Branch}}")
	viper.SetDefault("naming.sanitize_chars", map[string]string{
		"/": "-",
		":": "-",
	})
	viper.SetDefault("ui.icons", true)
	viper.SetDefault("ui.tilde_home", true)
	viper.SetDefault("tmux.enabled", true)
	viper.SetDefault("tmux.auto_create_session", true)
	viper.SetDefault("tmux.detach_on_create", true)
	viper.SetDefault("tmux.auto_cleanup_completed", false)
	viper.SetDefault("tmux.tmux_command", "tmux")
	viper.SetDefault("tmux.default_shell", "/bin/bash")
	viper.SetDefault("tmux.session_timeout", "24h")
	viper.SetDefault("tmux.keep_alive", true)
	viper.SetDefault("tmux.history_limit", 50000)
	viper.SetDefault("tmux.history_auto_save", true)
	viper.SetDefault("tmux.history_save_dir", "~/.gwq/history")

	// Claude defaults
	viper.SetDefault("claude.executable", "claude")
	viper.SetDefault("claude.skip_permissions", true)
	viper.SetDefault("claude.timeout", "2h")
	viper.SetDefault("claude.max_iterations", 3)
	viper.SetDefault("claude.config_dir", "~/.config/gwq/claude")
	viper.SetDefault("claude.additional_args", []string{})
	viper.SetDefault("claude.max_parallel", 3)
	viper.SetDefault("claude.max_development_tasks", 2)
	viper.SetDefault("claude.max_review_tasks", 1)
	viper.SetDefault("claude.max_cpu_percent", 80)
	viper.SetDefault("claude.max_memory_mb", 4096)
	viper.SetDefault("claude.task_timeout", "2h")

	// Claude queue defaults
	viper.SetDefault("claude.queue.max_queue_size", 50)
	viper.SetDefault("claude.queue.queue_dir", "~/.config/gwq/claude/queue")
	viper.SetDefault("claude.queue.priority_boost_after", "1h")
	viper.SetDefault("claude.queue.starvation_prevention", true)
	viper.SetDefault("claude.queue.dependency_timeout", "30m")
	viper.SetDefault("claude.queue.max_dependency_depth", 5)
	viper.SetDefault("claude.queue.validate_dependencies", true)
	viper.SetDefault("claude.queue.validate_task_files", true)
	viper.SetDefault("claude.queue.required_fields", []string{"objectives", "instructions", "verification_commands"})

	// Claude worktree defaults
	viper.SetDefault("claude.worktree.auto_create_worktree", true)
	viper.SetDefault("claude.worktree.require_existing_worktree", false)
	viper.SetDefault("claude.worktree.validate_branch_exists", true)

	// Claude review defaults
	viper.SetDefault("claude.review.enabled", true)
	viper.SetDefault("claude.review.review_patterns", []string{"*.go", "*.js", "*.ts", "*.py"})
	viper.SetDefault("claude.review.exclude_patterns", []string{"*_test.go", "vendor/*"})
	viper.SetDefault("claude.review.review_prompt", "Please review the code changes focusing on security, bugs, performance, and readability.")
	viper.SetDefault("claude.review.auto_fix", true)
	viper.SetDefault("claude.review.max_fix_attempts", 2)

	// Claude headless execution defaults
	viper.SetDefault("claude.headless.log_retention_days", 30)
	viper.SetDefault("claude.headless.max_log_size_mb", 100)
	viper.SetDefault("claude.headless.auto_cleanup", true)
	viper.SetDefault("claude.headless.fuzzy_finder", "fzf")

	// Claude headless formatting defaults
	viper.SetDefault("claude.headless.formatting.show_tool_details", true)
	viper.SetDefault("claude.headless.formatting.show_cost_breakdown", true)
	viper.SetDefault("claude.headless.formatting.show_timing_info", true)
	viper.SetDefault("claude.headless.formatting.max_content_length", 1000)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configPath := filepath.Join(configDir, configName+"."+configType)
			if err := viper.SafeWriteConfig(); err != nil {
				if err := viper.WriteConfigAs(configPath); err != nil {
					return fmt.Errorf("failed to create config file: %w", err)
				}
			}
		} else {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	return nil
}

// Load loads and returns the current configuration.
func Load() (*models.Config, error) {
	var cfg models.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.Worktree.BaseDir = os.ExpandEnv(cfg.Worktree.BaseDir)
	if strings.HasPrefix(cfg.Worktree.BaseDir, "~/") {
		home, _ := os.UserHomeDir()
		cfg.Worktree.BaseDir = filepath.Join(home, cfg.Worktree.BaseDir[2:])
	}

	// Expand Claude configuration paths
	cfg.Claude.ConfigDir = os.ExpandEnv(cfg.Claude.ConfigDir)
	if strings.HasPrefix(cfg.Claude.ConfigDir, "~/") {
		home, _ := os.UserHomeDir()
		cfg.Claude.ConfigDir = filepath.Join(home, cfg.Claude.ConfigDir[2:])
	}

	cfg.Claude.Queue.QueueDir = os.ExpandEnv(cfg.Claude.Queue.QueueDir)
	if strings.HasPrefix(cfg.Claude.Queue.QueueDir, "~/") {
		home, _ := os.UserHomeDir()
		cfg.Claude.Queue.QueueDir = filepath.Join(home, cfg.Claude.Queue.QueueDir[2:])
	}

	return &cfg, nil
}

// Set sets a configuration value by key.
func Set(key string, value any) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

// GetValue retrieves a configuration value by key.
func GetValue(key string) any {
	return viper.Get(key)
}

// AllSettings returns all configuration settings.
func AllSettings() map[string]any {
	return viper.AllSettings()
}

// Get returns the current loaded configuration, loading it if necessary.
func Get() *models.Config {
	cfg, err := Load()
	if err != nil {
		// Initialize with viper defaults if config cannot be loaded
		var defaultCfg models.Config
		if err := viper.Unmarshal(&defaultCfg); err != nil {
			// Fallback to empty config if unmarshal fails
			return &models.Config{}
		}

		// Apply path expansions to defaults
		defaultCfg.Worktree.BaseDir = os.ExpandEnv(defaultCfg.Worktree.BaseDir)
		if strings.HasPrefix(defaultCfg.Worktree.BaseDir, "~/") {
			home, _ := os.UserHomeDir()
			defaultCfg.Worktree.BaseDir = filepath.Join(home, defaultCfg.Worktree.BaseDir[2:])
		}

		defaultCfg.Claude.ConfigDir = os.ExpandEnv(defaultCfg.Claude.ConfigDir)
		if strings.HasPrefix(defaultCfg.Claude.ConfigDir, "~/") {
			home, _ := os.UserHomeDir()
			defaultCfg.Claude.ConfigDir = filepath.Join(home, defaultCfg.Claude.ConfigDir[2:])
		}

		defaultCfg.Claude.Queue.QueueDir = os.ExpandEnv(defaultCfg.Claude.Queue.QueueDir)
		if strings.HasPrefix(defaultCfg.Claude.Queue.QueueDir, "~/") {
			home, _ := os.UserHomeDir()
			defaultCfg.Claude.Queue.QueueDir = filepath.Join(home, defaultCfg.Claude.Queue.QueueDir[2:])
		}

		return &defaultCfg
	}
	return cfg
}
