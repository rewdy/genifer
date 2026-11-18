package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

func filterKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func filterTestModels() []provider.Model {
	return []provider.Model{
		{ID: "alpha", Name: "Alpha", Capabilities: provider.Capabilities{SupportsTransparentBackground: true, SupportsSeed: true}},
		{ID: "beta", Name: "Beta", Capabilities: provider.Capabilities{SupportsAnimatedGIF: true, AcceptsReferenceImages: true}},
		{ID: "gamma", Name: "Gamma", Capabilities: provider.Capabilities{OutputFormats: []string{"svg"}}},
		{ID: "delta", Name: "Delta", Capabilities: provider.Capabilities{}},
	}
}

func filterPickerModel(models []provider.Model) Model {
	m := New(Deps{})
	m.modelsReady = true
	m.models = models
	m.rebuildItems()
	return m
}

func visibleModelIDs(m Model) []string {
	var ids []string
	for _, it := range m.picker.VisibleItems() {
		if mi, ok := it.(modelItem); ok {
			ids = append(ids, mi.m.ID)
		}
	}
	return ids
}

func TestPickerFilterMatchesCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name    string
		filters pickerFilters
		caps    provider.Capabilities
		want    bool
	}{
		{"none active", pickerFilters{}, provider.Capabilities{}, true},
		{"transparent", pickerFilters{Transparent: true}, provider.Capabilities{SupportsTransparentBackground: true}, true},
		{"transparent missing", pickerFilters{Transparent: true}, provider.Capabilities{}, false},
		{"gif", pickerFilters{AnimatedGIF: true}, provider.Capabilities{SupportsAnimatedGIF: true}, true},
		{"refs", pickerFilters{References: true}, provider.Capabilities{AcceptsReferenceImages: true}, true},
		{"seed", pickerFilters{Seed: true}, provider.Capabilities{SupportsSeed: true}, true},
		{"svg", pickerFilters{SVG: true}, provider.Capabilities{OutputFormats: []string{"svg"}}, true},
		{"svg missing", pickerFilters{SVG: true}, provider.Capabilities{OutputFormats: []string{"png"}}, false},
		{"multiple", pickerFilters{Transparent: true, Seed: true}, provider.Capabilities{SupportsTransparentBackground: true, SupportsSeed: true}, true},
		{"multiple partial", pickerFilters{Transparent: true, Seed: true}, provider.Capabilities{SupportsTransparentBackground: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.filters.matches(tc.caps); got != tc.want {
				t.Errorf("matches = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPickerFilterToggleKeys(t *testing.T) {
	f := pickerFilters{}
	for _, tc := range []struct {
		key  string
		want pickerFilters
	}{
		{"1", pickerFilters{Transparent: true}},
		{"2", pickerFilters{Transparent: true, AnimatedGIF: true}},
		{"3", pickerFilters{Transparent: true, AnimatedGIF: true, References: true}},
		{"4", pickerFilters{Transparent: true, AnimatedGIF: true, References: true, Seed: true}},
		{"5", pickerFilters{Transparent: true, AnimatedGIF: true, References: true, Seed: true, SVG: true}},
	} {
		if !f.toggle(tc.key) {
			t.Fatalf("toggle(%s) reported no change", tc.key)
		}
		if f != tc.want {
			t.Fatalf("after %s: %+v, want %+v", tc.key, f, tc.want)
		}
	}
	if !f.toggle("0") || f.active() {
		t.Fatalf("0 should clear all filters, got %+v", f)
	}
	if f.toggle("0") {
		t.Error("0 with no active filters should be a no-op")
	}
	if f.toggle("x") {
		t.Error("unknown key should not change filters")
	}
}

func TestPickerCapabilityFiltersNarrowModels(t *testing.T) {
	m := filterPickerModel(filterTestModels())

	for _, tc := range []struct {
		key  string
		want []string
	}{
		{"1", []string{"alpha"}},
		{"2", []string{"beta"}},
		{"3", []string{"beta"}},
		{"4", []string{"alpha"}},
		{"5", []string{"gamma"}},
	} {
		t.Run(tc.key, func(t *testing.T) {
			next, _ := m.handlePickerKey(filterKey(tc.key))
			got := visibleModelIDs(next.(Model))
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("filter %s: visible = %v, want %v", tc.key, got, tc.want)
			}
			clear, _ := next.(Model).handlePickerKey(filterKey("0"))
			if len(visibleModelIDs(clear.(Model))) != 4 {
				t.Error("0 should restore all models")
			}
		})
	}
}

func TestPickerCapabilityFiltersCombine(t *testing.T) {
	m := filterPickerModel(filterTestModels())
	next, _ := m.handlePickerKey(filterKey("1"))
	m = next.(Model)
	next, _ = m.handlePickerKey(filterKey("4"))
	m = next.(Model)
	if got := visibleModelIDs(m); strings.Join(got, ",") != "alpha" {
		t.Errorf("transparent + seed = %v, want alpha", got)
	}
	if !strings.Contains(m.status, "transparent, seed") {
		t.Errorf("status = %q, want active filter names", m.status)
	}
}

func TestPickerCapabilityFilterWithTextQuery(t *testing.T) {
	m := filterPickerModel(filterTestModels())
	next, _ := m.handlePickerKey(filterKey("2"))
	m = next.(Model)
	m.picker.SetFilterText("beta")
	if got := visibleModelIDs(m); strings.Join(got, ",") != "beta" {
		t.Errorf("gif + text query = %v, want beta", got)
	}
	m.picker.SetFilterText("alpha")
	if len(visibleModelIDs(m)) != 0 {
		t.Errorf("gif + non-matching text = %v, want none", gotIDs(visibleModelIDs(m)))
	}
	if view := ansi.Strip(m.pickerView()); !strings.Contains(view, "No models match") {
		t.Errorf("empty result message missing; got %q", view)
	}
}

func gotIDs(ids []string) string { return strings.Join(ids, ",") }

func TestPickerCapabilityFilterPreservesFavoritesAndSelection(t *testing.T) {
	models := filterTestModels()
	m := New(Deps{})
	m.modelsReady = true
	m.models = models
	m.favorites = map[string]struct{}{"delta": {}}
	m.picker.Select(3) // delta is last before favorites-first reorder
	m.rebuildItems()
	m.selectPickItemByID("beta")
	m.rebuildItems()

	// With no filters, favorites sort first. Filter to GIF-capable models and
	// the selected beta model must remain selected.
	next, _ := m.handlePickerKey(filterKey("2"))
	m = next.(Model)
	ids := visibleModelIDs(m)
	if strings.Join(ids, ",") != "beta" {
		t.Errorf("visible = %v, want beta", ids)
	}
	if it, ok := m.picker.SelectedItem().(modelItem); !ok || it.m.ID != "beta" {
		t.Errorf("selection lost: %+v", m.picker.SelectedItem())
	}

	// Adding a second filter that beta also satisfies keeps it selected.
	next, _ = m.handlePickerKey(filterKey("3"))
	m = next.(Model)
	if got := visibleModelIDs(m); strings.Join(got, ",") != "beta" {
		t.Errorf("gif + refs = %v, want beta", got)
	}
	if it, ok := m.picker.SelectedItem().(modelItem); !ok || it.m.ID != "beta" {
		t.Errorf("selection lost: %+v", m.picker.SelectedItem())
	}

	// A filter no selected model satisfies yields an empty, navigable list.
	next, _ = m.handlePickerKey(filterKey("5"))
	m = next.(Model)
	if len(visibleModelIDs(m)) != 0 {
		t.Errorf("gif + refs + svg should match nothing, got %v", visibleModelIDs(m))
	}
}

func TestPickerCapabilityEmptyFilterMessage(t *testing.T) {
	m := filterPickerModel([]provider.Model{{ID: "plain", Name: "Plain"}})
	next, _ := m.handlePickerKey(filterKey("2"))
	m = next.(Model)
	if len(visibleModelIDs(m)) != 0 {
		t.Fatal("gif filter should exclude the plain model")
	}
	view := ansi.Strip(m.pickerView())
	if !strings.Contains(view, "No models match") || !strings.Contains(view, "0 to clear") {
		t.Errorf("empty filter message missing; got %q", view)
	}
	if !strings.Contains(m.status, "No models match") {
		t.Errorf("status = %q, want no-match message", m.status)
	}
}

func TestPickerFilterHints(t *testing.T) {
	m := filterPickerModel(filterTestModels())
	footer := ansi.Strip(m.footerView())
	for _, want := range []string{"1-5", "cap filters", "0", "clear"} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer missing %q; got %q", want, footer)
		}
	}
	view := ansi.Strip(m.pickerView())
	for _, want := range []string{"1 transparent", "2 GIF", "3 refs", "4 seed", "5 SVG", "0 clear"} {
		if !strings.Contains(view, want) {
			t.Errorf("picker legend missing %q; got %q", want, view)
		}
	}
}

func TestPickerFiltersDoNotAffectComposeControls(t *testing.T) {
	m := filterPickerModel(filterTestModels())
	next, _ := m.handlePickerKey(filterKey("2"))
	m = next.(Model)
	next, _ = m.handlePickerKey(filterKey("enter"))
	m = next.(Model)
	if m.phase != phaseCompose {
		t.Fatalf("phase = %v, want compose", m.phase)
	}
	if m.currentModelID() != "beta" {
		t.Errorf("selected model = %q, want beta", m.currentModelID())
	}
	// An active filter is a picker concern only; it must not leak into the
	// form's capabilities.
	if controlsFor(m.currentCaps()).ShowAnimatedGIF != true {
		t.Error("selected GIF model should show the GIF control")
	}
}

// Ensure the list delegate still sees filtered items with full model data.
func TestPickerFilterKeepsModelMetadata(t *testing.T) {
	m := filterPickerModel(filterTestModels())
	next, _ := m.handlePickerKey(filterKey("5"))
	m = next.(Model)
	it, ok := m.picker.SelectedItem().(modelItem)
	if !ok {
		t.Fatal("no selected model item")
	}
	if it.m.ID != "gamma" || len(it.m.Capabilities.OutputFormats) != 1 {
		t.Errorf("filtered item lost metadata: %+v", it)
	}
}
