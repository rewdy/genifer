package tui

import (
	"testing"

	"github.com/rewdy/genifer/internal/provider"
)

func TestRefImageAppendRemove(t *testing.T) {
	var m Model

	if m.removeLastRefImage() {
		t.Error("removeLastRefImage on empty set should report false")
	}

	m.addRefImage(provider.ReferenceImage{MediaType: "image/png"})
	m.addRefImage(provider.ReferenceImage{MediaType: "image/jpeg"})
	if len(m.refImages) != 2 {
		t.Fatalf("len = %d, want 2", len(m.refImages))
	}

	if !m.removeLastRefImage() {
		t.Error("removeLastRefImage should report true when images exist")
	}
	if len(m.refImages) != 1 {
		t.Fatalf("len = %d, want 1", len(m.refImages))
	}
	// The most-recent (jpeg) was removed; png remains.
	if m.refImages[0].MediaType != "image/png" {
		t.Errorf("remaining MediaType = %q, want image/png", m.refImages[0].MediaType)
	}

	if !m.removeLastRefImage() {
		t.Error("removeLastRefImage should remove the last image")
	}
	if len(m.refImages) != 0 {
		t.Errorf("len = %d, want 0 (back to no-image state)", len(m.refImages))
	}
}
