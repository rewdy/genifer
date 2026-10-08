package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPresent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
provider: openrouter
auto_open: true
output_dir: "/tmp/genifer-out"
openrouter:
  api_key: "{env:OPENROUTER_API_KEY}"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider != "openrouter" {
		t.Errorf("Provider = %q", cfg.Provider)
	}
	if !cfg.AutoOpen {
		t.Error("AutoOpen = false, want true")
	}
	if cfg.OutputDir != "/tmp/genifer-out" {
		t.Errorf("OutputDir = %q", cfg.OutputDir)
	}
	if cfg.OpenRouter.APIKey != "{env:OPENROUTER_API_KEY}" {
		t.Errorf("APIKey = %q", cfg.OpenRouter.APIKey)
	}
}

func TestLoadMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	cfg, err := Load(path)
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("err = %v, want ErrConfigNotFound", err)
	}
	// Still returns usable defaults.
	if cfg.Provider != "openrouter" {
		t.Errorf("default Provider = %q, want openrouter", cfg.Provider)
	}
}

func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("provider: [unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error for malformed YAML, got nil")
	}
	if errors.Is(err, ErrConfigNotFound) {
		t.Fatal("malformed file must not be reported as not-found")
	}
}
