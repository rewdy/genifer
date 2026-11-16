package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

type retryModelsProvider struct {
	provider.Provider
	calls int
	err   error
}

func (p *retryModelsProvider) Models(context.Context) ([]provider.Model, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return testModels, nil
}

func TestModelListRetry(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(map[bool]string{false: "failure", true: "empty"}[empty], func(t *testing.T) {
			p := &retryModelsProvider{err: errors.New("offline")}
			m := New(Deps{Provider: p, StatePath: filepath.Join(t.TempDir(), "state.json")})
			initial := modelsLoadedMsg{}
			if !empty {
				initial = loadModels(p)().(modelsLoadedMsg)
			}
			updated, _ := m.Update(initial)
			m = updated.(Model)
			for attempt := 0; attempt < 2; attempt++ {
				if !strings.Contains(m.pickerView(), "Press r to retry") || !strings.Contains(ansi.Strip(m.footerView()), "r retry") {
					t.Fatal("missing retry hint")
				}
				updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
				m = updated.(Model)
				if cmd == nil || m.modelsReady || m.modelsErr != nil || m.status != "Loading models..." {
					t.Fatal("retry did not reset loading state")
				}
				if strings.Contains(ansi.Strip(m.footerView()), "r retry") {
					t.Fatal("retry advertised while loading")
				}
				_, duplicate := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
				if duplicate != nil {
					t.Fatal("duplicate request while loading")
				}
				before := p.calls
				if attempt == 1 {
					p.err = nil
				}
				updated, _ = m.Update(cmd())
				m = updated.(Model)
				if p.calls != before+1 {
					t.Fatal("retry did not call provider exactly once")
				}
			}
			if m.modelsErr != nil || !m.modelsReady || len(m.picker.Items()) != len(testModels) {
				t.Fatal("retry did not restore picker")
			}
			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
			if cmd != nil {
				t.Fatal("healthy picker retried")
			}
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
			updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
			if updated.(Model).picker.FilterValue() != "r" {
				t.Fatal("r did not type into filter")
			}
		})
	}
}
