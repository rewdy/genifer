package tui

import (
	"os"

	"github.com/rewdy/genifer/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// configField identifies the focused row in the config view.
type configField int

const (
	fieldSaveLocation configField = iota
	fieldAutoOpen
	fieldHistory
	configFieldCount
)

// configState holds the in-tool config editor's working values. It is seeded
// from the loaded Config when the view opens and written back to config.yaml on
// confirm.
type configState struct {
	cursor   configField
	saveLoc  textinput.Model
	autoOpen bool
	history  bool
	err      string // last validation error, shown until the next edit
}

// newConfigState builds the editor state from the current session settings.
// saveLoc is pre-filled with the raw output_dir the user configured (which may
// be empty, meaning "use the default"), not the resolved path, so confirming
// without edits does not rewrite a relative/keyword value into an absolute one.
func newConfigState(rawOutputDir string, autoOpen, history bool) configState {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "default (~/Pictures/genifer)"
	ti.SetValue(rawOutputDir)
	return configState{
		cursor:   fieldSaveLocation,
		saveLoc:  ti,
		autoOpen: autoOpen,
		history:  history,
	}
}

// openConfig enters the config view, seeding the editor from current settings.
func (m Model) openConfig(from phase) Model {
	m.cfg = newConfigState(string(m.deps.Config.OutputDir), m.deps.Config.AutoOpen, m.deps.Config.HistoryEnabled())
	m.cfg.saveLoc.Focus()
	m.configPrev = from
	m.phase = phaseConfig
	m.status = "Edit settings · ↑/↓ move · space toggles · enter saves · esc cancels"
	return m
}

// handleConfigKey drives the config view: navigation between fields, toggling
// the booleans, confirming (validate + write + apply), and esc to return.
func (m Model) handleConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = m.configPrev
		m.status = ""
		return m, nil
	case "up", "shift+tab":
		m.cfg.saveLoc.Blur()
		if m.cfg.cursor > 0 {
			m.cfg.cursor--
		}
		m.syncSaveLocFocus()
		return m, nil
	case "down", "tab":
		if m.cfg.cursor < configFieldCount-1 {
			m.cfg.cursor++
		}
		m.syncSaveLocFocus()
		return m, nil
	case " ":
		// Toggle the focused boolean field; ignored on the text field.
		switch m.cfg.cursor {
		case fieldAutoOpen:
			m.cfg.autoOpen = !m.cfg.autoOpen
			return m, nil
		case fieldHistory:
			m.cfg.history = !m.cfg.history
			return m, nil
		}
	case "enter":
		return m.confirmConfig()
	}

	// Otherwise, feed the save-location text field when it is focused.
	if m.cfg.cursor == fieldSaveLocation {
		var cmd tea.Cmd
		m.cfg.saveLoc, cmd = m.cfg.saveLoc.Update(msg)
		m.cfg.err = "" // editing clears a stale validation error
		return m, cmd
	}
	return m, nil
}

// syncSaveLocFocus keeps the text-input focus in step with the cursor so the
// caret only blinks when the save-location row is active.
func (m *Model) syncSaveLocFocus() {
	if m.cfg.cursor == fieldSaveLocation {
		m.cfg.saveLoc.Focus()
	} else {
		m.cfg.saveLoc.Blur()
	}
}

// confirmConfig validates the save location, writes config.yaml, and applies
// the settings to the running session. On an invalid save location it reports
// the problem, keeps the prior values, and does not write the file.
func (m Model) confirmConfig() (tea.Model, tea.Cmd) {
	raw := m.cfg.saveLoc.Value()
	def, err := config.DefaultOutputDir()
	if err != nil {
		m.cfg.err = "Could not resolve the default save location: " + err.Error()
		return m, nil
	}
	resolved, err := config.ValidateOutputDir(raw, os.Getwd, os.UserHomeDir, def)
	if err != nil {
		m.cfg.err = "Unusable save location — " + err.Error()
		return m, nil // keep prior values, do not write
	}

	if err := config.WriteSettings(m.deps.ConfigPath, config.EditableSettings{
		OutputDir: raw,
		AutoOpen:  m.cfg.autoOpen,
		History:   m.cfg.history,
	}); err != nil {
		m.cfg.err = "Could not write config: " + err.Error()
		return m, nil
	}

	// Apply live: the next generation uses the new location and auto-open /
	// history preferences without a restart.
	m.outputDir = resolved
	m.deps.Config.OutputDir = config.Value(raw)
	m.deps.Config.AutoOpen = m.cfg.autoOpen
	h := m.cfg.history
	m.deps.Config.History = &h

	m.phase = m.configPrev
	m.status = "Settings saved"
	return m, nil
}

// updateConfig forwards non-key messages (e.g. cursor blink) to the text input.
func (m Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.cfg.cursor == fieldSaveLocation {
		var cmd tea.Cmd
		m.cfg.saveLoc, cmd = m.cfg.saveLoc.Update(msg)
		return m, cmd
	}
	return m, nil
}

// configView renders the editor: a save-location field and two toggles, with
// the focused row marked and any validation error shown.
func (m Model) configView() string {
	rows := []string{
		m.configRow(fieldSaveLocation, "Save location", m.cfg.saveLoc.View()),
		m.configRow(fieldAutoOpen, "Auto-open on complete", toggleLabel(m.cfg.autoOpen)),
		m.configRow(fieldHistory, "Keep prompt history", toggleLabel(m.cfg.history)),
	}
	body := "Settings:\n\n" + lipgloss.JoinVertical(lipgloss.Left, rows...)
	if m.cfg.err != "" {
		body += "\n\n" + styleErr.Render(m.cfg.err)
	}
	// Make it explicit what confirming does and which file is edited, with a
	// blank line above (the footer adds the separation below).
	body += "\n\n" + styleFaint.Render("Saving updates your config file: "+displayPath(m.deps.ConfigPath))
	return body
}

// configRow renders one labeled field, marking it when focused.
func (m Model) configRow(f configField, label, value string) string {
	marker := "  "
	labelStyle := styleFaint
	if m.cfg.cursor == f {
		marker = styleKey.Render("› ")
		labelStyle = styleKey
	}
	return marker + labelStyle.Render(label) + "  " + value
}

// toggleLabel renders a boolean as an on/off pill.
func toggleLabel(on bool) string {
	if on {
		return styleOK.Render("[on]")
	}
	return styleFaint.Render("[off]")
}
