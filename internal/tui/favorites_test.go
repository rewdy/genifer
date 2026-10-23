package tui

import (
	"reflect"
	"testing"

	"github.com/rewdy/genifer/internal/provider"
)

func idList(models []provider.Model) []string {
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	return ids
}

func TestFavoritesFirst(t *testing.T) {
	models := []provider.Model{
		{ID: "a/one"}, {ID: "b/two"}, {ID: "c/three"}, {ID: "d/four"},
	}
	cases := []struct {
		name  string
		favs  map[string]struct{}
		input []provider.Model
		want  []string
	}{
		{"none", nil, models, []string{"a/one", "b/two", "c/three", "d/four"}},
		{"empty set", map[string]struct{}{}, models, []string{"a/one", "b/two", "c/three", "d/four"}},
		{"one favorite", map[string]struct{}{"c/three": {}}, models, []string{"c/three", "a/one", "b/two", "d/four"}},
		{"several favorites keep provider order", map[string]struct{}{"d/four": {}, "b/two": {}}, models, []string{"b/two", "d/four", "a/one", "c/three"}},
		{"all favorites", map[string]struct{}{"a/one": {}, "b/two": {}, "c/three": {}, "d/four": {}}, models, []string{"a/one", "b/two", "c/three", "d/four"}},
		{"stale ids ignored", map[string]struct{}{"x/gone": {}, "b/two": {}}, models, []string{"b/two", "a/one", "c/three", "d/four"}},
		{"empty input", map[string]struct{}{"a/one": {}}, nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := favoritesFirst(tc.input, tc.favs)
			if len(got) != len(tc.want) {
				t.Fatalf("favoritesFirst = %v, want %v", idList(got), tc.want)
			}
			for i, id := range tc.want {
				if got[i].ID != id {
					t.Fatalf("favoritesFirst = %v, want %v", idList(got), tc.want)
				}
			}
		})
	}
}

func TestFavoritesFirstDoesNotMutateInput(t *testing.T) {
	models := []provider.Model{{ID: "a/one"}, {ID: "b/two"}}
	before := idList(models)
	_ = favoritesFirst(models, map[string]struct{}{"b/two": {}})
	if got := idList(models); !reflect.DeepEqual(got, before) {
		t.Errorf("input mutated: %v, want %v", got, before)
	}
}
