package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rewdy/genifer/internal/provider"
)

// Compose shows file guidance for a capable model and omits it for
// an incapable one; pills appear once an image is attached.
func TestComposeViewHintAndPills(t *testing.T) {
	capable := ansi.Strip(composeModel(true, fakePaster{}).composeView())
	if !strings.Contains(capable, "Reference file: @/path/to/image") {
		t.Errorf("capable compose view missing file hint; got %q", capable)
	}
	if strings.Contains(capable, "paste image") {
		t.Error("compose duplicates the footer paste shortcut")
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

func TestComposeDetailsFollowPromptAndKeepMutedWraps(t *testing.T) {
	m := composeModel(true, fakePaster{})
	m.width = 40
	m.prompt.SetWidth(m.contentWidth())
	m.prompt.SetValue("MY PROMPT")
	m.models[0].Capabilities.AspectRatios = []string{"1:1"}
	m.models[0].Capabilities.SupportsSeed = true
	m.aspectRatio = "1:1"
	rendered := m.composeView()
	plain := ansi.Strip(rendered)
	promptAt := strings.Index(plain, "MY PROMPT")
	for _, detail := range []string{"Aspect: 1:1", "Seed supported", "References optional", "Reference file:"} {
		if at := strings.Index(plain, detail); at <= promptAt {
			t.Errorf("%q must follow the prompt: %s", detail, plain)
		}
	}
	// Each physical details row must set its own muted foreground, including
	// rows created by wrapping the reference guidance at this narrow width.
	details := strings.SplitN(rendered, "\n\n", 3)[2]
	for _, row := range strings.Split(details, "\n") {
		if !strings.Contains(row, "139;133;160") {
			t.Errorf("details row lost muted foreground: %q", row)
		}
		if ansi.StringWidth(row) > m.contentWidth() {
			t.Errorf("details row exceeds content width: %q", row)
		}
	}
}
