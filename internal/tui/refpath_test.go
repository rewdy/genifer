package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 5.1: the scanner extracts sigil paths and cleans the prompt.
func TestExtractRefPaths(t *testing.T) {
	home, _ := os.UserHomeDir()

	cases := []struct {
		name        string
		prompt      string
		wantPaths   []string
		wantCleaned string
	}{
		{
			name:        "single path",
			prompt:      "a red bird\n@/tmp/ref.png",
			wantPaths:   []string{"/tmp/ref.png"},
			wantCleaned: "a red bird",
		},
		{
			name:        "multiple paths",
			prompt:      "@/a.png\nmake it blue\n@/b.jpg",
			wantPaths:   []string{"/a.png", "/b.jpg"},
			wantCleaned: "make it blue",
		},
		{
			name:        "tilde expansion",
			prompt:      "sketch\n@~/pics/ref.png",
			wantPaths:   []string{filepath.Join(home, "pics/ref.png")},
			wantCleaned: "sketch",
		},
		{
			name:        "no sigil lines",
			prompt:      "just a prompt\nwith two lines",
			wantPaths:   nil,
			wantCleaned: "just a prompt\nwith two lines",
		},
		{
			name:        "sigil amid text with leading spaces",
			prompt:      "line one\n  @/indented.png\nline three",
			wantPaths:   []string{"/indented.png"},
			wantCleaned: "line one\nline three",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			paths, cleaned := extractRefPaths(tc.prompt)
			if strings.Join(paths, "|") != strings.Join(tc.wantPaths, "|") {
				t.Errorf("paths = %v, want %v", paths, tc.wantPaths)
			}
			if cleaned != tc.wantCleaned {
				t.Errorf("cleaned = %q, want %q", cleaned, tc.wantCleaned)
			}
		})
	}
}

// 5.2: submitting a prompt with a valid @path attaches the image and strips the
// sigil line from the prompt.
func TestSubmitAttachesPathImage(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "ref.png")
	if err := os.WriteFile(imgPath, makePNG(t), 0o600); err != nil {
		t.Fatal(err)
	}

	m := composeModel(true, fakePaster{})
	m.prompt.SetValue("a red bird\n@" + imgPath)

	updated, _ := m.submitCompose()
	um := updated.(Model)

	if len(um.refImages) != 1 {
		t.Fatalf("refImages = %d, want 1", len(um.refImages))
	}
	if um.refImages[0].MediaType != "image/png" {
		t.Errorf("MediaType = %q, want image/png", um.refImages[0].MediaType)
	}
	if um.phase != phaseReview {
		t.Errorf("phase = %v, want review", um.phase)
	}
	if strings.Contains(um.prompt.Value(), "@") {
		t.Errorf("sigil line not stripped: %q", um.prompt.Value())
	}
	if strings.TrimSpace(um.prompt.Value()) != "a red bird" {
		t.Errorf("cleaned prompt = %q, want %q", um.prompt.Value(), "a red bird")
	}
}

// 5.3: a non-existent path keeps the user in compose with a non-fatal status.
func TestSubmitMissingPathBlocks(t *testing.T) {
	m := composeModel(true, fakePaster{})
	m.prompt.SetValue("a red bird\n@/no/such/file.png")

	updated, _ := m.submitCompose()
	um := updated.(Model)

	if um.phase != phaseCompose {
		t.Errorf("phase = %v, want compose (submit should be blocked)", um.phase)
	}
	if len(um.refImages) != 0 {
		t.Errorf("refImages = %d, want 0", len(um.refImages))
	}
	if um.status == "" || um.status == "Review your prompt" {
		t.Errorf("expected a non-fatal error status, got %q", um.status)
	}
	// The sigil line is preserved so the user can fix it.
	if !strings.Contains(um.prompt.Value(), "@/no/such/file.png") {
		t.Error("sigil line should be preserved on failure")
	}
}

// 5.3: a path to a non-image file is rejected and blocks the submit.
func TestSubmitNonImagePathBlocks(t *testing.T) {
	dir := t.TempDir()
	txtPath := filepath.Join(dir, "notimage.txt")
	if err := os.WriteFile(txtPath, []byte("hello, not an image"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := composeModel(true, fakePaster{})
	m.prompt.SetValue("a red bird\n@" + txtPath)

	updated, _ := m.submitCompose()
	um := updated.(Model)

	if um.phase != phaseCompose {
		t.Errorf("phase = %v, want compose", um.phase)
	}
	if len(um.refImages) != 0 {
		t.Errorf("refImages = %d, want 0", len(um.refImages))
	}
}

// 5.4: a path-attached image renders as a pill and can be removed uniformly.
func TestPathImagePillAndRemoval(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "ref.png")
	if err := os.WriteFile(imgPath, makePNG(t), 0o600); err != nil {
		t.Fatal(err)
	}

	m := composeModel(true, fakePaster{})
	m.prompt.SetValue("a red bird\n@" + imgPath)
	updated, _ := m.submitCompose()
	um := updated.(Model)

	// Back to compose to inspect the pill rendering.
	um.phase = phaseCompose
	rendered := um.composeView()
	if !strings.Contains(rendered, "Image 1") {
		t.Errorf("path-attached image should render a pill; got %q", rendered)
	}

	// Removal applies to the path-attached image the same as a pasted one.
	up, _ := um.handleComposeKey(key("ctrl+r"))
	um2 := up.(Model)
	if len(um2.refImages) != 0 {
		t.Errorf("refImages = %d, want 0 after removal", len(um2.refImages))
	}
}
