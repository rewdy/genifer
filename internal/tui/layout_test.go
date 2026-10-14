package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

func TestNarrowComposeLayout(t *testing.T) {
	for _, width := range []int{12, 20, 24, 40, 60, 80, 120} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			m := composeModel(true, fakePaster{})
			m.models[0].ID = "qwen/qwen-image-3-pro"
			m.models[0].Capabilities.AspectRatios = []string{"1:1", "16:9"}
			m.models[0].Capabilities.SupportsSeed = true
			m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
			updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 50})
			m = updated.(Model)
			for _, prompt := range []string{"", "A long prompt with 日本語 and enough words to wrap across the input."} {
				m.prompt.SetValue(prompt)
				for i, line := range strings.Split(m.View(), "\n") {
					if got := ansi.StringWidth(line); got != width {
						t.Errorf("row %d has width %d, want %d: %q", i, got, width, ansi.Strip(line))
					}
				}
			}
			if width >= 40 {
				want := "Model: qwen/qwen-image-3-pro"
				if !strings.Contains(ansi.Strip(m.View()), want) {
					t.Errorf("model label unexpectedly wrapped: want %q", want)
				}
			}
		})
	}
}

func TestFooterWrapsBetweenShortcuts(t *testing.T) {
	m := composeModel(true, fakePaster{})
	m.width = 40
	foot := ansi.Strip(m.footerView())
	for _, shortcut := range []string{"ctrl+s review", "ctrl+p history list", "ctrl+v paste image"} {
		if !strings.Contains(foot, shortcut) {
			t.Errorf("shortcut split or missing: %q in %q", shortcut, foot)
		}
	}
}
