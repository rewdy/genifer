package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/config"
)

// TestInitialPhase asserts the model starts in onboarding only on first run.
func TestInitialPhase(t *testing.T) {
	first := New(Deps{FirstRun: true, ConfigPath: "/tmp/x/config.yaml"})
	if first.phase != phaseOnboarding {
		t.Errorf("first-run phase = %v, want phaseOnboarding", first.phase)
	}
	normal := New(Deps{FirstRun: false})
	if normal.phase != phasePicker {
		t.Errorf("config-present phase = %v, want phasePicker", normal.phase)
	}
}

// TestAPIKeyValue checks each method maps to the right config directive.
func TestAPIKeyValue(t *testing.T) {
	cases := []struct {
		method keyMethod
		raw    string
		want   config.Value
	}{
		{methodEnv, "OPENROUTER_API_KEY", "{env:OPENROUTER_API_KEY}"},
		{methodCmd, "op read op://v/k", "{cmd:op read op://v/k}"},
		{methodPaste, "sk-or-literal", "sk-or-literal"},
	}
	for _, tc := range cases {
		if got := apiKeyValue(tc.method, tc.raw); got != tc.want {
			t.Errorf("apiKeyValue(%v, %q) = %q, want %q", tc.method, tc.raw, got, tc.want)
		}
	}
}

// drive sends a sequence of key presses through the model's key handler.
func driveKeys(m Model, keys ...string) Model {
	for _, s := range keys {
		var tm tea.Model
		tm, _ = m.handleKey(keyMsg(s))
		m = tm.(Model)
	}
	return m
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// TestOnboardingWritesEnvChoice drives the OpenRouter env-var branch end to end
// and asserts the written config carries the {env:...} directive.
func TestOnboardingWritesEnvChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := New(Deps{FirstRun: true, ConfigPath: path})

	// stepType: OpenRouter is index 0; select it.
	m = driveKeys(m, "enter")
	if m.onboard.step != stepChoose {
		t.Fatalf("step = %v, want stepChoose", m.onboard.step)
	}
	// stepChoose: method 0 is env; select it, then type the var name.
	m = driveKeys(m, "enter")
	if m.onboard.step != stepInput {
		t.Fatalf("step = %v, want stepInput", m.onboard.step)
	}
	m.onboard.input.SetValue("OPENROUTER_API_KEY")

	tm, cmd := m.handleKey(keyMsg("enter")) // confirm input
	m = tm.(Model)
	if cmd == nil {
		t.Fatal("expected a write command, got nil")
	}
	msg := cmd()
	if wm, ok := msg.(onboardWrittenMsg); !ok || wm.err != nil {
		t.Fatalf("write result = %#v, want success", msg)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Type != config.TypeOpenRouter {
		t.Fatalf("Providers = %+v, want one openrouter instance", cfg.Providers)
	}
	if cfg.Providers[0].APIKey != "{env:OPENROUTER_API_KEY}" {
		t.Errorf("api_key = %q, want {env:OPENROUTER_API_KEY}", cfg.Providers[0].APIKey)
	}
}

// TestOnboardingWritesLocalChoice drives the local-provider branch end to end
// and asserts the written config carries the a1111 instance with the address.
func TestOnboardingWritesLocalChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := New(Deps{FirstRun: true, ConfigPath: path})

	// stepType: move to Local (index 1) and select it; goes straight to input.
	m = driveKeys(m, "down", "enter")
	if m.onboard.step != stepInput {
		t.Fatalf("step = %v, want stepInput", m.onboard.step)
	}
	if m.onboard.selectedType() != typeLocal {
		t.Fatalf("selectedType = %v, want typeLocal", m.onboard.selectedType())
	}
	m.onboard.input.SetValue("http://10.0.0.2:7860")

	tm, cmd := m.handleKey(keyMsg("enter"))
	m = tm.(Model)
	if cmd == nil {
		t.Fatal("expected a write command, got nil")
	}
	if wm, ok := cmd().(onboardWrittenMsg); !ok || wm.err != nil {
		t.Fatalf("write result unexpected")
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Type != config.TypeA1111 {
		t.Fatalf("Providers = %+v, want one a1111 instance", cfg.Providers)
	}
	if cfg.Providers[0].BaseURL != "http://10.0.0.2:7860" {
		t.Errorf("base_url = %q, want http://10.0.0.2:7860", cfg.Providers[0].BaseURL)
	}
}

// TestOnboardingPasteRequiresConfirm asserts the paste branch stops at a
// confirmation step before writing, then writes a literal on confirm.
func TestOnboardingPasteRequiresConfirm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	m := New(Deps{FirstRun: true, ConfigPath: path})

	// stepType: choose OpenRouter.
	m = driveKeys(m, "enter")
	// stepChoose: move cursor to the paste option (index 2), then select.
	m = driveKeys(m, "down", "down", "enter")
	if m.onboard.selectedMethod() != methodPaste {
		t.Fatalf("selected = %v, want methodPaste", m.onboard.selectedMethod())
	}
	m.onboard.input.SetValue("sk-or-secret")

	// Enter on input should go to confirm, NOT write yet.
	tm, cmd := m.handleKey(keyMsg("enter"))
	m = tm.(Model)
	if m.onboard.step != stepConfirm {
		t.Fatalf("step = %v, want stepConfirm", m.onboard.step)
	}
	if cmd != nil {
		t.Fatal("should not write before confirmation")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("config written before confirmation")
	}

	// Confirm with y -> writes literal.
	tm, cmd = m.handleKey(keyMsg("y"))
	m = tm.(Model)
	if cmd == nil {
		t.Fatal("expected write command after confirm")
	}
	if msg, ok := cmd().(onboardWrittenMsg); !ok || msg.err != nil {
		t.Fatalf("write result = %#v, want success", msg)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].APIKey != "sk-or-secret" {
		t.Errorf("Providers = %+v, want literal sk-or-secret", cfg.Providers)
	}
}

// TestOnboardingNeverOverwrites asserts an existing config is left untouched
// even when onboarding writes.
func TestOnboardingNeverOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	existing := []byte("providers:\n  - key: or\n    type: openrouter\n    api_key: \"{env:PREEXISTING}\"\n")
	if err := os.WriteFile(path, existing, 0o600); err != nil {
		t.Fatal(err)
	}
	m := New(Deps{FirstRun: true, ConfigPath: path})
	m = driveKeys(m, "enter") // OpenRouter type
	m = driveKeys(m, "enter") // env method
	m.onboard.input.SetValue("NEW_VAR")
	tm, cmd := m.handleKey(keyMsg("enter"))
	m = tm.(Model)
	if cmd == nil {
		t.Fatal("expected write command")
	}
	cmd() // execute the write

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(existing) {
		t.Errorf("existing config was overwritten:\n%s", got)
	}
}

// 8.3: after onboarding writes the rebuilt config (the legacy-recovery entry
// point sets FirstRun), the onboardWrittenMsg advances the model into the
// picker and kicks off model loading.
func TestOnboardingAdvancesToPicker(t *testing.T) {
	m := New(Deps{
		FirstRun:   true,
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
	})
	if m.phase != phaseOnboarding {
		t.Fatalf("start phase = %v, want onboarding", m.phase)
	}
	updated, cmd := m.Update(onboardWrittenMsg{})
	m = updated.(Model)
	if m.phase != phasePicker {
		t.Errorf("phase after write = %v, want picker", m.phase)
	}
	if cmd == nil {
		t.Error("expected a model-loading command after onboarding")
	}
}
