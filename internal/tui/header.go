package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Wordmark colors: "gen" in pink, "ifer" in purple.
const (
	colorGen  = lipgloss.Color("#ff5fd2") // pink
	colorIfer = lipgloss.Color("#9a5ae6") // purple
)

// bannerLines is the "genifer" wordmark in the figlet Calvin S font. All rows
// are normalized to the same display width (bannerCols) so the pattern splice
// never overflows the header width on one row.
var bannerLines = []string{
	"┌─┐┌─┐┌┐┌┬┌─┐┌─┐┬─┐",
	"│ ┬├┤ ││││├┤ ├┤ ├┬┘",
	"└─┘└─┘┘└┘┴└  └─┘┴└─",
}

// bannerCols is the display width of the wordmark rows (all equal).
const bannerCols = 19

// genSplit is the rune column where "gen" ends and "ifer" begins. Columns
// [0,genSplit) render pink; [genSplit,end) render purple. Column 9 is the "i".
const genSplit = 9

// patternGap is the number of blank columns between the wordmark and the start
// of the decorative dot pattern.
const patternGap = 4

// dotSpacing is the column period of the dot lattice; dotOffset shifts odd rows
// so the dots form an offset lattice rather than a straight grid.
const (
	dotSpacing = 4
	dotOffset  = 2
)

// patternFadeFrom/To are the endpoints of the dot pattern's pink->purple fade.
var (
	patternFadeFrom = [3]int{0xff, 0x5f, 0xd2} // pink
	patternFadeTo   = [3]int{0x9a, 0x5a, 0xe6} // purple
)

// headerBG is the background color used across the whole app so the user's
// terminal background never clashes with the TUI.
const headerBG = lipgloss.Color("#17141f")

// RenderHeader renders the full-width styled header for the given width.
func RenderHeader(width int, version string) string {
	if width < 1 {
		width = 80
	}

	tag := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdbcff")).
		Bold(true).
		Background(headerBG).
		Render("genifer" + versionSuffix(version))

	// Content width inside the header's horizontal padding (Padding(1, 2)).
	contentWidth := width - 4
	patternWidth := contentWidth - bannerCols - patternGap
	gap := strings.Repeat(" ", patternGap)

	rows := make([]string, len(bannerLines))
	for i, line := range bannerLines {
		row := tintLine(line) // padded to bannerCols
		if patternWidth > 0 {
			row += gap + patternRow(i, patternWidth)
		}
		rows[i] = row
	}
	word := lipgloss.JoinVertical(lipgloss.Left, rows...)

	inner := lipgloss.JoinVertical(lipgloss.Left, tag, "", word)

	return lipgloss.NewStyle().
		Width(width).
		Padding(1, 2).
		Background(headerBG).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#6d5bd0")).
		BorderBackground(headerBG).
		BorderBottom(true).BorderTop(false).BorderLeft(false).BorderRight(false).
		Render(inner)
}

func versionSuffix(v string) string {
	if v == "" {
		return ""
	}
	return "  " + v
}

// tintLine colors a banner row: the "gen" columns pink and the "ifer" columns
// purple, over the shared header background. The row is padded to bannerCols
// display columns so every row aligns and the pattern splices at a fixed
// column.
func tintLine(line string) string {
	runes := []rune(line)
	split := genSplit
	if split > len(runes) {
		split = len(runes)
	}
	genStyle := lipgloss.NewStyle().Foreground(colorGen).Background(headerBG).Bold(true)
	iferStyle := lipgloss.NewStyle().Foreground(colorIfer).Background(headerBG).Bold(true)
	out := genStyle.Render(string(runes[:split])) + iferStyle.Render(string(runes[split:]))

	// Pad to the uniform banner width (box-drawing glyphs are width 1).
	if pad := bannerCols - lipgloss.Width(out); pad > 0 {
		out += lipgloss.NewStyle().Background(headerBG).Render(strings.Repeat(" ", pad))
	}
	return out
}

// patternRow builds one row (rowIdx) of the offset dot lattice, `cols` wide,
// each dot faded pink->purple across the width. Spaces keep the header
// background; only the dots are tinted.
func patternRow(rowIdx, cols int) string {
	if cols <= 0 {
		return ""
	}
	// Odd rows are shifted by dotOffset to form an offset lattice.
	phase := 0
	if rowIdx%2 == 1 {
		phase = dotOffset
	}
	var b strings.Builder
	for c := 0; c < cols; c++ {
		if (c-phase)%dotSpacing == 0 && c >= phase {
			frac := 0.0
			if cols > 1 {
				frac = float64(c) / float64(cols-1)
			}
			b.WriteString(lipgloss.NewStyle().
				Foreground(fadeColor(frac)).
				Background(headerBG).
				Render("•"))
		} else {
			b.WriteString(lipgloss.NewStyle().Background(headerBG).Render(" "))
		}
	}
	return b.String()
}

// fadeColor interpolates the pink->purple pattern fade at position frac (0..1).
func fadeColor(frac float64) lipgloss.Color {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	lerp := func(a, b int) int { return a + int(float64(b-a)*frac+0.5) }
	r := lerp(patternFadeFrom[0], patternFadeTo[0])
	g := lerp(patternFadeFrom[1], patternFadeTo[1])
	bl := lerp(patternFadeFrom[2], patternFadeTo[2])
	return lipgloss.Color(rgbHex(r, g, bl))
}

// rgbHex formats r,g,b as a #rrggbb string.
func rgbHex(r, g, b int) string {
	const hexdigits = "0123456789abcdef"
	buf := []byte{'#', 0, 0, 0, 0, 0, 0}
	vals := [3]int{r, g, b}
	for i, v := range vals {
		buf[1+i*2] = hexdigits[(v>>4)&0xf]
		buf[2+i*2] = hexdigits[v&0xf]
	}
	return string(buf)
}
