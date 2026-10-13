package tui

import (
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// composeWithHistory builds a compose-phase model with the given capabilities
// and n seeded history entries ("prompt-...-0" newest), ready for recall.
func composeWithHistory(t *testing.T, caps provider.Capabilities, n int) Model {
	t.Helper()
	m := historyModel(t, caps, config.Default())
	writeHistory(t, m.deps.HistoryPath, n)
	m.phase = phaseCompose
	m.prompt.Focus()
	return m
}

func upKey() tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyUp} }
func downKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyDown} }

// 1.1 / 2.1: Up at the first line starts recall (snapshots the draft, applies
// the newest), further Ups walk older and clamp, and Down past the newest
// restores the draft.
func TestRecallStepOlderNewerAndRestore(t *testing.T) {
	m := composeWithHistory(t, provider.Capabilities{}, 3)
	m.prompt.SetValue("my draft")
	m.prompt.CursorStart() // cursor on the first line

	// Up → newest entry, draft snapshotted.
	up1, _ := m.handleComposeKey(upKey())
	m = up1.(Model)
	if !m.recallActive() || m.recallIndex != 0 {
		t.Fatalf("recallIndex = %d, want 0 active", m.recallIndex)
	}
	if m.recallDraft != "my draft" {
		t.Errorf("draft snapshot = %q, want %q", m.recallDraft, "my draft")
	}
	newest := m.prompt.Value()
	if !strings.HasSuffix(newest, "-0") {
		t.Errorf("newest recalled prompt = %q, want the newest entry", newest)
	}

	// Up again → older entry (index 1).
	up2, _ := m.handleComposeKey(upKey())
	m = up2.(Model)
	if m.recallIndex != 1 {
		t.Errorf("recallIndex = %d, want 1", m.recallIndex)
	}

	// Up to the oldest (index 2) and then clamp.
	up3, _ := m.handleComposeKey(upKey())
	m = up3.(Model)
	up4, _ := m.handleComposeKey(upKey())
	m = up4.(Model)
	if m.recallIndex != 2 {
		t.Errorf("recallIndex clamped = %d, want 2 (oldest of 3)", m.recallIndex)
	}

	// Walk back down to the newest.
	for i := 0; i < 2; i++ {
		next, _ := m.handleComposeKey(downKey())
		m = next.(Model)
	}
	if m.recallIndex != 0 {
		t.Fatalf("recallIndex after stepping newer = %d, want 0", m.recallIndex)
	}

	// Down past the newest → restore the draft, end session.
	restore, _ := m.handleComposeKey(downKey())
	m = restore.(Model)
	if m.recallActive() {
		t.Error("session should end after restoring the draft")
	}
	if m.prompt.Value() != "my draft" {
		t.Errorf("restored prompt = %q, want %q", m.prompt.Value(), "my draft")
	}
}

// 1.1: an empty draft is restored exactly.
func TestRecallRestoresEmptyDraft(t *testing.T) {
	m := composeWithHistory(t, provider.Capabilities{}, 2)
	m.prompt.SetValue("")
	m.prompt.CursorStart()

	up, _ := m.handleComposeKey(upKey())
	m = up.(Model)
	if m.prompt.Value() == "" {
		t.Fatal("expected a recalled entry to populate the prompt")
	}
	down, _ := m.handleComposeKey(downKey())
	m = down.(Model)
	if m.prompt.Value() != "" {
		t.Errorf("restored prompt = %q, want empty", m.prompt.Value())
	}
	if m.recallActive() {
		t.Error("session should end on restore")
	}
}

