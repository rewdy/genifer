package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

func TestVectorRequirementsVisibleAndBounded(t *testing.T) {
	min, max := 1, 10
	for _, width := range []int{100, 32, 20} {
		for _, screen := range []phase{phaseCompose, phaseReview} {
			m := capModel(provider.Capabilities{AcceptsReferenceImages: true, ReferenceMin: &min, ReferenceMax: &max, OutputFormats: []string{"svg"}, ReferenceMediaTypes: []string{"image/png", "image/jpeg", "image/webp"}, ReferenceMinWidth: 256, ReferenceMinHeight: 256})
			updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 80})
			m = updated.(Model)
			m.phase = screen
			m.prompt.SetValue("bird")
			body := ansi.Strip(m.bodyView())
			unwrapped := strings.ReplaceAll(body, "\n", "")
			// Wrapping can split words at very narrow widths, so compare without whitespace.
			compact := strings.Join(strings.Fields(unwrapped), "")
			for _, want := range []string{"Referencesrequired:atleast1", "maximum10", "0attached", "Output:SVG"} {
				if !strings.Contains(compact, want) {
					t.Fatalf("width %d phase %v missing %s: %s", width, screen, want, body)
				}
			}
			for _, line := range strings.Split(m.View(), "\n") {
				if ansi.StringWidth(line) > width {
					t.Fatalf("width %d line %d: %s", width, ansi.StringWidth(line), line)
				}
			}
		}
	}
}
func TestOptionalAndUnknownOutputGuidance(t *testing.T) {
	for _, formats := range [][]string{nil, {"png", "svg"}} {
		m := capModel(provider.Capabilities{AcceptsReferenceImages: true, OutputFormats: formats})
		body := ansi.Strip(m.composeView() + m.reviewView())
		if !strings.Contains(body, "References optional") || strings.Contains(body, "Output: SVG") {
			t.Fatal(body)
		}
	}
	if !strings.Contains(capHint(provider.Capabilities{AcceptsReferenceImages: true, ReferenceMin: newInt(1), OutputFormats: []string{"svg"}}), "refs required SVG") {
		t.Fatal("picker hint missing requirements")
	}
}
func newInt(n int) *int { return &n }
