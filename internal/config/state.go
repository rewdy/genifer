package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// State holds app-owned, last-used selections. It is written by genifer and
// must never be mixed into config.yaml.
type State struct {
	// LastModel is the id of the most recently selected model.
	LastModel string `json:"last_model,omitempty"`
	// LastAspectRatio is the most recently generated aspect ratio (e.g.
	// "16:9"). Preferred as the default when a model offers it.
	LastAspectRatio string `json:"last_aspect_ratio,omitempty"`
	// LastTransparentBackground is true when the user last requested a transparent
	// background. Preferred as the default when a model offers it.
	LastTransparentBackground bool `json:"last_transparent_background,omitempty"`
	// Favorites holds the ids of models the user marked as favorites. Order is
	// not meaningful: display order comes from the provider's list. Ids for
	// models the provider no longer offers are kept so they light up again if the
	// model returns.
	Favorites []string `json:"favorites,omitempty"`
}

// FavoriteSet returns the favorites as a membership set, ignoring empty ids
// and collapsing duplicates.
func (s State) FavoriteSet() map[string]struct{} {
	set := make(map[string]struct{}, len(s.Favorites))
	for _, id := range s.Favorites {
		if id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

// StatePath returns the full path to state.json within the config dir.
func StatePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// LoadState reads state.json. A missing or unparsable file is not an error:
// it yields a zero State so the app starts cleanly and overwrites it on the
// next Save.
func LoadState(path string) State {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}
	}
	return s
}

// SaveState writes state.json, creating the config directory if needed. It
// writes only to the given path and never touches config.yaml.
func SaveState(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: creating state dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encoding state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("config: writing %s: %w", path, err)
	}
	return nil
}
