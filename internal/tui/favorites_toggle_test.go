package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
)

// 3.1: f toggles the highlighted model, persists it, and keeps the highlight on
// that model even though it moved.
func TestFavoriteTogglePersistsAndKeepsHighlight(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := config.SaveState(statePath, config.State{LastModel: "b/two"}); err != nil {
		t.Fatal(err)
	}
	m := New(Deps{StatePath: statePath})
	updated, _ := m.Update(modelsLoadedMsg{models: testModels})
	m = updated.(Model)
	m.selectPickItemByID("b/two")

	updated, _ = m.handlePickerKey(key("f"))
	m = updated.(Model)

	st := config.LoadState(statePath)
	if len(st.Favorites) != 1 || st.Favorites[0] != "b/two" {
		t.Fatalf("persisted favorites = %v, want [b/two]", st.Favorites)
	}
	if st.LastModel != "b/two" {
		t.Errorf("LastModel = %q, want b/two (favoriting must preserve other selections)", st.LastModel)
	}
	if got := pickerIDs(m); got[0] != "b/two" {
		t.Fatalf("item order = %v, want b/two first", got)
	}
	if it, ok := m.picker.SelectedItem().(modelItem); !ok || it.m.ID != "b/two" {
		t.Fatalf("highlight = %+v, want b/two", m.picker.SelectedItem())
	}

	// Toggling again removes it and returns it to its normal position.
	updated, _ = m.handlePickerKey(key("f"))
	m = updated.(Model)
	st = config.LoadState(statePath)
	if len(st.Favorites) != 0 {
		t.Fatalf("favorites after untoggle = %v, want empty", st.Favorites)
	}
	if got := pickerIDs(m); got[0] != "a/one" {
		t.Fatalf("item order = %v, want provider order restored", got)
	}
	if it, _ := m.picker.SelectedItem().(modelItem); it.favorite {
		t.Error("item still marked favorite after untoggle")
	}
}

// 3.2: while the picker filter is active, f is typing, not a toggle.
func TestFavoriteToggleInactiveWhileFiltering(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m := New(Deps{StatePath: statePath})
	updated, _ := m.Update(modelsLoadedMsg{models: testModels})
	m = updated.(Model)
	m.selectPickItemByID("a/one")

	updated, _ = m.handlePickerKey(key("/"))
	m = updated.(Model)
	if m.picker.FilterState() != list.Filtering {
		t.Fatalf("filter state = %v, want Filtering", m.picker.FilterState())
	}

	updated, _ = m.handlePickerKey(key("f"))
	m = updated.(Model)

	if got := m.picker.FilterInput.Value(); got != "f" {
		t.Errorf("filter value = %q, want %q", got, "f")
	}
	if len(m.favorites) != 0 {
		t.Errorf("favorites = %v, want none", m.favorites)
	}
	if _, err := os.Stat(statePath); err == nil {
		if st := config.LoadState(statePath); len(st.Favorites) != 0 {
			t.Errorf("persisted favorites = %v, want none", st.Favorites)
		}
	}
}

// 4.2: toggling a favorite never writes config.yaml.
func TestFavoriteToggleLeavesConfigUntouched(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("provider: openrouter\nauto_open: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	m := New(Deps{StatePath: filepath.Join(dir, "state.json"), ConfigPath: cfgPath})
	updated, _ := m.Update(modelsLoadedMsg{models: []provider.Model{{ID: "a/one", Name: "One"}}})
	m = updated.(Model)
	updated, _ = m.handlePickerKey(key("f"))
	_ = updated.(Model)

	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("config.yaml changed:\nbefore: %q\nafter:  %q", before, after)
	}
}
