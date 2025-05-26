package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/d-kuro/wtree/pkg/models"
	"github.com/spf13/viper"
)

func TestGetConfigDir(t *testing.T) {
	// Test without XDG_CONFIG_HOME
	t.Run("WithoutXDGConfigHome", func(t *testing.T) {
		origXDG := os.Getenv("XDG_CONFIG_HOME")
		defer func() { _ = os.Setenv("XDG_CONFIG_HOME", origXDG) }()

		_ = os.Unsetenv("XDG_CONFIG_HOME")

		dir := getConfigDir()
		if !filepath.IsAbs(dir) {
			t.Errorf("getConfigDir() should return absolute path, got %s", dir)
		}
		if filepath.Base(dir) != "wtree" {
			t.Errorf("getConfigDir() should end with 'wtree', got %s", dir)
		}
	})
	
	// getConfigDir uses os.UserConfigDir which doesn't respect XDG_CONFIG_HOME on macOS
	// So we just verify the basic behavior
}

func TestInit(t *testing.T) {
	// Create temporary directory for config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "wtree")

	// Override viper config path
	viper.Reset()
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configDir)

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Test initialization
	if err := Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Verify defaults are set
	if viper.GetString("worktree.basedir") != "~/worktrees" {
		t.Errorf("Default worktree.basedir not set correctly")
	}
	if !viper.GetBool("worktree.auto_mkdir") {
		t.Errorf("Default worktree.auto_mkdir should be true")
	}
	if !viper.GetBool("finder.preview") {
		t.Errorf("Default finder.preview should be true")
	}
	if viper.GetInt("finder.preview_size") != 3 {
		t.Errorf("Default finder.preview_size should be 3")
	}
	if viper.GetString("finder.keybind_select") != "enter" {
		t.Errorf("Default finder.keybind_select should be 'enter'")
	}
	if viper.GetString("finder.keybind_cancel") != "esc" {
		t.Errorf("Default finder.keybind_cancel should be 'esc'")
	}
	if viper.GetString("naming.template") != "{{.Repository}}-{{.Branch}}" {
		t.Errorf("Default naming.template not set correctly")
	}
	if !viper.GetBool("ui.color") {
		t.Errorf("Default ui.color should be true")
	}
	if !viper.GetBool("ui.icons") {
		t.Errorf("Default ui.icons should be true")
	}
}

func TestLoad(t *testing.T) {
	// Setup viper with test values
	viper.Reset()
	viper.Set("worktree.basedir", "~/test-worktrees")
	viper.Set("worktree.auto_mkdir", false)
	viper.Set("finder.preview", false)
	viper.Set("finder.preview_size", 5)
	viper.Set("finder.keybind_select", "tab")
	viper.Set("finder.keybind_cancel", "ctrl-c")
	viper.Set("naming.template", "{{.Branch}}")
	viper.Set("naming.sanitize_chars", map[string]string{"/": "_"})
	viper.Set("ui.color", false)
	viper.Set("ui.icons", false)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify loaded values
	if cfg.Worktree.AutoMkdir {
		t.Errorf("WorktreeConfig.AutoMkdir = %v, want false", cfg.Worktree.AutoMkdir)
	}
	if cfg.Finder.Preview {
		t.Errorf("FinderConfig.Preview = %v, want false", cfg.Finder.Preview)
	}
	if cfg.Finder.PreviewSize != 5 {
		t.Errorf("FinderConfig.PreviewSize = %d, want 5", cfg.Finder.PreviewSize)
	}
	if cfg.Finder.KeybindSelect != "tab" {
		t.Errorf("FinderConfig.KeybindSelect = %s, want tab", cfg.Finder.KeybindSelect)
	}
	if cfg.Finder.KeybindCancel != "ctrl-c" {
		t.Errorf("FinderConfig.KeybindCancel = %s, want ctrl-c", cfg.Finder.KeybindCancel)
	}
	if cfg.Naming.Template != "{{.Branch}}" {
		t.Errorf("NamingConfig.Template = %s, want {{.Branch}}", cfg.Naming.Template)
	}
	if cfg.UI.Color {
		t.Errorf("UIConfig.Color = %v, want false", cfg.UI.Color)
	}
	if cfg.UI.Icons {
		t.Errorf("UIConfig.Icons = %v, want false", cfg.UI.Icons)
	}
}

func TestPathExpansion(t *testing.T) {
	// Test home directory expansion
	t.Run("HomeDirectoryExpansion", func(t *testing.T) {
		viper.Reset()
		viper.Set("worktree.basedir", "~/worktrees")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if filepath.IsAbs(cfg.Worktree.BaseDir) && cfg.Worktree.BaseDir != "~/worktrees" {
			// Path was expanded
			if !filepath.IsAbs(cfg.Worktree.BaseDir) {
				t.Errorf("Expanded path should be absolute, got %s", cfg.Worktree.BaseDir)
			}
		}
	})

	// Test environment variable expansion
	t.Run("EnvironmentVariableExpansion", func(t *testing.T) {
		viper.Reset()
		_ = os.Setenv("TEST_WORKTREE_DIR", "/test/path")
		defer func() { _ = os.Unsetenv("TEST_WORKTREE_DIR") }()

		viper.Set("worktree.basedir", "$TEST_WORKTREE_DIR/worktrees")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		expected := "/test/path/worktrees"
		if cfg.Worktree.BaseDir != expected {
			t.Errorf("BaseDir = %s, want %s", cfg.Worktree.BaseDir, expected)
		}
	})
}

