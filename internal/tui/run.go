package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/rewdy/genifer/internal/preview"
)

// Run launches the full-screen TUI. One output adapter owns native preview
// graphics and clears them before the alternate screen is restored on exit.
func Run(d Deps) error {
	protocol := preview.Detect(term.IsTerminal(os.Stdout.Fd()), os.Getenv)
	var options []tea.ProgramOption
	options = append(options, tea.WithAltScreen())
	var output *preview.Writer
	if protocol != preview.None {
		registry := new(preview.Registry)
		loader := preview.NewLoader(registry)
		d.Preview = &PreviewDeps{Geometry: func() (preview.Geometry, error) { return preview.ReadGeometry(os.Stdout.Fd()) }, Load: loader.Load}
		output = preview.NewWriter(os.Stdout, protocol, registry)
		options = append(options, tea.WithOutput(output))
		defer output.Close()
	}
	p := tea.NewProgram(New(d), options...)
	final, err := p.Run()
	if m, ok := final.(Model); ok && m.previewCancel != nil {
		m.previewCancel()
	}
	return err
}
