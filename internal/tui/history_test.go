package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func boolPtr(b bool) *bool { return &b }

// historyModel builds a Model with a configured history path and the given
// capabilities for its single selected model.
func historyModel(t *testing.T, caps provider.Capabilities, cfg config.Config) Model {
	t.Helper()
	m := New(Deps{Config: cfg, HistoryPath: filepath.Join(t.TempDir(), "history-cache.json")})
	m.models = []provider.Model{{ID: "test/model", Name: "Test Model", Capabilities: caps}}
	m.selected = 0
	m.rebuildItems()
	m.picker.Select(0)
	m.width, m.height = 80, 40
	m.history.SetSize(76, 20)
	return m
}

// seedPtr returns a pointer to v.
func seedPtr(v int64) *int64 { return &v }

// 3.1: a successful generation writes a history entry; failure/cancel and a
// disabled config write nothing.
func TestRecordHistoryOnSuccess(t *testing.T) {
	run := func(m Model) {
		if cmd := m.recordHistoryCmd(); cmd != nil {
			cmd() // execute the background write synchronously
		}
	}

	// Success → one entry with the draft/outcome details.
	m := historyModel(t, provider.Capabilities{SupportsSeed: true}, config.Default())
	m.lastDraft = gen.Draft{Prompt: "a cat", Model: "test/model", AspectRatio: "1:1", Seed: seedPtr(7)}
	m.outcome = gen.Outcome{Path: "/tmp/a.png", Seed: seedPtr(7)}
	run(m)
	got := config.LoadHistory(m.deps.HistoryPath)
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(got.Entries))
	}
	e := got.Entries[0]
	if e.Prompt != "a cat" || e.Model != "test/model" || e.Path != "/tmp/a.png" {
		t.Errorf("entry mismatch: %+v", e)
	}
	if e.Seed == nil || *e.Seed != 7 {
		t.Errorf("seed = %v, want 7", e.Seed)
	}

	// Failure → nothing written.
	mf := historyModel(t, provider.Capabilities{}, config.Default())
	mf.lastDraft = gen.Draft{Prompt: "x", Model: "test/model"}
	mf.outcome = gen.Outcome{Failure: gen.FailureGeneration}
	if cmd := mf.recordHistoryCmd(); cmd != nil {
		t.Error("no history command expected on failure")
	}

	// Disabled → nothing written.
	md := historyModel(t, provider.Capabilities{}, config.Config{History: boolPtr(false)})
	md.lastDraft = gen.Draft{Prompt: "x", Model: "test/model"}
	md.outcome = gen.Outcome{Path: "/tmp/x.png"}
	if cmd := md.recordHistoryCmd(); cmd != nil {
		t.Error("no history command expected when disabled")
	}
}

