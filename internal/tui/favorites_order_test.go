package tui

import (
	"path/filepath"
	"testing"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
)

// pickerModel builds a picker-backed model from a saved state and applies
// modelsLoadedMsg, as the app does when the provider's list arrives.
func pickerModel(t *testing.T, st config.State, models []provider.Model) Model {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := config.SaveState(statePath, st); err != nil {
		t.Fatal(err)
	}
	m := New(Deps{StatePath: statePath})
	updated, _ := m.Update(modelsLoadedMsg{models: models})
	return updated.(Model)
}

func pickerIDs(m Model) []string {
	items := m.picker.Items()
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.(modelItem).m.ID
	}
	return ids
}

var testModels = []provider.Model{
	{ID: "a/one", Name: "One"},
	{ID: "b/two", Name: "Two"},
	{ID: "c/three", Name: "Three"},
}

// 2.3: favorited models sort first, the item count still matches the model
// count, and each item carries the favorite flag the delegate renders.
func TestPickerOrdersFavoritesFirst(t *testing.T) {
	m := pickerModel(t, config.State{Favorites: []string{"c/three"}}, testModels)

	ids := pickerIDs(m)
	if len(ids) != len(testModels) {
		t.Fatalf("picker items = %d, want %d", len(ids), len(testModels))
	}
	want := []string{"c/three", "a/one", "b/two"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("item order = %v, want %v", ids, want)
		}
	}
	if !m.picker.Items()[0].(modelItem).favorite {
		t.Error("favorited model item has favorite=false")
	}
	if m.picker.Items()[1].(modelItem).favorite {
		t.Error("non-favorited model item has favorite=true")
	}
}

// 2.2: the remembered model is highlighted by id, not by its index in the
// provider list, so favorites reordering does not change what is pre-selected.
func TestPreselectFollowsModelAcrossFavoritesReorder(t *testing.T) {
	m := pickerModel(t, config.State{
		LastModel: "a/one",
		Favorites: []string{"c/three"},
	}, testModels)

	if got := m.picker.Index(); got != 1 {
		t.Fatalf("picker index = %d, want 1 (a/one is second after the favorite)", got)
	}
	if it, ok := m.picker.SelectedItem().(modelItem); !ok || it.m.ID != "a/one" {
		t.Fatalf("selected item = %+v, want a/one", m.picker.SelectedItem())
	}
}

// 4.1: a favorite the provider no longer lists adds no row and no error.
func TestStaleFavoriteIsIgnored(t *testing.T) {
	m := pickerModel(t, config.State{Favorites: []string{"x/gone", "b/two"}}, testModels)

	ids := pickerIDs(m)
	if len(ids) != len(testModels) {
		t.Fatalf("picker items = %d, want %d (stale favorite must not add a row)", len(ids), len(testModels))
	}
	want := []string{"b/two", "a/one", "c/three"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("item order = %v, want %v", ids, want)
		}
	}
	if m.modelsErr != nil {
		t.Fatalf("modelsErr = %v, want nil", m.modelsErr)
	}
}
