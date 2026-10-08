package tui

import (
	"strings"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
	"github.com/charmbracelet/x/ansi"
)

// 3.4: compose view shows the paste hint for a capable model and omits it for
// an incapable one; pills appear once an image is attached.
func TestComposeViewHintAndPills(t *testing.T) {
	capable := ansi.Strip(composeModel(true, fakePaster{}).composeView())
	if !strings.Contains(capable, "paste image: ctrl+v") {
		t.Errorf("capable compose view missing paste hint; got %q", capable)
	}
	if strings.Contains(capable, "Image 1") {
		t.Error("no pill should show before any image is attached")
	}

	incapable := ansi.Strip(composeModel(false, fakePaster{}).composeView())
	if strings.Contains(incapable, "paste image") {
		t.Errorf("incapable compose view should not show a paste hint; got %q", incapable)
	}

	withImg := composeModel(true, fakePaster{})
	withImg.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	withImg.addRefImage(provider.ReferenceImage{MediaType: "image/jpeg"})
	rendered := ansi.Strip(withImg.composeView())
	for _, want := range []string{"Image 1", "Image 2", "2 attached"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("compose view missing %q; got %q", want, rendered)
		}
	}
}

// 3.5: footer shows paste and (when attached) remove hints for a capable model
// in the compose phase.
func TestFooterComposeHints(t *testing.T) {
	m := composeModel(true, fakePaster{})
	foot := ansi.Strip(m.footerView())
	if !strings.Contains(foot, "ctrl+v") || !strings.Contains(foot, "paste image") {
		t.Errorf("footer missing paste hint; got %q", foot)
	}
	if strings.Contains(foot, "remove image") {
		t.Error("remove hint should not show with no image attached")
	}

	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	foot = ansi.Strip(m.footerView())
	if !strings.Contains(foot, "remove image") {
		t.Errorf("footer missing remove hint when an image is attached; got %q", foot)
	}

	incapable := ansi.Strip(composeModel(false, fakePaster{}).footerView())
	if strings.Contains(incapable, "paste image") {
		t.Errorf("incapable footer should not show paste hint; got %q", incapable)
	}
}
