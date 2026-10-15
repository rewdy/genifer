package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
)

func TestResultOpenFolder(t *testing.T) {
	for _, launchErr := range []error{nil, errors.New("no file manager")} {
		m := resultModel(provider.Capabilities{})
		m.outcome.Path = filepath.Join(t.TempDir(), "image.svg")
		calls := 0
		m.deps.OpenFolder = func(ctx context.Context, path string) error {
			calls++
			if path != m.outcome.Path || ctx.Err() != nil {
				t.Errorf("wrong folder input %q", path)
			}
			return launchErr
		}
		m.deps.Open = func(context.Context, string, string) error { t.Fatal("image viewer used for folder"); return nil }
		if !strings.Contains(ansi.Strip(m.footerView()), "f open folder") {
			t.Fatal("missing folder hint")
		}
		updated, cmd := m.handleResultKey(key("f"))
		if cmd == nil {
			t.Fatal("missing command")
		}
		if calls != 0 {
			t.Fatal("I/O happened before command execution")
		}
		updated, _ = updated.(Model).Update(cmd())
		got := updated.(Model)
		if calls != 1 || got.phase != phaseResult || got.outcome.Path != m.outcome.Path {
			t.Fatal("result not preserved")
		}
		if launchErr != nil && !strings.Contains(got.status, launchErr.Error()) {
			t.Fatal(got.status)
		}
	}
}
func TestFailedResultCannotOpenFolder(t *testing.T) {
	m := resultModel(provider.Capabilities{})
	m.outcome = gen.Outcome{Failure: gen.FailureGeneration}
	if strings.Contains(m.footerView(), "open folder") {
		t.Fatal("failure offers open folder")
	}
	_, cmd := m.handleResultKey(key("f"))
	if cmd != nil {
		t.Fatal("failure opens folder")
	}
}
