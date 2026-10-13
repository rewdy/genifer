package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// historyItem adapts a config.HistoryEntry to a list.Item, filtered on the
// prompt text.
type historyItem struct {
	entry config.HistoryEntry
}

func (i historyItem) Title() string       { return i.entry.Prompt }
func (i historyItem) FilterValue() string { return i.entry.Prompt }
func (i historyItem) Description() string { return "" }

// historyDelegate renders one entry per line: the prompt (truncated) plus a
// muted relative timestamp, mirroring the model picker's single-line style.
type historyDelegate struct{}

func (historyDelegate) Height() int                         { return 1 }
func (historyDelegate) Spacing() int                        { return 0 }
func (historyDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d historyDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(historyItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e6e1f2")).Background(headerBG)
	cursor := "  "
	if selected {
		promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5fd2")).Bold(true).Background(headerBG)
		cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5fd2")).Bold(true).Background(headerBG).Render("> ")
	}
	// Collapse newlines so a multi-line prompt stays on one row.
	prompt := promptStyle.Render(strings.ReplaceAll(it.entry.Prompt, "\n", " "))

	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b85a0")).Background(headerBG)
	line := cursor + prompt
	if when := relativeTime(it.entry.CreatedAt, timeNow()); when != "" {
		line += detailStyle.Render("  " + when)
	}

	if wd := m.Width(); wd > 0 {
		line = ansi.Truncate(line, wd, "…")
		if pad := wd - ansi.StringWidth(line); pad > 0 {
			line += rowFill(pad)
		}
	}
	fmt.Fprint(w, line)
}

var _ list.ItemDelegate = historyDelegate{}

// relativeTime renders a short, human relative age ("just now", "3h ago",
// "2d ago"). An empty/zero time renders nothing.
func relativeTime(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours())/24)
	}
}

// loadRecentEntries reads the history store and returns at most historyLoadLimit
// of the most-recent entries (newest-first) along with whether the store holds
// more than were returned. An empty slice is returned when history is disabled.
func (m Model) loadRecentEntries() (entries []config.HistoryEntry, moreExist bool) {
	if !m.deps.Config.HistoryEnabled() {
		return nil, false
	}
	cache := config.LoadHistory(m.deps.HistoryPath)
	total := len(cache.Entries)
	entries = cache.Entries
	if total > historyLoadLimit {
		entries = entries[:historyLoadLimit]
	}
	return entries, total > len(entries)
}

// openHistory loads the history store, keeps the newest historyLoadLimit
// entries into the list, records whether more exist on disk, and switches to
// phaseHistory (remembering where to return on esc).
func (m Model) openHistory(from phase) Model {
	m.historyPrev = from
	m.phase = phaseHistory

	if !m.deps.Config.HistoryEnabled() {
		m.historyEntries = nil
		m.historyMoreExist = false
		m.history.SetItems(nil)
		m.status = "History is turned off"
		return m
	}

	entries, moreExist := m.loadRecentEntries()
	m.historyEntries = entries
	m.historyMoreExist = moreExist

	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = historyItem{entry: e}
	}
	m.history.SetItems(items)
	m.history.Select(0)

	if len(entries) == 0 {
		m.status = "No history yet"
	} else {
		m.status = fmt.Sprintf("%d recent prompt(s) · / to filter · enter to reuse", len(entries))
	}
	return m
}

// handleHistoryKey drives the history view: esc returns to the prior phase,
// enter reuses the selected entry, and everything else goes to the list.
func (m Model) handleHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// While filtering, let the list own the keys (except global ctrl+c, handled
	// upstream) so typing and esc-to-clear behave normally.
	if m.history.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.history, cmd = m.history.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.phase = m.historyPrev
		if m.phase == phaseCompose {
			m.prompt.Focus()
		}
		m.status = ""
		return m, nil
	case "enter":
		if it, ok := m.history.SelectedItem().(historyItem); ok {
			return m.reuseHistoryEntry(it.entry), nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.history, cmd = m.history.Update(msg)
	return m, cmd
}

// reuseHistoryEntry pre-fills compose from a history entry: the prompt always,
// and aspect ratio / seed only where the currently selected model supports
// them.
func (m Model) reuseHistoryEntry(e config.HistoryEntry) Model {
	m.prompt.SetValue(e.Prompt)
	caps := m.currentCaps()

	m.aspectRatio = ""
	if len(caps.AspectRatios) > 0 && e.AspectRatio != "" && indexOfString(caps.AspectRatios, e.AspectRatio) >= 0 {
		m.aspectRatio = e.AspectRatio
	} else if len(caps.AspectRatios) > 0 {
		m.aspectRatio = pickAspectRatio(caps.AspectRatios, e.AspectRatio)
	}

	m.reuseSeed = nil
	if caps.SupportsSeed && e.Seed != nil {
		s := *e.Seed
		m.reuseSeed = &s
	}

	m.phase = phaseCompose
	m.prompt.Focus()
	m.status = "Reusing a past prompt"
	return m
}

// historyPointer renders the message naming the on-disk history file so the
// user can find the full record there.
func (m Model) historyPointer() string {
	return "Full history can be found in " + displayHistoryPath(m.deps.HistoryPath)
}

// displayHistoryPath returns the history file path for display, shortening the
// user's home dir to ~ when possible, and falling back to the canonical config
// location when no path is set.
func displayHistoryPath(path string) string {
	if path == "" {
		return "~/.config/genifer/history-cache.json"
	}
	if home := homeDir(); home != "" && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// homeDir returns the user's home directory, or "" when it cannot be resolved.
func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return ""
}

