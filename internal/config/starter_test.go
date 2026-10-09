package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteStarterWhenAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	created, err := WriteStarter(path, StarterSpec{})
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
	existing := []byte("providers:\n  - key: or\n    type: openrouter\n# user content\n")
	if err := os.WriteFile(path, existing, 0o600); err != nil {
		t.Fatal(err)
	}
	created, err := WriteStarter(path, StarterSpec{})
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

// TestStarterRoundTrip verifies a written starter loads back into the expected
// single provider instance, for both an OpenRouter and an a1111 starter.
func TestStarterRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		spec       StarterSpec
		wantKey    string
		wantType   string
		wantAPIKey Value
		wantBase   string
	}{
		{
			name:       "openrouter default",
			spec:       StarterSpec{},
			wantKey:    "openrouter",
			wantType:   TypeOpenRouter,
			wantAPIKey: DefaultAPIKey,
		},
		{
			name:       "openrouter custom key method",
			spec:       StarterSpec{Key: "or", Type: TypeOpenRouter, APIKey: "{cmd:op read op://v/k}"},
			wantKey:    "or",
			wantType:   TypeOpenRouter,
			wantAPIKey: "{cmd:op read op://v/k}",
		},
		{
			name:     "a1111 default base url",
			spec:     StarterSpec{Type: TypeA1111},
			wantKey:  "local",
			wantType: TypeA1111,
			wantBase: DefaultA1111BaseURL,
		},
		{
			name:     "a1111 custom base url",
			spec:     StarterSpec{Key: "gpu-box", Type: TypeA1111, BaseURL: "http://10.0.0.2:7860"},
			wantKey:  "gpu-box",
			wantType: TypeA1111,
			wantBase: "http://10.0.0.2:7860",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if _, err := WriteStarter(path, tt.spec); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load of generated starter failed: %v", err)
			}
			if len(cfg.Providers) != 1 {
				t.Fatalf("len(Providers) = %d, want 1", len(cfg.Providers))
			}
			p := cfg.Providers[0]
			if p.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", p.Key, tt.wantKey)
			}
			if p.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", p.Type, tt.wantType)
			}
			if tt.wantAPIKey != "" && p.APIKey != tt.wantAPIKey {
				t.Errorf("APIKey = %q, want %q", p.APIKey, tt.wantAPIKey)
			}
			if tt.wantBase != "" && p.BaseURL != tt.wantBase {
				t.Errorf("BaseURL = %q, want %q", p.BaseURL, tt.wantBase)
			}
		})
	}
}
