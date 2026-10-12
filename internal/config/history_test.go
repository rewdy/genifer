package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "history-cache.json")
	seed := int64(42)
	in := HistoryCache{Entries: []HistoryEntry{
		{Prompt: "a cat", Model: "m", AspectRatio: "1:1", Seed: &seed, Path: "/tmp/a.png", CreatedAt: time.Now()},
	}}
	if err := SaveHistory(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := LoadHistory(path)
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(got.Entries))
	}
	e := got.Entries[0]
	if e.Prompt != "a cat" || e.Model != "m" || e.AspectRatio != "1:1" || e.Path != "/tmp/a.png" {
		t.Errorf("round-trip mismatch: %+v", e)
	}
	if e.Seed == nil || *e.Seed != 42 {
		t.Errorf("seed = %v, want 42", e.Seed)
	}
}

// 1.1: missing, unparsable, and wrong-version files all yield an empty history.
func TestHistoryMissingUnparsableWrongVersion(t *testing.T) {
	// Missing.
	if got := LoadHistory(filepath.Join(t.TempDir(), "absent.json")); len(got.Entries) != 0 {
		t.Errorf("missing history should be empty, got %+v", got.Entries)
	}

	// Unparsable.
	bad := filepath.Join(t.TempDir(), "history-cache.json")
	os.WriteFile(bad, []byte("{garbage"), 0o600)
	if got := LoadHistory(bad); len(got.Entries) != 0 {
		t.Errorf("unparsable history should be empty, got %+v", got.Entries)
	}

	// Wrong version.
	wrong := filepath.Join(t.TempDir(), "history-cache.json")
	os.WriteFile(wrong, []byte(`{"version":999,"entries":[{"prompt":"x","model":"m"}]}`), 0o600)
	if got := LoadHistory(wrong); len(got.Entries) != 0 {
		t.Errorf("wrong-version history should be empty, got %+v", got.Entries)
	}
}

// 1.2: Append prepends newest-first and never drops previously recorded entries.
func TestHistoryAppendRetainsAll(t *testing.T) {
	var c HistoryCache
	for i := 0; i < 200; i++ {
		c = c.Append(HistoryEntry{Prompt: string(rune('a' + i%26)), Model: "m"})
	}
	if len(c.Entries) != 200 {
		t.Fatalf("entries = %d, want 200 (nothing pruned)", len(c.Entries))
	}
	// Newest is the last appended ("a"+199%26) at index 0; the very first
	// appended entry must still be present at the tail.
	last := c.Append(HistoryEntry{Prompt: "newest", Model: "m"})
	if last.Entries[0].Prompt != "newest" {
		t.Errorf("newest entry not at front: %q", last.Entries[0].Prompt)
	}
	if len(last.Entries) != 201 {
		t.Errorf("entries = %d, want 201", len(last.Entries))
	}
}
