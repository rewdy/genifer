package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteStarterWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	created, err := WriteStarter(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("created = false, want true when file is absent")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file written: %v", err)
	}
}

func TestWriteStarterSkipsWhenPresent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	existing := []byte("provider: openrouter\n# user content\n")
	if err := os.WriteFile(path, existing, 0o600); err != nil {
		t.Fatal(err)
	}
	created, err := WriteStarter(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatal("created = true, want false when file exists")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(existing) {
		t.Errorf("existing file was modified:\n%s", got)
	}
}

func TestWriteStarterAPIKeyChoice(t *testing.T) {
	tests := []struct {
		name   string
		apiKey Value
		want   Value
	}{
		{"default", "", DefaultAPIKey},
		{"env", "{env:MY_KEY}", "{env:MY_KEY}"},
		{"cmd", "{cmd:op read op://v/k}", "{cmd:op read op://v/k}"},
		{"literal", "sk-literal-123", "sk-literal-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if _, err := WriteStarter(path, tt.apiKey); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load of generated starter failed: %v", err)
			}
			if cfg.OpenRouter.APIKey != tt.want {
				t.Errorf("api_key = %q, want %q", cfg.OpenRouter.APIKey, tt.want)
			}
		})
	}
}

// TestStarterLoadsAsValidConfig guards against the template drifting out of
// sync with the schema: the generated file must parse cleanly through Load.
func TestStarterLoadsAsValidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if _, err := WriteStarter(path, ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("generated starter did not load as valid config: %v", err)
	}
	if cfg.Provider != "openrouter" {
		t.Errorf("Provider = %q, want openrouter", cfg.Provider)
	}
}
