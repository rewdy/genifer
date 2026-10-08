package tui

import (
	"testing"

	"github.com/rewdy/genifer/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
)

// composeModel builds a Model sitting in the compose phase with a single model
// selected, whose reference-image capability is set by refCapable.
func composeModel(refCapable bool, paster imagePaster) Model {
	m := New(Deps{Paster: paster})
	m.models = []provider.Model{{
		ID:           "test/model",
		Name:         "Test Model",
		Capabilities: provider.Capabilities{AcceptsReferenceImages: refCapable},
	}}
	m.selected = 0
	m.phase = phaseCompose
	m.prompt.Focus()
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "ctrl+v":
		return tea.KeyMsg{Type: tea.KeyCtrlV}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// 3.1: paste key on an incapable model attaches nothing and does not trigger
// the paste flow — the key falls through to the textarea instead of being
// consumed by paste handling.
func TestPasteKeyIgnoredForIncapableModel(t *testing.T) {
	m := composeModel(false, fakePaster{data: makePNG(t)})
	updated, _ := m.handleComposeKey(key("ctrl+v"))
	um := updated.(Model)
	if len(um.refImages) != 0 {
		t.Errorf("refImages = %d, want 0", len(um.refImages))
	}
	if um.status == "Reading clipboard…" {
		t.Error("paste flow should not start for an incapable model")
	}
}

// 3.2: a successful paste result appends the validated image and clears to an
// attached status.
func TestPasteResultSuccessAttaches(t *testing.T) {
	m := composeModel(true, fakePaster{data: makePNG(t)})
	ref, err := validateRefImage(makePNG(t))
	if err != nil {
		t.Fatalf("fixture invalid: %v", err)
	}
	updated, _ := m.Update(pastedImageMsg{ref: ref})
	um := updated.(Model)
	if len(um.refImages) != 1 {
		t.Fatalf("refImages = %d, want 1", len(um.refImages))
	}
	if um.refImages[0].MediaType != "image/png" {
		t.Errorf("MediaType = %q, want image/png", um.refImages[0].MediaType)
	}
}

// 3.2: a failed paste result sets a transient, non-fatal status and attaches
// nothing.
func TestPasteResultFailureStatus(t *testing.T) {
	m := composeModel(true, fakePaster{err: ErrNoClipboardImage})
	updated, _ := m.Update(pastedImageMsg{err: ErrNoClipboardImage})
	um := updated.(Model)
	if len(um.refImages) != 0 {
		t.Errorf("refImages = %d, want 0 on failure", len(um.refImages))
	}
	if um.status != "No image in clipboard" {
		t.Errorf("status = %q, want %q", um.status, "No image in clipboard")
	}
	if um.phase != phaseCompose {
		t.Error("a paste failure must not change phase")
	}
}

// 3.2: the paste key on a capable model issues a command (reads the clipboard).
func TestPasteKeyCapableIssuesCommand(t *testing.T) {
	m := composeModel(true, fakePaster{data: makePNG(t)})
	_, cmd := m.handleComposeKey(key("ctrl+v"))
	if cmd == nil {
		t.Fatal("expected a paste command for a capable model")
	}
	// Running the command yields a success message carrying the image.
	msg := cmd()
	pm, ok := msg.(pastedImageMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want pastedImageMsg", msg)
	}
	if pm.err != nil || pm.ref.MediaType != "image/png" {
		t.Errorf("unexpected paste result: ref=%+v err=%v", pm.ref, pm.err)
	}
}

// 3.3: the remove key drops the most-recent image and returns to no-image.
func TestRemoveKeyDropsImage(t *testing.T) {
	m := composeModel(true, fakePaster{})
	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	m.addRefImage(provider.ReferenceImage{MediaType: "image/jpeg"})

	updated, _ := m.handleComposeKey(key("ctrl+r"))
	um := updated.(Model)
	if len(um.refImages) != 1 {
		t.Fatalf("refImages = %d, want 1 after one remove", len(um.refImages))
	}

	updated, _ = um.handleComposeKey(key("ctrl+r"))
	um = updated.(Model)
	if len(um.refImages) != 0 {
		t.Errorf("refImages = %d, want 0 after clearing", len(um.refImages))
	}
}
