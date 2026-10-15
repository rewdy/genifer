package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/preview"
	"github.com/rewdy/genifer/internal/provider"
)

func executePreview(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, next := range batch {
			m = executePreview(t, m, next)
		}
		return m
	}
	switch msg.(type) {
	case previewLoadedMsg, previewGeometryMsg:
		updated, next := m.Update(msg)
		return executePreview(t, updated.(Model), next)
	}
	return m
}
func previewModel(t *testing.T) (Model, *preview.Registry) {
	t.Helper()
	registry := new(preview.Registry)
	loader := preview.NewLoader(registry)
	m := resultModel(provider.Capabilities{})
	m.width, m.height = 80, 50
	m.previewGeometry = preview.Geometry{Columns: 80, Rows: 50, PixelWidth: 800, PixelHeight: 1000}
	m.deps.Preview = &PreviewDeps{Geometry: func() (preview.Geometry, error) { return m.previewGeometry, nil }, Load: loader.Load}
	path := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(path, makePNG(t), 0600); err != nil {
		t.Fatal(err)
	}
	m.outcome = gen.Outcome{Path: path, CostUSD: 0.05}
	return m, registry
}
func TestResultPreviewLayoutAndNativeOutput(t *testing.T) {
	for _, protocol := range []preview.Protocol{preview.Kitty, preview.ITerm2} {
		t.Run(fmt.Sprint(protocol), func(t *testing.T) {
			m, registry := previewModel(t)
			cmd := m.beginPreview()
			m = executePreview(t, m, cmd)
			placement := m.previewPlacement()
			if placement.ID == 0 {
				t.Fatal("no preview")
			}
			plain := ansi.Strip(m.View())
			if m.previewItem.Height != 150 || m.previewItem.Width != 150 {
				t.Fatal(m.previewItem)
			}
			savedRow := 0
			for row, line := range strings.Split(plain, "\n") {
				if strings.Contains(line, "Saved ") {
					savedRow = row + 1
				}
				if ansi.StringWidth(line) != m.width {
					t.Fatalf("bad row width %d", ansi.StringWidth(line))
				}
			}
			if savedRow != placement.Row+placement.Rows+1 {
				t.Fatalf("saved row %d; preview %+v", savedRow, placement)
			}
			for _, text := range []string{"cost:", "open folder", "edit prompt", "try new model", "new image", "re-generate", "quit"} {
				if !strings.Contains(plain, text) {
					t.Fatal(text)
				}
			}
			if strings.Contains(m.View(), "\x1b_G") || strings.Contains(m.View(), "1337;File") {
				t.Fatal("raw protocol in styled view")
			}
			var output bytes.Buffer
			writer := preview.NewWriter(&output, protocol, registry)
			if _, err := writer.Write([]byte(m.View())); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(output.String(), "genifer-preview;") {
				t.Fatal("internal marker leaked")
			}
			if protocol == preview.Kitty && !strings.Contains(output.String(), "a=T") {
				t.Fatal("missing Kitty image")
			}
			if protocol == preview.ITerm2 && !strings.Contains(output.String(), "inline=1") {
				t.Fatal("missing inline image")
			}
			writer.Close()
		})
	}
}
func TestPreviewNoSpaceOrGeometry(t *testing.T) {
	for _, width := range []int{20, 80} {
		for _, height := range []int{12, 20, 50} {
			m, _ := previewModel(t)
			m.width, m.height = width, height
			m.outcome.Path = strings.Repeat("long-directory/", 12) + "image.png"
			m.previewGeometry = preview.Geometry{Columns: width, Rows: height, PixelWidth: width * 10, PixelHeight: height * 20}
			w, h, _ := m.previewBox()
			if height == 12 && h != 0 {
				t.Fatalf("short terminal has room: %d %d", w, h)
			}
			if w > width*10 || h > height*20 {
				t.Fatal("box overflow")
			}
		}
	}
	m, _ := previewModel(t)
	m.previewGeometry = preview.Geometry{}
	if cmd := m.beginPreview(); cmd != nil {
		t.Fatal("loading with no geometry")
	}
}
func TestPreviewSilentFallbackAndSVG(t *testing.T) {
	for _, kind := range []string{"svg", "missing", "failed", "disabled"} {
		t.Run(kind, func(t *testing.T) {
			m, _ := previewModel(t)
			switch kind {
			case "svg":
				m.outcome.Path = filepath.Join(t.TempDir(), "image.svg")
				os.WriteFile(m.outcome.Path, []byte(`<svg/>`), 0600)
			case "missing":
				m.outcome.Path = "/does/not/exist.png"
			case "failed":
				m.outcome.Failure = gen.FailureGeneration
			case "disabled":
				m.deps.Preview = nil
			}
			before := m.status
			cmd := m.beginPreview()
			m = executePreview(t, m, cmd)
			if m.previewPlacement().ID != 0 || m.status != before {
				t.Fatal("failure changed screen")
			}
			// Marker aside, unsupported previews preserve exactly the text-only frame.
			withPreview := ansi.Strip(m.View())
			m.deps.Preview = nil
			if withPreview != ansi.Strip(m.View()) {
				t.Fatal("fallback reserved space or changed text")
			}
		})
	}
}
func TestPreviewNavigationCancelsAndDiscardsLateResult(t *testing.T) {
	for _, keyName := range []string{"p", "m", "n", "g", "q", "ctrl+c"} {
		t.Run(keyName, func(t *testing.T) {
			m, _ := previewModel(t)
			var ctx context.Context
			m.deps.Preview.Load = func(c context.Context, _ string, _, _ int) (preview.Item, error) {
				ctx = c
				return preview.Item{ID: 1, Width: 100, Height: 100}, nil
			}
			cmd := m.beginPreview()
			msg := cmd().(previewLoadedMsg)
			if ctx == nil {
				t.Fatal("no context")
			}
			k := key(keyName)
			if keyName == "ctrl+c" {
				k = tea.KeyMsg{Type: tea.KeyCtrlC}
			}
			updated, _ := m.Update(k)
			got := updated.(Model)
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatal("load not cancelled")
			}
			updated, _ = got.Update(msg)
			got = updated.(Model)
			if got.previewItem.ID != 0 {
				t.Fatal("late result accepted")
			}
		})
	}
}
func TestPreviewResizeAndStaleGeometry(t *testing.T) {
	m, _ := previewModel(t)
	old := m.previewGeometryRevision
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 60})
	m = updated.(Model)
	if cmd == nil || m.previewItem.ID != 0 {
		t.Fatal("resize did not refresh")
	}
	updated, _ = m.Update(previewGeometryMsg{revision: old, geometry: preview.Geometry{Columns: 100, Rows: 60, PixelWidth: 1000, PixelHeight: 1200}})
	m = updated.(Model)
	if m.previewGeometry.Columns != 0 {
		t.Fatal("stale geometry accepted")
	}
	updated, cmd = m.Update(previewGeometryMsg{revision: m.previewGeometryRevision, geometry: preview.Geometry{Columns: 100, Rows: 60, PixelWidth: 1000, PixelHeight: 1200}})
	m = executePreview(t, updated.(Model), cmd)
	if m.previewPlacement().ID == 0 {
		t.Fatal("resize did not reload")
	}
}
func TestStatusRedrawRefitsPreview(t *testing.T) {
	m, _ := previewModel(t)
	m = executePreview(t, m, m.beginPreview())
	original := m.previewRevision
	updated, cmd := m.Update(folderOpenedMsg{err: errors.New(strings.Repeat("long folder error ", 30))})
	m = executePreview(t, updated.(Model), cmd)
	if m.previewRevision == original {
		t.Fatal("wrapped status did not refit")
	}
	if m.previewItem.ID != 0 && m.previewPlacement().ID == 0 {
		t.Fatal("prepared image does not fit")
	}
}
func TestGeneratedPreviewStartsWithoutBlockingOtherActions(t *testing.T) {
	m, _ := previewModel(t)
	m.phase = phaseGenerating
	m.deps.Config.AutoOpen = true
	opens := 0
	m.deps.Open = func(context.Context, string, string) error { opens++; return nil }
	updated, cmd := m.Update(generatedMsg{outcome: m.outcome})
	m = updated.(Model)
	if m.phase != phaseResult || m.previewItem.ID != 0 {
		t.Fatal("success waited for preview")
	}
	m = executePreview(t, m, cmd)
	if opens != 1 || m.previewPlacement().ID == 0 {
		t.Fatal("auto-open or preview lost")
	}
}
