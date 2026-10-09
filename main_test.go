package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rewdy/genifer/internal/config"
)

// 2.3: the load-outcome branch — missing, legacy, malformed, valid.
func TestLoadConfigOutcomes(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		cfg, outcome, err := loadConfig(path)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if outcome != loadMissing {
			t.Errorf("outcome = %v, want loadMissing", outcome)
		}
		if len(cfg.Providers) != 1 {
			t.Errorf("want default one-instance config, got %+v", cfg.Providers)
		}
	})

	t.Run("legacy backs up and enters first run", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, []byte("provider: openrouter\nopenrouter:\n  api_key: \"sk-x\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, outcome, err := loadConfig(path)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if outcome != loadLegacy {
			t.Errorf("outcome = %v, want loadLegacy", outcome)
		}
		if _, err := os.Stat(path + ".bak"); err != nil {
			t.Errorf("expected backup at %s.bak: %v", path, err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("legacy config should have been renamed away")
		}
		if len(cfg.Providers) != 1 {
			t.Errorf("want default config after legacy recovery, got %+v", cfg.Providers)
		}
	})

	t.Run("malformed is a hard error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte("providers: [unterminated\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := loadConfig(path); err == nil {
			t.Fatal("want a hard error for malformed config")
		}
	})

	t.Run("valid loads", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		content := "providers:\n  - key: or\n    type: openrouter\n    api_key: \"{env:OPENROUTER_API_KEY}\"\n"
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, outcome, err := loadConfig(path)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if outcome != loadValid {
			t.Errorf("outcome = %v, want loadValid", outcome)
		}
		if len(cfg.Providers) != 1 || cfg.Providers[0].Key != "or" {
			t.Errorf("unexpected providers %+v", cfg.Providers)
		}
	})
}

// 5.1: buildProviders constructs the registry and reports unknown types.
func TestBuildProviders(t *testing.T) {
	cfg := config.Config{Providers: []config.ProviderConfig{
		{Key: "or", Type: config.TypeOpenRouter, APIKey: "{env:X}"},
		{Key: "local", Type: config.TypeA1111, BaseURL: "http://127.0.0.1:7860"},
	}}
	reg, keys, err := buildProviders(cfg)
	if err != nil {
		t.Fatalf("buildProviders: %v", err)
	}
	if len(reg) != 2 || reg["or"] == nil || reg["local"] == nil {
		t.Errorf("registry = %+v, want or+local", reg)
	}
	if len(keys) != 2 || keys[0] != "or" || keys[1] != "local" {
		t.Errorf("keys = %v, want [or local] in order", keys)
	}

	bad := config.Config{Providers: []config.ProviderConfig{{Key: "mystery", Type: "nope"}}}
	if _, _, err := buildProviders(bad); err == nil {
		t.Fatal("want an error for an unknown provider type")
	}
}
