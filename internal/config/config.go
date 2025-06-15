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
	viper.SetDefault("ui.icons", true)
	viper.SetDefault("ui.tilde_home", true)

	// Claude defaults
	viper.SetDefault("claude.executable", "claude")
	viper.SetDefault("claude.config_dir", "~/.config/gwq/claude")
	viper.SetDefault("claude.max_parallel", 3)
	viper.SetDefault("claude.max_development_tasks", 2)

	// Claude queue defaults
	viper.SetDefault("claude.queue.queue_dir", "~/.config/gwq/claude/queue")

	// Claude worktree defaults
	viper.SetDefault("claude.worktree.auto_create_worktree", true)
	viper.SetDefault("claude.worktree.require_existing_worktree", false)
	viper.SetDefault("claude.worktree.validate_branch_exists", true)

	// Claude execution defaults
	viper.SetDefault("claude.execution.auto_cleanup", true)

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
