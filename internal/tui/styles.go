package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// headerBG is the background color used across the whole app so the user's
// terminal background never clashes with the TUI. It is the single source of
// truth for the app surface; every widget style derives from it.
const headerBG = lipgloss.Color("#1e1c32")

// inputBG is one shade lighter than the app background, used for input surfaces
// (the prompt textarea and the picker's filter field) so they read as areas you
// can type into rather than as plain background.
const inputBG = lipgloss.Color("#2a2745")

// ansiReset is the SGR sequence lipgloss uses to clear formatting.
const ansiReset = "\x1b[0m"

// onAppBG returns s with the app background applied.
func onAppBG(s lipgloss.Style) lipgloss.Style {
	return s.Background(headerBG)
}

// onInputBG returns s with the input-field background applied.
func onInputBG(s lipgloss.Style) lipgloss.Style {
	return s.Background(inputBG)
}

// promptTextareaStyles builds the focused/blurred styles for the prompt input,
// each painted over the input background so the textarea reads as a distinct
// input area and never falls back to the terminal default (its stock CursorLine
// background is a hardcoded black).
func promptTextareaStyles() (textarea.Style, textarea.Style) {
	onBG := func(s textarea.Style) textarea.Style {
		s.Base = onInputBG(s.Base)
		s.CursorLine = onInputBG(s.CursorLine)
		s.CursorLineNumber = onInputBG(s.CursorLineNumber)
		s.EndOfBuffer = onInputBG(s.EndOfBuffer)
		s.LineNumber = onInputBG(s.LineNumber)
		s.Placeholder = onInputBG(s.Placeholder)
		s.Prompt = onInputBG(s.Prompt)
		s.Text = onInputBG(s.Text)
		return s
	}
	focused, blurred := textarea.DefaultStyles()
	return onBG(focused), onBG(blurred)
}

// pickerStyles builds the list styles painted over the app background, covering
// the filter bar, pagination dots, and status text the defaults leave bare.
func pickerStyles() list.Styles {
	s := list.DefaultStyles()
	s.TitleBar = onAppBG(s.TitleBar)
	s.Title = onAppBG(s.Title)
	s.Spinner = onAppBG(s.Spinner)
	s.FilterPrompt = onAppBG(s.FilterPrompt)
	s.FilterCursor = onAppBG(s.FilterCursor)
	s.DefaultFilterCharacterMatch = onAppBG(s.DefaultFilterCharacterMatch)
	s.StatusBar = onAppBG(s.StatusBar)
	s.StatusEmpty = onAppBG(s.StatusEmpty)
	s.StatusBarActiveFilter = onAppBG(s.StatusBarActiveFilter)
	s.StatusBarFilterCount = onAppBG(s.StatusBarFilterCount)
	s.NoItems = onAppBG(s.NoItems)
	s.PaginationStyle = onAppBG(s.PaginationStyle)
	s.HelpStyle = onAppBG(s.HelpStyle)
	s.ActivePaginationDot = onAppBG(s.ActivePaginationDot)
	s.InactivePaginationDot = onAppBG(s.InactivePaginationDot)
	s.ArabicPagination = onAppBG(s.ArabicPagination)
	s.DividerDot = onAppBG(s.DividerDot)
	return s
}

// filterInputStyles paints the picker's filter text input over the input
// background so it reads as a distinct input area.
func filterInputStyles(t textinput.Model) textinput.Model {
	t.PromptStyle = onInputBG(t.PromptStyle)
	t.TextStyle = onInputBG(t.TextStyle)
	t.PlaceholderStyle = onInputBG(t.PlaceholderStyle)
	t.CursorStyle = onInputBG(t.CursorStyle)
	t.CompletionStyle = onInputBG(t.CompletionStyle)
	return t
}

// colorSeq returns the SGR sequence that sets c for the terminal's active color
// profile, or "" when it has no color support.
func colorSeq(c lipgloss.Color) string {
	out := lipgloss.NewStyle().Background(c).Render("\x00")
	if i := strings.IndexByte(out, 0); i > 0 {
		return out[:i]
	}
	return ""
}

// backgroundSeq returns the SGR sequence that sets the app background.
func backgroundSeq() string {
	return colorSeq(headerBG)
}

// rowFill returns a string of n spaces painted with the app background, used to
// extend a shorter-than-width row without exposing the terminal default.
func rowFill(n int) string {
	if n <= 0 {
		return ""
	}
	return lipgloss.NewStyle().Background(headerBG).Render(strings.Repeat(" ", n))
}

// paintOver repaints content over background c: it re-asserts c after every
// reset in each line (so any cells a widget left bare inherit it) and pads the
// line to width. It repairs widgets whose own view leaves padding unstyled (for
// example the prompt textarea's placeholder, whose lines are not padded to
// width). It is a no-op on terminals without color support or before the first
// resize.
func paintOver(content string, width int, c lipgloss.Color) string {
	seq := colorSeq(c)
	if width <= 0 || seq == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.ReplaceAll(line, ansiReset, ansiReset+seq)
		pad := ""
		if w := ansi.StringWidth(line); w < width {
			pad = strings.Repeat(" ", width-w)
		}
		lines[i] = seq + line + pad + ansiReset
	}
	return strings.Join(lines, "\n")
}

// paintBackground is the whole-surface backstop: the app background is
// re-asserted across every line so no embedded widget can leave the terminal's
// default background showing through.
func paintBackground(content string, width int) string {
	return paintOver(content, width, headerBG)
}
