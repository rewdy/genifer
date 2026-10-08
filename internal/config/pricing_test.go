package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

func TestPricingCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "pricing-cache.json")
	in := PricingCache{
		FetchedAt: time.Now(),
		Prices: map[string]provider.Price{
			"a/b": {Unit: provider.PricePerImage, USD: 0.018},
			"c/d": {Unit: provider.PricePerToken},
		},
	}
	if err := SavePricingCache(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := LoadPricingCache(path)
	if len(got.Prices) != 2 || got.Prices["a/b"].USD != 0.018 {
		t.Errorf("round-trip mismatch: %+v", got.Prices)
	}
}

func TestPricingCacheFreshness(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	fresh := PricingCache{FetchedAt: now.Add(-1 * time.Hour)}
	if !fresh.Fresh(now) {
		t.Error("1h-old cache should be fresh")
	}
	stale := PricingCache{FetchedAt: now.Add(-25 * time.Hour)}
	if stale.Fresh(now) {
		t.Error("25h-old cache should be stale")
	}
	var zero PricingCache
	if zero.Fresh(now) {
		t.Error("zero-time cache should be stale")
	}
}

func TestPricingCacheMissingOrInvalid(t *testing.T) {
	// Missing file -> empty, stale.
	got := LoadPricingCache(filepath.Join(t.TempDir(), "absent.json"))
	if got.Prices == nil || len(got.Prices) != 0 || got.Fresh(time.Now()) {
		t.Errorf("missing cache should be empty+stale, got %+v", got)
	}

	// Invalid JSON -> empty, stale.
	path := filepath.Join(t.TempDir(), "pricing-cache.json")
	os.WriteFile(path, []byte("{garbage"), 0o600)
	got = LoadPricingCache(path)
	if got.Prices == nil || len(got.Prices) != 0 {
		t.Errorf("invalid cache should be empty, got %+v", got)
	}
}

func TestPricingCacheWrongVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pricing-cache.json")
	// version 999 should be rejected
	os.WriteFile(path, []byte(`{"version":999,"fetched_at":"2026-10-07T12:00:00Z","prices":{"a/b":{"Unit":2,"USD":0.01}}}`), 0o600)
	got := LoadPricingCache(path)
	if len(got.Prices) != 0 {
		t.Errorf("wrong-version cache should be ignored, got %+v", got.Prices)
	}
}
