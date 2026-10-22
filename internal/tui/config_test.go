package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// configModel builds a Model with a real config.yaml on disk (starter content)
// and an initial output dir, sitting in the picker.
func configModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if _, err := config.WriteStarter(cfgPath, config.DefaultAPIKey); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	m := New(Deps{
		Config:     cfg,
		ConfigPath: cfgPath,
		OutputDir:  filepath.Join(dir, "images"),
	})
	m.models = []provider.Model{{ID: "test/model", Name: "Test Model"}}
	m.selected = 0
	m.rebuildItems()
	m.picker.Select(0)
	m.width, m.height = 80, 40
	m.phase = phasePicker
	return m
}

func spaceKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeySpace} }

// typeInto sends each rune of s to the model via handleConfigKey.
func typeInto(m Model, s string) Model {
	for _, r := range s {
		out, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = out.(Model)
	}
	return m
}

// 3.1: the config view renders the current values from the loaded Config.
func TestConfigViewShowsCurrentValues(t *testing.T) {
	m := configModel(t)
	// history default is enabled, auto_open default false.
	m = m.openConfig(phasePicker)
	out := ansi.Strip(m.configView())
	for _, want := range []string{"Save location", "Auto-open on complete", "Keep prompt history", "Smart file names", "File-name model", "[off]", "[on]", "https://openrouter.ai/models"} {
		if !strings.Contains(out, want) {
			t.Errorf("config view missing %q; got:\n%s", want, out)
		}
	}
	// The view states what saving does and names the config file.
	if !strings.Contains(out, "Saving updates your config file:") {
		t.Errorf("config view missing the config-path note; got:\n%s", out)
	}
	if !strings.Contains(out, m.deps.ConfigPath) && !strings.Contains(out, "config.yaml") {
		t.Errorf("config view note missing the file path; got:\n%s", out)
	}
}

// 3.2 + 2.1 + 2.2: open → edit → confirm writes config.yaml, returns to the
// prior phase, and applies live (output dir + auto-open updated in session).
func TestConfigOpenEditConfirmWritesAndApplies(t *testing.T) {
	m := configModel(t)
	newDir := filepath.Join(t.TempDir(), "chosen")

	// Open from the picker with "c".
	opened, _ := m.handlePickerKey(key("c"))
	m = opened.(Model)
	if m.phase != phaseConfig {
		t.Fatalf("phase = %v, want config", m.phase)
	}

	// Clear the (empty) save-location field and type a new absolute path.
	m = typeInto(m, newDir)

	// Move to auto-open and toggle it on.
	down, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyDown})
	m = down.(Model)
	tog, _ := m.handleConfigKey(spaceKey())
	m = tog.(Model)

	// Confirm.
	confirmed, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = confirmed.(Model)

	// Returned to the picker.
	if m.phase != phasePicker {
		t.Errorf("phase after confirm = %v, want picker", m.phase)
	}
	// 2.1: session output dir updated.
	if m.outputDir != newDir {
		t.Errorf("outputDir = %q, want %q", m.outputDir, newDir)
	}
	// 2.2: auto-open applied in-memory.
	if !m.deps.Config.AutoOpen {
		t.Error("AutoOpen not applied to in-memory config")
	}
	// Written to disk: reloading reflects the change.
	reloaded, err := config.Load(m.deps.ConfigPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.AutoOpen {
		t.Error("auto_open not persisted to config.yaml")
	}
	if reloaded.OutputDir != config.Value(newDir) {
		t.Errorf("output_dir persisted = %q, want %q", reloaded.OutputDir, newDir)
	}
}

