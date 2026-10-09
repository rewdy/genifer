package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultAPIKey is the recommended api_key directive used in a fresh OpenRouter
// starter instance when the caller supplies no explicit choice.
const DefaultAPIKey Value = "{env:OPENROUTER_API_KEY}"

// DefaultA1111BaseURL is the WebUI address offered by default for a local
// (a1111-compatible) starter instance.
const DefaultA1111BaseURL = "http://127.0.0.1:7860"

// StarterSpec describes the single provider instance written into a fresh
// starter config. Type selects which detail is emitted: for TypeOpenRouter the
// api_key directive, for TypeA1111 the base_url.
type StarterSpec struct {
	// Key is the instance key written into the starter. Empty falls back to a
	// type-appropriate default ("openrouter" or "local").
	Key string
	// Type is the provider type. Empty falls back to TypeOpenRouter.
	Type string
	// APIKey is the OpenRouter api_key directive. Empty falls back to
	// DefaultAPIKey. Unused for TypeA1111.
	APIKey Value
	// BaseURL is the a1111 WebUI address. Empty falls back to
	// DefaultA1111BaseURL. Unused for TypeOpenRouter.
	BaseURL string
}

// DefaultStarterSpec returns the spec used when no explicit choice is supplied:
// a single OpenRouter instance with the recommended api_key default.
func DefaultStarterSpec() StarterSpec {
	return StarterSpec{Key: "openrouter", Type: TypeOpenRouter, APIKey: DefaultAPIKey}
}

// normalize fills in type-appropriate defaults for empty fields.
func (s StarterSpec) normalize() StarterSpec {
	if s.Type == "" {
		s.Type = TypeOpenRouter
	}
	switch s.Type {
	case TypeOpenRouter:
		if s.Key == "" {
			s.Key = "openrouter"
		}
		if s.APIKey == "" {
			s.APIKey = DefaultAPIKey
		}
	case TypeA1111:
		if s.Key == "" {
			s.Key = "local"
		}
		if s.BaseURL == "" {
			s.BaseURL = DefaultA1111BaseURL
		}
	}
	return s
}

// starterHeader is the leading commented block, shared by all starter files. It
// documents the provider-list shape and the non-provider settings.
const starterHeader = `# genifer configuration. You own this file; genifer never overwrites it.
# See docs/config.md for the full reference.

# Providers are a list of keyed, typed instances. Each entry has a unique "key"
# (its identity in the picker and in saved state), a "type" (how it is treated),
# and the fields that type needs. Add more instances by appending to this list.
#
#   - key: or
#     type: openrouter
#     api_key: "{env:OPENROUTER_API_KEY}"
#   - key: local
#     type: a1111
#     base_url: "http://127.0.0.1:7860"
providers:
`

// starterFooter documents the remaining (non-provider) settings.
const starterFooter = `
# Where images are written. Options:
#   omit         -> ~/Pictures/genifer (default)
#   "." or "pwd" -> the directory you launched genifer from
#   relative     -> resolved against the launch directory
#   absolute     -> used as-is
# output_dir: "."

# Open generated images automatically, with no prompt.
auto_open: false

# Override the per-OS command used to open images. Omit to use the OS default
# (macOS: open, Linux: xdg-open, Windows: start).
# open_command: "feh"
`

// openRouterInstance is the commented provider-list entry for an OpenRouter
// instance. The two %s are the key and the api_key value.
const openRouterInstance = `  # Hosted OpenRouter provider.
  - key: %s
    type: openrouter
    # Prefer a directive over a literal key:
    #   {env:NAME}  -> value of environment variable NAME
    #   {cmd:...}   -> trimmed stdout of a command (e.g. a secrets manager)
    api_key: %q
`

// a1111Instance is the commented provider-list entry for a local
// A1111-compatible instance. The two %s are the key and the base_url.
const a1111Instance = `  # Local A1111-compatible WebUI (A1111, Forge, or ComfyUI with the shim).
  - key: %s
    type: a1111
    # Address of the running WebUI.
    base_url: %q
`

// StarterContent returns the commented starter config.yaml body with the given
// provider instance filled in. Empty spec fields fall back to type-appropriate
// defaults. The result is valid YAML that Load accepts — the starter test
// enforces this.
func StarterContent(spec StarterSpec) string {
	spec = spec.normalize()
	var instance string
	switch spec.Type {
	case TypeA1111:
		instance = fmt.Sprintf(a1111Instance, spec.Key, spec.BaseURL)
	default: // TypeOpenRouter
		instance = fmt.Sprintf(openRouterInstance, spec.Key, string(spec.APIKey))
	}
	return starterHeader + instance + starterFooter
}

// WriteStarter creates a commented starter config.yaml at path, but only when
// no file exists there. It returns created=true when it wrote the file, or
// created=false (with a nil error) when a file already existed and was left
// untouched — enforcing the user-owned, never-overwrite invariant. The parent
// directory is created if needed. spec selects the provider instance written
// into the file; pass the zero StarterSpec{} for the recommended OpenRouter
// default.
func WriteStarter(path string, spec StarterSpec) (created bool, err error) {
	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil // already exists: never overwrite
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return false, fmt.Errorf("config: checking %s: %w", path, statErr)
	}

	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return false, fmt.Errorf("config: creating config dir: %w", mkErr)
	}

	// O_EXCL guards against a race where the file appears between the stat and
	// the write, preserving never-overwrite even then.
	f, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if openErr != nil {
		if errors.Is(openErr, os.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("config: creating %s: %w", path, openErr)
	}
	defer f.Close()

	if _, writeErr := f.WriteString(StarterContent(spec)); writeErr != nil {
		return false, fmt.Errorf("config: writing %s: %w", path, writeErr)
	}
	return true, nil
}
