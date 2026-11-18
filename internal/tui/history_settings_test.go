package tui

import (
	"testing"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/gen"
	"github.com/rewdy/genifer/internal/provider"
)

func TestRecordHistoryCapturesSettings(t *testing.T) {
	m := historyModel(t, provider.Capabilities{SupportsAnimatedGIF: true}, config.Default())
	m.lastDraft = gen.Draft{
		Prompt:       "a spinning logo",
		Model:        "test/model",
		OutputFormat: "gif",
	}
	m.outcome = gen.Outcome{Path: "/tmp/a.gif"}
	cmd := m.recordHistoryCmd()
	if cmd == nil {
		t.Fatal("expected a history write command")
	}
	cmd()

	entries := config.LoadHistory(m.deps.HistoryPath).Entries
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got := entries[0].OutputFormat; got != "gif" {
		t.Errorf("output format = %q, want gif", got)
	}

	transparent := historyModel(t, provider.Capabilities{SupportsTransparentBackground: true}, config.Default())
	transparent.lastDraft = gen.Draft{Prompt: "cutout", Background: "transparent", OutputFormat: "png"}
	transparent.outcome = gen.Outcome{Path: "/tmp/b.png"}
	if cmd := transparent.recordHistoryCmd(); cmd != nil {
		cmd()
	}
	entries = config.LoadHistory(transparent.deps.HistoryPath).Entries
	if got := entries[0].Background; got != "transparent" {
		t.Errorf("background = %q, want transparent", got)
	}
	if got := entries[0].OutputFormat; got != "png" {
		t.Errorf("output format = %q, want png", got)
	}
}

func TestHistoryReuseRestoresSettings(t *testing.T) {
	both := provider.Capabilities{SupportsAnimatedGIF: true, SupportsTransparentBackground: true}
	withGIF := config.HistoryEntry{Prompt: "spinner", OutputFormat: "gif"}

	m := historyModel(t, both, config.Default()).reuseHistoryEntry(withGIF)
	if !m.animatedGIF || m.transparentBackground {
		t.Fatalf("gif restore: animated=%v transparent=%v", m.animatedGIF, m.transparentBackground)
	}
	if d := m.composeDraft(); d.OutputFormat != "gif" || d.Background != "" {
		t.Errorf("draft format/background = %q/%q, want gif/empty", d.OutputFormat, d.Background)
	}

	withTransparency := config.HistoryEntry{Prompt: "cutout", Background: "transparent", OutputFormat: "png"}
	m = historyModel(t, both, config.Default()).reuseHistoryEntry(withTransparency)
	if !m.transparentBackground || m.animatedGIF {
		t.Fatalf("transparency restore: animated=%v transparent=%v", m.animatedGIF, m.transparentBackground)
	}
}

func TestHistoryReuseGatesUnsupportedSettings(t *testing.T) {
	// GIF wins over transparency when both were recorded, but is ignored when
	// the active model cannot produce GIF output.
	entry := config.HistoryEntry{Prompt: "mixed", Background: "transparent", OutputFormat: "gif"}

	m := historyModel(t, provider.Capabilities{SupportsTransparentBackground: true}, config.Default())
	m = m.reuseHistoryEntry(entry)
	if m.animatedGIF || !m.transparentBackground {
		t.Fatalf("unsupported gif should fall back to transparency: animated=%v transparent=%v", m.animatedGIF, m.transparentBackground)
	}

	m = historyModel(t, provider.Capabilities{SupportsAnimatedGIF: true, SupportsTransparentBackground: true}, config.Default())
	m = m.reuseHistoryEntry(entry)
	if !m.animatedGIF || m.transparentBackground {
		t.Fatalf("gif should win over transparency: animated=%v transparent=%v", m.animatedGIF, m.transparentBackground)
	}

	unsupported := historyModel(t, provider.Capabilities{}, config.Default()).reuseHistoryEntry(entry)
	if unsupported.animatedGIF || unsupported.transparentBackground {
		t.Fatal("unsupported settings should be ignored")
	}
}

func TestHistoryOldEntryKeepsCurrentSettings(t *testing.T) {
	m := historyModel(t, provider.Capabilities{SupportsTransparentBackground: true}, config.Default())
	m.transparentBackground = true
	m = m.reuseHistoryEntry(config.HistoryEntry{Prompt: "old"})
	if !m.transparentBackground {
		t.Error("an entry without recorded settings should leave the current default in place")
	}
}

func TestRecallRestoresSettingsAndDraftSnapshot(t *testing.T) {
	caps := provider.Capabilities{SupportsAnimatedGIF: true, SupportsTransparentBackground: true}
	m := historyModel(t, caps, config.Default())
	m.phase = phaseCompose
	m.prompt.SetValue("my draft")
	m.prompt.CursorStart()
	m.transparentBackground = true

	cache := config.HistoryCache{}.Append(config.HistoryEntry{Prompt: "remembered gif", OutputFormat: "gif"})
	if err := config.SaveHistory(m.deps.HistoryPath, cache); err != nil {
		t.Fatal(err)
	}

	up, _ := m.handleComposeKey(upKey())
	m = up.(Model)
	if m.prompt.Value() != "remembered gif" || !m.animatedGIF || m.transparentBackground {
		t.Fatalf("recall: prompt=%q animated=%v transparent=%v", m.prompt.Value(), m.animatedGIF, m.transparentBackground)
	}

	down, _ := m.handleComposeKey(downKey())
	m = down.(Model)
	if m.prompt.Value() != "my draft" {
		t.Errorf("draft prompt = %q, want my draft", m.prompt.Value())
	}
	if m.animatedGIF || !m.transparentBackground {
		t.Errorf("draft settings: animated=%v transparent=%v, want false/true", m.animatedGIF, m.transparentBackground)
	}
}
