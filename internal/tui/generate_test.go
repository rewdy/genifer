package tui

import (
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
	"github.com/charmbracelet/x/ansi"
)

// 4.1: the composed draft carries the attached reference images and prompt.
func TestComposeDraftCarriesRefImages(t *testing.T) {
	m := composeModel(true, fakePaster{})
	m.prompt.SetValue("a red bird")
	m.addRefImage(provider.ReferenceImage{Data: []byte("a"), MediaType: "image/png"})
	m.addRefImage(provider.ReferenceImage{Data: []byte("b"), MediaType: "image/jpeg"})

	d := m.composeDraft()
	if d.Model != "test/model" {
		t.Errorf("Model = %q, want test/model", d.Model)
	}
	if len(d.ReferenceImages) != 2 {
		t.Fatalf("ReferenceImages = %d, want 2", len(d.ReferenceImages))
	}

	req, err := d.Request()
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if len(req.ReferenceImages) != 2 {
		t.Errorf("request ReferenceImages = %d, want 2", len(req.ReferenceImages))
	}
}

// 4.2: the review view shows the attached count when images are present, and
// nothing when none are.
func TestReviewViewShowsCount(t *testing.T) {
	m := composeModel(true, fakePaster{})
	m.prompt.SetValue("a red bird")
	m.phase = phaseReview

	none := ansi.Strip(m.reviewView())
	if strings.Contains(none, "reference image(s) attached") {
		t.Error("review view should not mention images when none attached")
	}

	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	withImg := ansi.Strip(m.reviewView())
	if !strings.Contains(withImg, "1 reference image(s) attached") {
		t.Errorf("review view missing count; got %q", withImg)
	}
}

// Review box bounds the prompt to the terminal width: a long prompt wraps
// inside the border, no rendered line exceeds the box's outer width, and the
// box spans more than one line (top/content/bottom) so the border encloses the
// wrapped text.
func TestReviewViewWrapsLongPrompt(t *testing.T) {
	const termWidth = 60
	m := composeModel(false, fakePaster{})
	m.width = termWidth
	m.phase = phaseReview
	m.prompt.SetValue(strings.Repeat("make a really cool image of a native american with a headdress ", 4))

	rendered := m.reviewView()
	// reviewView bounds the content box to inner = max(1, (width-4)-4); the
	// rounded border adds 2 columns, so every box row renders at inner+2 and
	// that must not exceed the available width (width-4).
	avail := termWidth - 4
	inner := avail - 4
	boxWidth := inner + 2

	lines := strings.Split(rendered, "\n")
	var boxLines, maxWidth int
	for _, ln := range lines {
		w := ansi.StringWidth(ln)
		if w > maxWidth {
			maxWidth = w
		}
		// Border + content rows all render at the full box width; count them to
		// confirm the box encloses multiple wrapped content rows.
		if w == boxWidth {
			boxLines++
		}
	}
	if maxWidth > avail {
		t.Errorf("a rendered line is %d wide, exceeds available width %d", maxWidth, avail)
	}
	if boxLines < 3 {
		t.Errorf("expected the box to span its full width across top, content, and bottom rows (got %d full-width rows)", boxLines)
	}
}

// Review box reflows to a narrower terminal width.
func TestReviewViewReflowsOnWidth(t *testing.T) {
	m := composeModel(false, fakePaster{})
	m.phase = phaseReview
	m.prompt.SetValue(strings.Repeat("wide prompt text ", 6))

	widthOf := func(w int) int {
		m.width = w
		var maxW int
		for _, ln := range strings.Split(m.reviewView(), "\n") {
			if x := ansi.StringWidth(ln); x > maxW {
				maxW = x
			}
		}
		return maxW
	}
	wide := widthOf(80)
	narrow := widthOf(40)
	if narrow >= wide {
		t.Errorf("box did not reflow narrower: narrow=%d wide=%d", narrow, wide)
	}
	if wide > 80-4 || narrow > 40-4 {
		t.Errorf("box exceeded outer width: wide=%d (max 76), narrow=%d (max 36)", wide, narrow)
	}
}
