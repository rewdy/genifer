package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderHeaderNonEmpty(t *testing.T) {
	out := RenderHeader(100, "v0.1.0")
	if strings.TrimSpace(out) == "" {
		t.Fatal("header rendered empty")
	}
	// Should contain the version suffix text somewhere.
	if !strings.Contains(out, "v0.1.0") {
		t.Error("header missing version")
	}
}

func TestRenderHeaderZeroWidth(t *testing.T) {
	// Must not panic on degenerate width.
	_ = RenderHeader(0, "")
}

func TestBannerSplitColors(t *testing.T) {
	// Every banner row must be at least genSplit wide so the pink/purple split
	// lands inside the row, and tintLine must emit both colors.
	for i, line := range bannerLines {
		if w := len([]rune(line)); w < genSplit {
			t.Errorf("bannerLines[%d] width %d is narrower than genSplit %d", i, w, genSplit)
		}
	}

	// Force truecolor so lipgloss emits deterministic escapes.
	out := tintLine(bannerLines[0])
	const pinkRGB = "255;95;210"   // #ff5fd2
	const purpleRGB = "154;89;230" // #9a5ae6 (lipgloss truecolor rounding)
	if !strings.Contains(out, pinkRGB) {
		t.Errorf("banner row should contain pink (gen) color %q; got %q", pinkRGB, out)
	}
	if !strings.Contains(out, purpleRGB) {
		t.Errorf("banner row should contain purple (ifer) color %q; got %q", purpleRGB, out)
	}
}

func TestHeaderPatternNoOverflow(t *testing.T) {
	// The decorative dot pattern must fill to the edge without wrapping any
	// row past the given width, and must always produce exactly 3 pattern rows
	// (one per banner row), at every terminal width.
	for _, w := range []int{40, 60, 80, 100, 120, 200} {
		out := RenderHeader(w, "v0.1.0")
		stripped := ansi.Strip(out)
		for _, line := range strings.Split(out, "\n") {
			if lw := ansi.StringWidth(line); lw > w {
				t.Errorf("width %d: line overflows to %d cols: %q", w, lw, ansi.Strip(line))
			}
		}
		dotRows := 0
		for _, line := range strings.Split(stripped, "\n") {
			if strings.Contains(line, "•") {
				dotRows++
			}
		}
		if dotRows != 3 {
			t.Errorf("width %d: expected 3 pattern rows, got %d", w, dotRows)
		}
	}
}

func TestPatternRowOffsetLattice(t *testing.T) {
	// Even rows start a dot at column 0; odd rows are shifted by dotOffset.
	even := ansi.Strip(patternRow(0, 12))
	odd := ansi.Strip(patternRow(1, 12))
	if []rune(even)[0] != '•' {
		t.Errorf("even row should start with a dot: %q", even)
	}
	if []rune(odd)[0] == '•' {
		t.Errorf("odd row should be offset (not start with a dot): %q", odd)
	}
	if []rune(odd)[dotOffset] != '•' {
		t.Errorf("odd row should have a dot at column %d: %q", dotOffset, odd)
	}
}
