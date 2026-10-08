package tui

import tea "github.com/charmbracelet/bubbletea"

// Run launches the full-screen TUI and blocks until the user quits. It uses the
// alt screen so the terminal is restored to its prior state on exit.
func Run(d Deps) error {
	p := tea.NewProgram(New(d), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
