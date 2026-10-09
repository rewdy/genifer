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
auto_open: true
output_dir: "/tmp/genifer-out"
providers:
  - key: or
    type: openrouter
    api_key: "{env:OPENROUTER_API_KEY}"
  - key: local
    type: a1111
    base_url: "http://127.0.0.1:7860"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(cfg.Providers))
	}
	if cfg.Providers[0].Key != "or" || cfg.Providers[0].Type != TypeOpenRouter {
		t.Errorf("Providers[0] = %+v", cfg.Providers[0])
	}
	if cfg.Providers[0].APIKey != "{env:OPENROUTER_API_KEY}" {
		t.Errorf("Providers[0].APIKey = %q", cfg.Providers[0].APIKey)
	}
	if cfg.Providers[1].Key != "local" || cfg.Providers[1].Type != TypeA1111 {
		t.Errorf("Providers[1] = %+v", cfg.Providers[1])
	}
	if cfg.Providers[1].BaseURL != "http://127.0.0.1:7860" {
		t.Errorf("Providers[1].BaseURL = %q", cfg.Providers[1].BaseURL)
	}
	if !cfg.AutoOpen {
		t.Error("AutoOpen = false, want true")
	}
	if cfg.OutputDir != "/tmp/genifer-out" {
		t.Errorf("OutputDir = %q", cfg.OutputDir)
	}
}

func TestLoadMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	cfg, err := Load(path)
	if !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("err = %v, want ErrConfigNotFound", err)
	}
	// Still returns usable defaults: a single OpenRouter instance.
	if len(cfg.Providers) != 1 || cfg.Providers[0].Type != TypeOpenRouter {
		t.Errorf("default Providers = %+v, want one openrouter instance", cfg.Providers)
	}
}

func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("providers: [unterminated\n"), 0o600); err != nil {
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

func TestLoadLegacyShape(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"provider only", "provider: openrouter\n"},
		{"openrouter block only", "openrouter:\n  api_key: \"sk-x\"\n"},
		{"both", "provider: openrouter\nopenrouter:\n  api_key: \"sk-x\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if !errors.Is(err, ErrLegacyConfig) {
				t.Fatalf("err = %v, want ErrLegacyConfig", err)
			}
		})
	}
}

func TestValidateProviders(t *testing.T) {
	tests := []struct {
		name      string
		providers []ProviderConfig
		wantErr   bool
	}{
		{
			name: "valid",
			providers: []ProviderConfig{
				{Key: "or", Type: TypeOpenRouter},
				{Key: "local", Type: TypeA1111},
			},
		},
		{
			name: "duplicate key",
			providers: []ProviderConfig{
				{Key: "or", Type: TypeOpenRouter},
				{Key: "or", Type: TypeA1111},
			},
			wantErr: true,
		},
		{
			name:      "empty key",
			providers: []ProviderConfig{{Key: "", Type: TypeOpenRouter}},
			wantErr:   true,
		},
		{
			name:      "empty type",
			providers: []ProviderConfig{{Key: "or", Type: ""}},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProviders(tt.providers)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateProviders err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
