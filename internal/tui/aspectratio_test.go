package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
)

// aspectModel builds a Model in the compose phase with a single model selected
// whose offered aspect ratios are `ratios`, and a current selection of `sel`.
func aspectModel(ratios []string, sel string) Model {
	m := New(Deps{})
	m.models = []taggedModel{{ProviderKey: "test", Model: provider.Model{
		ID:           "test/model",
		Name:         "Test Model",
		Capabilities: provider.Capabilities{AspectRatios: ratios},
	}}}
	m.selected = 0
	m.phase = phaseCompose
	m.aspectRatio = sel
	m.prompt.Focus()
	return m
}

// 3.2: choosing a ratio-capable model from the picker sets a non-empty default
// selection, preferring the remembered last-used ratio when offered.
func TestChoosingModelDefaultsAspectRatio(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := config.SaveState(statePath, config.State{LastAspectRatio: "16:9"}); err != nil {
		t.Fatal(err)
	}
	m := New(Deps{StatePath: statePath})
	m.models = []taggedModel{{ProviderKey: "test", Model: provider.Model{
		ID:           "test/model",
		Name:         "Test Model",
		Capabilities: provider.Capabilities{AspectRatios: []string{"1:1", "16:9", "9:16"}},
	}}}
	m.rebuildItems()
	m.picker.Select(0)

	updated, _ := m.handlePickerKey(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.phase != phaseCompose {
		t.Fatalf("phase = %v, want compose", um.phase)
	}
	if um.aspectRatio != "16:9" {
		t.Errorf("aspectRatio = %q, want 16:9 (remembered)", um.aspectRatio)
	}
}

// 3.2: with no remembered ratio, the default is the model's first offered.
func TestChoosingModelDefaultsToFirstRatio(t *testing.T) {
	m := New(Deps{StatePath: filepath.Join(t.TempDir(), "state.json")})
	m.models = []taggedModel{{ProviderKey: "test", Model: provider.Model{
		ID:           "test/model",
		Capabilities: provider.Capabilities{AspectRatios: []string{"4:3", "1:1"}},
	}}}
	m.rebuildItems()
	m.picker.Select(0)

	updated, _ := m.handlePickerKey(tea.KeyMsg{Type: tea.KeyEnter})
	if got := updated.(Model).aspectRatio; got != "4:3" {
		t.Errorf("aspectRatio = %q, want 4:3 (first offered)", got)
	}
}

// 3.3: the composed draft carries the selected aspect ratio through to the
// provider request.
func TestComposeDraftCarriesAspectRatio(t *testing.T) {
	m := aspectModel([]string{"1:1", "16:9"}, "16:9")
	m.prompt.SetValue("a red bird")

	d := m.composeDraft()
	if d.AspectRatio != "16:9" {
		t.Errorf("draft AspectRatio = %q, want 16:9", d.AspectRatio)
	}
	req, err := d.Request()
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if req.AspectRatio != "16:9" {
		t.Errorf("request AspectRatio = %q, want 16:9", req.AspectRatio)
	}
}

// 4.1: ctrl+a from compose opens the dialog, seeding the cursor to the current
// selection; up/down move it; enter confirms; esc cancels.
func TestAspectDialogOpenMoveConfirm(t *testing.T) {
	m := aspectModel([]string{"1:1", "16:9", "9:16"}, "16:9")

	// Open: ctrl+a enters phaseAspect with the cursor on the current value.
	opened, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlA})
	m = opened.(Model)
	if m.phase != phaseAspect {
		t.Fatalf("phase = %v, want phaseAspect", m.phase)
	}
	if m.aspectCursor != 1 {
		t.Errorf("aspectCursor = %d, want 1 (index of 16:9)", m.aspectCursor)
	}

	// Move down to 9:16, then confirm.
	down, _ := m.handleAspectKey(tea.KeyMsg{Type: tea.KeyDown})
	m = down.(Model)
	if m.aspectCursor != 2 {
		t.Errorf("aspectCursor after down = %d, want 2", m.aspectCursor)
	}
	confirmed, _ := m.handleAspectKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = confirmed.(Model)
	if m.phase != phaseCompose {
		t.Errorf("phase after enter = %v, want phaseCompose", m.phase)
	}
	if m.aspectRatio != "9:16" {
		t.Errorf("aspectRatio after confirm = %q, want 9:16", m.aspectRatio)
	}
}

// 4.1: cursor movement is clamped at both ends.
func TestAspectDialogCursorClamped(t *testing.T) {
	m := aspectModel([]string{"1:1", "16:9"}, "1:1")
	m.phase = phaseAspect
	m.aspectCursor = 0

	up, _ := m.handleAspectKey(tea.KeyMsg{Type: tea.KeyUp})
	if got := up.(Model).aspectCursor; got != 0 {
		t.Errorf("cursor after up at top = %d, want 0", got)
	}
	m.aspectCursor = 1
	down, _ := m.handleAspectKey(tea.KeyMsg{Type: tea.KeyDown})
	if got := down.(Model).aspectCursor; got != 1 {
		t.Errorf("cursor after down at bottom = %d, want 1", got)
	}
}

