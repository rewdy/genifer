// Command preview-check exercises native image redraws without API calls.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/preview"
)

type model struct {
	registry              *preview.Registry
	id                    uint64
	width, height         int
	cellWidth, cellHeight int
	visible               bool
	status                int
}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "p":
			m.visible = !m.visible
		case " ":
			m.status++
		}
	}
	return m, nil
}
func (m model) View() string {
	var text strings.Builder
	text.WriteString("Genifer native-preview fixture (no API calls)\n\n")
	placement := preview.Placement{}
	columns := int(math.Ceil(240 / float64(m.cellWidth)))
	rows := int(math.Ceil(150 / float64(m.cellHeight)))
	if m.visible && m.height >= rows+8 && m.width >= columns+2 {
		placement = preview.Placement{ID: m.id, Row: 3, Column: 1, Rows: rows, Columns: columns}
		text.WriteString(strings.Repeat("\n", rows+1))
	} else {
		text.WriteString("Text-only screen\n\n")
	}
	fmt.Fprintf(&text, "Saved fixture · status redraw %d · terminal %d x %d cells\n\n", m.status, m.width, m.height)
	text.WriteString("space: redraw · p: hide/show · resize window · q: quit\n")
	// First-line marker is inserted only after constructing the text frame.
	return preview.Marker(placement) + text.String()
}
func fixture() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 240, 150))
	for y := 0; y < 150; y++ {
		for x := 0; x < 240; x++ {
			c := color.NRGBA{R: uint8(x), G: uint8(y * 255 / 150), B: 180, A: 255}
			if x >= 40 && x < 200 && y >= 35 && y < 115 {
				c = color.NRGBA{R: 240, G: 70, B: 100, A: 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}
func main() {
	app := flag.Bool("app", false, "exercise the real success screen with a local fixture provider")
	protocol := flag.String("protocol", "kitty", "kitty (Ghostty) or iterm2 (iTerm2)")
	cellWidth := flag.Int("cell-width", 8, "terminal cell width in pixels (fixture reservation only)")
	cellHeight := flag.Int("cell-height", 16, "terminal cell height in pixels (fixture reservation only)")
	flag.Parse()
	if *app {
		if err := runApp(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	var selected preview.Protocol
	switch *protocol {
	case "kitty":
		selected = preview.Kitty
	case "iterm2":
		selected = preview.ITerm2
	default:
		fmt.Fprintln(os.Stderr, "protocol must be kitty or iterm2")
		os.Exit(2)
	}
	if *cellWidth < 1 || *cellHeight < 1 {
		fmt.Fprintln(os.Stderr, "cell dimensions must be positive")
		os.Exit(2)
	}
	registry := new(preview.Registry)
	id := registry.Store(fixture())
	output := preview.NewWriter(os.Stdout, selected, registry)
	p := tea.NewProgram(model{registry: registry, id: id, visible: true, cellWidth: *cellWidth, cellHeight: *cellHeight}, tea.WithOutput(output), tea.WithAltScreen())
	_, err := p.Run()
	closeErr := output.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintln(os.Stderr, closeErr)
		os.Exit(1)
	}
}
