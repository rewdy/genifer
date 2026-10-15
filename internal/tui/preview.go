package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/preview"
)

// PreviewDeps is enabled only when Run has installed a native output adapter.
// Tests inject geometry and local image loading without touching a real TTY.
type PreviewDeps struct {
	Geometry func() (preview.Geometry, error)
	Load     func(context.Context, string, int, int) (preview.Item, error)
}
type previewGeometryMsg struct {
	revision uint64
	geometry preview.Geometry
	err      error
}
type previewLoadedMsg struct {
	revision uint64
	item     preview.Item
	err      error
}

func (m Model) geometryCmd() tea.Cmd {
	if m.deps.Preview == nil || m.deps.Preview.Geometry == nil {
		return nil
	}
	revision, read := m.previewGeometryRevision, m.deps.Preview.Geometry
	return func() tea.Msg { g, err := read(); return previewGeometryMsg{revision, g, err} }
}
func (m *Model) clearPreview() {
	if m.previewCancel != nil {
		m.previewCancel()
		m.previewCancel = nil
	}
	m.previewRevision++
	m.previewItem = preview.Item{}
	m.previewBudgetWidth, m.previewBudgetHeight = 0, 0
}
func (m Model) previewBox() (int, int, int) {
	if !m.previewGeometry.Valid(m.width, m.height) {
		return 0, 0, 0
	}
	header := lipgloss.Height(RenderHeader(m.width, m.deps.Version))
	result := lipgloss.Height(ansi.Wrap(m.resultTextView(), m.contentWidth(), ""))
	footer := lipgloss.Height(m.footerView())
	return m.previewGeometry.FrameBox(m.contentWidth(), header, result, footer)
}
func (m *Model) beginPreview() tea.Cmd {
	m.clearPreview()
	if m.deps.Preview == nil || m.deps.Preview.Load == nil || m.phase != phaseResult || m.outcome.Failure != gen.FailureNone || m.outcome.Path == "" {
		return nil
	}
	width, height, _ := m.previewBox()
	if width < 1 || height < 1 {
		return nil
	}
	m.previewBudgetWidth, m.previewBudgetHeight = width, height
	revision, path, load := m.previewRevision, m.outcome.Path, m.deps.Preview.Load
	ctx, cancel := context.WithCancel(context.Background())
	m.previewCancel = cancel
	return func() tea.Msg {
		item, err := load(ctx, path, width, height)
		return previewLoadedMsg{revision, item, err}
	}
}
func (m Model) previewPlacement() preview.Placement {
	if m.deps.Preview == nil || m.phase != phaseResult || m.outcome.Failure != gen.FailureNone || m.quit || m.previewItem.ID == 0 {
		return preview.Placement{}
	}
	width, height, row := m.previewBox()
	item := m.previewItem
	if item.Width < 1 || item.Height < 1 || item.Width > width || item.Height > height {
		return preview.Placement{}
	}
	size := m.previewGeometry.Size(item.Width, item.Height)
	if size.Rows < 1 || size.Columns < 1 {
		return preview.Placement{}
	}
	return preview.Placement{ID: item.ID, Row: row, Column: horizontalPadding(m.width) + 1, Rows: size.Rows, Columns: size.Columns}
}
func (m Model) resultView() string {
	text := m.resultTextView()
	if p := m.previewPlacement(); p.ID != 0 {
		return strings.Repeat("\n", p.Rows+1) + text
	}
	return text
}