// writeHistory seeds the history file with n entries, newest-first by prompt
// index (entry 0 = "prompt-0" newest).
func writeHistory(t *testing.T, path string, n int) {
	t.Helper()
	var c config.HistoryCache
	// Append oldest first so the newest ends up at the front.
	for i := n - 1; i >= 0; i-- {
		c = c.Append(config.HistoryEntry{
			Prompt:    "prompt-" + string(rune('A'+i%26)) + "-" + itoa(i),
			Model:     "test/model",
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}
	if err := config.SaveHistory(path, c); err != nil {
		t.Fatal(err)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// 4.1: the view loads at most 25 entries newest-first; empty and disabled
// states show their messages.
func TestHistoryViewLoadLimitAndStates(t *testing.T) {
	// 30 on disk → load exactly 25.
	m := historyModel(t, provider.Capabilities{}, config.Default())
	writeHistory(t, m.deps.HistoryPath, 30)
	m = m.openHistory(phasePicker)
	if len(m.historyEntries) != historyLoadLimit {
		t.Fatalf("loaded %d entries, want %d", len(m.historyEntries), historyLoadLimit)
	}
	if !m.historyMoreExist {
		t.Error("historyMoreExist should be true when 30 > 25 loaded")
	}
	// Newest first: entry 0 is the most-recent ("prompt-...-0").
	if !strings.HasSuffix(m.historyEntries[0].Prompt, "-0") {
		t.Errorf("newest-first violated; front = %q", m.historyEntries[0].Prompt)
	}

	// Empty history message.
	empty := historyModel(t, provider.Capabilities{}, config.Default())
	empty = empty.openHistory(phasePicker)
	if out := ansi.Strip(empty.historyView()); !strings.Contains(out, "No history yet") {
		t.Errorf("empty history view missing message; got %q", out)
	}

	// Disabled message.
	disabled := historyModel(t, provider.Capabilities{}, config.Config{History: boolPtr(false)})
	disabled = disabled.openHistory(phasePicker)
	if out := ansi.Strip(disabled.historyView()); !strings.Contains(out, "turned off") {
		t.Errorf("disabled history view missing message; got %q", out)
	}
}

// 4.1: the full-history pointer appears at end-of-list when more exist, and on
// an empty filter result.
func TestHistoryPointerMessage(t *testing.T) {
	m := historyModel(t, provider.Capabilities{}, config.Default())
	writeHistory(t, m.deps.HistoryPath, 30)
	m = m.openHistory(phasePicker)

	// Not at end initially (cursor at 0, 25 loaded) → no pointer.
	if out := ansi.Strip(m.historyView()); strings.Contains(out, "Full history can be found") {
		t.Errorf("pointer shown too early; got %q", out)
	}

	// Move cursor to the last loaded entry → pointer appears (more exist).
	m.history.Select(len(m.historyEntries) - 1)
	if out := ansi.Strip(m.historyView()); !strings.Contains(out, "Full history can be found") {
		t.Errorf("pointer missing at end-of-list; got %q", out)
	}

	// Filter to something that matches nothing loaded → pointer appears.
	fm := historyModel(t, provider.Capabilities{}, config.Default())
	writeHistory(t, fm.deps.HistoryPath, 30)
	fm = fm.openHistory(phasePicker)
	fm.history.SetFilterText("zzzzz-no-match")
	if out := ansi.Strip(fm.historyView()); !strings.Contains(out, "Full history can be found") {
		t.Errorf("pointer missing on empty filter; got %q", out)
	}
}

// 4.2: open → select → return transitions.
func TestHistoryOpenSelectReturn(t *testing.T) {
	m := historyModel(t, provider.Capabilities{}, config.Default())
	writeHistory(t, m.deps.HistoryPath, 3)
	m.phase = phasePicker

	// Open from the picker with "h".
	opened, _ := m.handlePickerKey(key("h"))
	om := opened.(Model)
	if om.phase != phaseHistory {
		t.Fatalf("phase = %v, want history", om.phase)
	}

	// esc returns to the picker.
	back, _ := om.handleHistoryKey(tea.KeyMsg{Type: tea.KeyEsc})
	bm := back.(Model)
	if bm.phase != phasePicker {
		t.Errorf("esc did not return to picker; phase = %v", bm.phase)
	}

	// enter selects an entry and goes to compose.
	sel, _ := om.handleHistoryKey(tea.KeyMsg{Type: tea.KeyEnter})
	sm := sel.(Model)
	if sm.phase != phaseCompose {
		t.Errorf("enter did not go to compose; phase = %v", sm.phase)
	}
	if sm.prompt.Value() == "" {
		t.Error("selecting an entry should populate the prompt")
	}
}

// 4.3: selecting pre-fills the prompt and applies aspect ratio and seed only
// when the active model supports them.
func TestHistoryReuseAppliesParamsOnlyWhenSupported(t *testing.T) {
	entry := config.HistoryEntry{Prompt: "a lighthouse", Model: "test/model", AspectRatio: "16:9", Seed: seedPtr(55)}

	// Capable model: prompt, aspect, and seed all applied.
	full := historyModel(t, provider.Capabilities{AspectRatios: []string{"1:1", "16:9"}, SupportsSeed: true}, config.Default())
	full = full.reuseHistoryEntry(entry)
	if full.prompt.Value() != "a lighthouse" {
		t.Errorf("prompt = %q", full.prompt.Value())
	}
	if full.aspectRatio != "16:9" {
		t.Errorf("aspect = %q, want 16:9", full.aspectRatio)
	}
	if full.reuseSeed == nil || *full.reuseSeed != 55 {
		t.Errorf("reuseSeed = %v, want 55", full.reuseSeed)
	}

	// Incapable model: prompt applied, aspect/seed not.
	none := historyModel(t, provider.Capabilities{}, config.Default())
	none = none.reuseHistoryEntry(entry)
	if none.prompt.Value() != "a lighthouse" {
		t.Errorf("prompt = %q", none.prompt.Value())
	}
	if none.aspectRatio != "" {
		t.Errorf("aspect = %q, want empty for no-aspect model", none.aspectRatio)
	}
	if none.reuseSeed != nil {
		t.Errorf("reuseSeed = %v, want nil for no-seed model", none.reuseSeed)
	}

	// The reused seed flows into the composed draft for a seed-capable model.
	d := full.composeDraft()
	if d.Seed == nil || *d.Seed != 55 {
		t.Errorf("composed draft seed = %v, want reused 55", d.Seed)
	}
}
