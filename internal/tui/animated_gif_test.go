package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

func animatedGIFModel(gif, transparent bool) Model {
	m := New(Deps{})
	m.models = []provider.Model{{
		ID:   "test/gif-model",
		Name: "Test Model",
		Capabilities: provider.Capabilities{
			SupportsAnimatedGIF:           gif,
			SupportsTransparentBackground: transparent,
		},
	}}
	m.selected = 0
	m.phase = phaseCompose
	m.prompt.Focus()
	return m
}

func TestAnimatedGIFToggle(t *testing.T) {
	m := animatedGIFModel(true, false)

	if strings.Contains(ansi.Strip(m.composeView()), "Animated: on") {
		t.Fatal("animated GIF should start off")
	}

	toggled, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = toggled.(Model)
	if !m.animatedGIF {
		t.Fatal("ctrl+g did not enable animated GIF")
	}
	if !strings.Contains(ansi.Strip(m.composeView()), "Animated: on") {
		t.Errorf("compose view missing on-state; got %q", ansi.Strip(m.composeView()))
	}
	if d := m.composeDraft(); d.OutputFormat != "gif" || d.Background != "" {
		t.Errorf("draft format/background = %q/%q, want gif/empty", d.OutputFormat, d.Background)
	}

	toggled, _ = m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = toggled.(Model)
	if m.animatedGIF {
		t.Fatal("second ctrl+g did not disable animated GIF")
	}
	if d := m.composeDraft(); d.OutputFormat != "" {
		t.Errorf("draft output format = %q, want empty when off", d.OutputFormat)
	}
}

func TestAnimatedGIFMutualExclusion(t *testing.T) {
	m := animatedGIFModel(true, true)

	toggled, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = toggled.(Model)
	if !m.transparentBackground {
		t.Fatal("ctrl+t did not enable transparency")
	}

	toggled, _ = m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = toggled.(Model)
	if !m.animatedGIF || m.transparentBackground {
		t.Fatal("enabling GIF should clear transparency")
	}
	if d := m.composeDraft(); d.OutputFormat != "gif" || d.Background != "" {
		t.Errorf("draft format/background = %q/%q, want gif/empty", d.OutputFormat, d.Background)
	}

	toggled, _ = m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = toggled.(Model)
	if !m.transparentBackground || m.animatedGIF {
		t.Fatal("enabling transparency should clear GIF")
	}
	if d := m.composeDraft(); d.OutputFormat != "png" || d.Background != "transparent" {
		t.Errorf("draft format/background = %q/%q, want png/transparent", d.OutputFormat, d.Background)
	}
}

func TestAnimatedGIFUnsupported(t *testing.T) {
	m := animatedGIFModel(false, false)

	toggled, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlG})
	m = toggled.(Model)
	if m.animatedGIF {
		t.Error("ctrl+g enabled GIF on an unsupported model")
	}
	if !strings.Contains(m.status, "does not support animated GIF") {
		t.Errorf("status = %q, want unsupported message", m.status)
	}
	if strings.Contains(ansi.Strip(m.composeView()), "Animated") {
		t.Errorf("compose view should omit GIF for unsupported model; got %q", ansi.Strip(m.composeView()))
	}
	if strings.Contains(ansi.Strip(m.footerView()), "ctrl+g") {
		t.Errorf("footer should omit GIF hint for unsupported model; got %q", ansi.Strip(m.footerView()))
	}
	if d := m.composeDraft(); d.OutputFormat != "" {
		t.Errorf("draft output format = %q for unsupported model, want empty", d.OutputFormat)
	}
}

func TestAnimatedGIFSupportedHints(t *testing.T) {
	m := animatedGIFModel(true, false)
	view := ansi.Strip(m.composeView())
	if !strings.Contains(view, "Animated: off") {
		t.Errorf("compose view missing GIF setting; got %q", view)
	}
	footer := ansi.Strip(m.footerView())
	if !strings.Contains(footer, "ctrl+g") || !strings.Contains(footer, "animated gif") {
		t.Errorf("footer missing GIF hint; got %q", footer)
	}
	if hint := capHint(m.currentCaps()); !strings.Contains(hint, "gif") {
		t.Errorf("capability hint = %q, want gif marker", hint)
	}
}

func TestSwitchingModelsClearsAnimatedGIF(t *testing.T) {
	m := animatedGIFModel(true, false)
	m.animatedGIF = true
	m.picker.SetItems([]list.Item{modelItem{m: provider.Model{
		ID: "other/model", Name: "Other", Capabilities: provider.Capabilities{SupportsAnimatedGIF: true},
	}}})

	next, _ := m.handlePickerKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if m.animatedGIF {
		t.Error("selecting another model should clear the GIF selection")
	}
}
