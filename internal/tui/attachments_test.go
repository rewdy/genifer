package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/provider"
)

func TestPathBatchAtomicValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ref.png")
	if err := os.WriteFile(path, makePNG(t), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, prompt string
		max          int
		wantCount    int
		review       bool
	}{
		{"mixed too many", "bird\n@" + path, 1, 1, false},
		{"mixed valid", "bird\n@" + path, 2, 2, true},
		{"invalid batch", "bird\n@" + path + "\n@" + filepath.Join(dir, "missing.png"), 3, 1, false},
		{"empty cleaned", "@" + path, 2, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := composeModel(true, fakePaster{})
			m.models[0].Capabilities.ReferenceMax = &tc.max
			ref, _ := validateRefImage(makePNG(t))
			m.addRefImage(ref)
			m.prompt.SetValue(tc.prompt)
			updated, cmd := m.submitCompose()
			if cmd == nil {
				t.Fatal("no command")
			}
			pending := updated.(Model)
			if pending.prompt.Value() != tc.prompt || len(pending.refImages) != 1 {
				t.Fatal("mutated before command")
			}
			updated, _ = pending.Update(cmd())
			got := updated.(Model)
			if len(got.refImages) != tc.wantCount || (got.phase == phaseReview) != tc.review {
				t.Fatalf("count %d phase %v status %s", len(got.refImages), got.phase, got.status)
			}
			if tc.wantCount == 1 && got.prompt.Value() != tc.prompt {
				t.Fatal("rejected batch consumed paths")
			}
			if tc.name == "empty cleaned" {
				if strings.Contains(got.prompt.Value(), "@") {
					t.Fatal("path retained after successful load")
				}
				got.prompt.SetValue("bird")
				again, cmd := got.submitCompose()
				if cmd != nil || len(again.(Model).refImages) != 2 {
					t.Fatal("duplicated attachment")
				}
			}
		})
	}
}

func TestAttachmentResultsDiscardedAfterDraftChanges(t *testing.T) {
	for _, kind := range []string{"paste", "paths"} {
		for _, change := range []string{"edit", "model", "navigate", "navigate back"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				m := composeModel(true, fakePaster{data: makePNG(t)})
				m.prompt.SetValue("bird")
				var result tea.Msg
				if kind == "paste" {
					result = m.pasteCmd()()
				} else {
					ref, _ := validateRefImage(makePNG(t))
					result = pathImagesMsg{token: m.attachmentToken(), refs: []provider.ReferenceImage{ref}, cleaned: "cleaned"}
				}
				switch change {
				case "edit":
					updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
					m = updated.(Model)
				case "model":
					m.models[0].ID = "other"
				case "navigate":
					updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
					m = updated.(Model)
				case "navigate back":
					updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
					m = updated.(Model)
					m.phase = phaseCompose
				}
				before := m.prompt.Value()
				updated, _ := m.Update(result)
				got := updated.(Model)
				if len(got.refImages) != 0 || got.prompt.Value() != before {
					t.Fatal("stale result applied")
				}
			})
		}
	}
}

func TestPasteConstraintFailurePreservesAttachments(t *testing.T) {
	m := composeModel(true, fakePaster{data: makePNG(t)})
	max := 1
	m.models[0].Capabilities.ReferenceMax = &max
	ref, _ := validateRefImage(makePNG(t))
	m.addRefImage(ref)
	updated, _ := m.Update(m.pasteCmd()())
	got := updated.(Model)
	if len(got.refImages) != 1 || !strings.Contains(got.status, "at most 1") {
		t.Fatalf("count %d status %s", len(got.refImages), got.status)
	}
}
