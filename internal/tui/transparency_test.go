package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

// transparencyModel builds a Model in the compose phase with a single selected
// model whose transparent-background support is set by supported.
func transparencyModel(supported bool) Model {
	m := New(Deps{})
	m.models = []provider.Model{{
		ID:           "test/model",
		Name:         "Test Model",
		Capabilities: provider.Capabilities{SupportsTransparentBackground: supported},
	}}
	m.selected = 0
	m.phase = phaseCompose
	m.prompt.Focus()
	return m
}

// ctrl+t toggles transparency for a supporting model and is reflected in the
// compose view and the composed draft.
func TestTransparencyToggle(t *testing.T) {
	m := transparencyModel(true)

	if strings.Contains(ansi.Strip(m.composeView()), "Transparent: on") {
		t.Fatal("transparency should start off")
	}

	toggled, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = toggled.(Model)
	if !m.transparentBackground {
		t.Fatal("ctrl+t did not enable transparency")
	}
	if !strings.Contains(ansi.Strip(m.composeView()), "Transparent: on") {
		t.Errorf("compose view missing on-state; got %q", ansi.Strip(m.composeView()))
	}
	// The settings line is the sole state indicator: a toggle must not also set
	// a transient status that would duplicate it.
	if strings.Contains(strings.ToLower(m.status), "transparent") {
		t.Errorf("toggle set a redundant status %q", m.status)
	}
	if d := m.composeDraft(); d.Background != "transparent" || d.OutputFormat != "png" {
		t.Errorf("draft background/format = %q/%q, want transparent/png", d.Background, d.OutputFormat)
	}

	toggled, _ = m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = toggled.(Model)
	if m.transparentBackground {
		t.Fatal("second ctrl+t did not disable transparency")
	}
	if d := m.composeDraft(); d.Background != "" {
		t.Errorf("draft background = %q, want empty when off", d.Background)
	}
}

// ctrl+t is inert for a model that does not support transparency: the toggle
// stays off, the compose view omits the setting, and the footer omits the hint.
func TestTransparencyUnsupported(t *testing.T) {
	m := transparencyModel(false)

	toggled, _ := m.handleComposeKey(tea.KeyMsg{Type: tea.KeyCtrlT})
	m = toggled.(Model)
	if m.transparentBackground {
		t.Error("ctrl+t enabled transparency on an unsupported model")
	}
	if strings.Contains(ansi.Strip(m.composeView()), "Transparent") {
		t.Errorf("compose view should omit transparency for unsupported model; got %q", ansi.Strip(m.composeView()))
	}
	if strings.Contains(ansi.Strip(m.footerView()), "ctrl+t") {
		t.Errorf("footer should omit transparency hint for unsupported model; got %q", ansi.Strip(m.footerView()))
	}
}

// The compose footer advertises ctrl+t only when the model supports it.
func TestTransparencyFooterHint(t *testing.T) {
	supported := ansi.Strip(transparencyModel(true).footerView())
	if !strings.Contains(supported, "ctrl+t") || !strings.Contains(supported, "transparent") {
		t.Errorf("footer missing transparency hint; got %q", supported)
	}
}