// recordHistoryCmd returns a background command that appends the just-completed
// generation to the history store, or nil when history is disabled, no path is
// configured, or the outcome did not succeed. Writing is best-effort: a failure
// is swallowed so a slow or read-only disk never disrupts the UI.
func (m Model) recordHistoryCmd() tea.Cmd {
	if !m.deps.Config.HistoryEnabled() || m.deps.HistoryPath == "" {
		return nil
	}
	if m.outcome.Failure != gen.FailureNone {
		return nil
	}
	d := m.lastDraft
	entry := config.HistoryEntry{
		Prompt:      d.Prompt,
		Model:       d.Model,
		AspectRatio: d.AspectRatio,
		Seed:        m.outcome.Seed,
		Path:        m.outcome.Path,
		CreatedAt:   timeNow(),
	}
	path := m.deps.HistoryPath
	return func() tea.Msg {
		cache := config.LoadHistory(path)
		_ = config.SaveHistory(path, cache.Append(entry))
		return nil
	}
}

// historyView renders the history list, or the appropriate empty/disabled
// message, and appends the full-history pointer when the user has paged to the
// end of the loaded entries (and more exist) or filtered to no matches.
func (m Model) historyView() string {
	if !m.deps.Config.HistoryEnabled() {
		return styleFaint.Render("History is turned off. Enable it with `history: true` in config.yaml.")
	}
	if len(m.historyEntries) == 0 {
		return styleFaint.Render("No history yet — your past prompts will appear here.")
	}

	body := "Past prompts:\n\n" + m.history.View()

	// Empty filter result: the filter matched none of the loaded entries, so
	// the match (if any) is in the older entries that live only on disk.
	if m.historyFilterActive() && len(m.history.VisibleItems()) == 0 {
		return body + "\n\n" + styleFaint.Render(m.historyPointer())
	}
	// End of the loaded entries with more on disk: tell the user where the rest
	// of their history lives.
	if m.historyMoreExist && m.historyAtEnd() {
		return body + "\n\n" + styleFaint.Render(m.historyPointer())
	}
	return body
}

// historyFilterActive reports whether a filter is being typed or applied.
func (m Model) historyFilterActive() bool {
	switch m.history.FilterState() {
	case list.Filtering, list.FilterApplied:
		return true
	default:
		return false
	}
}

// historyAtEnd reports whether the cursor is on the last loaded entry.
func (m Model) historyAtEnd() bool {
	items := m.history.VisibleItems()
	return len(items) > 0 && m.history.Index() == len(items)-1
}

// --- inline Up/Down recall -------------------------------------------------

// recallActive reports whether a recall session is in progress (the prompt
// currently holds a recalled entry rather than the user's draft).
func (m Model) recallActive() bool { return m.recallIndex != recallInactive }

// recallEntries returns the entries recall walks over. During an active session
// it uses the snapshot taken at session start (m.historyEntries); otherwise it
// loads the current recent entries so a fresh session reflects the latest
// history.
func (m Model) recallEntries() []config.HistoryEntry {
	if m.recallActive() {
		return m.historyEntries
	}
	entries, _ := m.loadRecentEntries()
	return entries
}

// recallPrevious steps to an older entry. Starting from the draft it snapshots
// the current prompt and applies the newest entry; subsequent calls walk toward
// older entries, clamped at the oldest loaded. A no-op when there is nothing to
// recall.
func (m Model) recallPrevious() Model {
	entries := m.recallEntries()
	if len(entries) == 0 {
		return m
	}
	if !m.recallActive() {
		m.recallDraft = m.prompt.Value()
		m.historyEntries = entries // freeze the set for this session
		m.recallIndex = 0
	} else if m.recallIndex < len(entries)-1 {
		m.recallIndex++
	} else {
		return m // already at the oldest loaded entry
	}
	return m.applyRecalled(entries[m.recallIndex])
}

// recallNext steps toward a newer entry. Stepping forward past the newest entry
// restores the snapshot draft and ends the session. A no-op when not recalling.
func (m Model) recallNext() Model {
	if !m.recallActive() {
		return m
	}
	entries := m.historyEntries
	if m.recallIndex > 0 {
		m.recallIndex--
		return m.applyRecalled(entries[m.recallIndex])
	}
	// Past the newest: restore the user's draft and end the session.
	return m.restoreRecallDraft()
}

// applyRecalled puts a recalled entry into compose via the shared reuse path
// (prompt + aspect ratio + seed where supported), keeping recallIndex intact.
func (m Model) applyRecalled(e config.HistoryEntry) Model {
	idx := m.recallIndex
	m = m.reuseHistoryEntry(e)
	// reuseHistoryEntry switches to compose and sets status for list selection;
	// recall stays in compose and keeps its own session index.
	m.recallIndex = idx
	m.phase = phaseCompose
	m.status = fmt.Sprintf("History %d of %d (↑ older · ↓ newer)", idx+1, len(m.historyEntries))
	m.prompt.CursorEnd()
	return m
}

// restoreRecallDraft returns the prompt to the snapshot draft and ends the
// session. A draft carries no recalled seed.
func (m Model) restoreRecallDraft() Model {
	m.prompt.SetValue(m.recallDraft)
	m.reuseSeed = nil
	m.recallIndex = recallInactive
	m.recallDraft = ""
	m.prompt.CursorEnd()
	m.status = "Compose your prompt"
	return m
}

// endRecallSession ends any active recall session without changing the prompt,
// used when the user edits the text so the next Up starts fresh from the newest.
func (m *Model) endRecallSession() {
	m.recallIndex = recallInactive
	m.recallDraft = ""
}
