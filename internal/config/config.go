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

	return &cfg, nil
}

// Set sets a configuration value by key.
func Set(key string, value any) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

// Get retrieves a configuration value by key.
func Get(key string) any {
	return viper.Get(key)
}

// GetString retrieves a string configuration value by key.
func GetString(key string) string {
	return viper.GetString(key)
}

// GetBool retrieves a boolean configuration value by key.
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetInt retrieves an integer configuration value by key.
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetStringMapString retrieves a string map configuration value by key.
func GetStringMapString(key string) map[string]string {
	return viper.GetStringMapString(key)
}

// AllSettings returns all configuration settings.
func AllSettings() map[string]any {
	return viper.AllSettings()
}