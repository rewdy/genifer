package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// wantAppBG is the truecolor SGR prefix for the app background in tests (the
// suite forces a truecolor profile in init).
func wantAppBG(t *testing.T) string {
	t.Helper()
	return wantBG(t, headerBG)
}

// wantBG returns the truecolor SGR prefix that sets the given background.
func wantBG(t *testing.T, c lipgloss.Color) string {
	t.Helper()
	out := lipgloss.NewStyle().Background(c).Render("\x00")
	i := strings.IndexByte(out, 0)
	if i <= 0 {
		t.Fatalf("no color sequence emitted for %v under a truecolor profile", c)
	}
	return out[:i]
}

// TestPromptTextareaUsesInputBackground checks the prompt input paints the
// lighter input background on every style — including CursorLine, whose stock
// value is a hardcoded black — and that a rendered placeholder line emits the
// input background rather than the terminal default.
func TestPromptTextareaUsesInputBackground(t *testing.T) {
	if inputBG == headerBG {
		t.Fatal("inputBG must differ from the app background")
	}
	focused, blurred := promptTextareaStyles()
	for _, tc := range []struct {
		name string
		s    textarea.Style
	}{{"focused", focused}, {"blurred", blurred}} {
		for _, f := range []struct {
			name string
			bg   lipgloss.Style
		}{
			{"Base", tc.s.Base},
			{"CursorLine", tc.s.CursorLine},
			{"CursorLineNumber", tc.s.CursorLineNumber},
			{"EndOfBuffer", tc.s.EndOfBuffer},
			{"LineNumber", tc.s.LineNumber},
			{"Placeholder", tc.s.Placeholder},
			{"Prompt", tc.s.Prompt},
			{"Text", tc.s.Text},
		} {
			if got := f.bg.GetBackground(); got != inputBG {
				t.Errorf("%s.%s background = %v, want %v", tc.name, f.name, got, inputBG)
			}
		}
	}

	m := composeModel(false, fakePaster{})
	m.prompt.SetWidth(40)
	out := m.prompt.View()
	if !strings.Contains(out, wantBG(t, inputBG)) {
		t.Errorf("rendered placeholder textarea missing input background; got %q", out)
	}
	if strings.Contains(out, "\x1b[48;2;0;0;0m") {
		t.Errorf("rendered textarea still uses the default black background; got %q", out)
	}
}

// TestPickerStylesUseAppBackground checks the model list paints the app
// background on its filter bar, pagination dots, and status, while its filter
// input uses the lighter input background.
func TestPickerStylesUseAppBackground(t *testing.T) {
	picker := list.New(nil, modelDelegate{}, 40, 20)
	picker.Styles = pickerStyles()
	picker.FilterInput = filterInputStyles(picker.FilterInput)

	appStyles := map[string]lipgloss.Style{
		"FilterPrompt":          picker.Styles.FilterPrompt,
		"FilterCursor":          picker.Styles.FilterCursor,
		"StatusBar":             picker.Styles.StatusBar,
		"NoItems":               picker.Styles.NoItems,
		"PaginationStyle":       picker.Styles.PaginationStyle,
		"ActivePaginationDot":   picker.Styles.ActivePaginationDot,
		"InactivePaginationDot": picker.Styles.InactivePaginationDot,
	}
	for name, s := range appStyles {
		if got := s.GetBackground(); got != headerBG {
			t.Errorf("%s background = %v, want %v", name, got, headerBG)
		}
	}

	inputStyles := map[string]lipgloss.Style{
		"FilterInput.TextStyle":   picker.FilterInput.TextStyle,
		"FilterInput.PromptStyle": picker.FilterInput.PromptStyle,
	}
	for name, s := range inputStyles {
		if got := s.GetBackground(); got != inputBG {
			t.Errorf("%s background = %v, want %v", name, got, inputBG)
		}
	}
}

// TestDelegateRowFillsWidth checks a short model row is padded to the list
// width with the app background rather than leaving the terminal default.
func TestDelegateRowFillsWidth(t *testing.T) {
	it := modelItem{m: provider.Model{Name: "Short"}}
	l := list.New([]list.Item{it}, modelDelegate{}, 40, 4)
	l.Select(0)

	var buf bytes.Buffer
	modelDelegate{}.Render(&buf, l, 0, it)
	out := buf.String()

	if w := ansi.StringWidth(out); w != 40 {
		t.Errorf("delegate row width = %d, want 40", w)
	}
	if !strings.Contains(out, wantAppBG(t)) {
		t.Errorf("delegate row missing app background padding; got %q", out)
	}
}

