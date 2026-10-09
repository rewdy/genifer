package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// modelDelegate is a compact, single-line list delegate that renders a model's
// name in the normal/selected color and its price + capabilities in a muted
// color, so the trailing detail reads as secondary to the name.
type modelDelegate struct{}

func (modelDelegate) Height() int                         { return 1 }
func (modelDelegate) Spacing() int                        { return 0 }
func (modelDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d modelDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(modelItem)
	if !ok {
		return
	}
	selected := index == m.Index()

	// Name: pink + bold when selected, soft lavender otherwise.
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e6e1f2")).Background(headerBG)
	cursor := "  "
	if selected {
		nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5fd2")).Bold(true).Background(headerBG)
		cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5fd2")).Bold(true).Background(headerBG).Render("> ")
	}
	name := nameStyle.Render(it.t.Model.Name)

	// Detail (provider tag + price + caps): always muted, regardless of
	// selection, so it reads as secondary to the name.
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b85a0")).Background(headerBG)
	var details []string
	if it.t.ProviderKey != "" {
		details = append(details, "@"+it.t.ProviderKey)
	}
	if price := priceLabel(it.price, it.priced, it.loading); price != "" {
		details = append(details, price)
	}
	if caps := capHint(it.t.Model.Capabilities); caps != "" {
		details = append(details, caps)
	}

	line := cursor + name
	if len(details) > 0 {
		line += detailStyle.Render("  " + strings.Join(details, "  "))
	}

	// Truncate to the list width so a long line never wraps into two rows, then
	// fill any remaining columns with the app background so the terminal's
	// default never shows through on a short row.
	if w := m.Width(); w > 0 {
		line = ansi.Truncate(line, w, "…")
		if pad := w - ansi.StringWidth(line); pad > 0 {
			line += rowFill(pad)
		}
	}
	fmt.Fprint(w, line)
}

var _ list.ItemDelegate = modelDelegate{}
