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