// 4.1: esc closes the dialog without changing the selection.
func TestAspectDialogCancel(t *testing.T) {
	m := aspectModel([]string{"1:1", "16:9", "9:16"}, "16:9")
	m.phase = phaseAspect
	m.aspectCursor = 2 // highlighted 9:16 but not confirmed

	cancelled, _ := m.handleAspectKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = cancelled.(Model)
	if m.phase != phaseCompose {
		t.Errorf("phase after esc = %v, want phaseCompose", m.phase)
	}
	if m.aspectRatio != "16:9" {
		t.Errorf("aspectRatio after cancel = %q, want 16:9 (unchanged)", m.aspectRatio)
	}
}

// 4.1: ctrl+a is a no-op for a model that offers no aspect ratios (the key
// falls through to the textarea instead of opening a dialog).
func TestAspectDialogIgnoredForNoRatios(t *testing.T) {
	m := aspectModel(nil, "")
	updated, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlA})
	if got := updated.(Model).phase; got != phaseCompose {
		t.Errorf("phase = %v, want phaseCompose (no dialog for no-ratio model)", got)
	}
}

// 4.2: the dialog lists every offered ratio with the current one marked.
func TestAspectViewListsAllRatios(t *testing.T) {
	m := aspectModel([]string{"1:1", "16:9", "9:16"}, "16:9")
	m.phase = phaseAspect
	m.aspectCursor = 1
	out := ansi.Strip(m.aspectView())
	for _, r := range []string{"1:1", "16:9", "9:16"} {
		if !strings.Contains(out, r) {
			t.Errorf("aspect dialog missing %q; got %q", r, out)
		}
	}
}

// 4.2: the compose view shows the selected aspect ratio and the open-dialog
// hint for a ratio-capable model, and nothing for a model with no ratios.
func TestComposeViewAspectSelector(t *testing.T) {
	withRatios := ansi.Strip(aspectModel([]string{"1:1", "16:9"}, "16:9").composeView())
	if !strings.Contains(withRatios, "16:9") {
		t.Errorf("compose view missing selected ratio; got %q", withRatios)
	}
	if !strings.Contains(withRatios, "ctrl+a") {
		t.Errorf("compose view missing change hint; got %q", withRatios)
	}

	noRatios := ansi.Strip(aspectModel(nil, "").composeView())
	if strings.Contains(noRatios, "ctrl+a") {
		t.Errorf("compose view should not show aspect hint for no-ratio model; got %q", noRatios)
	}
}

// 4.3: the compose footer shows the aspect-ratio hint only when the control is
// visible.
func TestFooterAspectHint(t *testing.T) {
	withRatios := ansi.Strip(aspectModel([]string{"1:1"}, "1:1").footerView())
	if !strings.Contains(withRatios, "ctrl+a") || !strings.Contains(withRatios, "aspect") {
		t.Errorf("footer missing aspect hint; got %q", withRatios)
	}
	noRatios := ansi.Strip(aspectModel(nil, "").footerView())
	if strings.Contains(noRatios, "aspect") {
		t.Errorf("footer should not show aspect hint for no-ratio model; got %q", noRatios)
	}
}

// 5.1: generating persists the selected aspect ratio while preserving the
// remembered model, and picking a model preserves a remembered ratio.
func TestPersistAspectRatioOnGenerate(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := config.SaveState(statePath, config.State{LastModel: "test/model"}); err != nil {
		t.Fatal(err)
	}
	m := aspectModel([]string{"1:1", "16:9"}, "16:9")
	m.deps.StatePath = statePath
	m.prompt.SetValue("a red bird")

	m.persistAspectRatio()

	got := config.LoadState(statePath)
	if got.LastAspectRatio != "16:9" {
		t.Errorf("LastAspectRatio = %q, want 16:9", got.LastAspectRatio)
	}
	if got.LastModel != "test/model" {
		t.Errorf("LastModel = %q, want test/model (preserved)", got.LastModel)
	}
}

// 5.1: picking a model preserves an already-remembered aspect ratio.
func TestPickingModelPreservesAspectRatio(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := config.SaveState(statePath, config.State{LastAspectRatio: "16:9"}); err != nil {
		t.Fatal(err)
	}
	m := New(Deps{StatePath: statePath})
	m.models = []taggedModel{{ProviderKey: "test", Model: provider.Model{
		ID:           "test/model",
		Capabilities: provider.Capabilities{AspectRatios: []string{"1:1", "16:9"}},
	}}}
	m.rebuildItems()
	m.picker.Select(0)

	_, _ = m.handlePickerKey(tea.KeyMsg{Type: tea.KeyEnter})

	got := config.LoadState(statePath)
	if got.LastModel != "test/model" {
		t.Errorf("LastModel = %q, want test/model", got.LastModel)
	}
	if got.LastAspectRatio != "16:9" {
		t.Errorf("LastAspectRatio = %q, want 16:9 (preserved on model pick)", got.LastAspectRatio)
	}
}
