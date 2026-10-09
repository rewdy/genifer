package config

import (
	"os"
	"path/filepath"
	"testing"
)

// docExampleConfig mirrors the example config.yaml in docs/config.md. If the
// documented example changes, keep this in sync — it guards that the doc is
// loadable.
const docExampleConfig = `providers:
  - key: or
    type: openrouter
    api_key: "{env:OPENROUTER_API_KEY}"
  - key: local
    type: a1111
    base_url: "http://127.0.0.1:7860"

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
	if len(cfg.Providers) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(cfg.Providers))
	}
	if cfg.Providers[0].Type != TypeOpenRouter || cfg.Providers[0].APIKey != "{env:OPENROUTER_API_KEY}" {
		t.Errorf("Providers[0] = %+v", cfg.Providers[0])
	}
	if cfg.Providers[1].Type != TypeA1111 || cfg.Providers[1].BaseURL != "http://127.0.0.1:7860" {
		t.Errorf("Providers[1] = %+v", cfg.Providers[1])
	}
}
