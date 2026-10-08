package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the user-owned configuration loaded from config.yaml. Every field
// that may hold a secret or vary by environment is a Value, so it supports the
// {env:} / {cmd:} directives.
type Config struct {
	// Provider selects the active provider implementation (default "openrouter").
	Provider string `yaml:"provider"`

	OpenRouter OpenRouterConfig `yaml:"openrouter"`

	// OutputDir is where generated images are written. May be a {env:}/{cmd:}
	// directive. Empty means use the built-in default.
	OutputDir Value `yaml:"output_dir"`

	// AutoOpen opens generated images in the system viewer without prompting.
	AutoOpen bool `yaml:"auto_open"`

	// OpenCommand overrides the per-OS default command used to open images.
	// May be a {env:}/{cmd:} directive. Empty means use the OS default.
	OpenCommand Value `yaml:"open_command"`
}

// OpenRouterConfig holds OpenRouter-specific settings.
type OpenRouterConfig struct {
	// APIKey resolves to the OpenRouter API key. Typically "{env:OPENROUTER_API_KEY}"
	// or a "{cmd:...}" directive; a literal key is discouraged.
	APIKey Value `yaml:"api_key"`

	// BaseURL overrides the API base (for testing). Empty means the default.
	BaseURL string `yaml:"base_url"`
}

// Default returns a Config populated with built-in defaults, used when no
// config file is present.
func Default() Config {
	return Config{
		Provider: "openrouter",
		OpenRouter: OpenRouterConfig{
			APIKey: "{env:OPENROUTER_API_KEY}",
		},
	}
}

// ErrConfigNotFound indicates no config.yaml existed at the resolved path.
var ErrConfigNotFound = errors.New("config: no config.yaml found")

// Load reads and parses config.yaml from the given path, applying defaults for
// unset fields. If the file does not exist it returns Default() along with
// ErrConfigNotFound so callers can report "no config found" while still
// running. A malformed file is a hard error (no silent fallback to defaults).
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), ErrConfigNotFound
		}
		return Config{}, fmt.Errorf("config: reading %s: %w", path, err)
	}

	cfg := Default()
	// Decode strictly so unknown keys surface typos rather than silently vanish.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parsing %s: %w", path, err)
	}
	if cfg.Provider == "" {
		cfg.Provider = "openrouter"
	}
	return cfg, nil
}

// Dir returns the genifer config directory (e.g. ~/.config/genifer on Unix),
// following the platform's user config directory convention.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: locating user config dir: %w", err)
	}
	return filepath.Join(base, "genifer"), nil
}

// ConfigPath returns the full path to config.yaml within the config dir.
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}