// 2.1: the next generation targets the updated output dir.
func TestConfigNewLocationAppliesToNextGeneration(t *testing.T) {
	m := configModel(t)
	newDir := filepath.Join(t.TempDir(), "next-gen")
	m = m.openConfig(phasePicker)
	m = typeInto(m, newDir)
	confirmed, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = confirmed.(Model)

	m.prompt.SetValue("a cat")
	gm, _ := m.generate()
	if gm.outputDir != newDir {
		t.Errorf("generation output dir = %q, want %q", gm.outputDir, newDir)
	}
	if gm.lastDraft.Prompt != "a cat" {
		t.Errorf("lastDraft prompt = %q", gm.lastDraft.Prompt)
	}
}

// 2.2: a toggled auto-open changes the completion behavior in-session (the
// generatedMsg handler issues an open command on success only when enabled).
func TestConfigAutoOpenAffectsCompletion(t *testing.T) {
	m := configModel(t)
	m.outputDir = t.TempDir()

	// Enable auto-open via the config view.
	m = m.openConfig(phasePicker)
	down, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyDown}) // to auto-open
	m = down.(Model)
	tog, _ := m.handleConfigKey(spaceKey())
	m = tog.(Model)
	confirmed, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = confirmed.(Model)

	if !m.deps.Config.AutoOpen {
		t.Fatal("auto-open should be enabled after confirm")
	}
	// A successful outcome with auto-open on returns a non-nil command batch
	// (the open command); with it off there is no open command.
	_, cmd := m.Update(generatedMsg{outcome: gen.Outcome{Path: "/tmp/out.png"}})
	if cmd == nil {
		t.Error("expected an open command when auto-open is enabled")
	}
}

// 3.3: an unusable save location is rejected — the file is not written and the
// prior values stay in effect.
func TestConfigRejectsInvalidSaveLocation(t *testing.T) {
	m := configModel(t)
	before, err := os.ReadFile(m.deps.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	origDir := m.outputDir

	// A path whose parent is a regular file cannot be created.
	fileParent := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(fileParent, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(fileParent, "sub")

	m = m.openConfig(phasePicker)
	m = typeInto(m, bad)
	confirmed, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = confirmed.(Model)

	// Still in the config view with an error, prior dir kept.
	if m.phase != phaseConfig {
		t.Errorf("phase = %v, want to stay in config on invalid input", m.phase)
	}
	if m.cfg.err == "" {
		t.Error("expected a validation error message")
	}
	if m.outputDir != origDir {
		t.Errorf("outputDir changed to %q despite invalid input", m.outputDir)
	}
	// config.yaml unchanged.
	after, err := os.ReadFile(m.deps.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("config.yaml was written despite an invalid save location")
	}
}

// 4.1: toggling smart file names and entering a model persists both to
// config.yaml and applies them live to the in-memory config.
func TestConfigSmartFilenamesRoundTrip(t *testing.T) {
	m := configModel(t)
	m = m.openConfig(phasePicker)

	// Move down to the smart file names toggle (save location -> auto-open ->
	// history -> smart file names) and toggle it on.
	for _, key := range []tea.KeyMsg{{Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyDown}} {
		down, _ := m.handleConfigKey(key)
		m = down.(Model)
	}
	tog, _ := m.handleConfigKey(spaceKey())
	m = tog.(Model)

	// Move to the filename-model field and type a model id.
	down, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyDown})
	m = down.(Model)
	m = typeInto(m, "openai/gpt-4o-mini")

	confirmed, _ := m.handleConfigKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = confirmed.(Model)

	if !m.deps.Config.SmartFilenames {
		t.Error("SmartFilenames not applied to in-memory config")
	}
	if m.deps.Config.FilenameModel != "openai/gpt-4o-mini" {
		t.Errorf("FilenameModel = %q, want openai/gpt-4o-mini", m.deps.Config.FilenameModel)
	}
	reloaded, err := config.Load(m.deps.ConfigPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reloaded.SmartFilenames {
		t.Error("smart_filenames not persisted to config.yaml")
	}
	if reloaded.FilenameModel != "openai/gpt-4o-mini" {
		t.Errorf("filename_model persisted = %q", reloaded.FilenameModel)
	}
}
