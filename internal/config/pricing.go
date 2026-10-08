package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rewdy/genifer/internal/provider"
)

// pricingCacheVersion is bumped when the on-disk format changes; a mismatch
// makes the cache ignored and re-fetched rather than mis-parsed.
const pricingCacheVersion = 1

// PricingTTL is how long a cached pricing snapshot is considered fresh.
const PricingTTL = 24 * time.Hour

// PricingCache is the app-owned, on-disk cache of per-model classified prices.
type PricingCache struct {
	Version   int                       `json:"version"`
	FetchedAt time.Time                 `json:"fetched_at"`
	Prices    map[string]provider.Price `json:"prices"`
}

// PricingCachePath returns the full path to pricing-cache.json in the config
// dir.
func PricingCachePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pricing-cache.json"), nil
}

// LoadPricingCache reads the cache. A missing, unparsable, or wrong-version
// file yields an empty, stale cache rather than an error, so the app always
// starts cleanly and simply refetches.
func LoadPricingCache(path string) PricingCache {
	empty := PricingCache{Version: pricingCacheVersion, Prices: map[string]provider.Price{}}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var c PricingCache
	if err := json.Unmarshal(data, &c); err != nil {
		return empty
	}
	if c.Version != pricingCacheVersion || c.Prices == nil {
		return empty
	}
	return c
}

// Fresh reports whether the cache was fetched within the TTL of now.
func (c PricingCache) Fresh(now time.Time) bool {
	if c.FetchedAt.IsZero() {
		return false
	}
	return now.Sub(c.FetchedAt) < PricingTTL
}

// SavePricingCache writes the cache, creating the config dir if needed.
func SavePricingCache(path string, c PricingCache) error {
	c.Version = pricingCacheVersion
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: creating pricing cache dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encoding pricing cache: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: writing %s: %w", path, err)
	}
	return nil
}
