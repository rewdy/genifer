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
	if err := os.WriteFile(cfgPath, []byte("providers:\n  - key: or\n    type: openrouter\n"), 0o600); err != nil {
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

func TestStateAspectRatioRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	in := State{LastModel: "google/gemini-2.5-flash-image", LastAspectRatio: "16:9"}
	if err := SaveState(path, in); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got := LoadState(path)
	if got.LastAspectRatio != "16:9" {
		t.Errorf("LastAspectRatio = %q, want %q", got.LastAspectRatio, "16:9")
	}
	if got.LastModel != in.LastModel {
		t.Errorf("LastModel = %q, want %q", got.LastModel, in.LastModel)
	}
}

func TestStateAspectRatioEmptyWhenMissing(t *testing.T) {
	if got := LoadState(filepath.Join(t.TempDir(), "absent.json")); got.LastAspectRatio != "" {
		t.Errorf("missing file: LastAspectRatio = %q, want empty", got.LastAspectRatio)
	}
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadState(path); got.LastAspectRatio != "" {
		t.Errorf("invalid file: LastAspectRatio = %q, want empty", got.LastAspectRatio)
	}
}

// TestStateProviderKeyPreservesOthers verifies that persisting the model and
// its provider key via read-modify-write keeps a previously remembered aspect
// ratio, and that the provider key round-trips.
func TestStateProviderKeyPreservesOthers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	// First remember an aspect ratio.
	if err := SaveState(path, State{LastAspectRatio: "16:9"}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	// Then persist a model + provider key, read-modify-write.
	s := LoadState(path)
	s.LastModel = "sd-xl"
	s.LastProviderKey = "local"
	if err := SaveState(path, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got := LoadState(path)
	if got.LastModel != "sd-xl" {
		t.Errorf("LastModel = %q, want sd-xl", got.LastModel)
	}
	if got.LastProviderKey != "local" {
		t.Errorf("LastProviderKey = %q, want local", got.LastProviderKey)
	}
	if got.LastAspectRatio != "16:9" {
		t.Errorf("LastAspectRatio = %q, want 16:9 (should be preserved)", got.LastAspectRatio)
	}
}