// TestComposePromptFillsInputBackgroundWithPlaceholder checks the input box is
// filled with the input background even while the placeholder is showing — the
// textarea's placeholder render path does not pad its lines, so the repair must
// make every prompt line full-width and input-background colored.
func TestComposePromptFillsInputBackgroundWithPlaceholder(t *testing.T) {
	const promptWidth = 50
	m := composeModel(false, fakePaster{})
	m.prompt.SetWidth(promptWidth)

	seqIn := wantBG(t, inputBG)
	var seen int
	for _, line := range strings.Split(m.composeView(), "\n") {
		if !strings.Contains(ansi.Strip(line), "┃") {
			continue
		}
		seen++
		if w := ansi.StringWidth(line); w != promptWidth {
			t.Errorf("prompt line width = %d, want %d: %q", w, promptWidth, ansi.Strip(line))
		}
		if !strings.Contains(line, ansiReset+seqIn) {
			t.Errorf("prompt line not re-asserted over the input background: %q", line)
		}
	}
	if seen == 0 {
		t.Fatal("no prompt lines found in compose view")
	}
}

// TestPaintBackgroundReassertsAndPads checks the backstop re-asserts the app
// background after a widget's reset (so its unstyled cells inherit it) and pads
// every line to the requested width.
func TestPaintBackgroundReassertsAndPads(t *testing.T) {
	seq := wantAppBG(t)
	// A widget line that ends in a reset and then leaves trailing cells bare.
	in := "hi" + ansiReset + "   \nsecond"
	out := paintBackground(in, 10)

	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	for i, line := range lines {
		if w := ansi.StringWidth(line); w != 10 {
			t.Errorf("line %d width = %d, want 10", i, w)
		}
		if !strings.HasPrefix(line, seq) {
			t.Errorf("line %d does not open with the app background: %q", i, line)
		}
	}
	if !strings.Contains(lines[0], ansiReset+seq) {
		t.Errorf("background not re-asserted after reset: %q", lines[0])
	}
	if got := paintBackground("x", 0); got != "x" {
		t.Errorf("width 0 should be a no-op; got %q", got)
	}
}

// TestViewLinesFullWidth renders a representative screen for each phase and
// checks every line is padded to exactly the terminal width with the app
// background, so the terminal default never shows through.
func TestViewLinesFullWidth(t *testing.T) {
	const width, height = 80, 24

	models := []provider.Model{{
		ID:           "test/model",
		Name:         "Test Model",
		Capabilities: provider.Capabilities{AcceptsReferenceImages: true, AspectRatios: []string{"1:1", "16:9"}},
	}}

	build := func(phase phase, mutate func(*Model)) Model {
		m := New(Deps{})
		m.models = models
		m.selected = 0
		m.modelsReady = true
		m.rebuildItems()
		m.picker.Select(0)
		m.width, m.height = width, height
		m.prompt.SetWidth(max(20, width-4))
		m.picker.SetSize(max(20, width-4), pickerHeight(height))
		m.prompt.Focus()
		m.phase = phase
		if mutate != nil {
			mutate(&m)
		}
		return m
	}

	cases := map[string]Model{
		"onboarding": func() Model {
			m := New(Deps{FirstRun: true})
			m.width, m.height = width, height
			return m
		}(),
		"picker":     build(phasePicker, nil),
		"compose":    build(phaseCompose, nil),
		"aspect":     build(phaseAspect, nil),
		"review":     build(phaseReview, func(m *Model) { m.prompt.SetValue("a prompt") }),
		"generating": build(phaseGenerating, nil),
		"result":     build(phaseResult, func(m *Model) { m.outcome = gen.Outcome{Path: "out.png"} }),
	}

	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			out := m.View()
			for i, line := range strings.Split(out, "\n") {
				if w := ansi.StringWidth(line); w != width {
					t.Errorf("line %d width = %d, want %d: %q", i, w, width, ansi.Strip(line))
				}
			}
		})
	}
}
