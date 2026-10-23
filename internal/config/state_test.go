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

func TestFavoriteSet(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"normal", []string{"a/one", "b/two"}, []string{"a/one", "b/two"}},
		{"duplicates", []string{"a/one", "a/one"}, []string{"a/one"}},
		{"blank ids ignored", []string{"", "a/one", ""}, []string{"a/one"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := State{Favorites: tc.in}.FavoriteSet()
			if len(got) != len(tc.want) {
				t.Fatalf("FavoriteSet(%v) = %v, want %v", tc.in, got, tc.want)
			}
			for _, id := range tc.want {
				if _, ok := got[id]; !ok {
					t.Errorf("FavoriteSet(%v) missing %q", tc.in, id)
				}
			}
		})
	}
}

func TestStateFavoritesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	in := State{
		LastModel:       "google/gemini-2.5-flash-image",
		LastAspectRatio: "16:9",
		Favorites:       []string{"a/one", "b/two"},
	}
	if err := SaveState(path, in); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got := LoadState(path)
	if len(got.Favorites) != 2 || got.Favorites[0] != "a/one" || got.Favorites[1] != "b/two" {
		t.Errorf("Favorites = %v, want %v", got.Favorites, in.Favorites)
	}
	if got.LastModel != in.LastModel {
		t.Errorf("LastModel = %q, want %q", got.LastModel, in.LastModel)
	}
	if got.LastAspectRatio != in.LastAspectRatio {
		t.Errorf("LastAspectRatio = %q, want %q", got.LastAspectRatio, in.LastAspectRatio)
	}
}

func TestSavingFavoritesPreservesOtherSelections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := SaveState(path, State{LastModel: "a/one", LastAspectRatio: "1:1"}); err != nil {
		t.Fatal(err)
	}
	s := LoadState(path)
	s.Favorites = []string{"b/two"}
	if err := SaveState(path, s); err != nil {
		t.Fatal(err)
	}
	got := LoadState(path)
	if got.LastModel != "a/one" || got.LastAspectRatio != "1:1" {
		t.Errorf("saving favorites clobbered selections: %+v", got)
	}
	if len(got.Favorites) != 1 || got.Favorites[0] != "b/two" {
		t.Errorf("Favorites = %v, want [b/two]", got.Favorites)
	}
}

func TestStateFavoritesEmptyWhenMissingOrInvalid(t *testing.T) {
	if got := LoadState(filepath.Join(t.TempDir(), "absent.json")); len(got.FavoriteSet()) != 0 {
		t.Errorf("missing file: favorites = %v, want empty", got.Favorites)
	}
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadState(path); len(got.FavoriteSet()) != 0 {
		t.Errorf("invalid file: favorites = %v, want empty", got.Favorites)
	}
}
