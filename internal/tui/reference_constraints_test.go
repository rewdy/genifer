package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
)

func TestReferenceConstraintsGuardEverySubmission(t *testing.T) {
	min := 1
	for _, action := range []string{"compose", "review", "retry", "regenerate"} {
		t.Run(action, func(t *testing.T) {
			m := capModel(provider.Capabilities{AcceptsReferenceImages: true, ReferenceMin: &min})
			m.prompt.SetValue("bird")
			m.lastDraft = gen.Draft{Model: m.currentModelID(), Prompt: "bird"}
			var updated tea.Model
			var cmd tea.Cmd
			switch action {
			case "compose":
				updated, cmd = m.submitCompose()
			case "review":
				m.phase = phaseReview
				updated, cmd = m.handleReviewKey(key("enter"))
			case "retry":
				m.phase = phaseResult
				m.outcome = gen.Outcome{Failure: gen.FailureGeneration}
				updated, cmd = m.handleResultKey(key("r"))
			case "regenerate":
				m.phase = phaseResult
				updated, cmd = m.handleResultKey(key("g"))
			}
			got := updated.(Model)
			if cmd != nil || got.phase != phaseCompose || got.prompt.Value() != "bird" || !strings.Contains(got.status, "at least 1") {
				t.Fatalf("phase %v cmd %v status %s", got.phase, cmd != nil, got.status)
			}
		})
	}
	// Zero minimum remains text-only capable.
	zero := 0
	m := capModel(provider.Capabilities{AcceptsReferenceImages: true, ReferenceMin: &zero})
	m.prompt.SetValue("bird")
	updated, _ := m.submitCompose()
	if updated.(Model).phase != phaseReview {
		t.Fatal("optional refs blocked")
	}
}

func TestModelSwitchPreservesInvalidReferencesForCorrection(t *testing.T) {
	ref, _ := validateRefImage(makePNG(t))
	max := 1
	for _, caps := range []provider.Capabilities{
		{AcceptsReferenceImages: true, ReferenceMax: &max},
		{AcceptsReferenceImages: true, ReferenceMinWidth: 256, ReferenceMinHeight: 256},
		{AcceptsReferenceImages: true, ReferenceMediaTypes: []string{"image/jpeg"}},
	} {
		m := capModel(caps)
		m.phase = phasePicker
		m.prompt.SetValue("bird")
		m.refImages = []provider.ReferenceImage{ref, ref}
		updated, _ := m.handlePickerKey(key("enter"))
		got := updated.(Model)
		if len(got.refImages) != 2 || got.prompt.Value() != "bird" || !strings.Contains(got.status, "invalid reference") {
			t.Fatalf("lost draft/status %s", got.status)
		}
		updated, cmd := got.submitCompose()
		if cmd != nil || updated.(Model).phase != phaseCompose {
			t.Fatal("invalid carried references accepted")
		}
	}
}

func TestProviderReferenceFailureReturnsToEditing(t *testing.T) {
	m := capModel(provider.Capabilities{AcceptsReferenceImages: true})
	ref, _ := validateRefImage(makePNG(t))
	m.lastDraft = gen.Draft{Prompt: "exact rejected draft", Model: m.currentModelID(), ReferenceImages: []provider.ReferenceImage{ref}}
	m.phase = phaseGenerating
	updated, cmd := m.Update(generatedMsg{outcome: gen.Outcome{Failure: gen.FailureInput, Err: provider.ErrInvalidReference}})
	got := updated.(Model)
	if cmd != nil || got.phase != phaseCompose || got.prompt.Value() != m.lastDraft.Prompt || len(got.refImages) != 1 {
		t.Fatal("rejected draft not restored")
	}
	if strings.Contains(got.footerView(), "retry") {
		t.Fatal("unchanged retry offered")
	}
}
