package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	if err := SaveState(path, State{LastModel: "google/gemini-2.5-flash-image"}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got := LoadState(path)
	if got.LastModel != "google/gemini-2.5-flash-image" {
		t.Errorf("LastModel = %q", got.LastModel)
	}
}

func TestStateMissing(t *testing.T) {
	got := LoadState(filepath.Join(t.TempDir(), "absent.json"))
	if got.LastModel != "" {
		t.Errorf("expected zero State for missing file, got %+v", got)
	}
}

func TestStateInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadState(path)
	if got.LastModel != "" {
		t.Errorf("expected zero State for invalid file, got %+v", got)
	}
}

func TestSaveStateDoesNotTouchConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("provider: openrouter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(cfgPath)
	if err := SaveState(filepath.Join(dir, "state.json"), State{LastModel: "x"}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(cfgPath)
	if string(before) != string(after) {
		t.Error("SaveState must not modify config.yaml")
	}
}
