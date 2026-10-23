package tui

import (
	"sort"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
)

// favoritesFirst returns models stably partitioned so favorited models come
// first. Within each group the provider's order is preserved, and ids in favs
// that the provider no longer offers are simply never matched. The input slice
// is not modified.
func favoritesFirst(models []provider.Model, favs map[string]struct{}) []provider.Model {
	if len(models) == 0 {
		return nil
	}
	out := make([]provider.Model, 0, len(models))
	for _, m := range models {
		if _, ok := favs[m.ID]; ok {
			out = append(out, m)
		}
	}
	for _, m := range models {
		if _, ok := favs[m.ID]; !ok {
			out = append(out, m)
		}
	}
	return out
}

// favoriteIDs renders a favorite set as a sorted slice for state.json, so the
// file stays stable and diff-friendly regardless of toggle order.
func favoriteIDs(favs map[string]struct{}) []string {
	ids := make([]string, 0, len(favs))
	for id := range favs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil
	}
	return ids
}

// toggleFavorite flips the favorite state of the model with id and persists the
// result to state.json, preserving every other remembered selection. It returns
// whether the model is now a favorite. A write failure is non-fatal: the session
// keeps the new state and the file is rewritten on the next successful save.
func (m *Model) toggleFavorite(id string) bool {
	if id == "" {
		return false
	}
	if _, ok := m.favorites[id]; ok {
		delete(m.favorites, id)
	} else {
		if m.favorites == nil {
			m.favorites = map[string]struct{}{}
		}
		m.favorites[id] = struct{}{}
	}
	s := config.LoadState(m.deps.StatePath)
	s.Favorites = favoriteIDs(m.favorites)
	_ = config.SaveState(m.deps.StatePath, s)
	_, fav := m.favorites[id]
	return fav
}