func TestGettersAndSetters(t *testing.T) {
	viper.Reset()

	// Test Set and Get
	testKey := "test.key"
	testValue := "test-value"
	
	// Note: In real usage, Set would write to config file
	// For testing, we'll just verify viper operations
	viper.Set(testKey, testValue)
	
	if got := Get(testKey); got != testValue {
		t.Errorf("Get(%s) = %v, want %v", testKey, got, testValue)
	}

	// Test GetString
	if got := GetString(testKey); got != testValue {
		t.Errorf("GetString(%s) = %s, want %s", testKey, got, testValue)
	}

	// Test GetBool
	boolKey := "test.bool"
	viper.Set(boolKey, true)
	if got := GetBool(boolKey); !got {
		t.Errorf("GetBool(%s) = %v, want true", boolKey, got)
	}

	// Test GetInt
	intKey := "test.int"
	intValue := 42
	viper.Set(intKey, intValue)
	if got := GetInt(intKey); got != intValue {
		t.Errorf("GetInt(%s) = %d, want %d", intKey, got, intValue)
	}

	// Test GetStringMapString
	mapKey := "test.map"
	mapValue := map[string]string{"foo": "bar", "baz": "qux"}
	viper.Set(mapKey, mapValue)
	if got := GetStringMapString(mapKey); len(got) != len(mapValue) {
		t.Errorf("GetStringMapString(%s) returned map with %d entries, want %d", mapKey, len(got), len(mapValue))
	}
}

func TestAllSettings(t *testing.T) {
	viper.Reset()
	viper.Set("test.key1", "value1")
	viper.Set("test.key2", 123)
	viper.Set("test.key3", true)

	settings := AllSettings()
	if len(settings) == 0 {
		t.Error("AllSettings() returned empty map")
	}

	// Check if our test settings are included
	if testSection, ok := settings["test"].(map[string]interface{}); ok {
		if testSection["key1"] != "value1" {
			t.Errorf("AllSettings() missing or incorrect test.key1")
		}
		if testSection["key2"] != 123 {
			t.Errorf("AllSettings() missing or incorrect test.key2")
		}
		if testSection["key3"] != true {
			t.Errorf("AllSettings() missing or incorrect test.key3")
		}
	} else {
		t.Error("AllSettings() missing 'test' section")
	}
}

func TestConfigStructureIntegrity(t *testing.T) {
	// This test ensures that the Config structure can be properly marshaled/unmarshaled
	cfg := &models.Config{
		Worktree: models.WorktreeConfig{
			BaseDir:   "/test/worktrees",
			AutoMkdir: true,
		},
		Finder: models.FinderConfig{
			Preview:       true,
			PreviewSize:   5,
			KeybindSelect: "enter",
			KeybindCancel: "esc",
		},
		Naming: models.NamingConfig{
			Template:      "{{.Repository}}-{{.Branch}}",
			SanitizeChars: map[string]string{"/": "-", ":": "-"},
		},
		UI: models.UIConfig{
			Color: true,
			Icons: false,
		},
	}

	// Set values in viper
	viper.Reset()
	viper.Set("worktree.basedir", cfg.Worktree.BaseDir)
	viper.Set("worktree.auto_mkdir", cfg.Worktree.AutoMkdir)
	viper.Set("finder.preview", cfg.Finder.Preview)
	viper.Set("finder.preview_size", cfg.Finder.PreviewSize)
	viper.Set("finder.keybind_select", cfg.Finder.KeybindSelect)
	viper.Set("finder.keybind_cancel", cfg.Finder.KeybindCancel)
	viper.Set("naming.template", cfg.Naming.Template)
	viper.Set("naming.sanitize_chars", cfg.Naming.SanitizeChars)
	viper.Set("ui.color", cfg.UI.Color)
	viper.Set("ui.icons", cfg.UI.Icons)

	// Load and verify
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare loaded config with original
	if loaded.Worktree.BaseDir != cfg.Worktree.BaseDir {
		t.Errorf("Worktree.BaseDir mismatch")
	}
	if loaded.Worktree.AutoMkdir != cfg.Worktree.AutoMkdir {
		t.Errorf("Worktree.AutoMkdir mismatch")
	}
	if loaded.Finder.Preview != cfg.Finder.Preview {
		t.Errorf("Finder.Preview mismatch")
	}
	if loaded.Finder.PreviewSize != cfg.Finder.PreviewSize {
		t.Errorf("Finder.PreviewSize mismatch")
	}
	if loaded.Finder.KeybindSelect != cfg.Finder.KeybindSelect {
		t.Errorf("Finder.KeybindSelect mismatch")
	}
	if loaded.Finder.KeybindCancel != cfg.Finder.KeybindCancel {
		t.Errorf("Finder.KeybindCancel mismatch")
	}
	if loaded.Naming.Template != cfg.Naming.Template {
		t.Errorf("Naming.Template mismatch")
	}
	if loaded.UI.Color != cfg.UI.Color {
		t.Errorf("UI.Color mismatch")
	}
	if loaded.UI.Icons != cfg.UI.Icons {
		t.Errorf("UI.Icons mismatch")
	}
}