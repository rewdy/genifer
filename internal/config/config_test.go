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

// 2.1: history defaults to enabled; an absent key means enabled, and only an
// explicit `history: false` disables it.
func TestHistoryEnabledResolution(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want bool
	}{
		{"absent", "provider: openrouter\n", true},
		{"true", "provider: openrouter\nhistory: true\n", true},
		{"false", "provider: openrouter\nhistory: false\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cfg.HistoryEnabled(); got != tc.want {
				t.Errorf("HistoryEnabled() = %v, want %v", got, tc.want)
			}
		})
	}

	// A Default() config (no file) is enabled.
	if !Default().HistoryEnabled() {
		t.Error("Default() should have history enabled")
	}
}

// 1.1: smart_filenames and filename_model load from YAML when set, and default
// to false/empty when unset.
func TestLoadSmartFilenamesKeys(t *testing.T) {
	cases := []struct {
		name            string
		yaml            string
		wantEnabled     bool
		wantModel       Value
	}{
		{"absent", "provider: openrouter\n", false, ""},
		{"set", "provider: openrouter\nsmart_filenames: true\nfilename_model: \"{env:SMART_NAME_MODEL}\"\n", true, "{env:SMART_NAME_MODEL}"},
		{"explicit-false", "provider: openrouter\nsmart_filenames: false\n", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.SmartFilenames != tc.wantEnabled {
				t.Errorf("SmartFilenames = %v, want %v", cfg.SmartFilenames, tc.wantEnabled)
			}
			if cfg.FilenameModel != tc.wantModel {
				t.Errorf("FilenameModel = %q, want %q", cfg.FilenameModel, tc.wantModel)
			}
		})
	}
}
