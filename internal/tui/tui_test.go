package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func init() {
	// Force a truecolor profile so lipgloss emits color escapes deterministically
	// in tests (otherwise a non-TTY test environment strips all color).
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestPreselectModel(t *testing.T) {
	models := []provider.Model{
		{ID: "a/one"}, {ID: "b/two"}, {ID: "c/three"},
	}
	// Last model present -> its index.
	if got := preselectModel(models, "b/two"); got != 1 {
		t.Errorf("preselect(b/two) = %d, want 1", got)
	}
	// Last model absent -> fall back to first.
	if got := preselectModel(models, "x/gone"); got != 0 {
		t.Errorf("preselect(absent) = %d, want 0", got)
	}
	// No remembered model -> first.
	if got := preselectModel(models, ""); got != 0 {
		t.Errorf("preselect(empty) = %d, want 0", got)
	}
	// Empty list -> -1.
	if got := preselectModel(nil, "a/one"); got != -1 {
		t.Errorf("preselect(empty list) = %d, want -1", got)
	}
}

func TestPickAspectRatio(t *testing.T) {
	offered := []string{"1:1", "16:9", "9:16"}
	// Last present -> last.
	if got := pickAspectRatio(offered, "16:9"); got != "16:9" {
		t.Errorf("pick(16:9) = %q, want 16:9", got)
	}
	// Last absent -> first offered.
	if got := pickAspectRatio(offered, "21:9"); got != "1:1" {
		t.Errorf("pick(absent) = %q, want 1:1", got)
	}
	// No remembered ratio -> first offered.
	if got := pickAspectRatio(offered, ""); got != "1:1" {
		t.Errorf("pick(empty) = %q, want 1:1", got)
	}
	// No offered ratios -> "".
	if got := pickAspectRatio(nil, "16:9"); got != "" {
		t.Errorf("pick(none offered) = %q, want empty", got)
	}
}

func TestControlsForRefCapable(t *testing.T) {
	caps := provider.Capabilities{
		AcceptsReferenceImages: true,
		AspectRatios:           []string{"1:1", "16:9"},
		SupportsSeed:           true,
	}
	c := controlsFor(caps)
	if !c.ShowReference {
		t.Error("ShowReference = false, want true")
	}
	if !c.ShowAspectRatio || len(c.AspectRatios) != 2 {
		t.Errorf("aspect controls wrong: %+v", c)
	}
	if !c.ShowSeed {
		t.Error("ShowSeed = false, want true")
	}
}

func TestControlsForRefIncapable(t *testing.T) {
	c := controlsFor(provider.Capabilities{})
	if c.ShowReference {
		t.Error("ShowReference = true, want false")
	}
	if c.ShowAspectRatio {
		t.Error("ShowAspectRatio = true, want false")
	}
	if c.ShowSeed {
		t.Error("ShowSeed = true, want false")
	}
}

func TestPriceLabel(t *testing.T) {
	cases := []struct {
		name    string
		p       provider.Price
		priced  bool
		loading bool
		want    string
	}{
		{"loading", provider.Price{}, false, true, "pricing…"},
		{"unresolved not loading", provider.Price{}, false, false, ""},
		{"free", provider.Price{Unit: provider.PriceFree}, true, false, "free"},
		{"per image", provider.Price{Unit: provider.PricePerImage, USD: 0.018}, true, false, "$0.018/img"},
		{"tiered", provider.Price{Unit: provider.PricePerImageTiered, USD: 0.041}, true, false, "$0.041/img"},
		{"per token", provider.Price{Unit: provider.PricePerToken}, true, false, "$/tok"},
		{"unknown", provider.Price{Unit: provider.PriceUnknown}, true, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := priceLabel(tc.p, tc.priced, tc.loading); got != tc.want {
				t.Errorf("priceLabel = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestModelDelegateRendersMutedDetail(t *testing.T) {
	it := modelItem{
		m:      provider.Model{Name: "ByteDance Seed: Seedream 5.0 Flash", Capabilities: provider.Capabilities{AcceptsReferenceImages: true, AspectRatios: []string{"1:1"}, SupportsSeed: true}},
		price:  provider.Price{Unit: provider.PricePerImage, USD: 0.018},
		priced: true,
	}
	other := modelItem{m: provider.Model{Name: "Other Model"}}

	// A list whose cursor is on index 0. Rendering index 1 is the UNSELECTED
	// case; rendering index 0 is the SELECTED case.
	l := list.New([]list.Item{it, other}, modelDelegate{}, 80, 4)
	l.Select(0)

	var selBuf, unselBuf bytes.Buffer
	modelDelegate{}.Render(&selBuf, l, 0, it)   // selected
	modelDelegate{}.Render(&unselBuf, l, 1, it) // unselected (same item, different index)
	sel, out := selBuf.String(), unselBuf.String()

	// Plain-text content includes name, price, and caps; single line.
	plain := ansi.Strip(out)
	for _, want := range []string{"Seedream 5.0 Flash", "$0.018/img", "img2img"} {
		if !strings.Contains(plain, want) {
			t.Errorf("row missing %q; got %q", want, plain)
		}
	}
	if strings.Contains(out, "\n") {
		t.Errorf("delegate row must be single-line, got %q", out)
	}

	// Colors are truecolor escapes (decimal RGB):
	//   muted #8b85a0 -> 139;133;160 ; selected pink #ff5fd2 -> 255;95;210
	const mutedRGB = "139;133;160"
	const pinkRGB = "255;95;210"

	if sel == out {
		t.Error("selected row should differ from unselected (name highlight)")
	}
	// Both states carry the muted detail color.
	if !strings.Contains(sel, mutedRGB) || !strings.Contains(out, mutedRGB) {
		t.Errorf("detail should use the muted color %q in both states", mutedRGB)
	}
	// Selected highlights the name in pink; unselected does not.
	if !strings.Contains(sel, pinkRGB) {
		t.Errorf("selected row should highlight the name in pink; got %q", sel)
	}
	if strings.Contains(out, pinkRGB) {
		t.Errorf("unselected row should not use the pink name color; got %q", out)
	}
}

func TestTrimPrice(t *testing.T) {
	cases := map[float64]string{
		0.018:  "0.018",
		0.0410: "0.041",
		0.6:    "0.6",
		1.0:    "1",
	}
	for in, want := range cases {
		if got := trimPrice(in); got != want {
			t.Errorf("trimPrice(%v) = %q, want %q", in, got, want)
		}
	}
}
