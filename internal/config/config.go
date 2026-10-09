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
	// Providers is the ordered list of configured provider instances. Each has
	// a unique Key (its identity in the UI and persisted state) and a Type that
	// determines how it is treated.
	Providers []ProviderConfig `yaml:"providers"`

	// OutputDir is where generated images are written. May be a {env:}/{cmd:}
	// directive. Empty means use the built-in default.
	OutputDir Value `yaml:"output_dir"`

	// AutoOpen opens generated images in the system viewer without prompting.
	AutoOpen bool `yaml:"auto_open"`

	// OpenCommand overrides the per-OS default command used to open images.
	// May be a {env:}/{cmd:} directive. Empty means use the OS default.
	OpenCommand Value `yaml:"open_command"`
}

// Provider type identifiers used in ProviderConfig.Type.
const (
	// TypeOpenRouter is the hosted OpenRouter provider.
	TypeOpenRouter = "openrouter"
	// TypeA1111 is the local A1111-compatible WebUI provider.
	TypeA1111 = "a1111"
)

// ProviderConfig is a flat union describing one configured provider instance.
// Fields not relevant to a given Type are simply unused by it.
type ProviderConfig struct {
	// Key is the instance identity (used in the picker and persisted state).
	// It must be unique and non-empty across all instances.
	Key string `yaml:"key"`

	// Type determines how the instance is treated (e.g. "openrouter", "a1111").
	Type string `yaml:"type"`

	// APIKey resolves to the provider API key (OpenRouter). Typically
	// "{env:OPENROUTER_API_KEY}" or a "{cmd:...}" directive; a literal key is
	// discouraged. Unused by types that need no key (e.g. a1111).
	APIKey Value `yaml:"api_key"`

	// BaseURL overrides the API base. Required by a1111 (the WebUI address);
	// optional for OpenRouter (test override). Empty means the type default.
	BaseURL string `yaml:"base_url"`
}

// Default returns a Config populated with built-in defaults, used when no
// config file is present: a single OpenRouter instance.
func Default() Config {
	return Config{
		Providers: []ProviderConfig{
			{
				Key:    "openrouter",
				Type:   TypeOpenRouter,
				APIKey: "{env:OPENROUTER_API_KEY}",
			},
		},
	}
}

// ErrConfigNotFound indicates no config.yaml existed at the resolved path.
var ErrConfigNotFound = errors.New("config: no config.yaml found")

// ErrLegacyConfig indicates the file is in the legacy single-provider shape (a
// top-level `provider:` and/or type-named block, with no `providers:` list).
// Callers back the file up and route the user into onboarding to rebuild it,
// rather than treating it as a parse error.
var ErrLegacyConfig = errors.New("config: legacy single-provider config detected")

// Load reads and parses config.yaml from the given path, applying defaults for
// unset fields. If the file does not exist it returns Default() along with
// ErrConfigNotFound so callers can report "no config found" while still
// running. A legacy single-provider file returns ErrLegacyConfig (recoverable).
// A malformed file is a hard error (no silent fallback to defaults).
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), ErrConfigNotFound
		}
		return Config{}, fmt.Errorf("config: reading %s: %w", path, err)
	}

	if isLegacyShape(data) {
		return Config{}, ErrLegacyConfig
	}

	cfg := Default()
	cfg.Providers = nil // don't merge the file onto the default instance
	// Decode strictly so unknown keys surface typos rather than silently vanish.
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parsing %s: %w", path, err)
	}
	if err := validateProviders(cfg.Providers); err != nil {
		return Config{}, fmt.Errorf("config: %s: %w", path, err)
	}
	return cfg, nil
}

// legacyProbe is a minimal decode used only to detect the legacy shape before
// strict decoding would reject its keys with a generic parse error.
type legacyProbe struct {
	Provider   *string                `yaml:"provider"`
	OpenRouter map[string]interface{} `yaml:"openrouter"`
	Providers  []interface{}          `yaml:"providers"`
}

// isLegacyShape reports whether data looks like the legacy single-provider
// config: a top-level `provider:` and/or an `openrouter:` block, with no
// `providers:` list. A current-shape file (which has `providers:`) is never
// legacy. Unparsable data is left to the strict decode to report.
func isLegacyShape(data []byte) bool {
	var p legacyProbe
	if err := yaml.Unmarshal(data, &p); err != nil {
		return false
	}
	if len(p.Providers) > 0 {
		return false
	}
	return p.Provider != nil || len(p.OpenRouter) > 0
}

// validateProviders enforces that every instance has a non-empty, unique Key
// and a non-empty Type, naming the offending instance on failure.
func validateProviders(providers []ProviderConfig) error {
	seen := make(map[string]bool, len(providers))
	for i, p := range providers {
		if p.Key == "" {
			return fmt.Errorf("provider instance #%d has an empty key", i+1)
		}
		if seen[p.Key] {
			return fmt.Errorf("duplicate provider key %q", p.Key)
		}
		seen[p.Key] = true
		if p.Type == "" {
			return fmt.Errorf("provider %q has an empty type", p.Key)
		}
	}
	return nil
}

// Dir returns the genifer config directory, ~/.config/genifer. It honors
// XDG_CONFIG_HOME when set, otherwise falls back to ~/.config — the same
// location on every platform, by design, rather than the OS-specific
// convention (e.g. ~/Library/Application Support on macOS).
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("config: locating home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
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

// BackupConfig renames path to a non-colliding backup alongside it, returning
// the backup path. It first tries "<path>.bak"; if that already exists it tries
// "<path>.bak.1", ".bak.2", and so on, so an existing backup is never
// overwritten. It is used to preserve a legacy config.yaml before onboarding
// rebuilds it.
func BackupConfig(path string) (string, error) {
	backup := path + ".bak"
	if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(path, backup); err != nil {
			return "", fmt.Errorf("config: backing up %s: %w", path, err)
		}
		return backup, nil
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s.bak.%d", path, i)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			if err := os.Rename(path, candidate); err != nil {
				return "", fmt.Errorf("config: backing up %s: %w", path, err)
			}
			return candidate, nil
		}
	}
}
