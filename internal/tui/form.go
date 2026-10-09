package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

// timeNow is overridable in tests.
var timeNow = time.Now

// priceLabel renders a model's classified price compactly for the picker.
//
//   - loading & not yet priced -> "pricing…"
//   - per image                -> "$0.018/img"
//   - per image, tiered        -> "$0.041/img" (lowest tier)
//   - per token                -> "$/tok"
//   - free                     -> "free"
//   - unknown / fetch failed    -> "" (nothing)
func priceLabel(p provider.Price, priced, loading bool) string {
	if !priced {
		if loading {
			return "pricing…"
		}
		return ""
	}
	switch p.Unit {
	case provider.PriceFree:
		return "free"
	case provider.PricePerImage, provider.PricePerImageTiered:
		return fmt.Sprintf("$%s/img", trimPrice(p.USD))
	case provider.PricePerToken:
		return "$/tok"
	default:
		return ""
	}
}

// trimPrice formats a USD amount compactly (no trailing zeros, up to 4 dp).
func trimPrice(v float64) string {
	s := fmt.Sprintf("%.4f", v)
	// strip trailing zeros, then a trailing dot
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// FormControls describes which generate-form controls are available for a
// given model's capabilities. The view renders exactly the enabled controls.
type FormControls struct {
	ShowAspectRatio bool
	AspectRatios    []string
	ShowSeed        bool
	ShowReference   bool
}

// controlsFor derives the available form controls from a model's capabilities.
func controlsFor(c provider.Capabilities) FormControls {
	return FormControls{
		ShowAspectRatio: len(c.AspectRatios) > 0,
		AspectRatios:    c.AspectRatios,
		ShowSeed:        c.SupportsSeed,
		ShowReference:   c.AcceptsReferenceImages,
	}
}

// preselectModel chooses which model index to highlight on launch: the last
// used model — matched by provider key and model id — if it is still offered,
// otherwise the first model (index 0), or -1 when the list is empty.
func preselectModel(models []taggedModel, lastProviderKey, lastModel string) int {
	if len(models) == 0 {
		return -1
	}
	if lastModel != "" {
		for i, m := range models {
			if m.Model.ID == lastModel && m.ProviderKey == lastProviderKey {
				return i
			}
		}
	}
	return 0
}

// pickAspectRatio chooses the default aspect-ratio selection for a model: the
// last-used ratio when the model still offers it, otherwise the model's first
// offered ratio, or "" when the model offers none.
func pickAspectRatio(offered []string, last string) string {
	if len(offered) == 0 {
		return ""
	}
	if last != "" {
		for _, r := range offered {
			if r == last {
				return last
			}
		}
	}
	return offered[0]
}

// indexOfString returns the index of s in list, or -1 when absent.
func indexOfString(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// modelsStatus builds the picker status line: the merged model count plus a
// note of any offline providers, so an unreachable instance is surfaced
// without blanking the picker.
func modelsStatus(count int, offline map[string]bool) string {
	s := fmt.Sprintf("%d models · / to filter", count)
	if len(offline) > 0 {
		keys := make([]string, 0, len(offline))
		for k := range offline {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		s += fmt.Sprintf(" · offline: %s", strings.Join(keys, ", "))
	}
	return s
}