// 2.1: Up/Down away from the edges move the cursor and do not recall.
func TestRecallEdgeAwareness(t *testing.T) {
	m := composeWithHistory(t, provider.Capabilities{}, 3)
	m.prompt.SetValue("line one\nline two\nline three")
	// Cursor is at the end (last line) after SetValue.

	// Up from the last line (not the first) → cursor moves, no recall.
	up, _ := m.handleComposeKey(upKey())
	um := up.(Model)
	if um.recallActive() {
		t.Error("Up away from the first line should not recall")
	}

	// Put cursor on the first line; Down should move within text, not recall
	// (and recall isn't active anyway).
	um.prompt.CursorStart()
	down, _ := um.handleComposeKey(downKey())
	dm := down.(Model)
	if dm.recallActive() {
		t.Error("Down when not recalling should never start recall")
	}
}

// 2.1: with no loaded history, Up/Down at the edges are inert (fall through).
func TestRecallNoHistoryIsInert(t *testing.T) {
	m := composeWithHistory(t, provider.Capabilities{}, 0)
	m.prompt.SetValue("")
	m.prompt.CursorStart()
	up, _ := m.handleComposeKey(upKey())
	um := up.(Model)
	if um.recallActive() {
		t.Error("recall must not activate with no history")
	}
	if um.prompt.Value() != "" {
		t.Errorf("prompt changed with no history: %q", um.prompt.Value())
	}
}

// 2.2: editing the prompt ends the session, so the next Up recalls the newest.
func TestRecallResetsOnEdit(t *testing.T) {
	m := composeWithHistory(t, provider.Capabilities{}, 3)
	m.prompt.SetValue("draft")
	m.prompt.CursorStart()

	// Recall two steps back.
	for i := 0; i < 2; i++ {
		next, _ := m.handleComposeKey(upKey())
		m = next.(Model)
	}
	if m.recallIndex != 1 {
		t.Fatalf("recallIndex = %d, want 1 before edit", m.recallIndex)
	}

	// Type a character → ends the session.
	typed, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = typed.(Model)
	if m.recallActive() {
		t.Error("editing should end the recall session")
	}

	// Next Up at the first line starts fresh from the newest.
	m.prompt.CursorStart()
	up, _ := m.handleComposeKey(upKey())
	m = up.(Model)
	if m.recallIndex != 0 {
		t.Errorf("recallIndex after edit+Up = %d, want 0 (newest)", m.recallIndex)
	}
}

// 2.1 / 1.1: a recalled entry applies aspect ratio and seed where the model
// supports them, flowing into the composed draft.
func TestRecallAppliesParamsWhenSupported(t *testing.T) {
	m := historyModel(t, provider.Capabilities{AspectRatios: []string{"1:1", "16:9"}, SupportsSeed: true}, config.Default())
	seed := int64(99)
	cache := config.HistoryCache{}.Append(config.HistoryEntry{
		Prompt: "a seeded prompt", Model: "test/model", AspectRatio: "16:9", Seed: &seed,
	})
	if err := config.SaveHistory(m.deps.HistoryPath, cache); err != nil {
		t.Fatal(err)
	}
	m.phase = phaseCompose
	m.prompt.SetValue("")
	m.prompt.CursorStart()

	up, _ := m.handleComposeKey(upKey())
	m = up.(Model)
	if m.aspectRatio != "16:9" {
		t.Errorf("aspect = %q, want 16:9", m.aspectRatio)
	}
	d := m.composeDraft()
	if d.Seed == nil || *d.Seed != 99 {
		t.Errorf("composed seed = %v, want 99", d.Seed)
	}
}

// 3.1: the compose footer advertises Up/Down history recall; recall does not
// activate outside compose.
func TestRecallFooterHintAndScope(t *testing.T) {
	foot := ansi.Strip(composeWithHistory(t, provider.Capabilities{}, 2).footerView())
	if !strings.Contains(foot, "history") {
		t.Errorf("compose footer missing history hint; got %q", foot)
	}

	// In the picker, Up is handled by the list, not recall.
	m := composeWithHistory(t, provider.Capabilities{}, 2)
	m.phase = phasePicker
	before := m.recallActive()
	out, _ := m.handlePickerKey(upKey())
	after := out.(Model).recallActive()
	if before || after {
		t.Error("recall must not activate in the picker")
	}
}
