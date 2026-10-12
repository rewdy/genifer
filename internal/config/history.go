package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// historyCacheVersion is bumped when the on-disk format changes; a mismatch
// makes the history ignored (treated as empty) rather than mis-parsed.
const historyCacheVersion = 1

// HistoryEntry is a record of one successful generation, enough to recall and
// reuse the prompt and its recoverable parameters later.
type HistoryEntry struct {
	Prompt      string    `json:"prompt"`
	Model       string    `json:"model"`
	AspectRatio string    `json:"aspect_ratio,omitempty"`
	Seed        *int64    `json:"seed,omitempty"`
	Path        string    `json:"path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// HistoryCache is the app-owned, on-disk record of past generations. It retains
// every entry: records are never pruned or deleted by the app. Entries are
// stored newest-first so the history view renders them directly.
type HistoryCache struct {
	Version int            `json:"version"`
	Entries []HistoryEntry `json:"entries"`
}

// HistoryPath returns the full path to history-cache.json in the config dir.
func HistoryPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history-cache.json"), nil
}

// LoadHistory reads the history file. A missing, unparsable, or wrong-version
// file yields an empty history rather than an error, so the app always starts
// cleanly.
func LoadHistory(path string) HistoryCache {
	empty := HistoryCache{Version: historyCacheVersion}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var c HistoryCache
	if err := json.Unmarshal(data, &c); err != nil {
		return empty
	}
	if c.Version != historyCacheVersion {
		return empty
	}
	return c
}

// SaveHistory writes the history file, creating the config dir if needed.
func SaveHistory(path string, c HistoryCache) error {
	c.Version = historyCacheVersion
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: creating history dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encoding history: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: writing %s: %w", path, err)
	}
	return nil
}

// Append returns a copy of the cache with entry prepended as the newest. Every
// prior entry is retained — history is never truncated or pruned.
func (c HistoryCache) Append(entry HistoryEntry) HistoryCache {
	entries := make([]HistoryEntry, 0, len(c.Entries)+1)
	entries = append(entries, entry)
	entries = append(entries, c.Entries...)
	return HistoryCache{Version: historyCacheVersion, Entries: entries}
}
