package tui

import (
	"fmt"
	"strings"

	"github.com/rewdy/genifer/internal/provider"
)

// pickerFilters are the capability filters available in the model picker.
// Each one excludes models that do not advertise the corresponding capability.
type pickerFilters struct {
	Transparent bool
	AnimatedGIF bool
	References  bool
	Seed        bool
	SVG         bool
}

func (f pickerFilters) active() bool {
	return f.Transparent || f.AnimatedGIF || f.References || f.Seed || f.SVG
}

// matches reports whether a model satisfies every active filter. A model that
// does not advertise a filtered capability is treated as unsupported.
func (f pickerFilters) matches(c provider.Capabilities) bool {
	if f.Transparent && !c.SupportsTransparentBackground {
		return false
	}
	if f.AnimatedGIF && !c.SupportsAnimatedGIF {
		return false
	}
	if f.References && !c.AcceptsReferenceImages {
		return false
	}
	if f.Seed && !c.SupportsSeed {
		return false
	}
	if f.SVG && indexOfString(c.OutputFormats, "svg") < 0 {
		return false
	}
	return true
}

// labels names the active filters for the status line.
func (f pickerFilters) labels() []string {
	var out []string
	if f.Transparent {
		out = append(out, "transparent")
	}
	if f.AnimatedGIF {
		out = append(out, "gif")
	}
	if f.References {
		out = append(out, "refs")
	}
	if f.Seed {
		out = append(out, "seed")
	}
	if f.SVG {
		out = append(out, "svg")
	}
	return out
}

func (f pickerFilters) summary() string {
	return strings.Join(f.labels(), ", ")
}

// pickerStatus summarizes the current picker selection set for the status line.
func (m Model) pickerStatus() string {
	if !m.pickerFilters.active() {
		return fmt.Sprintf("%d models · / to filter", len(m.models))
	}
	count := len(m.picker.VisibleItems())
	if count == 0 {
		return "No models match the active filters · 0 to clear"
	}
	return fmt.Sprintf("%d models · filters: %s", count, m.pickerFilters.summary())
}

// toggle applies a picker filter key. Keys 1-5 toggle individual filters and 0
// clears them all. It reports whether the filter set changed.
func (f *pickerFilters) toggle(key string) bool {
	switch key {
	case "1":
		f.Transparent = !f.Transparent
	case "2":
		f.AnimatedGIF = !f.AnimatedGIF
	case "3":
		f.References = !f.References
	case "4":
		f.Seed = !f.Seed
	case "5":
		f.SVG = !f.SVG
	case "0":
		if !f.active() {
			return false
		}
		*f = pickerFilters{}
	default:
		return false
	}
	return true
}
