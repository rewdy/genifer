package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultAPIKey is the recommended api_key directive used in a fresh starter
// config when the caller supplies no explicit choice.
const DefaultAPIKey Value = "{env:OPENROUTER_API_KEY}"

// starterTemplate is the commented config.yaml written for new users. The
// single %s is the api_key value. It documents every setting with its default
// and must stay valid YAML that Load accepts — the starter test enforces this.
const starterTemplate = `# genifer configuration. You own this file; genifer never overwrites it.
# See docs/config.md for the full reference.

# Active provider. Only "openrouter" is supported today.
provider: openrouter

openrouter:
  # The OpenRouter API key. Prefer a directive over a literal key:
  #   {env:NAME}  -> value of environment variable NAME
  #   {cmd:...}   -> trimmed stdout of a command (e.g. a secrets manager)
  api_key: %q

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

// StarterContent returns the commented starter config.yaml body with apiKey
// substituted into the api_key setting. An empty apiKey falls back to
// DefaultAPIKey.
func StarterContent(apiKey Value) string {
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	return fmt.Sprintf(starterTemplate, string(apiKey))
}

// WriteStarter creates a commented starter config.yaml at path, but only when
// no file exists there. It returns created=true when it wrote the file, or
// created=false (with a nil error) when a file already existed and was left
// untouched — enforcing the user-owned, never-overwrite invariant. The parent
// directory is created if needed. apiKey selects the api_key directive written
// into the file; pass "" for the recommended default.
func WriteStarter(path string, apiKey Value) (created bool, err error) {
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

	if _, writeErr := f.WriteString(StarterContent(apiKey)); writeErr != nil {
		return false, fmt.Errorf("config: writing %s: %w", path, writeErr)
	}
	return true, nil
}
