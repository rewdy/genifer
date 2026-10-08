package config

import (
	"os"
	"path/filepath"
	"testing"
)

// docExampleConfig mirrors the example config.yaml in docs/config.md. If the
// documented example changes, keep this in sync — it guards that the doc is
// loadable.
const docExampleConfig = `provider: openrouter

openrouter:
  api_key: "{env:OPENROUTER_API_KEY}"

output_dir: "{env:HOME}/Pictures/genifer"

auto_open: false
`

func TestDocExampleLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(docExampleConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("documented example config.yaml failed to load: %v", err)
	}
	if cfg.Provider != "openrouter" {
		t.Errorf("Provider = %q", cfg.Provider)
	}
	if cfg.OpenRouter.APIKey != "{env:OPENROUTER_API_KEY}" {
		t.Errorf("APIKey = %q", cfg.OpenRouter.APIKey)
	}
}
